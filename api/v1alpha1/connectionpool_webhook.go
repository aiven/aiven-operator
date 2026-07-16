// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var connectionpoollog = logf.Log.WithName("connectionpool-resource")

func (in *ConnectionPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &ConnectionPool{}).
		WithDefaulter(&ConnectionPoolWebhook{}).
		WithValidator(&ConnectionPoolWebhook{}).
		Complete()
}

type ConnectionPoolWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-connectionpool,mutating=true,failurePolicy=fail,groups=aiven.io,resources=connectionpools,verbs=create;update,versions=v1alpha1,name=mconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) Default(_ context.Context, obj *ConnectionPool) error {
	connectionpoollog.Info("default", "name", obj.Name)

	if obj.Spec.PoolSize == 0 {
		obj.Spec.PoolSize = 10
	}
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-connectionpool,mutating=false,failurePolicy=fail,groups=aiven.io,resources=connectionpools,versions=v1alpha1,name=vconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateCreate(_ context.Context, obj *ConnectionPool) (admission.Warnings, error) {
	connectionpoollog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateUpdate(_ context.Context, _, newObj *ConnectionPool) (admission.Warnings, error) {
	connectionpoollog.Info("validate update", "name", newObj.Name)
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ConnectionPoolWebhook) ValidateDelete(_ context.Context, obj *ConnectionPool) (admission.Warnings, error) {
	connectionpoollog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
