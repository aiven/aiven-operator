package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// parseMigrationPort parses the "port" value from a migration secret. The error
// names the offending Secret so the reconciler can surface a useful message.
func parseMigrationPort(secretName, raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("migration secret %q: invalid or missing port: %w", secretName, err)
	}
	return port, nil
}

// readMigrationSecret fetches the Secret referenced by MigrationSecretSource
// and returns its data as strings keyed by secret key name.
// Note: all values are trimmed of leading and trailing whitespace.
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

// deleteMigrationSecret removes the Secret referenced by MigrationSecretSource.
// NotFound is treated as success so the operation is idempotent across reconciles.
func deleteMigrationSecret(ctx context.Context, k8s client.Writer, namespace string, ref *v1alpha1.MigrationSecretSource) error {
	if ref == nil {
		return nil
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: ref.Name, Namespace: namespace},
	}
	if err := k8s.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete migration secret %q: %w", ref.Name, err)
	}
	return nil
}

// maybeDeleteMigrationSecret deletes the referenced migration Secret when
// DeleteAfterMigration is opted in and the MigrationComplete condition is True. No-op otherwise.
func (h *genericServiceHandler) maybeDeleteMigrationSecret(ctx context.Context, o serviceAdapter, mp migrationSecretProvider) error {
	src := mp.getMigrationSecretSource()
	if src == nil || !src.DeleteAfterMigration {
		return nil
	}
	if !meta.IsStatusConditionTrue(o.getServiceStatus().Conditions, v1alpha1.ConditionTypeMigrationComplete) {
		return nil
	}
	if err := mp.deleteMigrationSecret(ctx); err != nil {
		return err
	}
	h.log.Info("migration secret deleted",
		"name", src.Name,
		"namespace", o.getObjectMeta().Namespace,
		"resource", o.getObjectMeta().Name,
	)
	return nil
}
