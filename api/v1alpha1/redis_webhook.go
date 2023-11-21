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
var redislog = logf.Log.WithName("redis-resource")

func (in *Redis) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-redis,mutating=true,failurePolicy=fail,groups=aiven.io,resources=redis,verbs=create;update,versions=v1alpha1,name=mredis.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Redis{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Redis) Default() {
	redislog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-redis,mutating=false,failurePolicy=fail,groups=aiven.io,resources=redis,versions=v1alpha1,name=vredis.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Redis{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Redis) ValidateCreate() error {
	redislog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Redis) ValidateUpdate(old runtime.Object) error {
	redislog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*Redis).Spec.Project {
		return errors.New("cannot update a Redis service, project field is immutable and cannot be updated")
	}

	if in.Spec.ConnInfoSecretTarget.Name != old.(*Redis).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Redis service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Redis) ValidateDelete() error {
	redislog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete Redis service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
