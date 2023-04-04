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
var mysqllog = logf.Log.WithName("mysql-resource")

func (in *MySQL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-mysql,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=mysqls,verbs=create;update,versions=v1alpha1,name=mmysql.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &MySQL{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *MySQL) Default() {
	mysqllog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-mysql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=mysqls,versions=v1alpha1,name=vmysql.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &MySQL{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *MySQL) ValidateCreate() error {
	mysqllog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *MySQL) ValidateUpdate(old runtime.Object) error {
	mysqllog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*MySQL).Spec.Project {
		return errors.New("cannot update a MySQL service, project field is immutable and cannot be updated")
	}

	if in.Spec.ConnInfoSecretTarget.Name != old.(*MySQL).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a MySQL service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *MySQL) ValidateDelete() error {
	mysqllog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete MySQL service, termination protection is on")
	}

	return nil
}
