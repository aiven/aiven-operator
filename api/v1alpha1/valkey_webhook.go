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
var valkeylog = logf.Log.WithName("valkey-resource")

func (in *Valkey) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Valkey{}).
		WithDefaulter(&ValkeyWebhook{}).
		WithValidator(&ValkeyWebhook{}).
		Complete()
}

type ValkeyWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-valkey,mutating=true,failurePolicy=fail,groups=aiven.io,resources=valkeys,verbs=create;update,versions=v1alpha1,name=mvalkey.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ValkeyWebhook) Default(_ context.Context, obj *Valkey) error {
	valkeylog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-valkey,mutating=false,failurePolicy=fail,groups=aiven.io,resources=valkeys,versions=v1alpha1,name=vvalkey.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ValkeyWebhook) ValidateCreate(_ context.Context, obj *Valkey) (admission.Warnings, error) {
	valkeylog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ValkeyWebhook) ValidateUpdate(_ context.Context, _, newObj *Valkey) (admission.Warnings, error) {
	valkeylog.Info("validate update", "name", newObj.Name)
	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ValkeyWebhook) ValidateDelete(_ context.Context, obj *Valkey) (admission.Warnings, error) {
	valkeylog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Valkey service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
