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
var kafkaconnectlog = logf.Log.WithName("kafkaconnect-resource")

func SetupKafkaConnectWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.KafkaConnect{}).
		WithDefaulter(&KafkaConnectWebhook{}).
		WithValidator(&KafkaConnectWebhook{}).
		Complete()
}

type KafkaConnectWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaconnect,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,verbs=create;update,versions=v1alpha1,name=mkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &KafkaConnectWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaConnectWebhook) Default(_ context.Context, obj runtime.Object) error {
	in := obj.(*v1alpha1.KafkaConnect)
	kafkaconnectlog.Info("default", "name", in.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkaconnect,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,versions=v1alpha1,name=vkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaConnectWebhook{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaConnect)
	kafkaconnectlog.Info("validate create", "name", in.Name)

	return nil, in.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	in := newObj.(*v1alpha1.KafkaConnect)
	old := oldObj.(*v1alpha1.KafkaConnect)
	kafkaconnectlog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.Spec.Project {
		return nil, errors.New("cannot update a KafkaConnect service, project field is immutable and cannot be updated")
	}

	return nil, in.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	in := obj.(*v1alpha1.KafkaConnect)
	kafkaconnectlog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete KafkaConnect service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
