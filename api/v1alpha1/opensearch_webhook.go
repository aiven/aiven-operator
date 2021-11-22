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
var opensearchlog = logf.Log.WithName("opensearch-resource")

func (r *OpenSearch) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-opensearch,mutating=true,failurePolicy=fail,groups=aiven.io,resources=opensearches,verbs=create;update,versions=v1alpha1,name=mopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenSearch{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OpenSearch) Default() {
	opensearchlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-opensearch,mutating=false,failurePolicy=fail,groups=aiven.io,resources=opensearches,versions=v1alpha1,name=vopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &OpenSearch{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenSearch) ValidateCreate() error {
	opensearchlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OpenSearch) ValidateUpdate(old runtime.Object) error {
	opensearchlog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*OpenSearch).Spec.Project {
		return errors.New("cannot update a OpenSearch service, project field is immutable and cannot be updated")
	}

	if r.Spec.ConnInfoSecretTarget.Name != old.(*OpenSearch).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a OpenSearch service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OpenSearch) ValidateDelete() error {
	opensearchlog.Info("validate delete", "name", r.Name)

	if r.Spec.TerminationProtection {
		return errors.New("cannot delete OpenSearch service, termination protection is on")
	}

	return nil
}
