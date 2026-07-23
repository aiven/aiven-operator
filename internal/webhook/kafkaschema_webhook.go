// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// log is for logging in this package.
var kafkaschemalog = logf.Log.WithName("kafkaschema-resource")

func SetupKafkaSchemaWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.KafkaSchema{}).
		WithDefaulter(&KafkaSchemaWebhook{}).
		WithValidator(&KafkaSchemaWebhook{}).
		Complete()
}

type KafkaSchemaWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaschema,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,verbs=create;update,versions=v1alpha1,name=mkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &KafkaSchemaWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.KafkaSchema)
	kafkaschemalog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaschema,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaschemas,versions=v1alpha1,name=vkafkaschema.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaSchemaWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaSchema)
	kafkaschemalog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.KafkaSchema)
	old := oldObj.(*v1alpha1.KafkaSchema)
	kafkaschemalog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.Spec.Project {
		return nil, errors.New("cannot update a KafkaSchema, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != old.Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaSchema, serviceName field is immutable and cannot be updated")
	}

	if in.Spec.SubjectName != old.Spec.SubjectName {
		return nil, errors.New("cannot update a KafkaSchema, subjectName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaSchemaWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaSchema)
	kafkaschemalog.Info("validate delete", "name", in.Name)

	return nil, nil
}
