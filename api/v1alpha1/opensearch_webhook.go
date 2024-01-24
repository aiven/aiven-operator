// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

var _ webhook.CustomValidator = &OpenSearch{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *OpenSearch) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	opensearchlog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *OpenSearch) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	opensearchlog.Info("validate update", "name", in.Name)

	if in.Spec.Project != oldObj.(*OpenSearch).Spec.Project {
		return nil, errors.New("cannot update a OpenSearch service, project field is immutable and cannot be updated")
	}

	if in.Spec.ConnInfoSecretTarget.Name != oldObj.(*OpenSearch).Spec.ConnInfoSecretTarget.Name {
		return nil, errors.New("cannot update a OpenSearch service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (in *OpenSearch) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	opensearchlog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete OpenSearch service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
