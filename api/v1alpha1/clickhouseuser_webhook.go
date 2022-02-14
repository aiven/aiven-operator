// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var clickhouseuserlog = logf.Log.WithName("clickhouseuser-resource")

func (r *ClickhouseUser) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-clickhouseuser,mutating=true,failurePolicy=fail,groups=aiven.io,resources=clickhouseusers,verbs=create;update,versions=v1alpha1,name=mclickhouseuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ClickhouseUser{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ClickhouseUser) Default() {
	clickhouseuserlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-clickhouseuser,mutating=false,failurePolicy=fail,groups=aiven.io,resources=clickhouseusers,versions=v1alpha1,name=vclickhouseuser.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ClickhouseUser{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ClickhouseUser) ValidateCreate() error {
	clickhouseuserlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ClickhouseUser) ValidateUpdate(old runtime.Object) error {
	clickhouseuserlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ClickhouseUser) ValidateDelete() error {
	clickhouseuserlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
