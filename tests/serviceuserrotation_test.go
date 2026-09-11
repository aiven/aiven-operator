// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

//go:build postgresql

package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
	"github.com/aiven/aiven-operator/controllers"
)

const (
	rotationPublishedAtKey     = "aiven-rotation-published-at"
	rotationPendingUsernameKey = "aiven-rotation-desired-username"
	rotationPendingPasswordKey = "aiven-rotation-desired-password"
	rotationWait               = 3 * time.Minute
	rotationPoll               = 2 * time.Second
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
		if previous != first {
			s.waitSQL(first, "authenticated")
		}
		previous = current
	}

	current := s.rotate(previous, s.rotation.Spec.Usernames[0])
	s.waitSQL(current, "authenticated")
	s.waitSQL(previous, "authenticated")
	require.False(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], current.secret.Data[s.prefix+"PASSWORD"]),
		"reusing a username must give it a new password")
	s.waitSQL(first, "password rejected")

	s.delete()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		for _, username := range s.rotation.Spec.Usernames {
			_, err := avnGen.ServiceUserGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
			assert.True(c, avngen.IsNotFound(err), "user %q must be deleted while the service still exists", username)
		}
	}, rotationWait, rotationPoll)
}

func TestServiceUserRotationPgRecovery(t *testing.T) {
	t.Parallel()
	s := newPgRotationTest(t)

	// Start with one existing pool member; the controller creates the others.
	_, err := avnGen.ServiceUserCreate(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, &service.ServiceUserCreateIn{
		Username: s.rotation.Spec.Usernames[0],
	})
	require.NoError(t, err)
	first := s.create()
	s.waitSQL(first, "authenticated")

	username, password := s.rotation.Spec.Usernames[1], rand.Text()
	_, err = avnGen.ServiceUserCredentialsModify(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, &service.ServiceUserCredentialsModifyIn{
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

	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = time.Minute
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
		cr.Spec.RotationInterval.Duration = time.Hour
	})
	pending := &corev1.Secret{}
	require.NoError(t, directClient.Get(s.ctx, key, pending))
	require.Equal(t, first.secret.Data[s.prefix+"USERNAME"], pending.Data[s.prefix+"USERNAME"])
	require.True(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], pending.Data[s.prefix+"PASSWORD"]),
		"the previous password must remain published until the commit")
	require.Equal(t, first.secret.Data[rotationPublishedAtKey], pending.Data[rotationPublishedAtKey])
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
			secret.Data[rotationPublishedAtKey] = []byte(time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano))
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
			secret.Data[rotationPublishedAtKey] = []byte(time.Now().Add(-2 * time.Hour).Format(time.RFC3339Nano))
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

func TestServiceUserRotationPgOrphan(t *testing.T) {
	t.Parallel()
	s := newPgRotationTest(t)
	published := s.create()
	s.waitSQL(published, "authenticated")
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Annotations = map[string]string{"controllers.aiven.io/deletion-policy": "Orphan"}
	})

	s.delete()
	for _, username := range s.rotation.Spec.Usernames {
		user, err := avnGen.ServiceUserGet(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
		require.NoError(t, err)
		require.Equal(t, username, user.Username)
	}
	s.waitSQL(published, "authenticated")
}

func TestServiceUserRotationKafka(t *testing.T) {
	t.Parallel()
	acquireCtx, cancelAcquire := testCtx()
	kafka, release, err := sharedResources.AcquireKafka(acquireCtx)
	cancelAcquire()
	require.NoError(t, err)
	t.Cleanup(release)
	enableRotationKafkaSASL(t, kafka)
	s := newRotationTest(t, kafka.Name)
	first := s.create()
	waitRotationKafka(t, s, first)
	waitRotationKafkaSASL(t, s, first, "authenticated")
	previous := first

	for _, username := range s.rotation.Spec.Usernames[1:] {
		current := s.rotate(previous, username)
		waitRotationKafka(t, s, current)
		waitRotationKafka(t, s, previous)
		waitRotationKafkaSASL(t, s, current, "authenticated")
		waitRotationKafkaSASL(t, s, previous, "authenticated")
		if previous != first {
			waitRotationKafkaSASL(t, s, first, "authenticated")
		}
		previous = current
	}

	current := s.rotate(previous, s.rotation.Spec.Usernames[0])
	waitRotationKafka(t, s, current)
	waitRotationKafka(t, s, previous)
	waitRotationKafkaSASL(t, s, current, "authenticated")
	waitRotationKafkaSASL(t, s, previous, "authenticated")
	require.False(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], current.secret.Data[s.prefix+"PASSWORD"]),
		"reusing a Kafka username must give it a new password")
	waitRotationKafkaSASL(t, s, first, "password rejected")
}

func TestServiceUserRotationValidation(t *testing.T) {
	t.Parallel()
	ctx, cancel := testCtx()
	defer cancel()
	rotation := rotationFromExample(t, randName("validation-service"))

	for _, tc := range []struct {
		name    string
		edit    func(*v1alpha1.ServiceUserRotation)
		message string
	}{
		{"empty pool", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames = nil }, "spec.usernames"},
		{"one user", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames = cr.Spec.Usernames[:1] }, "spec.usernames"},
		{"duplicate users", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames[1] = cr.Spec.Usernames[0] }, "usernames must be unique"},
		{"built-in user", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames[0] = "avnadmin" }, "avnadmin cannot be managed"},
		{"empty username", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames[0] = "" }, "spec.usernames"},
		{"short interval", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.RotationInterval.Duration = time.Minute - time.Second }, "rotationInterval must be at least 1m"},
		{"empty Secret name", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ConnInfoSecretTarget.Name = "" }, "connInfoSecretTarget.name must not be empty"},
	} {
		t.Run("Rejects "+tc.name, func(t *testing.T) {
			cr := rotation.DeepCopy()
			tc.edit(cr)
			err := k8sClient.Create(ctx, cr, client.DryRunAll)
			require.True(t, apierrors.IsInvalid(err), "expected schema validation error, got %v", err)
			require.ErrorContains(t, err, tc.message)
		})
	}

	// Transition rules need a persisted object. A nonexistent service and
	// Orphan cleanup let these checks run without provisioning Aiven resources.
	rotation.Annotations = map[string]string{"controllers.aiven.io/deletion-policy": "Orphan"}
	rotation.Spec.ConnInfoSecretTarget.Prefix = ""
	session := NewSession(ctx, k8sClient)
	defer session.Destroy(t)
	require.NoError(t, session.ApplyObjects(rotation))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NoError(c, k8sClient.Get(ctx, client.ObjectKeyFromObject(rotation), &v1alpha1.ServiceUserRotation{}))
	}, rotationWait, rotationPoll)

	t.Run("Rejects changes to immutable fields", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			edit    func(*v1alpha1.ServiceUserRotation)
			message string
		}{
			{"renamed user", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Usernames[0] = randName("another-user") }, "usernames is immutable"},
			{"reordered pool", func(cr *v1alpha1.ServiceUserRotation) { slices.Reverse(cr.Spec.Usernames) }, "usernames is immutable"},
			{"Secret name", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ConnInfoSecretTarget.Name = randName("another-secret") }, "connInfoSecretTarget.name is immutable"},
			{"Secret prefix", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ConnInfoSecretTarget.Prefix = "OTHER_" }, "connInfoSecretTarget.prefix is immutable"},
			{"project", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.Project = "another-project" }, "Value is immutable"},
			{"service", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ServiceName = "another-service" }, "Value is immutable"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					cr := &v1alpha1.ServiceUserRotation{}
					if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(rotation), cr); err != nil {
						return err
					}
					tc.edit(cr)
					return k8sClient.Update(ctx, cr, client.DryRunAll)
				})
				require.True(t, apierrors.IsInvalid(err), "expected schema validation error, got %v", err)
				require.ErrorContains(t, err, tc.message)
			})
		}
	})

	t.Run("Accepts valid updates", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			edit func(*v1alpha1.ServiceUserRotation)
		}{
			{"one-minute interval", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.RotationInterval.Duration = time.Minute }},
			{"explicit default prefix", func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ConnInfoSecretTarget.Prefix = "SERVICEUSER_" }},
			{"Secret metadata", func(cr *v1alpha1.ServiceUserRotation) {
				cr.Spec.ConnInfoSecretTarget.Labels = map[string]string{"app": "consumer"}
				cr.Spec.ConnInfoSecretTarget.Annotations = map[string]string{"example.com/owner": "team"}
			}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					cr := &v1alpha1.ServiceUserRotation{}
					if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(rotation), cr); err != nil {
						return err
					}
					tc.edit(cr)
					return k8sClient.Update(ctx, cr, client.DryRunAll)
				})
				require.NoError(t, err)
			})
		}
	})
}

type rotationTest struct {
	t        *testing.T
	ctx      context.Context
	session  Session
	rotation *v1alpha1.ServiceUserRotation
	prefix   string
}

type rotationPublication struct {
	secret      *corev1.Secret
	publishedAt time.Time
}

func rotationFromExample(t *testing.T, serviceName string) *v1alpha1.ServiceUserRotation {
	t.Helper()
	src, err := loadExampleYaml("serviceuserrotation.yaml", map[string]string{
		"metadata.name":                  randName("rotation"),
		"spec.project":                   cfg.Project,
		"spec.serviceName":               serviceName,
		"spec.authSecretRef.name":        secretRefName,
		"spec.authSecretRef.key":         secretRefKey,
		"spec.connInfoSecretTarget.name": randName("rotation-secret"),
	})
	require.NoError(t, err)
	cr := &v1alpha1.ServiceUserRotation{}
	require.NoError(t, yaml.UnmarshalStrict([]byte(src), cr))
	require.Equal(t, "aiven.io/v1alpha1", cr.APIVersion)
	require.Equal(t, "ServiceUserRotation", cr.Kind)
	require.GreaterOrEqual(t, len(cr.Spec.Usernames), 2)
	cr.Namespace = defaultNamespace
	return cr
}

func newRotationTest(t *testing.T, serviceName string) *rotationTest {
	t.Helper()
	ctx, cancel := testCtx()
	t.Cleanup(cancel)
	cr := rotationFromExample(t, serviceName)
	for i := range cr.Spec.Usernames {
		cr.Spec.Usernames[i] = randName(fmt.Sprintf("rotation-%d", i))
	}
	cr.Spec.RotationInterval.Duration = time.Hour
	s := &rotationTest{
		t: t, ctx: ctx, session: NewSession(ctx, k8sClient), rotation: cr,
		prefix: cr.Spec.ConnInfoSecretTarget.Prefix,
	}
	if s.prefix == "" {
		s.prefix = "SERVICEUSER_"
	}
	t.Cleanup(s.cleanup)
	return s
}

func (s *rotationTest) rotate(previous *rotationPublication, username string) *rotationPublication {
	s.t.Helper()
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = time.Minute
	})
	current := s.waitPublication(username, previous.publishedAt)
	require.False(s.t, current.publishedAt.Before(previous.publishedAt.Add(time.Minute)),
		"rotation must wait until the published interval has elapsed")
	require.Equal(s.t, previous.secret.UID, current.secret.UID)

	// Keep subsequent rotations outside the connection retry window.
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = time.Hour
	})
	held := s.waitPublication(username, time.Time{})
	require.True(s.t, current.publishedAt.Equal(held.publishedAt), "changing the interval must preserve the publication")
	require.True(s.t, bytes.Equal(current.secret.Data[s.prefix+"PASSWORD"], held.secret.Data[s.prefix+"PASSWORD"]),
		"changing the interval must preserve the password")
	return current
}

func (s *rotationTest) create() *rotationPublication {
	s.t.Helper()
	require.NoError(s.t, s.session.ApplyObjects(s.rotation))
	return s.waitPublication(s.rotation.Spec.Usernames[0], time.Time{})
}

func (s *rotationTest) update(edit func(*v1alpha1.ServiceUserRotation)) {
	s.t.Helper()
	require.NoError(s.t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr := &v1alpha1.ServiceUserRotation{}
		if err := k8sClient.Get(s.ctx, client.ObjectKeyFromObject(s.rotation), cr); err != nil {
			return err
		}
		edit(cr)
		if err := k8sClient.Update(s.ctx, cr); err != nil {
			return err
		}
		*s.rotation = *cr
		return nil
	}))
}

func (s *rotationTest) updateSecret(edit func(*corev1.Secret)) {
	s.t.Helper()
	require.NoError(s.t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret := &corev1.Secret{}
		key := client.ObjectKey{Namespace: s.rotation.Namespace, Name: s.rotation.Spec.ConnInfoSecretTarget.Name}
		if err := k8sClient.Get(s.ctx, key, secret); err != nil {
			return err
		}
		edit(secret)
		return k8sClient.Update(s.ctx, secret)
	}))
}

func (s *rotationTest) refresh(previous *rotationPublication) *rotationPublication {
	s.t.Helper()
	// The new label proves that the controller processed this change, even
	// when the published credentials and status have nothing to update.
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		if cr.Spec.ConnInfoSecretTarget.Labels == nil {
			cr.Spec.ConnInfoSecretTarget.Labels = make(map[string]string)
		}
		cr.Spec.ConnInfoSecretTarget.Labels["tests.aiven.io/refresh"] = rand.Text()
	})
	current := s.waitPublication(string(previous.secret.Data[s.prefix+"USERNAME"]), time.Time{})
	require.Equal(s.t, previous.secret.UID, current.secret.UID)
	require.True(s.t, previous.publishedAt.Equal(current.publishedAt), "refresh must preserve the publication time")
	require.True(s.t, bytes.Equal(previous.secret.Data[s.prefix+"PASSWORD"], current.secret.Data[s.prefix+"PASSWORD"]),
		"refresh must preserve the published password")
	return current
}

func (s *rotationTest) waitPublication(username string, after time.Time) *rotationPublication {
	s.t.Helper()
	var publication *rotationPublication
	require.EventuallyWithT(s.t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		secret := &corev1.Secret{}
		key := client.ObjectKey{Namespace: s.rotation.Namespace, Name: s.rotation.Spec.ConnInfoSecretTarget.Name}
		require.NoError(c, k8sClient.Get(ctx, key, secret))
		require.Equal(c, username, string(secret.Data[s.prefix+"USERNAME"]))
		for _, suffix := range []string{"HOST", "PORT", "USERNAME", "PASSWORD", "CA_CERT"} {
			require.NotEmpty(c, secret.Data[s.prefix+suffix], "missing connection field %s", suffix)
		}
		_, pendingUser := secret.Data[rotationPendingUsernameKey]
		_, pendingPassword := secret.Data[rotationPendingPasswordKey]
		require.False(c, pendingUser, "publication must clear the pending username")
		require.False(c, pendingPassword, "publication must clear the pending password")
		publishedAt, err := time.Parse(time.RFC3339Nano, string(secret.Data[rotationPublishedAtKey]))
		require.NoError(c, err)
		require.True(c, publishedAt.After(after), "waiting for a publication after %s", after)
		require.True(c, metav1.IsControlledBy(secret, s.rotation))
		require.Equal(c, corev1.SecretTypeOpaque, secret.Type)
		require.Equal(c, s.rotation.Spec.ConnInfoSecretTarget.Labels, secret.Labels)
		require.Equal(c, s.rotation.Spec.ConnInfoSecretTarget.Annotations, secret.Annotations)

		cr := &v1alpha1.ServiceUserRotation{}
		require.NoError(c, k8sClient.Get(ctx, client.ObjectKeyFromObject(s.rotation), cr))
		require.GreaterOrEqual(c, cr.Generation, s.rotation.Generation)
		require.Equal(c, s.rotation.Spec.RotationInterval, cr.Spec.RotationInterval)
		require.True(c, meta.FindStatusCondition(cr.Status.Conditions, controllers.ConditionTypeError) == nil,
			"the controller must clear its Error condition")
		require.Equal(c, username, cr.Status.ActiveUsername)
		require.True(c, publishedAt.Truncate(time.Second).Equal(cr.Status.LastRotationAt.Time), "status must report this publication")
		next := publishedAt.Add(cr.Spec.RotationInterval.Duration).Truncate(time.Second)
		require.True(c, next.Equal(cr.Status.NextRotationAt.Time), "status must use the current rotation interval")
		publication = &rotationPublication{secret: secret, publishedAt: publishedAt}
	}, rotationWait, rotationPoll, "waiting for published credentials for %q", username)
	return publication
}

func (s *rotationTest) delete() {
	s.t.Helper()
	require.NoError(s.t, k8sClient.Delete(s.ctx, s.rotation))
	require.EventuallyWithT(s.t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(s.rotation), &v1alpha1.ServiceUserRotation{})
		assert.True(c, apierrors.IsNotFound(err), "rotation finalization must complete")
	}, rotationWait, rotationPoll)
}

func (s *rotationTest) cleanup() {
	s.t.Helper()
	s.session.Destroy(s.t)
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(s.rotation), &v1alpha1.ServiceUserRotation{})
	if !apierrors.IsNotFound(err) {
		s.t.Error("rotation still exists or cannot be checked; leave its users to its finalizer")
		return
	}

	for _, username := range s.rotation.Spec.Usernames {
		err := avnGen.ServiceUserDelete(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
		assert.True(s.t, err == nil || avngen.IsNotFound(err), "cleaning up user %q must succeed", username)
	}
	assert.EventuallyWithT(s.t, func(c *assert.CollectT) {
		attemptCtx, cancelAttempt := context.WithTimeout(ctx, 10*time.Second)
		defer cancelAttempt()
		for _, username := range s.rotation.Spec.Usernames {
			_, err := avnGen.ServiceUserGet(attemptCtx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username)
			assert.True(c, avngen.IsNotFound(err), "user %q must be absent after cleanup", username)
		}
	}, rotationWait, rotationPoll)

	// envtest has no garbage collector for the owned connection Secret.
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: s.rotation.Spec.ConnInfoSecretTarget.Name, Namespace: s.rotation.Namespace,
	}}
	assert.NoError(s.t, client.IgnoreNotFound(k8sClient.Delete(ctx, secret)))
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

func enableRotationKafkaSASL(t *testing.T, kafka *v1alpha1.Kafka) {
	t.Helper()
	ctx, cancel := testCtx()
	defer cancel()
	current := &v1alpha1.Kafka{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), current))
	orig := current.Spec.UserConfig.DeepCopy()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()
		assert.NoError(t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
			cr := &v1alpha1.Kafka{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), cr); err != nil {
				return err
			}
			cr.Spec.UserConfig = orig
			return k8sClient.Update(ctx, cr)
		}))
	})
	require.NoError(t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr := &v1alpha1.Kafka{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), cr); err != nil {
			return err
		}
		if cr.Spec.UserConfig == nil {
			cr.Spec.UserConfig = &kafkauserconfig.KafkaUserConfig{}
		}
		cr.Spec.UserConfig.KafkaAuthenticationMethods = &kafkauserconfig.KafkaAuthenticationMethods{
			Certificate: new(true), Sasl: new(true),
		}
		if cr.Spec.UserConfig.KafkaSaslMechanisms == nil {
			cr.Spec.UserConfig.KafkaSaslMechanisms = &kafkauserconfig.KafkaSaslMechanisms{}
		}
		cr.Spec.UserConfig.KafkaSaslMechanisms.Plain = new(true)
		return k8sClient.Update(ctx, cr)
	}))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		attemptCtx, cancelAttempt := context.WithTimeout(ctx, 10*time.Second)
		defer cancelAttempt()
		svc, err := avnGen.ServiceGet(attemptCtx, kafka.Spec.Project, kafka.Name)
		require.NoError(c, err)
		assert.Contains(c, serviceRunningStatesAiven, svc.State)
		methods, ok := svc.UserConfig["kafka_authentication_methods"].(map[string]any)
		require.True(c, ok)
		assert.Equal(c, true, methods["sasl"])
		assert.Equal(c, true, methods["certificate"])
		mechanisms, ok := svc.UserConfig["kafka_sasl_mechanisms"].(map[string]any)
		require.True(c, ok)
		assert.Equal(c, true, mechanisms["plain"])
		assert.True(c, slices.ContainsFunc(svc.Components, func(component service.ComponentOut) bool {
			return component.Component == "kafka" && component.KafkaAuthenticationMethod == service.KafkaAuthenticationMethodTypeSasl
		}), "Kafka must expose a SASL endpoint")
	}, waitRunningTimeout, rotationPoll)
}

func waitRotationKafkaSASL(t *testing.T, s *rotationTest, publication *rotationPublication, expected string) {
	t.Helper()
	data := publication.secret.Data
	for _, suffix := range []string{"SASL_HOST", "SASL_PORT", "USERNAME", "PASSWORD", "CA_CERT"} {
		require.NotEmpty(t, data[s.prefix+suffix], "missing connection field %s", suffix)
	}
	ca := x509.NewCertPool()
	require.True(t, ca.AppendCertsFromPEM(data[s.prefix+"CA_CERT"]), "the published CA must be valid PEM")
	address := net.JoinHostPort(string(data[s.prefix+"SASL_HOST"]), string(data[s.prefix+"SASL_PORT"]))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		dialer := tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: ca}}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		require.NoError(c, err)
		defer conn.Close()
		deadline, ok := ctx.Deadline()
		require.True(c, ok)
		require.NoError(c, conn.SetDeadline(deadline))

		// SaslHandshake v1 selects PLAIN; SaslAuthenticate v0 checks the password.
		// https://kafka.apache.org/39/design/protocol/#The_Messages_SaslAuthenticate
		require.Zero(c, rotationKafkaRequest(c, conn, 17, 1, append([]byte{0, 5}, "PLAIN"...)))
		auth := []byte("\x00" + string(data[s.prefix+"USERNAME"]) + "\x00" + string(data[s.prefix+"PASSWORD"]))
		payload := append(binary.BigEndian.AppendUint32(nil, uint32(len(auth))), auth...)
		code := rotationKafkaRequest(c, conn, 36, 0, payload)
		outcome := fmt.Sprintf("Kafka error %d", code)
		switch code {
		case 0:
			outcome = "authenticated"
		case 58: // SASL_AUTHENTICATION_FAILED
			outcome = "password rejected"
		}
		assert.Equal(c, expected, outcome)
	}, time.Minute, rotationPoll, "the published Kafka password must determine SASL authentication")
}

// rotationKafkaRequest reads the error code shared by the non-flexible SASL responses.
// Discard the remaining response without logging it: authentication errors may contain credentials.
func rotationKafkaRequest(c *assert.CollectT, conn net.Conn, apiKey, version int16, payload []byte) int16 {
	request := struct {
		Length         int32
		APIKey         int16
		Version        int16
		CorrelationID  int32
		ClientIDLength int16
	}{APIKey: apiKey, Version: version, CorrelationID: int32(apiKey)}
	request.Length = int32(binary.Size(request) - 4 + len(payload))
	var frame bytes.Buffer
	require.NoError(c, binary.Write(&frame, binary.BigEndian, request))
	frame.Write(payload)
	_, err := frame.WriteTo(conn)
	require.NoError(c, err)
	var response struct {
		Length        int32
		CorrelationID int32
		ErrorCode     int16
	}
	require.NoError(c, binary.Read(conn, binary.BigEndian, &response))
	require.Equal(c, request.CorrelationID, response.CorrelationID)
	require.GreaterOrEqual(c, response.Length, int32(6))
	_, err = io.CopyN(io.Discard, conn, int64(response.Length)-6)
	require.NoError(c, err)
	return response.ErrorCode
}

func waitRotationKafka(t *testing.T, s *rotationTest, publication *rotationPublication) {
	t.Helper()
	data := publication.secret.Data
	cert, err := tls.X509KeyPair(data[s.prefix+"ACCESS_CERT"], data[s.prefix+"ACCESS_KEY"])
	require.NoError(t, err)
	ca := x509.NewCertPool()
	require.True(t, ca.AppendCertsFromPEM(data[s.prefix+"CA_CERT"]), "the published CA must be valid PEM")
	address := net.JoinHostPort(string(data[s.prefix+"HOST"]), string(data[s.prefix+"PORT"]))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		user, err := avnGen.ServiceUserGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
			string(data[s.prefix+"USERNAME"]), service.ServiceUserGetIncludeSecrets(true))
		require.NoError(c, err)
		require.NotNil(c, user.AccessCert)
		require.True(c, bytes.Equal([]byte(*user.AccessCert), data[s.prefix+"ACCESS_CERT"]),
			"the published certificate must belong to the published user")

		requestedCertificate := false
		dialer := tls.Dialer{Config: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    ca,
			GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
				requestedCertificate = true
				return &cert, nil
			},
		}}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		require.NoError(c, err)
		defer conn.Close()
		// ApiVersions is also available before SASL authentication. Require mTLS.
		require.True(c, requestedCertificate, "the broker must request the published client certificate")
		deadline, ok := ctx.Deadline()
		require.True(c, ok)
		require.NoError(c, conn.SetDeadline(deadline))

		// ApiVersions v0 needs no topic ACLs. A protocol response confirms that
		// the broker accepted the TLS connection, including TLS 1.3's final flight.
		// https://kafka.apache.org/34/design/protocol/#The_Messages_ApiVersions
		request := struct {
			Length         int32
			APIKey         int16
			Version        int16
			CorrelationID  int32
			ClientIDLength int16
		}{APIKey: 18, CorrelationID: 1}
		request.Length = int32(binary.Size(request) - 4)
		require.NoError(c, binary.Write(conn, binary.BigEndian, request))
		var response struct {
			Length        int32
			CorrelationID int32
			ErrorCode     int16
			APICount      int32
		}
		require.NoError(c, binary.Read(conn, binary.BigEndian, &response))
		assert.Equal(c, request.CorrelationID, response.CorrelationID)
		assert.Zero(c, response.ErrorCode)
		assert.Positive(c, response.APICount)
		assert.Equal(c, int32(10)+6*response.APICount, response.Length)
	}, time.Minute, rotationPoll, "the published Kafka certificate and key must authenticate a fresh connection")
}
