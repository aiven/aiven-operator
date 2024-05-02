// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var serviceintegrationendpointlog = logf.Log.WithName("serviceintegrationendpoint-resource")

func (in *ServiceIntegrationEndpoint) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegrationendpoint,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrationendpoints,verbs=create;update,versions=v1alpha1,name=mserviceintegrationendpoint.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceIntegrationEndpoint{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ServiceIntegrationEndpoint) Default() {
	serviceintegrationendpointlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegrationendpoint,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrationendpoints,versions=v1alpha1,name=vserviceintegrationendpoint.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceIntegrationEndpoint{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegrationEndpoint) ValidateCreate() error {
	serviceintegrationendpointlog.Info("validate create", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return err
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegrationEndpoint) ValidateUpdate(old runtime.Object) error {
	serviceintegrationendpointlog.Info("validate update", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return err
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ServiceIntegrationEndpoint) ValidateDelete() error {
	serviceintegrationendpointlog.Info("validate delete", "name", in.Name)

	return nil
}
