// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var kafkaacllog = logf.Log.WithName("kafkaacl-resource")

func (in *KafkaACL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &KafkaACL{}).
		WithDefaulter(&KafkaACLWebhook{}).
		WithValidator(&KafkaACLWebhook{}).
		Complete()
}

type KafkaACLWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaacl,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaacls,verbs=create;update,versions=v1alpha1,name=mkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaACLWebhook) Default(_ context.Context, obj *KafkaACL) error {
	kafkaacllog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaacl,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaacls,versions=v1alpha1,name=vkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateCreate(_ context.Context, obj *KafkaACL) (admission.Warnings, error) {
	kafkaacllog.Info("validate create", "name", obj.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateUpdate(_ context.Context, _, newObj *KafkaACL) (admission.Warnings, error) {
	kafkaacllog.Info("validate update", "name", newObj.Name)

	// TODO: validate that the spec does not get updated; this will fail on the aiven api

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateDelete(_ context.Context, obj *KafkaACL) (admission.Warnings, error) {
	kafkaacllog.Info("validate delete", "name", obj.Name)

	return nil, nil
}
