// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var databaselog = logf.Log.WithName("database-resource")

func (in *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		WithDefaulter(&DatabaseWebhook{}).
		WithValidator(&DatabaseWebhook{}).
		Complete()
}

type DatabaseWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-database,mutating=true,failurePolicy=fail,groups=aiven.io,resources=databases,verbs=create;update,versions=v1alpha1,name=mdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &DatabaseWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *DatabaseWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*Database)
	databaselog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-database,mutating=false,failurePolicy=fail,groups=aiven.io,resources=databases,versions=v1alpha1,name=vdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &DatabaseWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*Database)
	databaselog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*Database)
	databaselog.Info("validate update", "name", in.Name)

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *DatabaseWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*Database)
	databaselog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Database, termination protection is on")
	}
	return nil, nil
}
