// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

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

func (in *OpenSearch) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-opensearch,mutating=true,failurePolicy=fail,groups=aiven.io,resources=opensearches,verbs=create;update,versions=v1alpha1,name=mopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &OpenSearch{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *OpenSearch) Default() {
	opensearchlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-opensearch,mutating=false,failurePolicy=fail,groups=aiven.io,resources=opensearches,versions=v1alpha1,name=vopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &OpenSearch{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *OpenSearch) ValidateCreate() error {
	opensearchlog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *OpenSearch) ValidateUpdate(_ runtime.Object) error {
	opensearchlog.Info("validate update", "name", in.Name)
	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *OpenSearch) ValidateDelete() error {
	opensearchlog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete OpenSearch service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
