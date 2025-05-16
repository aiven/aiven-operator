// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var connectionpoollog = logf.Log.WithName("connectionpool-resource")

func (in *ConnectionPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-connectionpool,mutating=true,failurePolicy=fail,groups=aiven.io,resources=connectionpools,verbs=create;update,versions=v1alpha1,name=mconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ConnectionPool{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ConnectionPool) Default() {
	connectionpoollog.Info("default", "name", in.Name)

	if in.Spec.PoolSize == 0 {
		in.Spec.PoolSize = 10
	}
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-connectionpool,mutating=false,failurePolicy=fail,groups=aiven.io,resources=connectionpools,versions=v1alpha1,name=vconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ConnectionPool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ConnectionPool) ValidateCreate() error {
	connectionpoollog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ConnectionPool) ValidateUpdate(_ runtime.Object) error {
	connectionpoollog.Info("validate update", "name", in.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ConnectionPool) ValidateDelete() error {
	connectionpoollog.Info("validate delete", "name", in.Name)

	return nil
}
