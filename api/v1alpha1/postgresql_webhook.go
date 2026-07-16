// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var pglog = logf.Log.WithName("postgresql-resource")

func (in *PostgreSQL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &PostgreSQL{}).
		WithDefaulter(&PostgreSQLWebhook{}).
		WithValidator(&PostgreSQLWebhook{}).
		Complete()
}

type PostgreSQLWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-postgresql,mutating=true,failurePolicy=fail,groups=aiven.io,resources=postgresqls,verbs=create;update,versions=v1alpha1,name=mpg.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *PostgreSQLWebhook) Default(_ context.Context, obj *PostgreSQL) error {
	pglog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-postgresql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=postgresqls,versions=v1alpha1,name=vpg.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateCreate(_ context.Context, obj *PostgreSQL) (admission.Warnings, error) {
	pglog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateUpdate(_ context.Context, _, newObj *PostgreSQL) (admission.Warnings, error) {
	pglog.Info("validate update", "name", newObj.Name)

	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateDelete(_ context.Context, obj *PostgreSQL) (admission.Warnings, error) {
	pglog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete PostgreSQL service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
