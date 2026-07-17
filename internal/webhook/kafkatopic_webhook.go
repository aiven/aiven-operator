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
var kafkatopiclog = logf.Log.WithName("kafkatopic-resource")

func SetupKafkaTopicWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.KafkaTopic{}).
		WithDefaulter(&KafkaTopicWebhook{}).
		WithValidator(&KafkaTopicWebhook{}).
		Complete()
}

type KafkaTopicWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkatopic,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,verbs=create;update,versions=v1alpha1,name=mkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &KafkaTopicWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaTopicWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.KafkaTopic)
	kafkatopiclog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkatopic,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,versions=v1alpha1,name=vkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaTopicWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaTopic)
	kafkatopiclog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.KafkaTopic)
	old := oldObj.(*v1alpha1.KafkaTopic)
	kafkatopiclog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.Spec.Project {
		return nil, errors.New("cannot update a KafkaTopic, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != old.Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaTopic, serviceName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaTopicWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaTopic)
	kafkatopiclog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete KafkaTopic, termination protection is on")
	}

	return nil, nil
}
