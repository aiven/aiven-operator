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
var kafkaschemalog = logf.Log.WithName("kafkaschema-resource")

func (in *KafkaSchema) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaschema,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,verbs=create;update,versions=v1alpha1,name=mkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaSchema{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *KafkaSchema) Default() {
	kafkaschemalog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaschema,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,versions=v1alpha1,name=vkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaSchema{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateCreate() error {
	kafkaschemalog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateUpdate(old runtime.Object) error {
	kafkaschemalog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*KafkaSchema).Spec.Project {
		return errors.New("cannot update a KafkaSchema, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != old.(*KafkaSchema).Spec.ServiceName {
		return errors.New("cannot update a KafkaSchema, serviceName field is immutable and cannot be updated")
	}

	if in.Spec.SubjectName != old.(*KafkaSchema).Spec.SubjectName {
		return errors.New("cannot update a KafkaSchema, subjectName field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateDelete() error {
	kafkaschemalog.Info("validate delete", "name", in.Name)

	return nil
}
