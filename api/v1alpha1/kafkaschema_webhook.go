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

func (r *KafkaSchema) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaschema,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,verbs=create;update,versions=v1alpha1,name=mkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaSchema{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KafkaSchema) Default() {
	kafkaschemalog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaschema,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,versions=v1alpha1,name=vkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaSchema{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaSchema) ValidateCreate() error {
	kafkaschemalog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaSchema) ValidateUpdate(old runtime.Object) error {
	kafkaschemalog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*KafkaSchema).Spec.Project {
		return errors.New("cannot update a KafkaSchema, project field is immutable and cannot be updated")
	}

	if r.Spec.ServiceName != old.(*KafkaSchema).Spec.ServiceName {
		return errors.New("cannot update a KafkaSchema, serviceName field is immutable and cannot be updated")
	}

	if r.Spec.SubjectName != old.(*KafkaSchema).Spec.SubjectName {
		return errors.New("cannot update a KafkaSchema, subjectName field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaSchema) ValidateDelete() error {
	kafkaschemalog.Info("validate delete", "name", r.Name)

	return nil
}
