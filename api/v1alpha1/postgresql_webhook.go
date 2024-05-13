// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var pglog = logf.Log.WithName("postgresql-resource")

func (in *PostgreSQL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-postgresql,mutating=true,failurePolicy=fail,groups=aiven.io,resources=postgresqls,verbs=create;update,versions=v1alpha1,name=mpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &PostgreSQL{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *PostgreSQL) Default() {
	pglog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-postgresql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=postgresqls,versions=v1alpha1,name=vpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &PostgreSQL{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *PostgreSQL) ValidateCreate() error {
	pglog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *PostgreSQL) ValidateUpdate(old runtime.Object) error {
	pglog.Info("validate update", "name", in.Name)
	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *PostgreSQL) ValidateDelete() error {
	pglog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete PostgreSQL service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
