// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	alloydbomniUtils "github.com/aiven/aiven-operator/utils/alloydbomni"
)

// log is for logging in this package.
var alloydbomnilog = logf.Log.WithName("alloydbomni-resource")

func (in *AlloyDBOmni) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-alloydbomni,mutating=true,failurePolicy=fail,groups=aiven.io,resources=alloydbomnis,verbs=create;update,versions=v1alpha1,name=malloydbomni.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &AlloyDBOmni{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *AlloyDBOmni) Default() {
	alloydbomnilog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-alloydbomni,mutating=false,failurePolicy=fail,groups=aiven.io,resources=alloydbomnis,versions=v1alpha1,name=valloydbomni.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &AlloyDBOmni{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *AlloyDBOmni) ValidateCreate() (admission.Warnings, error) {
	alloydbomnilog.Info("validate create", "name", in.Name)

	if err := alloydbomniUtils.ValidateServiceAccountCredentials(in.Spec.ServiceAccountCredentials); err != nil {
		return nil, fmt.Errorf("invalid serviceAccountCredentials: %w", err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *AlloyDBOmni) ValidateUpdate(_ runtime.Object) (admission.Warnings, error) {
	alloydbomnilog.Info("validate update", "name", in.Name)

	if err := alloydbomniUtils.ValidateServiceAccountCredentials(in.Spec.ServiceAccountCredentials); err != nil {
		return nil, fmt.Errorf("invalid serviceAccountCredentials: %w", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *AlloyDBOmni) ValidateDelete() (admission.Warnings, error) {
	alloydbomnilog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete AlloyDBOmni service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
