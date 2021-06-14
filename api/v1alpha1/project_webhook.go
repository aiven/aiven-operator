// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var projectlog = logf.Log.WithName("project-resource")

func (r *Project) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-project,mutating=true,failurePolicy=fail,groups=aiven.io,resources=projects,verbs=create;update,versions=v1alpha1,name=mproject.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Project{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Project) Default() {
	projectlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-project,mutating=false,failurePolicy=fail,groups=aiven.io,resources=projects,versions=v1alpha1,name=vproject.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Project{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateCreate() error {
	projectlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateUpdate(old runtime.Object) error {
	projectlog.Info("validate update", "name", r.Name)

	if r.Spec.CopyFromProject != old.(*Project).Spec.CopyFromProject {
		return errors.New("'copyFromProject' can only be set during creation of a project")
	}

	if r.Spec.ConnInfoSecretTarget.Name != old.(*Project).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Project, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateDelete() error {
	projectlog.Info("validate delete", "name", r.Name)

	if r.Status.AccountID == "" && r.Status.EstimatedBalance != "0.00" {
		return errors.New("project with an open balance cannot be deleted")
	}

	return nil
}
