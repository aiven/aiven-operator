// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var serviceintegrationlog = logf.Log.WithName("serviceintegration-resource")

func (in *ServiceIntegration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegration,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,verbs=create;update,versions=v1alpha1,name=mserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceIntegration{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ServiceIntegration) Default() {
	serviceintegrationlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegration,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,versions=v1alpha1,name=vserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceIntegration{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegration) ValidateCreate() error {
	serviceintegrationlog.Info("validate create", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return err
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegration) ValidateUpdate(old runtime.Object) error {
	serviceintegrationlog.Info("validate update", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return err
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegration) ValidateDelete() error {
	serviceintegrationlog.Info("validate delete", "name", in.Name)

	return nil
}
