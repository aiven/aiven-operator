// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestSecretWatchController_secretDataChanged(t *testing.T) {
	t.Parallel()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	controller := &SecretWatchController{
		Log: ctrl.Log.WithName("test"),
	}

	tests := []struct {
		name     string
		oldData  map[string][]byte
		newData  map[string][]byte
		expected bool
	}{
		{
			name:     "no change in data",
			oldData:  map[string][]byte{"password": []byte("oldpass")},
			newData:  map[string][]byte{"password": []byte("oldpass")},
			expected: false,
		},
		{
			name:     "password changed",
			oldData:  map[string][]byte{"password": []byte("oldpass")},
			newData:  map[string][]byte{"password": []byte("newpass")},
			expected: true,
		},
		{
			name:     "key added",
			oldData:  map[string][]byte{"password": []byte("pass")},
			newData:  map[string][]byte{"password": []byte("pass"), "username": []byte("user")},
			expected: true,
		},
		{
			name:     "key removed",
			oldData:  map[string][]byte{"password": []byte("pass"), "username": []byte("user")},
			newData:  map[string][]byte{"password": []byte("pass")},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldSec := &corev1.Secret{Data: tt.oldData}
			newSec := &corev1.Secret{Data: tt.newData}

			updateEvent := event.UpdateEvent{
				ObjectOld: oldSec,
				ObjectNew: newSec,
			}

			result := controller.secretDataChanged(updateEvent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecretWatchController_getResourcesWithSecretSource(t *testing.T) {
	t.Parallel()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	controller := &SecretWatchController{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Log:    ctrl.Log.WithName("test"),
	}

	resources := controller.getResourcesWithSecretSource()

	foundTypes := make(map[reflect.Type]bool)
	for _, resource := range resources {
		foundTypes[reflect.TypeOf(resource)] = true
	}

	assert.True(t, foundTypes[reflect.TypeOf(&v1alpha1.ServiceUser{})], "should find ServiceUser")
	assert.True(t, foundTypes[reflect.TypeOf(&v1alpha1.ClickhouseUser{})], "should find ClickhouseUser")
}

func TestConnInfoSecretRefIndexFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource client.Object
		expected []string
	}{
		{
			name: "ServiceUser with secretSource",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "my-secret",
						PasswordKey: "password",
					},
				},
			},
			expected: []string{"default/my-secret"},
		},
		{
			name: "ServiceUser with secretSource in different namespace",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "my-secret",
						Namespace:   "other-namespace",
						PasswordKey: "password",
					},
				},
			},
			expected: []string{"other-namespace/my-secret"},
		},
		{
			name: "ServiceUser without secretSource",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: nil,
				},
			},
			expected: nil,
		},
		{
			name: "ClickhouseUser with secretSource",
			resource: &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ch-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "ch-secret",
						PasswordKey: "password",
					},
				},
			},
			expected: []string{"default/ch-secret"},
		},
		{
			name: "ClickhouseUser with cross-namespace secretSource",
			resource: &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ch-user",
					Namespace: "app-namespace",
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "shared-secret",
						Namespace:   "secrets-namespace",
						PasswordKey: "password",
					},
				},
			},
			expected: []string{"secrets-namespace/shared-secret"},
		},
		{
			name: "Non-SecretSourceResource",
			resource: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connInfoSecretRefIndexFunc(tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecretWatchController_resourceMatchesSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource SecretSourceResource
		secret   *corev1.Secret
		expected bool
	}{
		{
			name: "ServiceUser matches secret in same namespace",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "my-secret",
						PasswordKey: "password",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
			},
			expected: true,
		},
		{
			name: "ServiceUser matches secret in different namespace",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "app-ns",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "my-secret",
						Namespace:   "secret-ns",
						PasswordKey: "password",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "secret-ns",
				},
			},
			expected: true,
		},
		{
			name: "ServiceUser does not match different secret name",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "different-secret",
						PasswordKey: "password",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
			},
			expected: false,
		},
		{
			name: "ClickhouseUser matches secret",
			resource: &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ch-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "ch-secret",
						PasswordKey: "password",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ch-secret",
					Namespace: "default",
				},
			},
			expected: true,
		},
		{
			name: "Resource without ConnInfoSecretSource",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: nil,
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-secret",
					Namespace: "default",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretSource := tt.resource.GetConnInfoSecretSource()
			if secretSource == nil {
				assert.False(t, tt.expected)
				return
			}

			sourceNamespace := secretSource.Namespace
			if sourceNamespace == "" {
				sourceNamespace = tt.resource.GetNamespace()
			}

			matches := secretSource.Name == tt.secret.Name && sourceNamespace == tt.secret.Namespace
			assert.Equal(t, tt.expected, matches)
		})
	}
}

func TestSecretWatchController_triggerReconciliation(t *testing.T) {
	t.Parallel()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	tests := []struct {
		name     string
		resource SecretSourceResource
		wantErr  bool
	}{
		{
			name: "trigger reconciliation for ServiceUser",
			resource: &v1alpha1.ServiceUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ServiceUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "my-secret",
						PasswordKey: "password",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "trigger reconciliation for ClickhouseUser",
			resource: &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ch-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "ch-secret",
						PasswordKey: "password",
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.resource).
				Build()

			controller := &SecretWatchController{
				Client: fakeClient,
				Log:    ctrl.Log.WithName("test"),
			}

			ctx := context.Background()
			err := controller.triggerReconciliation(ctx, tt.resource)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// verify the annotation was added
				updatedResource := tt.resource.DeepCopyObject().(client.Object)
				err = fakeClient.Get(ctx, client.ObjectKeyFromObject(tt.resource), updatedResource)
				require.NoError(t, err)

				annotations := updatedResource.GetAnnotations()
				assert.NotNil(t, annotations)
				assert.Contains(t, annotations, "controllers.aiven.io/secret-source-updated")
				assert.NotEmpty(t, annotations["controllers.aiven.io/secret-source-updated"])

				// verify processed generation annotation was cleared
				assert.NotContains(t, annotations, "controllers.aiven.io/generation-was-processed")
			}
		})
	}
}

// TestAnnotationHandling tests annotation management scenarios
func TestAnnotationHandling(t *testing.T) {
	t.Parallel()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Preserve existing annotations", func(t *testing.T) {
		namespace := "test-namespace"
		secretName := "annotation-test-secret"

		sourceSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"password": []byte("annotation-password-123"),
			},
		}

		serviceUser := &v1alpha1.ServiceUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "annotation-service-user",
				Namespace: namespace,
				Annotations: map[string]string{
					"existing-annotation": "existing-value",
				},
			},
			Spec: v1alpha1.ServiceUserSpec{
				ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
					Name:        secretName,
					PasswordKey: "password",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(sourceSecret, serviceUser).
			Build()

		controller := &SecretWatchController{
			Client: fakeClient,
			Log:    ctrl.Log.WithName("test"),
		}

		ctx := context.Background()
		err := controller.triggerReconciliation(ctx, serviceUser)
		require.NoError(t, err)

		updatedServiceUser := &v1alpha1.ServiceUser{}
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceUser), updatedServiceUser)
		require.NoError(t, err)

		annotations := updatedServiceUser.GetAnnotations()
		assert.NotNil(t, annotations)
		assert.Equal(t, "existing-value", annotations["existing-annotation"])
		assert.Contains(t, annotations, "controllers.aiven.io/secret-source-updated")
		assert.NotEmpty(t, annotations["controllers.aiven.io/secret-source-updated"])

		// verify the processed generation annotation was cleared
		assert.NotContains(t, annotations, "controllers.aiven.io/generation-was-processed")
	})

	t.Run("Valid timestamp format", func(t *testing.T) {
		namespace := "test-namespace"

		serviceUser := &v1alpha1.ServiceUser{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "timestamp-service-user",
				Namespace: namespace,
			},
			Spec: v1alpha1.ServiceUserSpec{
				ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
					Name:        "test-secret",
					PasswordKey: "password",
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(serviceUser).
			Build()

		controller := &SecretWatchController{
			Client: fakeClient,
			Log:    ctrl.Log.WithName("test"),
		}

		ctx := context.Background()
		err := controller.triggerReconciliation(ctx, serviceUser)
		require.NoError(t, err)

		updatedServiceUser := &v1alpha1.ServiceUser{}
		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(serviceUser), updatedServiceUser)
		require.NoError(t, err)

		annotations := updatedServiceUser.GetAnnotations()
		require.NotNil(t, annotations)

		timestampStr := annotations["controllers.aiven.io/secret-source-updated"]
		assert.NotEmpty(t, timestampStr)
		assert.Regexp(t, `^\d+$`, timestampStr, "Timestamp should be numeric")
	})
}
