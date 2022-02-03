// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var kafkalog = logf.Log.WithName("kafka-resource")

func (r *Kafka) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafka,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkas,verbs=create;update,versions=v1alpha1,name=mkafka.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Kafka{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Kafka) Default() {
	kafkalog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafka,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkas,versions=v1alpha1,name=vkafka.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Kafka{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Kafka) ValidateCreate() error {
	kafkalog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Kafka) ValidateUpdate(old runtime.Object) error {
	kafkalog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*Kafka).Spec.Project {
		return errors.New("cannot update a Kafka service, project field is immutable and cannot be updated")
	}

	if r.Spec.ConnInfoSecretTarget.Name != old.(*Kafka).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Kafka service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Kafka) ValidateDelete() error {
	kafkalog.Info("validate delete", "name", r.Name)

	if r.Spec.TerminationProtection {
		return errors.New("cannot delete Kafka service, termination protection is on")
	}

	return nil
}
