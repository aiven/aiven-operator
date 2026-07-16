// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceintegrationlog = logf.Log.WithName("serviceintegration-resource")

func (in *ServiceIntegration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &ServiceIntegration{}).
		WithDefaulter(&ServiceIntegrationWebhook{}).
		WithValidator(&ServiceIntegrationWebhook{}).
		Complete()
}

type ServiceIntegrationWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegration,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,verbs=create;update,versions=v1alpha1,name=mserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) Default(_ context.Context, obj *ServiceIntegration) error {
	serviceintegrationlog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegration,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,versions=v1alpha1,name=vserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateCreate(_ context.Context, obj *ServiceIntegration) (admission.Warnings, error) {
	serviceintegrationlog.Info("validate create", "name", obj.Name)

	// We need the validation here only
	_, err := obj.GetUserConfig()
	return nil, err
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateUpdate(_ context.Context, _, newObj *ServiceIntegration) (admission.Warnings, error) {
	serviceintegrationlog.Info("validate update", "name", newObj.Name)

	// We need the validation here only
	_, err := newObj.GetUserConfig()
	return nil, err
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateDelete(_ context.Context, obj *ServiceIntegration) (admission.Warnings, error) {
	serviceintegrationlog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
