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
var kafkaschemalog = logf.Log.WithName("kafkaschema-resource")

func (in *KafkaSchema) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &KafkaSchema{}).
		WithDefaulter(&KafkaSchemaWebhook{}).
		WithValidator(&KafkaSchemaWebhook{}).
		Complete()
}

type KafkaSchemaWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaschema,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,verbs=create;update,versions=v1alpha1,name=mkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) Default(_ context.Context, obj *KafkaSchema) error {
	kafkaschemalog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaschema,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,versions=v1alpha1,name=vkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateCreate(_ context.Context, obj *KafkaSchema) (admission.Warnings, error) {
	kafkaschemalog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateUpdate(_ context.Context, oldObj, newObj *KafkaSchema) (admission.Warnings, error) {
	kafkaschemalog.Info("validate update", "name", newObj.Name)

	if newObj.Spec.Project != oldObj.Spec.Project {
		return nil, errors.New("cannot update a KafkaSchema, project field is immutable and cannot be updated")
	}

	if newObj.Spec.ServiceName != oldObj.Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaSchema, serviceName field is immutable and cannot be updated")
	}

	if newObj.Spec.SubjectName != oldObj.Spec.SubjectName {
		return nil, errors.New("cannot update a KafkaSchema, subjectName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateDelete(_ context.Context, obj *KafkaSchema) (admission.Warnings, error) {
	kafkaschemalog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
