// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clickhousedatabaselog = logf.Log.WithName("clickhousedatabase-resource")

func (in *ClickhouseDatabase) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-clickhousedatabase,mutating=true,failurePolicy=fail,groups=aiven.io,resources=clickhousedatabases,verbs=create;update,versions=v1alpha1,name=mclickhousedatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ClickhouseDatabase{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *ClickhouseDatabase) Default() {
	clickhousedatabaselog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-clickhousedatabase,mutating=false,failurePolicy=fail,groups=aiven.io,resources=clickhousedatabases,versions=v1alpha1,name=vclickhousedatabase.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ClickhouseDatabase{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *ClickhouseDatabase) ValidateCreate() error {
	clickhousedatabaselog.Info("validate create", "name", in.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *ClickhouseDatabase) ValidateUpdate(old runtime.Object) error {
	clickhousedatabaselog.Info("validate update", "name", in.Name)

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *ClickhouseDatabase) ValidateDelete() error {
	clickhousedatabaselog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete ClickhouseDatabase, termination protection is on")
	}
	return nil
}
