package controllers

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// readMigrationSecret fetches the Secret referenced by MigrationSecretSource
// and returns its data as trimmed strings keyed by secret key name.
func readMigrationSecret(ctx context.Context, k8s client.Reader, namespace string, ref *v1alpha1.MigrationSecretSource) (map[string]string, error) {
	secret := &corev1.Secret{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("cannot get migration secret %q: %w", ref.Name, err)
	}

	data := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		data[k] = strings.TrimSpace(string(v))
	}
	return data, nil
}
