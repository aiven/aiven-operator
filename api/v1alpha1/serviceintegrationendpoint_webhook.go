// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceintegrationendpointlog = logf.Log.WithName("serviceintegrationendpoint-resource")

func (in *ServiceIntegrationEndpoint) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		WithDefaulter(&ServiceIntegrationEndpointWebhook{}).
		WithValidator(&ServiceIntegrationEndpointWebhook{}).
		Complete()
}

type ServiceIntegrationEndpointWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegrationendpoint,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrationendpoints,verbs=create;update,versions=v1alpha1,name=mserviceintegrationendpoint.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ServiceIntegrationEndpointWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ServiceIntegrationEndpointWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*ServiceIntegrationEndpoint)
	serviceintegrationendpointlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegrationendpoint,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrationendpoints,versions=v1alpha1,name=vserviceintegrationendpoint.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ServiceIntegrationEndpointWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationEndpointWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*ServiceIntegrationEndpoint)
	serviceintegrationendpointlog.Info("validate create", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return nil, err
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationEndpointWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*ServiceIntegrationEndpoint)
	serviceintegrationendpointlog.Info("validate update", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return nil, err
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationEndpointWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*ServiceIntegrationEndpoint)
	serviceintegrationendpointlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
