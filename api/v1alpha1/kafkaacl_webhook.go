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
var kafkaacllog = logf.Log.WithName("kafkaacl-resource")

func (r *KafkaACL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-k8s-operator-aiven-io-v1alpha1-kafkaacl,mutating=true,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=kafkaacls,verbs=create;update,versions=v1alpha1,name=mkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaACL{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *KafkaACL) Default() {
	kafkaacllog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-k8s-operator-aiven-io-v1alpha1-kafkaacl,mutating=false,failurePolicy=fail,groups=k8s-operator.aiven.io,resources=kafkaacls,versions=v1alpha1,name=vkafkaacl.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &KafkaACL{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaACL) ValidateCreate() error {
	kafkaacllog.Info("validate create", "name", r.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaACL) ValidateUpdate(old runtime.Object) error {
	kafkaacllog.Info("validate update", "name", r.Name)

	return errors.New("cannot update a KafkaACL, it can only be created or deleted")
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *KafkaACL) ValidateDelete() error {
	kafkaacllog.Info("validate delete", "name", r.Name)

	return nil
}
