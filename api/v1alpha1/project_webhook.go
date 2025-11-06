// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var projectlog = logf.Log.WithName("project-resource")

func (in *Project) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-project,mutating=true,failurePolicy=fail,groups=aiven.io,resources=projects,verbs=create;update,versions=v1alpha1,name=mproject.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Project{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Project) Default() {
	projectlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-project,mutating=false,failurePolicy=fail,groups=aiven.io,resources=projects,versions=v1alpha1,name=vproject.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Project{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Project) ValidateCreate() (admission.Warnings, error) {
	projectlog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Project) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	projectlog.Info("validate update", "name", in.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Project) ValidateDelete() (admission.Warnings, error) {
	projectlog.Info("validate delete", "name", in.Name)

	if in.Spec.AccountID == "" && in.Status.EstimatedBalance != "0.00" {
		return nil, errors.New("project with an open balance cannot be deleted")
	}

	return nil, nil
}
