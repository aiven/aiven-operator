// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var kafkatopiclog = logf.Log.WithName("kafkatopic-resource")

func (in *KafkaTopic) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkatopic,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,verbs=create;update,versions=v1alpha1,name=mkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaTopic{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *KafkaTopic) Default() {
	kafkatopiclog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkatopic,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,versions=v1alpha1,name=vkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaTopic{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateCreate() error {
	kafkatopiclog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateUpdate(old runtime.Object) error {
	kafkatopiclog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*KafkaTopic).Spec.Project {
		return errors.New("cannot update a KafkaTopic, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != old.(*KafkaTopic).Spec.ServiceName {
		return errors.New("cannot update a KafkaTopic, serviceName field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateDelete() error {
	kafkatopiclog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete KafkaTopic, termination protection is on")
	}

	return nil
}
