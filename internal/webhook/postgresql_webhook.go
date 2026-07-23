// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// log is for logging in this package.
var pglog = logf.Log.WithName("postgresql-resource")

func SetupPostgreSQLWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.PostgreSQL{}).
		WithDefaulter(&PostgreSQLWebhook{}).
		WithValidator(&PostgreSQLWebhook{}).
		Complete()
}

type PostgreSQLWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-postgresql,mutating=true,failurePolicy=fail,groups=aiven.io,resources=postgresqls,verbs=create;update,versions=v1alpha1,name=mpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &PostgreSQLWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *PostgreSQLWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.PostgreSQL)
	pglog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-postgresql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=postgresqls,versions=v1alpha1,name=vpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &PostgreSQLWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.PostgreSQL)
	pglog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.PostgreSQL)
	pglog.Info("validate update", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *PostgreSQLWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.PostgreSQL)
	pglog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete PostgreSQL service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
