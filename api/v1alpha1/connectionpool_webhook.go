// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var connectionpoollog = logf.Log.WithName("connectionpool-resource")

func (r *ConnectionPool) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-v1alpha1-connectionpool,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=connectionpools,verbs=create;update,versions=v1alpha1,name=mconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ConnectionPool{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ConnectionPool) Default() {
	connectionpoollog.Info("default", "name", r.Name)

	if r.Spec.PoolSize == 0 {
		r.Spec.PoolSize = 10
	}
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-k8s-operator-aiven-io-v1alpha1-connectionpool,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=connectionpools,versions=v1alpha1,name=vconnectionpool.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ConnectionPool{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ConnectionPool) ValidateCreate() error {
	connectionpoollog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ConnectionPool) ValidateUpdate(old runtime.Object) error {
	connectionpoollog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*ConnectionPool).Spec.Project {
		return errors.New("cannot update a ConnectionPool, project field is immutable and cannot be updated")
	}

	if r.Spec.ServiceName != old.(*ConnectionPool).Spec.ServiceName {
		return errors.New("cannot update a ConnectionPool, serviceName field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ConnectionPool) ValidateDelete() error {
	connectionpoollog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
