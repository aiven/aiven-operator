// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

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
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaconnector,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaconnectors,verbs=create;update,versions=v1alpha1,name=mkafkaconnector.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaConnector{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *KafkaConnector) Default() {
	kafkaconnectorlog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkaconnector,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaconnectors,versions=v1alpha1,name=vkafkaconnector.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaConnector{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaConnector) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkaconnectorlog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaConnector) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	kafkaconnectorlog.Info("validate update", "name", in.Name)

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaConnector) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkaconnectorlog.Info("validate delete", "name", in.Name)

	return nil, nil
}
