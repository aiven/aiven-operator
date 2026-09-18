// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

//go:build postgresql

package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestServiceUserRotationPg(t *testing.T) {
	t.Parallel()
	s := newPgRotationTest(t)
	first := s.create()
	s.waitSQL(first, "authenticated")
	previous := first

	for _, username := range s.rotation.Spec.Usernames[1:] {
		current := s.rotate(previous, username)

		s.waitSQL(current, "authenticated")
		s.waitSQL(previous, "authenticated")
		previous = current
	}

	current := s.rotate(previous, s.rotation.Spec.Usernames[0])
	s.waitSQL(current, "authenticated")
	s.waitSQL(previous, "authenticated")
	require.False(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], current.secret.Data[s.prefix+"PASSWORD"]),
		"reusing a username must give it a new password")
	s.waitSQL(first, "password rejected")

	s.delete()
	for _, username := range s.rotation.Spec.Usernames {
		user, err := avnGen.ServiceUserGet(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
		require.NoError(t, err)
		require.Equal(t, username, user.Username)
	}
	s.waitSQL(current, "authenticated")
	s.waitSQL(previous, "authenticated")
}

func TestServiceUserRotationPgRecovery(t *testing.T) {
	t.Parallel()
	s := newPgRotationTest(t)
	first := s.create()
	s.waitSQL(first, "authenticated")

	username, password := s.rotation.Spec.Usernames[1], rand.Text()
	_, err := avnGen.ServiceUserCredentialsModify(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, &service.ServiceUserCredentialsModifyIn{
		NewPassword: &password,
		Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
	})
	require.True(t, err == nil, "preparing the candidate password in Aiven must succeed")

	// Seed the durable state left when the password request succeeded but
	// publication did not. Replaying it must use the same password.
	s.updateSecret(func(secret *corev1.Secret) {
		secret.Data[rotationPendingUsernameKey] = []byte(username)
		secret.Data[rotationPendingPasswordKey] = []byte(password)
		secret.Data["UNRELATED"] = []byte("preserve")
	})

	second := s.waitPublication(username, first.publishedAt)
	require.True(t, bytes.Equal([]byte(password), second.secret.Data[s.prefix+"PASSWORD"]),
		"recovery must publish the saved password")
	require.True(t, second.publishedAt.Before(first.publishedAt.Add(s.rotation.Spec.RotationInterval.Duration)),
		"a saved candidate must bypass the future rotation deadline")
	require.Equal(t, first.secret.UID, second.secret.UID)
	require.Equal(t, []byte("preserve"), second.secret.Data["UNRELATED"])
	s.waitSQL(second, "authenticated")
	s.waitSQL(first, "authenticated")
}

func TestServiceUserRotationPgPublicationConflict(t *testing.T) {
	// This test pauses the rotation controller's worker; keep it outside the
	// parallel tests so their reconciliations can run without waiting on it.
	s := newPgRotationTest(t)
	first := s.create()
	s.waitSQL(first, "authenticated")
	username := s.rotation.Spec.Usernames[1]
	key := client.ObjectKeyFromObject(first.secret)
	directClient, err := client.New(restConfig, client.Options{Scheme: k8sClient.Scheme()})
	require.NoError(t, err)

	gateCtx, cancelGate := context.WithCancel(s.ctx)
	t.Cleanup(func() {
		secretUpdateHooks.Delete(key)
		cancelGate()
	})
	ready := make(chan *corev1.Secret, 1)
	allowWrite := make(chan struct{})
	written := make(chan error, 1)
	allowReconcile := make(chan struct{})
	secretUpdateHooks.Store(key, secretUpdateHook(func(ctx context.Context, secret *corev1.Secret, update func() error) error {
		if string(secret.Data[s.prefix+"USERNAME"]) != username {
			return update()
		}
		if _, pending := secret.Data[rotationPendingUsernameKey]; pending {
			return update()
		}
		secretUpdateHooks.Delete(key)
		ready <- secret.DeepCopy()
		select {
		case <-allowWrite:
		case <-gateCtx.Done():
			return gateCtx.Err()
		case <-ctx.Done():
			return ctx.Err()
		}
		err := update()
		written <- err
		// Hold the error until the test checks persisted state, before the
		// controller can return and retry the reconciliation.
		select {
		case <-allowReconcile:
		case <-gateCtx.Done():
		case <-ctx.Done():
		}
		return err
	}))

	// Move the saved publication back so shortening the interval makes rotation due.
	publishedAt := time.Now().Add(-time.Hour)
	s.updateSecret(func(secret *corev1.Secret) {
		secret.Data[rotationPublishedAtKey] = []byte(publishedAt.Format(time.RFC3339Nano))
	})
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = time.Hour
	})
	var attempted *corev1.Secret
	select {
	case attempted = <-ready:
	case <-time.After(rotationWait):
		t.Fatal("the controller did not reach credential publication")
	case <-s.ctx.Done():
		t.Fatal("test context expired while waiting for credential publication")
	}
	// Keep the next scheduled rotation outside the connection checks. The
	// saved candidate must still be retried after the publication conflict.
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = 2 * time.Hour
	})
	pending := &corev1.Secret{}
	require.NoError(t, directClient.Get(s.ctx, key, pending))
	require.Equal(t, first.secret.Data[s.prefix+"USERNAME"], pending.Data[s.prefix+"USERNAME"])
	require.True(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], pending.Data[s.prefix+"PASSWORD"]),
		"the previous password must remain published until the commit")
	require.Equal(t, []byte(publishedAt.Format(time.RFC3339Nano)), pending.Data[rotationPublishedAtKey])
	require.Equal(t, username, string(pending.Data[rotationPendingUsernameKey]))
	require.True(t, len(pending.Data[rotationPendingPasswordKey]) > 0, "the pending password must be saved")
	require.True(t, bytes.Equal(pending.Data[rotationPendingPasswordKey], attempted.Data[s.prefix+"PASSWORD"]),
		"the publication must use the saved candidate password")
	s.waitSQL(&rotationPublication{secret: attempted}, "authenticated")

	pending.Data["UNRELATED"] = []byte("concurrent-write")
	require.NoError(t, directClient.Update(s.ctx, pending))
	require.NotEqual(t, attempted.ResourceVersion, pending.ResourceVersion)
	close(allowWrite)
	select {
	case err = <-written:
		require.True(t, apierrors.IsConflict(err), "the API server must reject the stale publication")
	case <-time.After(rotationWait):
		t.Fatal("the publication request did not complete")
	case <-s.ctx.Done():
		t.Fatal("test context expired while waiting for the publication result")
	}
	afterConflict := &corev1.Secret{}
	require.NoError(t, directClient.Get(s.ctx, key, afterConflict))
	require.Equal(t, pending.ResourceVersion, afterConflict.ResourceVersion)
	require.True(t, maps.EqualFunc(pending.Data, afterConflict.Data, bytes.Equal),
		"a rejected publication must preserve the active credentials, timestamp, pending pair, and competing write")
	close(allowReconcile)

	current := s.waitPublication(username, first.publishedAt)
	require.Equal(t, first.secret.UID, current.secret.UID)
	require.True(t, bytes.Equal(pending.Data[rotationPendingPasswordKey], current.secret.Data[s.prefix+"PASSWORD"]),
		"retrying publication must reuse the saved candidate password")
	require.Equal(t, []byte("concurrent-write"), current.secret.Data["UNRELATED"])
	s.waitSQL(current, "authenticated")
	s.waitSQL(first, "authenticated")
}

func TestServiceUserRotationPgInvalidCandidate(t *testing.T) {
	t.Parallel()

	t.Run("missing password", func(t *testing.T) {
		s := newPgRotationTest(t)
		first := s.create()
		s.waitSQL(first, "authenticated")
		username := s.rotation.Spec.Usernames[1]
		setCandidate := func(secret *corev1.Secret) {
			secret.Data[rotationPendingUsernameKey] = []byte(username)
			delete(secret.Data, rotationPendingPasswordKey)
		}
		s.updateSecret(setCandidate)
		s.refresh(first)
		s.waitSQL(first, "authenticated")

		// Save the invalid pair and an overdue publication together so the
		// controller must regenerate the candidate in the same reconciliation.
		s.updateSecret(func(secret *corev1.Secret) {
			setCandidate(secret)
			secret.Data[rotationPublishedAtKey] = []byte(time.Now().Add(-2 * s.rotation.Spec.RotationInterval.Duration).Format(time.RFC3339Nano))
		})
		next := s.waitPublication(username, first.publishedAt)
		require.Equal(t, first.secret.UID, next.secret.UID)
		s.waitSQL(next, "authenticated")
		s.waitSQL(first, "authenticated")
	})

	t.Run("user outside the pool", func(t *testing.T) {
		s := newPgRotationTest(t)
		first := s.create()
		s.waitSQL(first, "authenticated")
		username, password := randName("outside-pool"), rand.Text()
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
			defer cancel()
			err := avnGen.ServiceUserDelete(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
			assert.True(t, err == nil || avngen.IsNotFound(err), "cleaning up the user outside the pool must succeed")
		})
		user, err := avnGen.ServiceUserCreate(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
			&service.ServiceUserCreateIn{Username: username})
		require.NoError(t, err)
		require.NotEmpty(t, user.Password)
		unmanaged := &rotationPublication{secret: first.secret.DeepCopy(), publishedAt: first.publishedAt}
		unmanaged.secret.Data[s.prefix+"USERNAME"] = []byte(username)
		unmanaged.secret.Data[s.prefix+"PASSWORD"] = []byte(user.Password)
		s.waitSQL(unmanaged, "authenticated")

		setCandidate := func(secret *corev1.Secret) {
			secret.Data[rotationPendingUsernameKey] = []byte(username)
			secret.Data[rotationPendingPasswordKey] = []byte(password)
		}
		s.updateSecret(setCandidate)
		s.refresh(first)
		s.waitSQL(first, "authenticated")
		s.waitSQL(unmanaged, "authenticated")

		s.updateSecret(func(secret *corev1.Secret) {
			setCandidate(secret)
			secret.Data[rotationPublishedAtKey] = []byte(time.Now().Add(-2 * s.rotation.Spec.RotationInterval.Duration).Format(time.RFC3339Nano))
		})
		next := s.waitPublication(s.rotation.Spec.Usernames[1], first.publishedAt)
		require.Equal(t, first.secret.UID, next.secret.UID)
		require.False(t, bytes.Equal([]byte(password), next.secret.Data[s.prefix+"PASSWORD"]),
			"an invalid candidate must be replaced with a newly generated password")
		s.waitSQL(next, "authenticated")
		s.waitSQL(first, "authenticated")
		s.waitSQL(unmanaged, "authenticated")
	})
}

func TestServiceUserRotationPgExternalChanges(t *testing.T) {
	t.Parallel()

	t.Run("password change", func(t *testing.T) {
		s := newPgRotationTest(t)
		first := s.create()
		s.waitSQL(first, "authenticated")
		username, password := s.rotation.Spec.Usernames[0], rand.Text()
		_, err := avnGen.ServiceUserCredentialsModify(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username,
			&service.ServiceUserCredentialsModifyIn{
				NewPassword: &password,
				Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
			})
		require.True(t, err == nil, "changing the active password outside rotation must succeed")
		changed := &rotationPublication{secret: first.secret.DeepCopy(), publishedAt: first.publishedAt}
		changed.secret.Data[s.prefix+"PASSWORD"] = []byte(password)

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			defer cancel()
			svc, err := avnGen.ServiceGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
				service.ServiceGetIncludeSecrets(true))
			require.NoError(c, err)
			idx := slices.IndexFunc(svc.Users, func(user service.UserOut) bool { return user.Username == username })
			require.GreaterOrEqual(c, idx, 0, "the active user must remain present")
			require.False(c, bytes.Equal([]byte(svc.Users[idx].Password), first.secret.Data[s.prefix+"PASSWORD"]),
				"the service observation must no longer report the published password")
		}, rotationWait, rotationPoll)
		s.waitSQL(changed, "authenticated")
		refreshed := s.refresh(first)
		s.waitSQL(refreshed, "password rejected")
		s.waitSQL(changed, "authenticated")

		next := s.rotate(refreshed, s.rotation.Spec.Usernames[1])
		s.waitSQL(next, "authenticated")
		s.waitSQL(changed, "authenticated")
	})

	t.Run("user recreation", func(t *testing.T) {
		s := newPgRotationTest(t)
		first := s.create()
		s.waitSQL(first, "authenticated")
		username := s.rotation.Spec.Usernames[0]
		require.NoError(t, avnGen.ServiceUserDelete(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username))
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			defer cancel()
			_, err := avnGen.ServiceUserGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
			require.True(c, avngen.IsNotFound(err), "the deleted user must be absent")
			cr := &v1alpha1.ServiceUserRotation{}
			require.NoError(c, k8sClient.Get(ctx, client.ObjectKeyFromObject(s.rotation), cr))
			condition := meta.FindStatusCondition(cr.Status.Conditions, controllers.ConditionTypeError)
			require.NotNil(c, condition, "rotation must report the missing active user")
			require.Contains(c, condition.Message, fmt.Sprintf("active user %q not found", username))
		}, rotationWait, rotationPoll)

		// The external owner restores the user. Rotation must leave its new password alone.
		_, err := avnGen.ServiceUserCreate(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
			&service.ServiceUserCreateIn{Username: username})
		require.NoError(t, err)
		changed := &rotationPublication{secret: first.secret.DeepCopy(), publishedAt: first.publishedAt}

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
			defer cancel()
			svc, err := avnGen.ServiceGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
				service.ServiceGetIncludeSecrets(true))
			require.NoError(c, err)
			idx := slices.IndexFunc(svc.Users, func(user service.UserOut) bool { return user.Username == username })
			require.GreaterOrEqual(c, idx, 0, "the active user must exist again")
			user := svc.Users[idx]
			require.False(c, bytes.Equal([]byte(user.Password), first.secret.Data[s.prefix+"PASSWORD"]),
				"the service observation must no longer report the published password")
			require.NotEmpty(c, user.Password)
			changed.secret.Data[s.prefix+"PASSWORD"] = []byte(user.Password)
		}, rotationWait, rotationPoll)
		s.waitSQL(changed, "authenticated")
		refreshed := s.refresh(first)
		s.waitSQL(refreshed, "password rejected")
		s.waitSQL(changed, "authenticated")

		next := s.rotate(refreshed, s.rotation.Spec.Usernames[1])
		s.waitSQL(next, "authenticated")
		s.waitSQL(changed, "authenticated")
	})
}

func TestServiceUserRotationPgServiceUsers(t *testing.T) {
	t.Parallel()
	s := newPgRotationTest(t)
	for _, username := range s.rotation.Spec.Usernames {
		src, err := loadExampleYaml("serviceuser-for-rotation.yaml", map[string]string{
			"metadata.name":           username,
			"spec.project":            s.rotation.Spec.Project,
			"spec.serviceName":        s.rotation.Spec.ServiceName,
			"spec.authSecretRef.name": secretRefName,
			"spec.authSecretRef.key":  secretRefKey,
		})
		require.NoError(t, err)
		require.NoError(t, s.session.Apply(src))
		user := &v1alpha1.ServiceUser{}
		require.NoError(t, s.session.GetRunning(user, username))
		require.Nil(t, user.Spec.ConnInfoSecretSource)
		require.NotNil(t, user.Spec.ConnInfoSecretTargetDisabled)
		require.True(t, *user.Spec.ConnInfoSecretTargetDisabled)
		_, err = s.session.GetSecret(username)
		require.True(t, apierrors.IsNotFound(err), "ServiceUser must not publish its own Secret")
	}

	require.NoError(t, s.session.ApplyObjects(s.rotation))
	first := s.waitPublication(s.rotation.Spec.Usernames[0], time.Time{})
	s.waitSQL(first, "authenticated")
	current := s.rotate(first, s.rotation.Spec.Usernames[1])
	s.waitSQL(current, "authenticated")

	// Reconcile both ServiceUsers after rotation and check that they keep the new passwords.
	for _, username := range s.rotation.Spec.Usernames {
		key := client.ObjectKey{Namespace: s.rotation.Namespace, Name: username}
		require.NoError(t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
			user := &v1alpha1.ServiceUser{}
			if err := k8sClient.Get(s.ctx, key, user); err != nil {
				return err
			}
			user.Spec.ConnInfoSecretTarget.Labels = map[string]string{"tests.aiven.io/refresh": rand.Text()}
			return k8sClient.Update(s.ctx, user)
		}))
		require.NoError(t, s.session.GetRunning(&v1alpha1.ServiceUser{}, username))
	}
	s.waitSQL(first, "authenticated")
	s.waitSQL(current, "authenticated")

	s.delete()
	for _, username := range s.rotation.Spec.Usernames {
		require.NoError(t, k8sClient.Get(s.ctx, client.ObjectKey{Namespace: s.rotation.Namespace, Name: username}, &v1alpha1.ServiceUser{}))
		user, err := avnGen.ServiceUserGet(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
		require.NoError(t, err)
		require.Equal(t, username, user.Username)
	}
	s.waitSQL(first, "authenticated")
	s.waitSQL(current, "authenticated")
}

type pgRotationTest struct {
	*rotationTest
	database string
}

func newPgRotationTest(t *testing.T) *pgRotationTest {
	t.Helper()
	acquireCtx, cancelAcquire := testCtx()
	pg, release, err := sharedResources.AcquirePostgreSQL(acquireCtx)
	cancelAcquire()
	require.NoError(t, err)
	t.Cleanup(release)

	s := &pgRotationTest{rotationTest: newRotationTest(t, pg.Name)}
	secret, err := s.session.GetSecret(pg.Name)
	require.NoError(t, err)
	s.database = string(secret.Data["POSTGRESQL_DATABASE"])
	require.NotEmpty(t, s.database)
	return s
}

func (s *pgRotationTest) waitSQL(publication *rotationPublication, expected string) {
	s.t.Helper()
	data := publication.secret.Data
	caPath := filepath.Join(s.t.TempDir(), "ca.pem")
	require.NoError(s.t, os.WriteFile(caPath, data[s.prefix+"CA_CERT"], 0o600))
	username := string(data[s.prefix+"USERNAME"])
	dsn := &url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(string(data[s.prefix+"HOST"]), string(data[s.prefix+"PORT"])),
		User:   url.UserPassword(username, string(data[s.prefix+"PASSWORD"])),
		Path:   "/" + s.database,
	}
	query := dsn.Query()
	query.Set("sslmode", "verify-full")
	query.Set("sslrootcert", caPath)
	query.Set("connect_timeout", "5")
	dsn.RawQuery = query.Encode()

	require.EventuallyWithT(s.t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		// A session opened before a password reset can remain authenticated.
		// Each attempt must establish a fresh connection.
		db, err := sql.Open("postgres", dsn.String())
		require.True(c, err == nil, "opening the PostgreSQL connection must succeed")
		defer db.Close()
		var currentUser string
		err = db.QueryRowContext(ctx, "SELECT current_user").Scan(&currentUser)
		outcome := "connection or query failed"
		var pgErr *pq.Error
		switch {
		case err == nil && currentUser == username:
			outcome = "authenticated"
		case err == nil:
			outcome = "unexpected database user"
		case errors.As(err, &pgErr):
			outcome = "PostgreSQL error " + string(pgErr.Code)
			if pgErr.Code == "28P01" {
				outcome = "password rejected"
			}
		}
		// Raw driver errors may include credentials; report only the outcome.
		assert.Equal(c, expected, outcome)
	}, time.Minute, rotationPoll)
}
