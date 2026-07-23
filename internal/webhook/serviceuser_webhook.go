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
var serviceuserlog = logf.Log.WithName("serviceuser-resource")

func SetupServiceUserWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.ServiceUser{}).
		WithDefaulter(&ServiceUserWebhook{}).
		WithValidator(&ServiceUserWebhook{}).
		Complete()
}

type ServiceUserWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceuser,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceusers,verbs=create;update,versions=v1alpha1,name=mserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ServiceUserWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ServiceUserWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.ServiceUser)
	serviceuserlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceuser,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceusers,versions=v1alpha1,name=vserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ServiceUserWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ServiceUser)
	serviceuserlog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.ServiceUser)
	serviceuserlog.Info("validate update", "name", in.Name)
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ServiceUserWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ServiceUser)
	serviceuserlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
