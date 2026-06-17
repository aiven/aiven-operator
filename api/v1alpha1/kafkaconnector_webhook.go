// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var kafkaconnectorlog = logf.Log.WithName("kafkaconnector-resource")

func (in *KafkaConnector) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		WithDefaulter(&KafkaConnectorWebhook{}).
		WithValidator(&KafkaConnectorWebhook{}).
		Complete()
}

type KafkaConnectorWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaconnector,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaconnectors,verbs=create;update,versions=v1alpha1,name=mkafkaconnector.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &KafkaConnectorWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaConnectorWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*KafkaConnector)
	kafkaconnectorlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkaconnector,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaconnectors,versions=v1alpha1,name=vkafkaconnector.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaConnectorWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectorWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*KafkaConnector)
	kafkaconnectorlog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectorWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*KafkaConnector)
	kafkaconnectorlog.Info("validate update", "name", in.Name)

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectorWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*KafkaConnector)
	kafkaconnectorlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
