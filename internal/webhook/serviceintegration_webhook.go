// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// log is for logging in this package.
var serviceintegrationlog = logf.Log.WithName("serviceintegration-resource")

func SetupServiceIntegrationWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.ServiceIntegration{}).
		WithDefaulter(&ServiceIntegrationWebhook{}).
		WithValidator(&ServiceIntegrationWebhook{}).
		Complete()
}

type ServiceIntegrationWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegration,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,verbs=create;update,versions=v1alpha1,name=mserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ServiceIntegrationWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.ServiceIntegration)
	serviceintegrationlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegration,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,versions=v1alpha1,name=vserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ServiceIntegrationWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ServiceIntegration)
	serviceintegrationlog.Info("validate create", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return nil, err
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.ServiceIntegration)
	serviceintegrationlog.Info("validate update", "name", in.Name)

	// We need the validation here only
	_, err := in.GetUserConfig()
	return nil, err
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceIntegrationWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ServiceIntegration)
	serviceintegrationlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
