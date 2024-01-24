// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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

var _ webhook.CustomValidator = &Database{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *Database) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	databaselog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *Database) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	databaselog.Info("validate update", "name", in.Name)

	if in.Spec.Project != oldObj.(*Database).Spec.Project {
		return nil, errors.New("cannot update a Database, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != oldObj.(*Database).Spec.ServiceName {
		return nil, errors.New("cannot update a Database, service_name field is immutable and cannot be updated")
	}

	if in.Spec.LcCollate != oldObj.(*Database).Spec.LcCollate {
		return nil, errors.New("cannot update a Database, lc_collate field is immutable and cannot be updated")
	}

	if in.Spec.LcCtype != oldObj.(*Database).Spec.LcCtype {
		return nil, errors.New("cannot update a Database, lc_ctype field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (in *Database) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	databaselog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Database, termination protection is on")
	}
	return nil, nil
}
