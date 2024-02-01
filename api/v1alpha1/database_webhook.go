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

	const defaultLC = "en_US.UTF-8"

	if in.Spec.LcCtype == "" {
		in.Spec.LcCtype = defaultLC
	}

	if in.Spec.LcCollate == "" {
		in.Spec.LcCollate = defaultLC
	}
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-database,mutating=false,failurePolicy=fail,groups=aiven.io,resources=databases,versions=v1alpha1,name=vdatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Database{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Database) ValidateCreate() error {
	databaselog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Database) ValidateUpdate(old runtime.Object) error {
	databaselog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*Database).Spec.Project {
		return errors.New("cannot update a Database, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != old.(*Database).Spec.ServiceName {
		return errors.New("cannot update a Database, service_name field is immutable and cannot be updated")
	}

	if in.Spec.LcCollate != old.(*Database).Spec.LcCollate {
		return errors.New("cannot update a Database, lc_collate field is immutable and cannot be updated")
	}

	if in.Spec.LcCtype != old.(*Database).Spec.LcCtype {
		return errors.New("cannot update a Database, lc_ctype field is immutable and cannot be updated")
	}

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
