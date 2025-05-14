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
var databaselog = logf.Log.WithName("database-resource")

func (in *Database) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-database,mutating=true,failurePolicy=fail,groups=aiven.io,resources=databases,verbs=create;update,versions=v1alpha1,name=mdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Database{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Database) Default() {
	databaselog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-database,mutating=false,failurePolicy=fail,groups=aiven.io,resources=databases,versions=v1alpha1,name=vdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Database{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Database) ValidateCreate() error {
	databaselog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Database) ValidateUpdate(_ runtime.Object) error {
	databaselog.Info("validate update", "name", in.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Database) ValidateDelete() error {
	databaselog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete Database, termination protection is on")
	}
	return nil
}
