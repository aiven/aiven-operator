package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/yaml"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceUserRotationReconciler(t *testing.T) {
	t.Parallel()

	newRotationScenario := func(t *testing.T) *rotationScenario {
		t.Helper()

		data, err := os.ReadFile(examplesDirPath + "/serviceuserrotation.yaml")
		require.NoError(t, err)
		cr := &v1alpha1.ServiceUserRotation{}
		require.NoError(t, yaml.UnmarshalStrict(data, cr))
		require.Equal(t, "aiven.io/v1alpha1", cr.APIVersion)
		require.Equal(t, "ServiceUserRotation", cr.Kind)
		require.NotEmpty(t, cr.Name)
		require.NotEmpty(t, cr.Spec.Project)
		require.NotEmpty(t, cr.Spec.ServiceName)
		require.NotEmpty(t, cr.Spec.ConnInfoSecretTarget.Name)
		require.Len(t, cr.Spec.Usernames, 2)
		require.Positive(t, cr.Spec.RotationInterval.Duration)
		require.NotNil(t, cr.Spec.AuthSecretRef)
		// Supply metadata normally assigned by Kubernetes.
		cr.Namespace, cr.UID, cr.Generation = "default", types.UID("rotation-uid"), 1

		auth := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: cr.Spec.AuthSecretRef.Name, Namespace: cr.Namespace},
			Data:       map[string][]byte{cr.Spec.AuthSecretRef.Key: []byte("test-token")},
		}
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))
		s := &rotationScenario{
			rotation: cr,
			auth:     auth,
			client: fake.NewClientBuilder().WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ServiceUserRotation{}).
				WithObjects(cr, auth).Build(),
			api: avngen.NewMockClient(t),
			service: &service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "pg",
				Components:  []service.ComponentOut{{Component: "pg", Host: "db.example.com", Port: 5432}},
			},
			recorder: record.NewFakeRecorder(100),
			// Exercise subsecond precision and a non-UTC offset on every publication.
			now: time.Date(2026, time.August, 17, 12, 0, 0, 123456789, time.FixedZone("publication", 19800)),
		}
		for _, username := range cr.Spec.Usernames {
			s.service.Users = append(s.service.Users, service.UserOut{Username: username, Password: "api-password"})
		}
		s.r = newServiceUserRotationReconciler(Controller{
			Client:       s.client,
			Scheme:       scheme,
			Recorder:     s.recorder,
			PollInterval: 2 * cr.Spec.RotationInterval.Duration,
			newAivenClient: func(token, _, _ string) (avngen.Client, error) {
				require.Equal(t, "test-token", token)
				return s.api, nil
			},
		}).(*ServiceUserRotationReconciler)
		s.r.now = func() time.Time { return s.now }
		s.r.generatePassword = func() string {
			s.generated++
			return fmt.Sprintf("generated-%d", s.generated)
		}
		return s
	}

	t.Run("Publishes initial credentials for an existing user", func(t *testing.T) {
		s := newRotationScenario(t)
		users := slices.Clone(s.service.Users)
		s.expectService()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, users[0].Username, rotationPasswordRequest("generated-1")).
			RunAndReturn(func(_ context.Context, _, _, username string, in *service.ServiceUserCredentialsModifyIn) (*service.ServiceUserCredentialsModifyOut, error) {
				pending := s.secret(t)
				require.Equal(t, map[string][]byte{
					"aiven-rotation-desired-username": []byte(username),
					"aiven-rotation-desired-password": []byte(*in.NewPassword),
				}, pending.Data)
				require.True(t, metav1.IsControlledBy(pending, s.rotation))
				return &service.ServiceUserCredentialsModifyOut{Users: users}, nil
			}).Once()
		s.expectCA()
		publications := 0
		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
				if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.rotation.Spec.ConnInfoSecretTarget.Name {
					publications++
					require.Equal(t, []byte("generated-1"), secret.Data[s.prefix()+"PASSWORD"])
					require.Equal(t, []byte("db.example.com"), secret.Data[s.prefix()+"HOST"])
					require.Equal(t, []byte(s.now.Format(time.RFC3339Nano)), secret.Data["aiven-rotation-published-at"])
					require.NotContains(t, secret.Data, "aiven-rotation-desired-password")
					require.NotContains(t, secret.Data, "aiven-rotation-desired-username")
				}
				return c.Update(ctx, obj, opts...)
			},
		})

		res, err := s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		secret := s.requirePublished(t, users[0].Username, "generated-1", s.now)
		require.Equal(t, 1, publications)
		require.Equal(t, 1, s.generated)
		require.Equal(t, []byte("5432"), secret.Data[s.prefix()+"PORT"])
		require.Equal(t, []byte("project-ca"), secret.Data[s.prefix()+"CA_CERT"])
		require.Empty(t, secret.Data[s.prefix()+"ACCESS_CERT"])
		require.Empty(t, secret.Data[s.prefix()+"ACCESS_KEY"])
		require.Empty(t, s.object(t).Finalizers)
		auth := &corev1.Secret{}
		require.NoError(t, s.client.Get(t.Context(), crclient.ObjectKeyFromObject(s.auth), auth))
		require.Empty(t, auth.Finalizers)
		statusJSON, err := json.Marshal(s.object(t).Status)
		require.NoError(t, err)
		require.NotContains(t, string(statusJSON), "generated-1")
	})

	t.Run("Rotates through the pool and starts one new interval after downtime", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		interval := s.rotation.Spec.RotationInterval.Duration
		for i, username := range s.rotation.Spec.Usernames[1:] {
			s.now = s.now.Add(interval)
			password := fmt.Sprintf("generated-%d", i+2)
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()

			res, err := s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, interval, res.RequeueAfter)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, i+2, s.generated)
		}

		// Miss several rotation intervals before returning to the first user.
		s.now = s.now.Add(6 * interval)
		username := s.rotation.Spec.Usernames[0]
		password := fmt.Sprintf("generated-%d", len(s.rotation.Spec.Usernames)+1)
		s.expectService()
		s.expectReset(username, password)
		s.expectCA()

		res, err := s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, interval, res.RequeueAfter)
		s.requirePublished(t, username, password, s.now)
		require.Equal(t, len(s.rotation.Spec.Usernames)+1, s.generated)
	})

	t.Run("When the service endpoint and CA change between rotations, updates the Secret and leaves passwords unchanged", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		publishedAt := s.now
		before := s.secret(t)
		before.Data["UNRELATED"] = []byte("keep")
		require.NoError(t, s.client.Update(t.Context(), before))
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
			cr.Spec.ConnInfoSecretTarget.Labels = map[string]string{"app": "consumer"}
			cr.Spec.ConnInfoSecretTarget.Annotations = map[string]string{"example.com/owner": "team"}
		})
		s.service.Users[0].Password = "changed-outside-the-operator"
		s.service.Components[0].Host = "new-db.example.com"
		s.now = s.now.Add(time.Minute)
		s.expectService()
		s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("new-ca", nil).Once()

		res, err := s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, s.rotation.Spec.RotationInterval.Duration-time.Minute, res.RequeueAfter)
		got := s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
		require.Equal(t, []byte("new-db.example.com"), got.Data[s.prefix()+"HOST"])
		require.Equal(t, []byte("new-ca"), got.Data[s.prefix()+"CA_CERT"])
		require.Equal(t, []byte("keep"), got.Data["UNRELATED"])
		require.Equal(t, s.rotation.Spec.ConnInfoSecretTarget.Labels, got.Labels)
		require.Equal(t, s.rotation.Spec.ConnInfoSecretTarget.Annotations, got.Annotations)
		require.Equal(t, 1, s.generated)

		// A subsequent observation must not write the same Secret or status again.
		beforeStatus := s.object(t)
		s.expectService()
		s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("new-ca", nil).Once()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, got.ResourceVersion, s.secret(t).ResourceVersion)
		require.Equal(t, beforeStatus.ResourceVersion, s.object(t).ResourceVersion)
	})

	for _, serviceType := range []string{"pg", "kafka"} {
		t.Run("Refreshes published connection details before retrying rotation/"+serviceType, func(t *testing.T) {
			s := newRotationScenario(t)
			if serviceType == "kafka" {
				s.useKafka()
			}
			s.publishInitial(t)
			published := s.secret(t)
			status := s.object(t).Status
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			username, password := s.rotation.Spec.Usernames[1], "generated-2"
			failure := errors.New("password request failed")
			for attempt := range 2 {
				s.service.Components[0].Host = fmt.Sprintf("new-%d.example.com", attempt)
				s.service.Users[0].Password = "changed-outside-the-operator"
				caCert := fmt.Sprintf("new-ca-%d", attempt)
				want := maps.Clone(published.Data)
				want[s.prefix()+"HOST"] = []byte(s.service.Components[0].Host)
				want[s.prefix()+"CA_CERT"] = []byte(caCert)
				want["aiven-rotation-desired-username"] = []byte(username)
				want["aiven-rotation-desired-password"] = []byte(password)
				if serviceType == "kafka" {
					s.service.Users[0].AccessCert = new(fmt.Sprintf("renewed-certificate-%d", attempt))
					s.service.Users[0].AccessKey = new(fmt.Sprintf("renewed-key-%d", attempt))
					want[s.prefix()+"ACCESS_CERT"] = []byte(*s.service.Users[0].AccessCert)
					want[s.prefix()+"ACCESS_KEY"] = []byte(*s.service.Users[0].AccessKey)
				}
				s.expectService()
				s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return(caCert, nil).Once()
				s.api.EXPECT().
					ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest(password)).
					RunAndReturn(func(context.Context, string, string, string, *service.ServiceUserCredentialsModifyIn) (*service.ServiceUserCredentialsModifyOut, error) {
						require.Equal(t, want, s.secret(t).Data)
						return nil, failure
					}).Once()

				res, err := s.reconcile(t)
				require.ErrorIs(t, err, failure)
				s.requireFailure(t, res, err)
				require.Equal(t, want, s.secret(t).Data)
				got := s.object(t).Status
				require.Equal(t, status.ActiveUsername, got.ActiveUsername)
				require.Equal(t, status.LastRotationAt, got.LastRotationAt)
				require.Equal(t, status.NextRotationAt, got.NextRotationAt)
				require.Equal(t, 2, s.generated)
				s.now = s.now.Add(time.Minute)
				s.restart()
			}

			s.expectService()
			s.expectReset(username, password)
			s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("new-ca-1", nil).Once()
			res, err := s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
			require.Equal(t, 2, s.generated)
		})
	}

	t.Run("When the active user disappears, reports the missing connection details and preserves the Secret", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		publishedAt := s.now
		published := s.secret(t)
		user := s.service.Users[0]
		s.service.Users = slices.Clone(s.service.Users[1:])
		s.expectService()
		s.expectCA()

		res, err := s.reconcile(t)
		require.ErrorContains(t, err, fmt.Sprintf("active user %q not found", user.Username))
		s.requireFailure(t, res, err)
		require.Equal(t, published.Data, s.secret(t).Data)
		require.Equal(t, 1, s.generated)

		// An external owner recreates the user with a different password.
		user.Password = "new-api-password"
		s.service.Users = append(s.service.Users, user)
		s.expectService()
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, user.Username, "generated-1", publishedAt)
		require.Equal(t, 1, s.generated)
	})

	t.Run("When the inactive user is missing, refreshes active details and retries its password request once rotation is due", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		publishedAt := s.now
		user := s.service.Users[1]
		s.service.Users = s.service.Users[:1]
		s.service.Components[0].Host = "new-db.example.com"
		s.now = s.now.Add(time.Minute)
		s.expectService()
		s.expectCA()

		_, err := s.reconcile(t)
		require.NoError(t, err)
		published := s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
		require.Equal(t, []byte("new-db.example.com"), published.Data[s.prefix()+"HOST"])
		require.Equal(t, 1, s.generated)

		s.now = publishedAt.Add(s.rotation.Spec.RotationInterval.Duration)
		s.service.Components[0].Host = "latest-db.example.com"
		s.expectService()
		s.expectCA()
		failure := newAivenError(404, "user not found")
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, user.Username, rotationPasswordRequest("generated-2")).
			Return(nil, failure).Once()
		res, err := s.reconcile(t)
		require.ErrorIs(t, err, failure)
		s.requireFailure(t, res, err)
		pending := s.secret(t)
		require.Equal(t, []byte("latest-db.example.com"), pending.Data[s.prefix()+"HOST"])
		require.Equal(t, published.Data[s.prefix()+"USERNAME"], pending.Data[s.prefix()+"USERNAME"])
		require.Equal(t, published.Data[s.prefix()+"PASSWORD"], pending.Data[s.prefix()+"PASSWORD"])
		require.Equal(t, published.Data["aiven-rotation-published-at"], pending.Data["aiven-rotation-published-at"])
		require.Equal(t, []byte(user.Username), pending.Data["aiven-rotation-desired-username"])
		require.Equal(t, []byte("generated-2"), pending.Data["aiven-rotation-desired-password"])

		s.service.Users = append(s.service.Users, user)
		s.restart()
		s.expectService()
		s.expectReset(user.Username, "generated-2")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, user.Username, "generated-2", s.now)
		require.Equal(t, 2, s.generated)
	})

	t.Run("Recovers status from the Secret and uses the current rotation interval", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		publishedAt := s.now
		secret := s.secret(t)
		cr := s.object(t)
		cr.Status.ActiveUsername = s.rotation.Spec.Usernames[1]
		cr.Status.NextRotationAt = metav1.NewTime(s.now.Add(100 * s.rotation.Spec.RotationInterval.Duration))
		require.NoError(t, s.client.Status().Update(t.Context(), cr))
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
			cr.Spec.RotationInterval.Duration *= 2
		})
		s.restart()
		s.now = s.now.Add(time.Minute)
		s.expectService()
		s.expectCA()

		res, err := s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, s.rotation.Spec.RotationInterval.Duration-time.Minute, res.RequeueAfter)
		s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
		require.Equal(t, secret.ResourceVersion, s.secret(t).ResourceVersion)

		// A future date in status cannot postpone a rotation due according to the Secret.
		cr = s.object(t)
		cr.Status.ActiveUsername = s.rotation.Spec.Usernames[1]
		cr.Status.NextRotationAt = metav1.NewTime(s.now.Add(100 * s.rotation.Spec.RotationInterval.Duration))
		require.NoError(t, s.client.Status().Update(t.Context(), cr))
		s.now = publishedAt.Add(s.rotation.Spec.RotationInterval.Duration)
		failure := errors.New("password request interrupted")
		s.expectService()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, s.rotation.Spec.Usernames[1], rotationPasswordRequest("generated-2")).
			Return(nil, failure).Once()
		s.expectCA()
		res, err = s.reconcile(t)
		require.ErrorIs(t, err, failure)
		s.requireFailure(t, res, err)
		// Failed attempts report the error without partially refreshing status.
		require.Equal(t, cr.Status.ActiveUsername, s.object(t).Status.ActiveUsername)

		s.expectService()
		s.expectReset(s.rotation.Spec.Usernames[1], "generated-2")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, s.rotation.Spec.Usernames[1], "generated-2", s.now)
		require.Equal(t, 2, s.generated)
	})

	t.Run("Initial publication failures", func(t *testing.T) {
		t.Run("Retries a failed password request", func(t *testing.T) {
			s := newRotationScenario(t)
			username, password := s.rotation.Spec.Usernames[0], "generated-1"
			failure := errors.New("password request failed")
			s.expectService()
			s.expectCA()
			s.api.EXPECT().
				ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest(password)).
				Return(nil, failure).Once()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			require.Equal(t, map[string][]byte{
				"aiven-rotation-desired-username": []byte(username),
				"aiven-rotation-desired-password": []byte(password),
			}, s.secret(t).Data)
			require.Equal(t, 1, s.generated)

			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, 1, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries when the service component is missing", func(t *testing.T) {
			s := newRotationScenario(t)
			username, password := s.rotation.Spec.Usernames[0], "generated-1"
			components := slices.Clone(s.service.Components)
			s.service.Components = nil
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()

			res, err := s.reconcile(t)
			require.ErrorContains(t, err, "service component")
			s.requireFailure(t, res, err)
			require.Equal(t, map[string][]byte{
				"aiven-rotation-desired-username": []byte(username),
				"aiven-rotation-desired-password": []byte(password),
			}, s.secret(t).Data)
			require.Equal(t, 1, s.generated)

			s.service.Components = components
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, 1, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries a failed CA request before generating credentials", func(t *testing.T) {
			s := newRotationScenario(t)
			failure := errors.New("CA request failed")
			s.expectService()
			s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("", failure).Once()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			require.True(t, apierrors.IsNotFound(s.client.Get(t.Context(), s.secretKey(), &corev1.Secret{})))
			require.Zero(t, s.generated)

			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(s.rotation.Spec.Usernames[0], "generated-1")
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", s.now)
			require.Equal(t, 1, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries a failed publication write", func(t *testing.T) {
			s := newRotationScenario(t)
			username, password := s.rotation.Spec.Usernames[0], "generated-1"
			failure := errors.New("publication write failed")
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
					if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.secretKey().Name {
						if _, pending := secret.Data["aiven-rotation-desired-username"]; !pending {
							return failure
						}
					}
					return c.Update(ctx, obj, opts...)
				},
			})
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			require.Equal(t, map[string][]byte{
				"aiven-rotation-desired-username": []byte(username),
				"aiven-rotation-desired-password": []byte(password),
			}, s.secret(t).Data)
			require.Equal(t, 1, s.generated)

			s.r.Client = s.client
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, 1, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Recovers a lost publication response without resetting credentials again", func(t *testing.T) {
			s := newRotationScenario(t)
			username, password := s.rotation.Spec.Usernames[0], "generated-1"
			failure := errors.New("publication response lost")
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
					if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.secretKey().Name {
						if _, pending := secret.Data["aiven-rotation-desired-username"]; !pending {
							require.NoError(t, c.Update(ctx, obj, opts...))
							return failure
						}
					}
					return c.Update(ctx, obj, opts...)
				},
			})
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			publishedAt := s.now

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			saved := s.secret(t)
			require.Equal(t, []byte(password), saved.Data[s.prefix()+"PASSWORD"])
			require.NotContains(t, saved.Data, "aiven-rotation-desired-password")
			require.Equal(t, 1, s.generated)

			s.r.Client = s.client
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, publishedAt)
			require.Equal(t, 1, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration-time.Minute, res.RequeueAfter)
		})
	})

	t.Run("Scheduled rotation failures", func(t *testing.T) {
		t.Run("Retries a failed password request despite an extended interval", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			before := s.secret(t).Data
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			username, password := s.rotation.Spec.Usernames[1], "generated-2"
			failure := errors.New("password request failed")
			s.expectService()
			s.expectCA()
			s.api.EXPECT().
				ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest(password)).
				Return(nil, failure).Once()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			want := maps.Clone(before)
			want["aiven-rotation-desired-username"] = []byte(username)
			want["aiven-rotation-desired-password"] = []byte(password)
			require.Equal(t, want, s.secret(t).Data)
			require.Equal(t, 2, s.generated)

			s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
				cr.Spec.RotationInterval.Duration *= 2
			})
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, 2, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries when the service component is missing before preparing a candidate", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			before := s.secret(t).Data
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			components := slices.Clone(s.service.Components)
			s.service.Components = nil
			s.expectService()
			s.expectCA()

			res, err := s.reconcile(t)
			require.ErrorContains(t, err, "service component")
			s.requireFailure(t, res, err)
			require.Equal(t, before, s.secret(t).Data)
			require.Equal(t, 1, s.generated)

			s.service.Components = components
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(s.rotation.Spec.Usernames[1], "generated-2")
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, s.rotation.Spec.Usernames[1], "generated-2", s.now)
			require.Equal(t, 2, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries a failed CA request before preparing a candidate", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			before := s.secret(t).Data
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			failure := errors.New("CA request failed")
			s.expectService()
			s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("", failure).Once()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			require.Equal(t, before, s.secret(t).Data)
			require.Equal(t, 1, s.generated)

			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(s.rotation.Spec.Usernames[1], "generated-2")
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, s.rotation.Spec.Usernames[1], "generated-2", s.now)
			require.Equal(t, 2, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Retries a failed publication write despite an extended interval", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			before := s.secret(t).Data
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			username, password := s.rotation.Spec.Usernames[1], "generated-2"
			failure := errors.New("publication write failed")
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
					if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.secretKey().Name {
						if _, pending := secret.Data["aiven-rotation-desired-username"]; !pending {
							return failure
						}
					}
					return c.Update(ctx, obj, opts...)
				},
			})
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			want := maps.Clone(before)
			want["aiven-rotation-desired-username"] = []byte(username)
			want["aiven-rotation-desired-password"] = []byte(password)
			require.Equal(t, want, s.secret(t).Data)
			require.Equal(t, 2, s.generated)

			s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
				cr.Spec.RotationInterval.Duration *= 2
			})
			s.r.Client = s.client
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, s.now)
			require.Equal(t, 2, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		})

		t.Run("Recovers a lost publication response without resetting credentials again", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
			username, password := s.rotation.Spec.Usernames[1], "generated-2"
			failure := errors.New("publication response lost")
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
					if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.secretKey().Name {
						if _, pending := secret.Data["aiven-rotation-desired-username"]; !pending {
							require.NoError(t, c.Update(ctx, obj, opts...))
							return failure
						}
					}
					return c.Update(ctx, obj, opts...)
				},
			})
			s.expectService()
			s.expectReset(username, password)
			s.expectCA()
			publishedAt := s.now

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			s.requireFailure(t, res, err)
			saved := s.secret(t)
			require.Equal(t, []byte(password), saved.Data[s.prefix()+"PASSWORD"])
			require.NotContains(t, saved.Data, "aiven-rotation-desired-password")
			require.Equal(t, 2, s.generated)

			s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
				cr.Spec.RotationInterval.Duration *= 2
			})
			s.r.Client = s.client
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, publishedAt)
			require.Equal(t, 2, s.generated)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration-time.Minute, res.RequeueAfter)
		})
	})

	t.Run("Losing initial Secret creation reuses the winner's candidate", func(t *testing.T) {
		s := newRotationScenario(t)
		s.expectService()
		s.expectCA()
		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			Create: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.CreateOption) error {
				if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.rotation.Spec.ConnInfoSecretTarget.Name {
					winner := secret.DeepCopy()
					winner.Data["aiven-rotation-desired-password"] = []byte("winner-password")
					require.NoError(t, c.Create(ctx, winner))
				}
				return c.Create(ctx, obj, opts...)
			},
		})

		res, err := s.reconcile(t)
		require.True(t, apierrors.IsAlreadyExists(err))
		s.requireFailure(t, res, err)
		require.Equal(t, []byte("winner-password"), s.secret(t).Data["aiven-rotation-desired-password"])
		s.restart()
		s.expectService()
		s.expectReset(s.rotation.Spec.Usernames[0], "winner-password")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, s.rotation.Spec.Usernames[0], "winner-password", s.now)
		require.Equal(t, 1, s.generated)
	})

	t.Run("Rejects a stale pending Secret before replaying its password request", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
		username := s.rotation.Spec.Usernames[1]
		s.expectService()
		s.expectCA()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest("generated-2")).
			Return(nil, errors.New("request interrupted")).Once()
		res, err := s.reconcile(t)
		s.requireFailure(t, res, err)
		stale := s.secret(t)

		s.expectService()
		s.expectReset(username, "generated-2")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		winner := s.secret(t)

		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
				if secret, ok := obj.(*corev1.Secret); ok && key == crclient.ObjectKeyFromObject(stale) {
					*secret = *stale.DeepCopy()
					return nil
				}
				return c.Get(ctx, key, obj, opts...)
			},
		})
		s.expectService()
		s.expectCA()
		res, err = s.reconcile(t)
		require.True(t, apierrors.IsConflict(err))
		s.requireFailure(t, res, err)
		require.Equal(t, winner.Data, s.secret(t).Data)
		require.Equal(t, winner.ResourceVersion, s.secret(t).ResourceVersion)
		require.Equal(t, 2, s.generated)
	})

	t.Run("A losing publication cannot overwrite credentials committed by another reconciliation", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
		username := s.rotation.Spec.Usernames[1]
		s.expectService()
		s.expectReset(username, "generated-2")
		s.expectCA()
		compete := true
		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
				secret, ok := obj.(*corev1.Secret)
				if ok && secret.Name == s.rotation.Spec.ConnInfoSecretTarget.Name && compete {
					if _, pending := secret.Data["aiven-rotation-desired-username"]; !pending {
						compete = false
						winner := secret.DeepCopy()
						winner.Data[s.prefix()+"PASSWORD"] = []byte("winner-password")
						winner.Data["UNRELATED"] = []byte("winner-data")
						require.NoError(t, c.Update(ctx, winner))
					}
				}
				return c.Update(ctx, obj, opts...)
			},
		})

		res, err := s.reconcile(t)
		require.True(t, apierrors.IsConflict(err))
		s.requireFailure(t, res, err)
		winner := s.secret(t)
		require.Equal(t, []byte("winner-password"), winner.Data[s.prefix()+"PASSWORD"])
		s.restart()
		s.expectService()
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, username, "winner-password", s.now)
		require.Equal(t, winner.Data, s.secret(t).Data)
		require.Equal(t, winner.ResourceVersion, s.secret(t).ResourceVersion)
		require.Equal(t, 2, s.generated)
	})

	t.Run("Kafka publishes returned certificates and refreshes optional endpoints without rotating", func(t *testing.T) {
		s := newRotationScenario(t)
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
			cr.Spec.ConnInfoSecretTarget.Prefix = "APP_"
		})
		s.useKafka()
		s.expectService()
		users := slices.Clone(s.service.Users)
		users[0].Password = ""
		users[0].AccessCert = new("issued-certificate")
		users[0].AccessKey = new("issued-key")
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, users[0].Username, rotationPasswordRequest("generated-1")).
			Return(&service.ServiceUserCredentialsModifyOut{Users: users}, nil).Once()
		s.expectCA()

		_, err := s.reconcile(t)
		require.NoError(t, err)
		publishedAt := s.now
		secret := s.requirePublished(t, users[0].Username, "generated-1", publishedAt)
		require.Equal(t, []byte("issued-certificate"), secret.Data["APP_ACCESS_CERT"])
		require.Equal(t, []byte("issued-key"), secret.Data["APP_ACCESS_KEY"])
		require.Equal(t, []byte("kafka.example.com"), secret.Data["APP_HOST"])
		require.Equal(t, []byte("9092"), secret.Data["APP_PORT"])
		require.Equal(t, []byte("sasl.example.com"), secret.Data["APP_SASL_HOST"])
		require.Equal(t, []byte("9093"), secret.Data["APP_SASL_PORT"])
		require.Equal(t, []byte("schema.example.com"), secret.Data["APP_SCHEMA_REGISTRY_HOST"])
		require.Equal(t, []byte("8081"), secret.Data["APP_SCHEMA_REGISTRY_PORT"])

		s.service.Users = users
		s.service.Users[0].AccessCert = new("renewed-certificate")
		s.service.Users[0].AccessKey = new("renewed-key")
		s.service.Components = s.service.Components[:1]
		s.now = s.now.Add(time.Minute)
		s.expectService()
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		secret = s.requirePublished(t, users[0].Username, "generated-1", publishedAt)
		require.Equal(t, []byte("renewed-certificate"), secret.Data["APP_ACCESS_CERT"])
		require.Equal(t, []byte("renewed-key"), secret.Data["APP_ACCESS_KEY"])
		for _, key := range []string{"APP_SASL_HOST", "APP_SASL_PORT", "APP_SCHEMA_REGISTRY_HOST", "APP_SCHEMA_REGISTRY_PORT"} {
			require.NotContains(t, secret.Data, key)
		}
		require.Equal(t, 1, s.generated)
	})

	t.Run("When the password response omits the Kafka user, keeps the candidate for retry", func(t *testing.T) {
		s := newRotationScenario(t)
		s.useKafka()
		s.expectService()
		username := s.rotation.Spec.Usernames[0]
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest("generated-1")).
			Return(&service.ServiceUserCredentialsModifyOut{Users: slices.Clone(s.service.Users[1:])}, nil).Once()
		s.expectCA()

		res, err := s.reconcile(t)
		s.requireFailure(t, res, err)
		require.Equal(t, map[string][]byte{
			"aiven-rotation-desired-username": []byte(username),
			"aiven-rotation-desired-password": []byte("generated-1"),
		}, s.secret(t).Data)

		s.now = s.now.Add(time.Minute)
		s.restart()
		s.expectService()
		s.expectReset(username, "generated-1")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, username, "generated-1", s.now)
		require.Equal(t, 1, s.generated)
	})

	for _, tc := range []struct {
		name string
		cert *string
		key  *string
	}{
		{"missing certificate", nil, new("key")},
		{"empty certificate", new(""), new("key")},
		{"missing key", new("certificate"), nil},
		{"empty key", new("certificate"), new("")},
	} {
		t.Run("When Kafka credentials aren't available yet, waits before publishing/"+tc.name, func(t *testing.T) {
			s := newRotationScenario(t)
			s.useKafka()
			s.expectService()
			users := slices.Clone(s.service.Users)
			users[0].AccessCert, users[0].AccessKey = tc.cert, tc.key
			username := s.rotation.Spec.Usernames[0]
			s.api.EXPECT().
				ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest("generated-1")).
				Return(&service.ServiceUserCredentialsModifyOut{Users: users}, nil).Once()
			s.expectCA()

			res, err := s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
			require.Empty(t, s.object(t).Status.Conditions)
			require.Equal(t, []string{
				fmt.Sprintf("Normal %s %s: Kafka user certificate and key are not yet available from the API", eventPreconditionsNotMet, errPreconditionNotMet),
			}, recorderEvents(s.recorder))
			require.Equal(t, map[string][]byte{
				"aiven-rotation-desired-username": []byte(username),
				"aiven-rotation-desired-password": []byte("generated-1"),
			}, s.secret(t).Data)

			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectReset(username, "generated-1")
			s.expectCA()
			_, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, "generated-1", s.now)
			require.Equal(t, 1, s.generated)
		})
	}

	for _, phase := range []string{"status write", "status response lost"} {
		t.Run("Recovers after "+phase+" without rotating again", func(t *testing.T) {
			s := newRotationScenario(t)
			failure := errors.New(phase + " failed")
			writes := 0
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, name string, obj crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					if name == "status" {
						writes++
						if writes == 1 {
							if phase == "status response lost" {
								require.NoError(t, c.SubResource(name).Update(ctx, obj, opts...))
							}
							return failure
						}
					}
					return c.SubResource(name).Update(ctx, obj, opts...)
				},
			})
			s.expectService()
			s.expectReset(s.rotation.Spec.Usernames[0], "generated-1")
			s.expectCA()

			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			require.Equal(t, ctrlruntime.Result{}, res)
			committed := s.secret(t)
			require.Equal(t, []byte("generated-1"), committed.Data[s.prefix()+"PASSWORD"])
			require.NotContains(t, committed.Data, "aiven-rotation-desired-password")
			publishedAt := s.now
			s.now = s.now.Add(time.Minute)
			s.restart()
			s.expectService()
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, s.rotation.Spec.RotationInterval.Duration-time.Minute, res.RequeueAfter)
			s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
			require.Equal(t, committed.ResourceVersion, s.secret(t).ResourceVersion)
			require.Equal(t, 1, s.generated)
			if phase == "status response lost" {
				require.Equal(t, 1, writes)
			} else {
				require.Equal(t, 2, writes)
			}
		})
	}

	t.Run("Preserves both an operation error and a status write error", func(t *testing.T) {
		s := newRotationScenario(t)
		operationErr := errors.New("service unavailable")
		statusErr := errors.New("status unavailable")
		s.api.EXPECT().ServiceGet(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, mock.Anything).
			Return(nil, operationErr).Once()
		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			SubResourceUpdate: func(_ context.Context, _ crclient.Client, name string, obj crclient.Object, _ ...crclient.SubResourceUpdateOption) error {
				require.Equal(t, "status", name)
				cr := obj.(*v1alpha1.ServiceUserRotation)
				require.Equal(t, operationErr.Error(), meta.FindStatusCondition(cr.Status.Conditions, ConditionTypeError).Message)
				return statusErr
			},
		})

		res, err := s.reconcile(t)
		require.ErrorIs(t, err, operationErr)
		require.ErrorIs(t, err, statusErr)
		require.Equal(t, ctrlruntime.Result{}, res)
		require.Equal(t, []string{"Warning ReconcileFailed " + operationErr.Error()}, recorderEvents(s.recorder))
	})

	for _, publication := range []string{"initial", "scheduled", "retry"} {
		t.Run("Starts the rotation interval after a slow CA request/"+publication, func(t *testing.T) {
			const interval = time.Hour
			s := newRotationScenario(t)
			s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
				cr.Spec.RotationInterval.Duration = interval
			})
			username, password := s.rotation.Spec.Usernames[0], "generated-1"
			if publication != "initial" {
				s.publishInitial(t)
				s.now = s.now.Add(interval)
				username, password = s.rotation.Spec.Usernames[1], "generated-2"
			}
			if publication == "retry" {
				s.expectService()
				s.expectCA()
				s.api.EXPECT().
					ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest(password)).
					Return(nil, errors.New("password request interrupted")).Once()
				res, err := s.reconcile(t)
				s.requireFailure(t, res, err)
				s.restart()
			}

			publishedAt := s.now.Add(2 * interval)
			s.expectService()
			s.expectReset(username, password)
			s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).
				RunAndReturn(func(context.Context, string) (string, error) {
					s.now = publishedAt
					return "project-ca", nil
				}).Once()

			res, err := s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, publishedAt)
			require.Equal(t, interval, res.RequeueAfter)
			generated := s.generated

			// Consumers retain the full interval to switch to the new credentials.
			s.now = publishedAt.Add(interval - time.Second)
			s.expectService()
			s.expectCA()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			s.requirePublished(t, username, password, publishedAt)
			require.Equal(t, time.Second, res.RequeueAfter)
			require.Equal(t, generated, s.generated)
		})
	}

	t.Run("A slow CA refresh preserves the existing rotation deadline", func(t *testing.T) {
		const interval = time.Hour
		s := newRotationScenario(t)
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
			cr.Spec.RotationInterval.Duration = interval
		})
		s.publishInitial(t)
		publishedAt := s.now
		s.now = publishedAt.Add(interval - time.Second)
		s.service.Components[0].Host = "new-db.example.com"
		s.expectService()
		s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).
			RunAndReturn(func(context.Context, string) (string, error) {
				s.now = s.now.Add(2 * interval)
				return "new-ca", nil
			}).Once()

		res, err := s.reconcile(t)
		require.NoError(t, err)
		secret := s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
		require.Equal(t, []byte("new-db.example.com"), secret.Data[s.prefix()+"HOST"])
		require.Equal(t, []byte("new-ca"), secret.Data[s.prefix()+"CA_CERT"])
		require.Equal(t, time.Nanosecond, res.RequeueAfter)
		require.Equal(t, 1, s.generated)
	})

	t.Run("Computes requeue after publication", func(t *testing.T) {
		const interval = time.Hour
		for _, tc := range []struct {
			name          string
			writeDuration time.Duration
			pollInterval  time.Duration
			minRequeue    time.Duration
			maxRequeue    time.Duration
		}{
			{"before deadline", time.Second, 2 * interval, interval - time.Second, interval - time.Second},
			{"at deadline", interval, 2 * interval, time.Nanosecond, time.Nanosecond},
			{"past deadline", interval + time.Second, 2 * interval, time.Nanosecond, time.Nanosecond},
			{"poll cap includes jitter", time.Second, time.Minute, time.Minute, 66 * time.Second},
			{"rotation deadline caps jitter", interval - time.Minute, time.Minute, time.Minute, time.Minute},
			{"default poll", time.Second, 0, DefaultPollInterval, DefaultPollInterval + DefaultPollInterval/10},
			{"negative poll", time.Second, -time.Minute, DefaultPollInterval, DefaultPollInterval + DefaultPollInterval/10},
		} {
			t.Run(tc.name, func(t *testing.T) {
				s := newRotationScenario(t)
				s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
					cr.Spec.RotationInterval.Duration = interval
				})
				s.r.PollInterval = tc.pollInterval
				publishedAt := s.now
				s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
					Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
						if err := c.Update(ctx, obj, opts...); err != nil {
							return err
						}
						if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.rotation.Spec.ConnInfoSecretTarget.Name {
							s.now = s.now.Add(tc.writeDuration)
						}
						return nil
					},
				})
				s.expectService()
				s.expectReset(s.rotation.Spec.Usernames[0], "generated-1")
				s.expectCA()

				res, err := s.reconcile(t)
				require.NoError(t, err)
				require.GreaterOrEqual(t, res.RequeueAfter, tc.minRequeue)
				require.LessOrEqual(t, res.RequeueAfter, tc.maxRequeue)
				s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
			})
		}
	})

	for _, elapsed := range []time.Duration{0, time.Second, 2 * time.Second} {
		t.Run("Computes requeue after refreshing connection details/"+elapsed.String(), func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			publishedAt := s.now
			s.now = publishedAt.Add(s.rotation.Spec.RotationInterval.Duration - time.Second)
			s.service.Components[0].Host = "new-db.example.com"
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, obj crclient.Object, opts ...crclient.UpdateOption) error {
					if err := c.Update(ctx, obj, opts...); err != nil {
						return err
					}
					if secret, ok := obj.(*corev1.Secret); ok && secret.Name == s.rotation.Spec.ConnInfoSecretTarget.Name {
						s.now = s.now.Add(elapsed)
					}
					return nil
				},
			})
			s.expectService()
			s.expectCA()

			res, err := s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, max(time.Nanosecond, time.Second-elapsed), res.RequeueAfter)
			secret := s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
			require.Equal(t, []byte("new-db.example.com"), secret.Data[s.prefix()+"HOST"])
			require.Equal(t, 1, s.generated)
		})
	}

	for _, tc := range []struct {
		name     string
		response *service.ServiceGetOut
		err      error
	}{
		{"not created", nil, newAivenError(404, "service missing")},
		{"rebuilding", &service.ServiceGetOut{State: "REBUILDING"}, nil},
	} {
		t.Run("When the service isn't ready yet, waits before publishing credentials/"+tc.name, func(t *testing.T) {
			s := newRotationScenario(t)
			s.api.EXPECT().ServiceGet(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, mock.Anything).
				Return(tc.response, tc.err).Twice()
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				SubResourceUpdate: func(_ context.Context, _ crclient.Client, _ string, _ crclient.Object, _ ...crclient.SubResourceUpdateOption) error {
					t.Fatal("waiting for the service must not write status")
					return nil
				},
			})
			for range 2 {
				res, err := s.reconcile(t)
				require.NoError(t, err)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
				require.Empty(t, s.object(t).Status.Conditions)
				events := recorderEvents(s.recorder)
				require.Len(t, events, 1)
				require.Contains(t, events[0], "Normal PreconditionsNotMet ")
				require.Zero(t, s.generated)
				s.now = s.now.Add(requeueTimeout)
			}
			require.True(t, apierrors.IsNotFound(s.client.Get(t.Context(), s.secretKey(), &corev1.Secret{})))

			s.r.Client = s.client
			s.service.State = service.ServiceStateTypeRebalancing
			s.publishInitial(t)
		})
	}

	serviceErr := errors.New("service request failed")
	for _, tc := range []struct {
		name     string
		response *service.ServiceGetOut
		err      error
		wantErr  error
	}{
		{"API failure", nil, serviceErr, serviceErr},
		{"power off", &service.ServiceGetOut{State: service.ServiceStateTypePoweroff}, nil, errServicePoweredOff},
	} {
		t.Run("Reports a stable error until the service becomes operational/"+tc.name, func(t *testing.T) {
			s := newRotationScenario(t)
			s.api.EXPECT().ServiceGet(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, mock.Anything).
				Return(tc.response, tc.err).Twice()

			res, err := s.reconcile(t)
			s.requireFailure(t, res, err)
			require.ErrorIs(t, err, tc.wantErr)
			failed := s.object(t)
			res, err = s.reconcile(t)
			s.requireFailure(t, res, err)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, failed.ResourceVersion, s.object(t).ResourceVersion)
			require.Zero(t, s.generated)

			s.service.State = "REBUILDING"
			s.expectService()
			res, err = s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
			require.Equal(t, failed.Status, s.object(t).Status)
			require.Equal(t, failed.ResourceVersion, s.object(t).ResourceVersion)

			s.service.State = service.ServiceStateTypeRebalancing
			s.publishInitial(t)
		})
	}

	t.Run("When the first user is missing, preserves the candidate after the API rejects the password request", func(t *testing.T) {
		s := newRotationScenario(t)
		users := slices.Clone(s.service.Users)
		s.service.Users = nil
		failure := newAivenError(404, "user not found")
		s.expectService()
		s.expectCA()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, users[0].Username, rotationPasswordRequest("generated-1")).
			Return(nil, failure).Once()

		res, err := s.reconcile(t)
		require.ErrorIs(t, err, failure)
		s.requireFailure(t, res, err)
		require.Equal(t, map[string][]byte{
			"aiven-rotation-desired-username": []byte(users[0].Username),
			"aiven-rotation-desired-password": []byte("generated-1"),
		}, s.secret(t).Data)
		require.Equal(t, 1, s.generated)

		s.service.Users = users
		s.restart()
		s.expectService()
		s.expectReset(users[0].Username, "generated-1")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, users[0].Username, "generated-1", s.now)
		require.Equal(t, 1, s.generated)
	})

	t.Run("Does not change an API password when saving the candidate fails", func(t *testing.T) {
		s := newRotationScenario(t)
		failure := errors.New("Secret creation failed")
		s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
			Create: func(_ context.Context, _ crclient.WithWatch, _ crclient.Object, _ ...crclient.CreateOption) error {
				return failure
			},
		})
		s.expectService()
		s.expectCA()
		res, err := s.reconcile(t)
		require.ErrorIs(t, err, failure)
		s.requireFailure(t, res, err)
		require.True(t, apierrors.IsNotFound(s.client.Get(t.Context(), s.secretKey(), &corev1.Secret{})))

		s.r.Client = s.client
		s.expectService()
		s.expectReset(s.rotation.Spec.Usernames[0], "generated-2")
		s.expectCA()
		_, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-2", s.now)
	})

	t.Run("Does not adopt an unrelated connection Secret", func(t *testing.T) {
		s := newRotationScenario(t)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: s.secretKey().Name, Namespace: s.secretKey().Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte("someone-else")},
		}
		require.NoError(t, s.client.Create(t.Context(), secret))
		s.expectService()
		res, err := s.reconcile(t)
		require.ErrorContains(t, err, "not controlled")
		s.requireFailure(t, res, err)
		require.Equal(t, secret.Data, s.secret(t).Data)
		require.Equal(t, secret.ResourceVersion, s.secret(t).ResourceVersion)
		require.Zero(t, s.generated)
	})

	t.Run("Invalid saved credentials", func(t *testing.T) {
		cases := []struct {
			name string
			edit func(map[string][]byte)
		}{
			{"built-in user", func(data map[string][]byte) { data["aiven-rotation-desired-username"] = []byte("avnadmin") }},
			{"user outside the pool", func(data map[string][]byte) { data["aiven-rotation-desired-username"] = []byte("outside-pool") }},
			{"empty username", func(data map[string][]byte) { data["aiven-rotation-desired-username"] = []byte{} }},
			{"missing username", func(data map[string][]byte) { delete(data, "aiven-rotation-desired-username") }},
			{"missing password", func(data map[string][]byte) { delete(data, "aiven-rotation-desired-password") }},
			{"empty password", func(data map[string][]byte) { data["aiven-rotation-desired-password"] = []byte{} }},
		}

		t.Run("Discards the candidate without rotating before the deadline", func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					s := newRotationScenario(t)
					s.publishInitial(t)
					publishedAt := s.now
					s.now = publishedAt.Add(s.rotation.Spec.RotationInterval.Duration / 2)
					// Existing users outside the configured pool must not be reset either.
					s.service.Users = append(s.service.Users,
						service.UserOut{Username: "avnadmin"},
						service.UserOut{Username: "outside-pool"},
					)
					s.service.Components[0].Host = "new-db.example.com"
					secret := s.secret(t)
					secret.Data["aiven-rotation-desired-username"] = []byte(s.rotation.Spec.Usernames[1])
					secret.Data["aiven-rotation-desired-password"] = []byte("saved-password")
					tc.edit(secret.Data)
					require.NoError(t, s.client.Update(t.Context(), secret))
					s.expectService()
					s.expectCA()

					res, err := s.reconcile(t)
					require.NoError(t, err)
					got := s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", publishedAt)
					require.Equal(t, []byte("new-db.example.com"), got.Data[s.prefix()+"HOST"])
					require.Equal(t, s.rotation.Spec.RotationInterval.Duration/2, res.RequeueAfter)
					require.Equal(t, 1, s.generated)
					s.api.AssertNumberOfCalls(t, "ServiceUserCredentialsModify", 1)
				})
			}
		})

		t.Run("Regenerates the candidate when rotation is due", func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					s := newRotationScenario(t)
					s.publishInitial(t)
					s.now = s.now.Add(s.rotation.Spec.RotationInterval.Duration)
					s.service.Users = append(s.service.Users,
						service.UserOut{Username: "avnadmin"},
						service.UserOut{Username: "outside-pool"},
					)
					s.service.Components[0].Host = "new-db.example.com"
					secret := s.secret(t)
					secret.Data["aiven-rotation-desired-username"] = []byte(s.rotation.Spec.Usernames[1])
					secret.Data["aiven-rotation-desired-password"] = []byte("saved-password")
					tc.edit(secret.Data)
					require.NoError(t, s.client.Update(t.Context(), secret))
					s.expectService()
					s.expectReset(s.rotation.Spec.Usernames[1], "generated-2")
					s.expectCA()

					res, err := s.reconcile(t)
					require.NoError(t, err)
					got := s.requirePublished(t, s.rotation.Spec.Usernames[1], "generated-2", s.now)
					require.Equal(t, []byte("new-db.example.com"), got.Data[s.prefix()+"HOST"])
					require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
					require.Equal(t, 2, s.generated)
					s.api.AssertNumberOfCalls(t, "ServiceUserCredentialsModify", 2)
				})
			}
		})
	})

	t.Run("Regenerates an incomplete candidate before the first publication", func(t *testing.T) {
		s := newRotationScenario(t)
		username := s.rotation.Spec.Usernames[0]
		s.expectService()
		s.expectCA()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest("generated-1")).
			Return(nil, errors.New("password request interrupted")).Once()
		res, err := s.reconcile(t)
		s.requireFailure(t, res, err)
		secret := s.secret(t)
		delete(secret.Data, "aiven-rotation-desired-password")
		require.NoError(t, s.client.Update(t.Context(), secret))
		s.restart()
		s.expectService()
		s.api.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest("generated-2")).
			RunAndReturn(func(_ context.Context, _, _, username string, in *service.ServiceUserCredentialsModifyIn) (*service.ServiceUserCredentialsModifyOut, error) {
				require.Equal(t, map[string][]byte{
					"aiven-rotation-desired-username": []byte(username),
					"aiven-rotation-desired-password": []byte(*in.NewPassword),
				}, s.secret(t).Data)
				return &service.ServiceUserCredentialsModifyOut{Users: slices.Clone(s.service.Users)}, nil
			}).Once()
		s.expectCA()

		res, err = s.reconcile(t)
		require.NoError(t, err)
		s.requirePublished(t, username, "generated-2", s.now)
		require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
		require.Equal(t, 2, s.generated)
	})

	t.Run("Rejects an invalid publication time before requesting a new password", func(t *testing.T) {
		s := newRotationScenario(t)
		s.publishInitial(t)
		secret := s.secret(t)
		secret.Data["aiven-rotation-published-at"] = []byte("not-a-time")
		require.NoError(t, s.client.Update(t.Context(), secret))
		s.expectService()
		res, err := s.reconcile(t)
		require.ErrorContains(t, err, "invalid rotation publication time")
		s.requireFailure(t, res, err)
		require.Equal(t, secret.Data, s.secret(t).Data)
		require.Equal(t, 1, s.generated)
	})

	t.Run("Returns successfully when the resource no longer exists", func(t *testing.T) {
		s := newRotationScenario(t)
		require.NoError(t, s.client.Delete(t.Context(), s.rotation))
		res, err := s.reconcile(t)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)
		require.Empty(t, recorderEvents(s.recorder))
	})

	for _, resource := range []string{"rotation", "connection Secret"} {
		t.Run("Stops on a Kubernetes read error/"+resource, func(t *testing.T) {
			s := newRotationScenario(t)
			failure := errors.New("Kubernetes read failed")
			target := crclient.ObjectKeyFromObject(s.rotation)
			if resource == "connection Secret" {
				target = s.secretKey()
				s.expectService()
			}
			s.r.Client = interceptor.NewClient(s.client, interceptor.Funcs{
				Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, obj crclient.Object, opts ...crclient.GetOption) error {
					if key == target {
						return failure
					}
					return c.Get(ctx, key, obj, opts...)
				},
			})
			res, err := s.reconcile(t)
			require.ErrorIs(t, err, failure)
			require.Equal(t, ctrlruntime.Result{}, res)
			if resource == "connection Secret" {
				s.requireFailure(t, res, err)
			} else {
				require.Empty(t, s.object(t).Status.Conditions)
				require.Empty(t, recorderEvents(s.recorder))
			}
			require.Zero(t, s.generated)
		})
	}

	t.Run("Reports a missing auth Secret", func(t *testing.T) {
		s := newRotationScenario(t)
		require.NoError(t, s.client.Delete(t.Context(), s.auth))
		res, err := s.reconcile(t)
		s.requireFailure(t, res, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Reports an API client initialization error", func(t *testing.T) {
		s := newRotationScenario(t)
		failure := errors.New("client initialization failed")
		s.r.Controller.newAivenClient = func(_, _, _ string) (avngen.Client, error) {
			return nil, failure
		}
		res, err := s.reconcile(t)
		s.requireFailure(t, res, err)
		require.ErrorIs(t, err, failure)
	})

	t.Run("Uses the default token before looking up the auth Secret", func(t *testing.T) {
		s := newRotationScenario(t)
		require.NoError(t, s.client.Delete(t.Context(), s.auth))
		s.r.DefaultToken = "default-token"
		s.r.Controller.newAivenClient = func(token, _, _ string) (avngen.Client, error) {
			require.Equal(t, "default-token", token)
			return s.api, nil
		}
		s.publishInitial(t)
	})

	t.Run("Reports a missing token", func(t *testing.T) {
		s := newRotationScenario(t)
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.AuthSecretRef = nil })
		res, err := s.reconcile(t)
		require.ErrorIs(t, err, errNoTokenProvided)
		s.requireFailure(t, res, err)
	})

	t.Run("Uses SERVICEUSER_ when the prefix is empty", func(t *testing.T) {
		s := newRotationScenario(t)
		s.configure(t, func(cr *v1alpha1.ServiceUserRotation) { cr.Spec.ConnInfoSecretTarget.Prefix = "" })
		s.publishInitial(t)
		require.Equal(t, []byte("generated-1"), s.secret(t).Data["SERVICEUSER_PASSWORD"])
		require.Empty(t, s.object(t).Spec.ConnInfoSecretTarget.Prefix)
	})

	t.Run("Deletion", func(t *testing.T) {
		t.Run("Deleting the resource stops rotation without API requests", func(t *testing.T) {
			s := newRotationScenario(t)
			s.publishInitial(t)
			require.NoError(t, s.client.Delete(t.Context(), s.rotation))
			res, err := s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, ctrlruntime.Result{}, res)
			require.True(t, apierrors.IsNotFound(s.client.Get(t.Context(), crclient.ObjectKeyFromObject(s.rotation), &v1alpha1.ServiceUserRotation{})))
			require.Empty(t, recorderEvents(s.recorder))
		})

		t.Run("Does nothing when only another controller's finalizer is present", func(t *testing.T) {
			s := newRotationScenario(t)
			s.configure(t, func(cr *v1alpha1.ServiceUserRotation) {
				cr.Finalizers = []string{"example.com/another-controller"}
			})
			require.NoError(t, s.client.Delete(t.Context(), s.rotation))
			res, err := s.reconcile(t)
			require.NoError(t, err)
			require.Equal(t, ctrlruntime.Result{}, res)
			require.Equal(t, []string{"example.com/another-controller"}, s.object(t).Finalizers)
			require.Empty(t, s.object(t).Status.Conditions)
		})
	})
}

type rotationScenario struct {
	rotation  *v1alpha1.ServiceUserRotation
	auth      *corev1.Secret
	client    crclient.WithWatch
	r         *ServiceUserRotationReconciler
	api       *avngen.MockClient
	service   *service.ServiceGetOut
	recorder  *record.FakeRecorder
	now       time.Time
	generated int
}

func (s *rotationScenario) restart() {
	now, generate := s.r.now, s.r.generatePassword
	s.r = newServiceUserRotationReconciler(s.r.Controller).(*ServiceUserRotationReconciler)
	s.r.now, s.r.generatePassword = now, generate
}

func (s *rotationScenario) reconcile(t *testing.T) (ctrlruntime.Result, error) {
	t.Helper()
	return s.r.Reconcile(t.Context(), ctrlruntime.Request{NamespacedName: crclient.ObjectKeyFromObject(s.rotation)})
}

func (s *rotationScenario) object(t *testing.T) *v1alpha1.ServiceUserRotation {
	t.Helper()
	cr := &v1alpha1.ServiceUserRotation{}
	require.NoError(t, s.client.Get(t.Context(), crclient.ObjectKeyFromObject(s.rotation), cr))
	return cr
}

func (s *rotationScenario) secretKey() crclient.ObjectKey {
	return crclient.ObjectKey{Namespace: s.rotation.Namespace, Name: s.rotation.Spec.ConnInfoSecretTarget.Name}
}

func (s *rotationScenario) secret(t *testing.T) *corev1.Secret {
	t.Helper()
	secret := &corev1.Secret{}
	require.NoError(t, s.client.Get(t.Context(), s.secretKey(), secret))
	return secret
}

func (s *rotationScenario) prefix() string {
	if prefix := s.rotation.Spec.ConnInfoSecretTarget.Prefix; prefix != "" {
		return prefix
	}
	return "SERVICEUSER_"
}

func (s *rotationScenario) configure(t *testing.T, edit func(*v1alpha1.ServiceUserRotation)) {
	t.Helper()
	cr := s.object(t)
	edit(cr)
	require.NoError(t, s.client.Update(t.Context(), cr))
	s.rotation = cr
}

func (s *rotationScenario) expectService() {
	observed := *s.service
	observed.Users = slices.Clone(s.service.Users)
	observed.Components = slices.Clone(s.service.Components)
	s.api.EXPECT().
		ServiceGet(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, [][2]string{service.ServiceGetIncludeSecrets(true)}).
		Return(&observed, nil).Once()
}

func (s *rotationScenario) expectCA() {
	s.api.EXPECT().ProjectKmsGetCA(mock.Anything, s.rotation.Spec.Project).Return("project-ca", nil).Once()
}

func rotationPasswordRequest(password string) *service.ServiceUserCredentialsModifyIn {
	return &service.ServiceUserCredentialsModifyIn{
		NewPassword: new(password),
		Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
	}
}

func (s *rotationScenario) expectReset(username, password string) {
	s.api.EXPECT().
		ServiceUserCredentialsModify(mock.Anything, s.rotation.Spec.Project, s.rotation.Spec.ServiceName, username, rotationPasswordRequest(password)).
		Return(&service.ServiceUserCredentialsModifyOut{Users: slices.Clone(s.service.Users)}, nil).Once()
}

func (s *rotationScenario) publishInitial(t *testing.T) {
	t.Helper()
	s.expectService()
	s.expectReset(s.rotation.Spec.Usernames[0], "generated-1")
	s.expectCA()
	res, err := s.reconcile(t)
	require.NoError(t, err)
	require.Equal(t, s.rotation.Spec.RotationInterval.Duration, res.RequeueAfter)
	s.requirePublished(t, s.rotation.Spec.Usernames[0], "generated-1", s.now)
}

func (s *rotationScenario) requirePublished(t *testing.T, username, password string, at time.Time) *corev1.Secret {
	t.Helper()
	secret := s.secret(t)
	require.Equal(t, corev1.SecretTypeOpaque, secret.Type)
	require.True(t, metav1.IsControlledBy(secret, s.rotation))
	require.Equal(t, []byte(username), secret.Data[s.prefix()+"USERNAME"])
	require.Equal(t, []byte(password), secret.Data[s.prefix()+"PASSWORD"])
	require.Equal(t, []byte(at.Format(time.RFC3339Nano)), secret.Data["aiven-rotation-published-at"])
	require.NotContains(t, secret.Data, "aiven-rotation-desired-username")
	require.NotContains(t, secret.Data, "aiven-rotation-desired-password")

	cr := s.object(t)
	require.Equal(t, username, cr.Status.ActiveUsername)
	require.True(t, at.Truncate(time.Second).Equal(cr.Status.LastRotationAt.Time), "last rotation: %s", cr.Status.LastRotationAt)
	next := at.Add(cr.Spec.RotationInterval.Duration).Truncate(time.Second)
	require.True(t, next.Equal(cr.Status.NextRotationAt.Time), "next rotation: %s", cr.Status.NextRotationAt)
	require.NotNil(t, cr.Status.Conditions)
	require.Empty(t, cr.Status.Conditions)
	return secret
}

func (s *rotationScenario) requireFailure(t *testing.T, result ctrlruntime.Result, err error) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, ctrlruntime.Result{}, result)
	cr := s.object(t)
	require.Len(t, cr.Status.Conditions, 1)
	condition := meta.FindStatusCondition(cr.Status.Conditions, ConditionTypeError)
	require.NotNil(t, condition)
	require.Equal(t, metav1.ConditionUnknown, condition.Status)
	require.Equal(t, "ReconcileFailed", condition.Reason)
	require.Equal(t, err.Error(), condition.Message)
	require.Equal(t, cr.Generation, condition.ObservedGeneration)
	require.False(t, condition.LastTransitionTime.IsZero())
	require.Equal(t, []string{"Warning ReconcileFailed " + err.Error()}, recorderEvents(s.recorder))
}

func (s *rotationScenario) useKafka() {
	s.service.ServiceType = "kafka"
	s.service.Components = []service.ComponentOut{
		{Component: "kafka", Host: "kafka.example.com", Port: 9092, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate},
		{Component: "kafka", Host: "sasl.example.com", Port: 9093, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl},
		{Component: "schema_registry", Host: "schema.example.com", Port: 8081},
	}
	for i := range s.service.Users {
		s.service.Users[i].AccessCert = new("observed-certificate-" + s.service.Users[i].Username)
		s.service.Users[i].AccessKey = new("observed-key-" + s.service.Users[i].Username)
	}
}
