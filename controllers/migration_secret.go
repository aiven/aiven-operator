package controllers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// parseMigrationPort parses the "port" value from a migration secret.
func parseMigrationPort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid or missing port in migration secret: %w", err)
	}
	return port, nil
}

// readMigrationSecret fetches the Secret referenced by MigrationSecretSource
// and returns its data as strings keyed by secret key name.
// Values are rejected if they contain leading or trailing whitespace.
func readMigrationSecret(ctx context.Context, k8s client.Reader, namespace string, ref *v1alpha1.MigrationSecretSource) (map[string]string, error) {
	secret := &corev1.Secret{}
	if err := k8s.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, secret); err != nil {
		return nil, fmt.Errorf("cannot get migration secret %q: %w", ref.Name, err)
	}

	data := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		s := string(v)
		if s != strings.TrimSpace(s) {
			return nil, fmt.Errorf("migration secret %q: key %q has leading or trailing whitespace; remove it from the Secret", ref.Name, k)
		}
		data[k] = s
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

// markMigrationApplied records that the migration config was sent to Aiven, unless a migration condition exists.
func markMigrationApplied(conditions *[]metav1.Condition, generation int64) {
	if meta.FindStatusCondition(*conditions, v1alpha1.ConditionTypeMigrationComplete) != nil {
		return
	}
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               v1alpha1.ConditionTypeMigrationComplete,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             v1alpha1.MigrationReasonInProgress,
		Message:            "Migration config applied",
	})
}

// updateMigrationCondition sets MigrationComplete from the migration status at Aiven.
func updateMigrationCondition(ctx context.Context, avnGen avngen.Client, project, name string, generation int64, conditions *[]metav1.Condition) error {
	migrationStatus, err := avnGen.ServiceGetMigrationStatus(ctx, project, name)
	if err != nil {
		if isNotFound(err) {
			// 404 means no active migration: done once the migration config was applied, not started otherwise.
			if meta.FindStatusCondition(*conditions, v1alpha1.ConditionTypeMigrationComplete) == nil {
				return nil
			}
			meta.SetStatusCondition(conditions, metav1.Condition{
				Type:               v1alpha1.ConditionTypeMigrationComplete,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: generation,
				Reason:             v1alpha1.MigrationReasonDone,
				Message:            "Migration completed (no active migration found)",
			})
			return nil
		}

		return fmt.Errorf("failed to get migration status: %w", err)
	}

	condition := metav1.Condition{
		Type:               v1alpha1.ConditionTypeMigrationComplete,
		ObservedGeneration: generation,
	}

	switch migrationStatus.Migration.Status {
	case service.MigrationStatusTypeDone:
		condition.Status = metav1.ConditionTrue
		condition.Reason = v1alpha1.MigrationReasonDone
		condition.Message = "Migration completed successfully"
	case service.MigrationStatusTypeFailed:
		condition.Status = metav1.ConditionFalse
		condition.Reason = v1alpha1.MigrationReasonFailed
		condition.Message = "Migration failed"
		if migrationStatus.Migration.Error != nil {
			condition.Message = *migrationStatus.Migration.Error
		}
	default:
		condition.Status = metav1.ConditionFalse
		condition.Reason = v1alpha1.MigrationReasonInProgress
		condition.Message = fmt.Sprintf("Migration status: %s", migrationStatus.Migration.Status)
	}

	meta.SetStatusCondition(conditions, condition)
	return nil
}
