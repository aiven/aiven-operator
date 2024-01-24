// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

var _ webhook.CustomValidator = &KafkaSchema{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkaschemalog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	kafkaschemalog.Info("validate update", "name", in.Name)

	if in.Spec.Project != oldObj.(*KafkaSchema).Spec.Project {
		return nil, errors.New("cannot update a KafkaSchema, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != oldObj.(*KafkaSchema).Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaSchema, serviceName field is immutable and cannot be updated")
	}

	if in.Spec.SubjectName != oldObj.(*KafkaSchema).Spec.SubjectName {
		return nil, errors.New("cannot update a KafkaSchema, subjectName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaSchema) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkaschemalog.Info("validate delete", "name", in.Name)

	return nil, nil
}
