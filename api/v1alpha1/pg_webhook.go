// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var pglog = logf.Log.WithName("pg-resource")

func (r *PG) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-v1alpha1-pg,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=pgs,verbs=create;update,versions=v1alpha1,name=mpg.kb.io

var _ webhook.Defaulter = &PG{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *PG) Default() {
	pglog.Info("default", "name", r.Name)
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-k8s-operator-aiven-io-v1alpha1-pg,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=pgs,versions=v1alpha1,name=vpg.kb.io

var _ webhook.Validator = &PG{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *PG) ValidateCreate() error {
	pglog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *PG) ValidateUpdate(old runtime.Object) error {
	pglog.Info("validate update", "name", r.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *PG) ValidateDelete() error {
	pglog.Info("validate delete", "name", r.Name)

	return nil
}
