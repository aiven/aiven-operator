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
var clickhouselog = logf.Log.WithName("clickhouse-resource")

func (in *Clickhouse) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-clickhouse,mutating=true,failurePolicy=fail,groups=aiven.io,resources=clickhouses,verbs=create;update,versions=v1alpha1,name=mclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &Clickhouse{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *Clickhouse) Default() {
	clickhouselog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-clickhouse,mutating=false,failurePolicy=fail,groups=aiven.io,resources=clickhouses,versions=v1alpha1,name=vclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Clickhouse{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Clickhouse) ValidateCreate() error {
	clickhouselog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Clickhouse) ValidateUpdate(old runtime.Object) error {
	clickhouselog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*Clickhouse).Spec.Project {
		return errors.New("cannot update a Clickhouse service, project field is immutable and cannot be updated")
	}

	if in.Spec.ConnInfoSecretTarget.Name != old.(*Clickhouse).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Clickhouse service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Clickhouse) ValidateDelete() error {
	clickhouselog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete Clickhouse service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
