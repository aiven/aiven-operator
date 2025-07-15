package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// PasswordSource defines an interface for resources that can provide password sources
type PasswordSource interface {
	GetConnInfoSecretSource() *v1alpha1.ConnInfoSecretSource
	GetNamespace() string
	metav1.Object
}

// PasswordModifier defines an interface for different password modification strategies
type PasswordModifier[T PasswordSource] interface {
	ModifyCredentials(ctx context.Context, resource T, password string) error
}

// PasswordManager handles password retrieval and modification for Aiven resources
type PasswordManager[T PasswordSource] struct {
	k8sClient client.Client
	modifier  PasswordModifier[T]
}

// NewPasswordManager creates a new password manager with the given modifier strategy
func NewPasswordManager[T PasswordSource](k8sClient client.Client, modifier PasswordModifier[T]) *PasswordManager[T] {
	return &PasswordManager[T]{
		k8sClient: k8sClient,
		modifier:  modifier,
	}
}

// GetPasswordFromSecret retrieves and validates the password from connInfoSecretSource
func (pm *PasswordManager[T]) GetPasswordFromSecret(ctx context.Context, resource T) (string, error) {
	secretSource := resource.GetConnInfoSecretSource()
	if secretSource == nil {
		return "", nil
	}

	sourceNamespace := secretSource.Namespace
	if sourceNamespace == "" {
		sourceNamespace = resource.GetNamespace()
	}

	sourceSecret := &corev1.Secret{}
	err := pm.k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretSource.Name,
		Namespace: sourceNamespace,
	}, sourceSecret)
	if err != nil {
		return "", fmt.Errorf("failed to read connInfoSecretSource %s/%s: %w", sourceNamespace, secretSource.Name, err)
	}

	passwordBytes, exists := sourceSecret.Data[secretSource.PasswordKey]
	if !exists {
		return "", fmt.Errorf("password not found in source secret %s/%s (expected %s key)", sourceNamespace, secretSource.Name, secretSource.PasswordKey)
	}

	newPassword := string(passwordBytes)

	// validate password length according to API requirements
	if len(newPassword) < 8 || len(newPassword) > 256 {
		return "", fmt.Errorf("password length must be between 8 and 256 characters, got %d characters from source secret %s/%s (key: %s)",
			len(newPassword), sourceNamespace, secretSource.Name, secretSource.PasswordKey)
	}

	return newPassword, nil
}

// ModifyCredentials modifies the credentials using the configured strategy
func (pm *PasswordManager[T]) ModifyCredentials(ctx context.Context, resource T, password string) error {
	return pm.modifier.ModifyCredentials(ctx, resource, password)
}
