// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var serviceuserlog = logf.Log.WithName("serviceuser-resource")

func (in *ServiceUser) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &ServiceUser{}).
		WithDefaulter(&ServiceUserWebhook{}).
		WithValidator(&ServiceUserWebhook{}).
		Complete()
}

type ServiceUserWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceuser,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceusers,verbs=create;update,versions=v1alpha1,name=mserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ServiceUserWebhook) Default(_ context.Context, obj *ServiceUser) error {
	serviceuserlog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceuser,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceusers,versions=v1alpha1,name=vserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateCreate(_ context.Context, obj *ServiceUser) (admission.Warnings, error) {
	serviceuserlog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateUpdate(_ context.Context, _, newObj *ServiceUser) (admission.Warnings, error) {
	serviceuserlog.Info("validate update", "name", newObj.Name)
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateDelete(_ context.Context, obj *ServiceUser) (admission.Warnings, error) {
	serviceuserlog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
