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
var kafkalog = logf.Log.WithName("kafka-resource")

func (in *Kafka) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Kafka{}).
		WithDefaulter(&KafkaWebhook{}).
		WithValidator(&KafkaWebhook{}).
		Complete()
}

type KafkaWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafka,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkas,verbs=create;update,versions=v1alpha1,name=mkafka.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaWebhook) Default(_ context.Context, obj *Kafka) error {
	kafkalog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafka,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkas,versions=v1alpha1,name=vkafka.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaWebhook) ValidateCreate(_ context.Context, obj *Kafka) (admission.Warnings, error) {
	kafkalog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaWebhook) ValidateUpdate(_ context.Context, _, newObj *Kafka) (admission.Warnings, error) {
	kafkalog.Info("validate update", "name", newObj.Name)
	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaWebhook) ValidateDelete(_ context.Context, obj *Kafka) (admission.Warnings, error) {
	kafkalog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Kafka service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
