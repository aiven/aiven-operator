// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceuserlog = logf.Log.WithName("serviceuser-resource")

func (in *ServiceUser) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceuser,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceusers,verbs=create;update,versions=v1alpha1,name=mserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceUser{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ServiceUser) Default() {
	serviceuserlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceuser,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceusers,versions=v1alpha1,name=vserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceUser{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceUser) ValidateCreate() (admission.Warnings, error) {
	serviceuserlog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceUser) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	serviceuserlog.Info("validate update", "name", in.Name)
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceUser) ValidateDelete() (admission.Warnings, error) {
	serviceuserlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
