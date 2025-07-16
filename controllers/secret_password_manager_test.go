package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestPasswordManager_GetPasswordFromSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		secretSource   *v1alpha1.ConnInfoSecretSource
		resourceNS     string
		secret         *corev1.Secret
		expectedResult string
		expectError    bool
	}{
		{
			name:           "No secret source returns empty string",
			secretSource:   nil,
			expectedResult: "",
			expectError:    false,
		},
		{
			name: "Valid password from secret in same namespace",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				PasswordKey: "PASSWORD",
			},
			resourceNS: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"PASSWORD": []byte("ValidPassword123!"),
				},
			},
			expectedResult: "ValidPassword123!",
			expectError:    false,
		},
		{
			name: "Valid password from secret in different namespace",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				Namespace:   "other-ns",
				PasswordKey: "PASSWORD",
			},
			resourceNS: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "other-ns",
				},
				Data: map[string][]byte{
					"PASSWORD": []byte("CrossNSPassword123!"),
				},
			},
			expectedResult: "CrossNSPassword123!",
			expectError:    false,
		},
		{
			name: "Secret not found",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "nonexistent-secret",
				PasswordKey: "PASSWORD",
			},
			resourceNS:     "default",
			secret:         nil,
			expectedResult: "",
			expectError:    true,
		},
		{
			name: "Password key not found in secret",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				PasswordKey: "MISSING_KEY",
			},
			resourceNS: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"OTHER_KEY": []byte("ValidPassword123!"),
				},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name: "Password too short",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				PasswordKey: "PASSWORD",
			},
			resourceNS: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"PASSWORD": []byte("short"),
				},
			},
			expectedResult: "",
			expectError:    true,
		},
		{
			name: "Password too long",
			secretSource: &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				PasswordKey: "PASSWORD",
			},
			resourceNS: "default",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"PASSWORD": []byte(string(make([]byte, 257))), // 257 characters
				},
			},
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var k8sClient client.Client
			if tt.secret != nil {
				k8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.secret).Build()
			} else {
				k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			}

			user := &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: tt.resourceNS,
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: tt.secretSource,
				},
			}

			result, err := GetPasswordFromSecret(context.Background(), k8sClient, user)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestPasswordManager_PasswordValidation(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		passwordLength int
		expectError    bool
	}{
		{
			name:           "Minimum valid password length (8 chars)",
			passwordLength: 8,
			expectError:    false,
		},
		{
			name:           "Maximum valid password length (256 chars)",
			passwordLength: 256,
			expectError:    false,
		},
		{
			name:           "Below minimum length (7 chars)",
			passwordLength: 7,
			expectError:    true,
		},
		{
			name:           "Above maximum length (257 chars)",
			passwordLength: 257,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := string(make([]byte, tt.passwordLength))
			for i := range password {
				password = password[:i] + "a" + password[i+1:]
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"PASSWORD": []byte(password),
				},
			}

			user := &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: &v1alpha1.ConnInfoSecretSource{
						Name:        "test-secret",
						PasswordKey: "PASSWORD",
					},
				},
			}

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(secret).Build()
			result, err := GetPasswordFromSecret(context.Background(), k8sClient, user)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, password, result)
			}
		})
	}
}

func TestPasswordManager_NamespaceResolution(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	tests := []struct {
		name         string
		resourceNS   string
		sourceNS     string
		expectedNS   string
		secretExists bool
	}{
		{
			name:         "Uses resource namespace when source namespace is empty",
			resourceNS:   "resource-ns",
			sourceNS:     "",
			expectedNS:   "resource-ns",
			secretExists: true,
		},
		{
			name:         "Uses source namespace when specified",
			resourceNS:   "resource-ns",
			sourceNS:     "source-ns",
			expectedNS:   "source-ns",
			secretExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var objects []client.Object
			if tt.secretExists {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: tt.expectedNS,
					},
					Data: map[string][]byte{
						"PASSWORD": []byte("ValidPassword123!"),
					},
				}
				objects = append(objects, secret)
			}

			secretSource := &v1alpha1.ConnInfoSecretSource{
				Name:        "test-secret",
				PasswordKey: "PASSWORD",
			}
			if tt.sourceNS != "" {
				secretSource.Namespace = tt.sourceNS
			}

			user := &v1alpha1.ClickhouseUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: tt.resourceNS,
				},
				Spec: v1alpha1.ClickhouseUserSpec{
					ConnInfoSecretSource: secretSource,
				},
			}

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			result, err := GetPasswordFromSecret(context.Background(), k8sClient, user)

			if tt.secretExists {
				require.NoError(t, err)
				assert.Equal(t, "ValidPassword123!", result)
			} else {
				require.Error(t, err)
			}
		})
	}
}
