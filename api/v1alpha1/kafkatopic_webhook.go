// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var kafkatopiclog = logf.Log.WithName("kafkatopic-resource")

func (in *KafkaTopic) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-kafkatopic,mutating=true,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,verbs=create;update,versions=v1alpha1,name=mkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &KafkaTopic{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *KafkaTopic) Default() {
	kafkatopiclog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-kafkatopic,mutating=false,failurePolicy=fail,groups=aiven.io,resources=kafkatopics,versions=v1alpha1,name=vkafkatopic.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.CustomValidator = &KafkaTopic{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkatopiclog.Info("validate create", "name", in.Name)

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	kafkatopiclog.Info("validate update", "name", in.Name)

	if in.Spec.Project != oldObj.(*KafkaTopic).Spec.Project {
		return nil, errors.New("cannot update a KafkaTopic, project field is immutable and cannot be updated")
	}

	if in.Spec.ServiceName != oldObj.(*KafkaTopic).Spec.ServiceName {
		return nil, errors.New("cannot update a KafkaTopic, serviceName field is immutable and cannot be updated")
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (in *KafkaTopic) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	kafkatopiclog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return nil, errors.New("cannot delete KafkaTopic, termination protection is on")
	}

	return nil, nil
}
