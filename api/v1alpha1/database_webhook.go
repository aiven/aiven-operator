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
var databaselog = logf.Log.WithName("database-resource")

func (in *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Database{}).
		WithDefaulter(&DatabaseWebhook{}).
		WithValidator(&DatabaseWebhook{}).
		Complete()
}

type DatabaseWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-database,mutating=true,failurePolicy=fail,groups=aiven.io,resources=databases,verbs=create;update,versions=v1alpha1,name=mdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *DatabaseWebhook) Default(_ context.Context, obj *Database) error {
	databaselog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-database,mutating=false,failurePolicy=fail,groups=aiven.io,resources=databases,versions=v1alpha1,name=vdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateCreate(_ context.Context, obj *Database) (admission.Warnings, error) {
	databaselog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateUpdate(_ context.Context, _, newObj *Database) (admission.Warnings, error) {
	databaselog.Info("validate update", "name", newObj.Name)

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateDelete(_ context.Context, obj *Database) (admission.Warnings, error) {
	databaselog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Database, termination protection is on")
	}
	return nil, nil
}
