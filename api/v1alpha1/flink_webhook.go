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
var flinklog = logf.Log.WithName("flink-resource")

func (in *Flink) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-flink,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=flinks,verbs=create;update,versions=v1alpha1,name=mflink.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Flink{}

func (in *Flink) Default() {
	flinklog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-flink,mutating=false,failurePolicy=fail,groups=aiven.io,resources=flinks,versions=v1alpha1,name=vflink.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Flink{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Flink) ValidateCreate() (admission.Warnings, error) {
	flinklog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Flink) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	flinklog.Info("validate update", "name", in.Name)
	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Flink) ValidateDelete() (admission.Warnings, error) {
	flinklog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Flink service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
