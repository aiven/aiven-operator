// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

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
var flinklog = logf.Log.WithName("flink-resource")

func (in *Flink) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		WithDefaulter(&FlinkWebhook{}).
		WithValidator(&FlinkWebhook{}).
		Complete()
}

type FlinkWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-flink,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=flinks,verbs=create;update,versions=v1alpha1,name=mflink.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &FlinkWebhook{}

func (h *FlinkWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*Flink)
	flinklog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-flink,mutating=false,failurePolicy=fail,groups=aiven.io,resources=flinks,versions=v1alpha1,name=vflink.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &FlinkWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *FlinkWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*Flink)
	flinklog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *FlinkWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*Flink)
	flinklog.Info("validate update", "name", in.Name)
	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *FlinkWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*Flink)
	flinklog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Flink service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
