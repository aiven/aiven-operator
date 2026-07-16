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
var clickhouselog = logf.Log.WithName("clickhouse-resource")

func (in *Clickhouse) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Clickhouse{}).
		WithDefaulter(&ClickhouseWebhook{}).
		WithValidator(&ClickhouseWebhook{}).
		Complete()
}

type ClickhouseWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-clickhouse,mutating=true,failurePolicy=fail,groups=aiven.io,resources=clickhouses,verbs=create;update,versions=v1alpha1,name=mclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *ClickhouseWebhook) Default(_ context.Context, obj *Clickhouse) error {
	clickhouselog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-clickhouse,mutating=false,failurePolicy=fail,groups=aiven.io,resources=clickhouses,versions=v1alpha1,name=vclickhouse.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ClickhouseWebhook) ValidateCreate(_ context.Context, obj *Clickhouse) (admission.Warnings, error) {
	clickhouselog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ClickhouseWebhook) ValidateUpdate(_ context.Context, _, newObj *Clickhouse) (admission.Warnings, error) {
	clickhouselog.Info("validate update", "name", newObj.Name)
	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *ClickhouseWebhook) ValidateDelete(_ context.Context, obj *Clickhouse) (admission.Warnings, error) {
	clickhouselog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Clickhouse service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
