// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var valkeylog = logf.Log.WithName("valkey-resource")

func (in *Valkey) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-valkey,mutating=true,failurePolicy=fail,groups=aiven.io,resources=valkeys,verbs=create;update,versions=v1alpha1,name=mvalkey.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Valkey{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Valkey) Default() {
	valkeylog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-valkey,mutating=false,failurePolicy=fail,groups=aiven.io,resources=valkeys,versions=v1alpha1,name=vvalkey.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Valkey{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Valkey) ValidateCreate() (admission.Warnings, error) {
	valkeylog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Valkey) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	valkeylog.Info("validate update", "name", in.Name)
	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Valkey) ValidateDelete() (admission.Warnings, error) {
	valkeylog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Valkey service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
