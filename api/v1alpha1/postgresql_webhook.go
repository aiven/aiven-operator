// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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

func (r *PostgreSQL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-postgresql,mutating=true,failurePolicy=fail,groups=aiven.io,resources=postgresqls,verbs=create;update,versions=v1alpha1,name=mpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &PostgreSQL{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PostgreSQL) Default() {
	pglog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-postgresql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=postgresqls,versions=v1alpha1,name=vpg.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &PostgreSQL{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PostgreSQL) ValidateCreate() error {
	pglog.Info("validate create", "name", r.Name)

	return r.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PostgreSQL) ValidateUpdate(old runtime.Object) error {
	pglog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*PostgreSQL).Spec.Project {
		return errors.New("cannot update a PostgreSQL service, project field is immutable and cannot be updated")
	}

	if r.Spec.ConnInfoSecretTarget.Name != old.(*PostgreSQL).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a PostgreSQL service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return r.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PostgreSQL) ValidateDelete() error {
	pglog.Info("validate delete", "name", r.Name)

	if r.Spec.TerminationProtection != nil && *r.Spec.TerminationProtection {
		return errors.New("cannot delete PostgreSQL service, termination protection is on")
	}

	if r.Spec.ProjectVPCID != "" && r.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
