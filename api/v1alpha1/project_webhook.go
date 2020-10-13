// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"fmt"
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

// +kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-aiven-io-v1alpha1-project,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io.aiven.io,resources=projects,verbs=create;update,versions=v1alpha1,name=mproject.kb.io

var _ webhook.Defaulter = &Project{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Project) Default() {
	projectlog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-k8s-operator-aiven-io-aiven-io-v1alpha1-project,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io.aiven.io,resources=projects,versions=v1alpha1,name=vproject.kb.io

var _ webhook.Validator = &Project{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateCreate() error {
	projectlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateUpdate(old runtime.Object) error {
	projectlog.Info("validate update", "name", r.Name)

	oldP, ok := old.(*Project)
	if !ok {
		return fmt.Errorf("expect old project object to be a %T instead of %T", oldP, old)
	}

	if r.Spec.Name != oldP.Spec.Name {
		return fmt.Errorf("project name cannot change after creation")
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Project) ValidateDelete() error {
	projectlog.Info("validate delete", "name", r.Name)

	return nil
}
