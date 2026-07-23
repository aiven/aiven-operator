// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package webhook

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// log is for logging in this package.
var kafkaacllog = logf.Log.WithName("kafkaacl-resource")

func SetupKafkaACLWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.KafkaACL{}).
		WithDefaulter(&KafkaACLWebhook{}).
		WithValidator(&KafkaACLWebhook{}).
		Complete()
}

type KafkaACLWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaacl,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaacls,verbs=create;update,versions=v1alpha1,name=mkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &KafkaACLWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaACLWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.KafkaACL)
	kafkaacllog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-kafkaacl,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaacls,versions=v1alpha1,name=vkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaACLWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaACL)
	kafkaacllog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateUpdate(_ context.Context, _, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.KafkaACL)
	kafkaacllog.Info("validate update", "name", in.Name)

	// TODO: validate that the spec does not get updated; this will fail on the aiven api

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaACLWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaACL)
	kafkaacllog.Info("validate delete", "name", in.Name)

	return nil, nil
}
