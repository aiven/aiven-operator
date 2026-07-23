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
var connectionpoollog = logf.Log.WithName("connectionpool-resource")

func SetupConnectionPoolWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.ConnectionPool{}).
		WithDefaulter(&ConnectionPoolWebhook{}).
		WithValidator(&ConnectionPoolWebhook{}).
		Complete()
}

type ConnectionPoolWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-connectionpool,mutating=true,failurePolicy=fail,groups=aiven.io,resources=connectionpools,verbs=create;update,versions=v1alpha1,name=mconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &ConnectionPoolWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.ConnectionPool)
	connectionpoollog.Info("default", "name", in.Name)

	if in.Spec.PoolSize == 0 {
		in.Spec.PoolSize = 10
	}
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-connectionpool,mutating=false,failurePolicy=fail,groups=aiven.io,resources=connectionpools,versions=v1alpha1,name=vconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ConnectionPoolWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ConnectionPool)
	connectionpoollog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.ConnectionPool)
	connectionpoollog.Info("validate update", "name", in.Name)
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.ConnectionPool)
	connectionpoollog.Info("validate delete", "name", in.Name)

	return nil, nil
}
