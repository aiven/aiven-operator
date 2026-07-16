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
var kafkaconnectlog = logf.Log.WithName("kafkaconnect-resource")

func (in *KafkaConnect) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &KafkaConnect{}).
		WithDefaulter(&KafkaConnectWebhook{}).
		WithValidator(&KafkaConnectWebhook{}).
		Complete()
}

type KafkaConnectWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkaconnect,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,verbs=create;update,versions=v1alpha1,name=mkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *KafkaConnectWebhook) Default(_ context.Context, obj *KafkaConnect) error {
	kafkaconnectlog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkaconnect,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkaconnects,versions=v1alpha1,name=vkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateCreate(_ context.Context, obj *KafkaConnect) (admission.Warnings, error) {
	kafkaconnectlog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateUpdate(_ context.Context, oldObj, newObj *KafkaConnect) (admission.Warnings, error) {
	kafkaconnectlog.Info("validate update", "name", newObj.Name)

	if newObj.Spec.Project != oldObj.Spec.Project {
		return nil, errors.New("cannot update a KafkaConnect service, project field is immutable and cannot be updated")
	}

	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *KafkaConnectWebhook) ValidateDelete(_ context.Context, obj *KafkaConnect) (admission.Warnings, error) {
	kafkaconnectlog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete KafkaConnect service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
