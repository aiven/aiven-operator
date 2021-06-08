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
var serviceuserlog = logf.Log.WithName("serviceuser-resource")

func (r *ServiceUser) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-v1alpha1-serviceuser,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=serviceusers,verbs=create;update,versions=v1alpha1,name=mserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceUser{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ServiceUser) Default() {
	serviceuserlog.Info("default", "name", r.Name)

}

//+kubebuilder:webhook:verbs=create;update,path=/validate-k8s-operator-aiven-io-v1alpha1-serviceuser,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=serviceusers,versions=v1alpha1,name=vserviceuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceUser{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceUser) ValidateCreate() error {
	serviceuserlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceUser) ValidateUpdate(old runtime.Object) error {
	serviceuserlog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*ConnectionPool).Spec.Project {
		return errors.New("cannot update a Service User, project field is immutable and cannot be updated")
	}

	if r.Spec.ServiceName != old.(*ConnectionPool).Spec.ServiceName {
		return errors.New("cannot update a Service User, serviceName field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceUser) ValidateDelete() error {
	serviceuserlog.Info("validate delete", "name", r.Name)

	return nil
}
