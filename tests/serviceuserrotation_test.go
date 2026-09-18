// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

//go:build suite

package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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
	"github.com/aiven/aiven-operator/controllers"
)

const (
	rotationPublishedAtKey     = "aiven-rotation-published-at"
	rotationPendingUsernameKey = "aiven-rotation-desired-username"
	rotationPendingPasswordKey = "aiven-rotation-desired-password"
	rotationWait               = 3 * time.Minute
	rotationPoll               = 2 * time.Second
)

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
	require.Len(t, cr.Spec.Usernames, 2)
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
	cr.Spec.RotationInterval.Duration = 2 * time.Hour
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
	// Exercise the last minute of an hour-long interval without waiting an hour.
	publishedAt := time.Now().Add(-time.Hour + time.Minute)
	s.updateSecret(func(secret *corev1.Secret) {
		secret.Data[rotationPublishedAtKey] = []byte(publishedAt.Format(time.RFC3339Nano))
	})
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = time.Hour
	})
	current := s.waitPublication(username, previous.publishedAt)
	require.False(s.t, current.publishedAt.Before(publishedAt.Add(time.Hour)),
		"rotation must wait until the published interval has elapsed")
	require.Equal(s.t, previous.secret.UID, current.secret.UID)

	// Keep subsequent rotations outside the connection retry window.
	s.update(func(cr *v1alpha1.ServiceUserRotation) {
		cr.Spec.RotationInterval.Duration = 2 * time.Hour
	})
	held := s.waitPublication(username, time.Time{})
	require.True(s.t, current.publishedAt.Equal(held.publishedAt), "changing the interval must preserve the publication")
	require.True(s.t, bytes.Equal(current.secret.Data[s.prefix+"PASSWORD"], held.secret.Data[s.prefix+"PASSWORD"]),
		"changing the interval must preserve the password")
	return current
}

func (s *rotationTest) create() *rotationPublication {
	s.t.Helper()
	for _, username := range s.rotation.Spec.Usernames {
		_, err := avnGen.ServiceUserCreate(s.ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
			&service.ServiceUserCreateIn{Username: username})
		require.NoError(s.t, err)
	}
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
		s.t.Error("rotation still exists or cannot be checked; stop it before deleting its users")
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
