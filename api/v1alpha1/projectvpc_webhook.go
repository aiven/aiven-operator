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
var projectvpclog = logf.Log.WithName("projectvpc-resource")

func (r *ProjectVPC) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-projectvpc,mutating=true,failurePolicy=fail,groups=aiven.io,resources=projectvpcs,verbs=create;update,versions=v1alpha1,name=mprojectvpc.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ProjectVPC{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ProjectVPC) Default() {
	projectvpclog.Info("default", "name", r.Name)

}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-projectvpc,mutating=false,failurePolicy=fail,groups=aiven.io,resources=projectvpcs,versions=v1alpha1,name=vprojectvpc.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ProjectVPC{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectVPC) ValidateCreate() error {
	projectvpclog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectVPC) ValidateUpdate(old runtime.Object) error {
	projectvpclog.Info("validate update", "name", r.Name)

	return errors.New("project vpc resource cannot be updated")
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ProjectVPC) ValidateDelete() error {
	projectvpclog.Info("validate delete", "name", r.Name)

	return nil
}
