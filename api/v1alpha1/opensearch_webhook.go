// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var opensearchlog = logf.Log.WithName("opensearch-resource")

func (in *OpenSearch) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &OpenSearch{}).
		WithDefaulter(&OpenSearchWebhook{}).
		WithValidator(&OpenSearchWebhook{}).
		Complete()
}

type OpenSearchWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-opensearch,mutating=true,failurePolicy=fail,groups=aiven.io,resources=opensearches,verbs=create;update,versions=v1alpha1,name=mopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *OpenSearchWebhook) Default(_ context.Context, obj *OpenSearch) error {
	opensearchlog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-opensearch,mutating=false,failurePolicy=fail,groups=aiven.io,resources=opensearches,versions=v1alpha1,name=vopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateCreate(_ context.Context, obj *OpenSearch) (admission.Warnings, error) {
	opensearchlog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateUpdate(_ context.Context, _, newObj *OpenSearch) (admission.Warnings, error) {
	opensearchlog.Info("validate update", "name", newObj.Name)
	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateDelete(_ context.Context, obj *OpenSearch) (admission.Warnings, error) {
	opensearchlog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete OpenSearch service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
