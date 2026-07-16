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
var kafkatopiclog = logf.Log.WithName("kafkatopic-resource")

func (in *KafkaTopic) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &KafkaTopic{}).
		WithDefaulter(&KafkaTopicWebhook{}).
		WithValidator(&KafkaTopicWebhook{}).
		Complete()
}

type KafkaTopicWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkatopic,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,verbs=create;update,versions=v1alpha1,name=mkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaTopicWebhook) Default(_ context.Context, obj *KafkaTopic) error {
	kafkatopiclog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkatopic,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,versions=v1alpha1,name=vkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateCreate(_ context.Context, obj *KafkaTopic) (admission.Warnings, error) {
	kafkatopiclog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateUpdate(_ context.Context, oldObj, newObj *KafkaTopic) (admission.Warnings, error) {
	kafkatopiclog.Info("validate update", "name", newObj.Name)

	if newObj.Spec.Project != oldObj.Spec.Project {
		return nil, errors.New("cannot update a KafkaTopic, project field is immutable and cannot be updated")
	}

	if newObj.Spec.ServiceName != oldObj.Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaTopic, serviceName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateDelete(_ context.Context, obj *KafkaTopic) (admission.Warnings, error) {
	kafkatopiclog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete KafkaTopic, termination protection is on")
	}

	return nil, nil
}
