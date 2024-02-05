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
var kafkaconnectlog = logf.Log.WithName("kafkaconnect-resource")

func (in *KafkaConnect) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaconnect,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,verbs=create;update,versions=v1alpha1,name=mkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaConnect{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *KafkaConnect) Default() {
	kafkaconnectlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkaconnect,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,versions=v1alpha1,name=vkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaConnect{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaConnect) ValidateCreate() error {
	kafkaconnectlog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaConnect) ValidateUpdate(old runtime.Object) error {
	kafkaconnectlog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*KafkaConnect).Spec.Project {
		return errors.New("cannot update a KafkaConnect service, project field is immutable and cannot be updated")
	}

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaConnect) ValidateDelete() error {
	kafkaconnectlog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete KafkaConnect service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
