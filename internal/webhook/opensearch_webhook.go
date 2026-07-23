// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// log is for logging in this package.
var opensearchlog = logf.Log.WithName("opensearch-resource")

func SetupOpenSearchWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.OpenSearch{}).
		WithDefaulter(&OpenSearchWebhook{}).
		WithValidator(&OpenSearchWebhook{}).
		Complete()
}

type OpenSearchWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-opensearch,mutating=true,failurePolicy=fail,groups=aiven.io,resources=opensearches,verbs=create;update,versions=v1alpha1,name=mopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &OpenSearchWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *OpenSearchWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.OpenSearch)
	opensearchlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-opensearch,mutating=false,failurePolicy=fail,groups=aiven.io,resources=opensearches,versions=v1alpha1,name=vopensearch.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &OpenSearchWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.OpenSearch)
	opensearchlog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.OpenSearch)
	opensearchlog.Info("validate update", "name", in.Name)
	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *OpenSearchWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.OpenSearch)
	opensearchlog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete OpenSearch service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
