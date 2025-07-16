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
	metav1.Object
	GetConnInfoSecretSource() *v1alpha1.ConnInfoSecretSource
}

// GetPasswordFromSecret retrieves and validates the password from connInfoSecretSource
func GetPasswordFromSecret(ctx context.Context, k8sClient client.Client, resource PasswordSource) (string, error) {
	secretSource := resource.GetConnInfoSecretSource()
	if secretSource == nil {
		return "", nil
	}

	sourceNamespace := secretSource.Namespace
	if sourceNamespace == "" {
		sourceNamespace = resource.GetNamespace()
	}

	sourceSecret := &corev1.Secret{}
	err := k8sClient.Get(ctx, types.NamespacedName{
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
