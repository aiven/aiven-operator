// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var kafkaconnectlog = logf.Log.WithName("kafkaconnect-resource")

func (r *KafkaConnect) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-v1alpha1-kafkaconnect,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=kafkaconnects,verbs=create;update,versions=v1alpha1,name=mkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaConnect{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KafkaConnect) Default() {
	kafkaconnectlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-k8s-operator-aiven-io-v1alpha1-kafkaconnect,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=kafkaconnects,versions=v1alpha1,name=vkafkaconnect.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaConnect{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaConnect) ValidateCreate() error {
	kafkaconnectlog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaConnect) ValidateUpdate(old runtime.Object) error {
	kafkaconnectlog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*KafkaConnect).Spec.Project {
		return errors.New("cannot update a KafkaConnect service, project field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaConnect) ValidateDelete() error {
	kafkaconnectlog.Info("validate delete", "name", r.Name)

	if r.Spec.TerminationProtection {
		return errors.New("cannot delete KafkaConnect service, termination protection is on")
	}

	return nil
}
