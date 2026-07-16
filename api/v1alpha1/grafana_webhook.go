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
var grafanalog = logf.Log.WithName("grafana-resource")

func (in *Grafana) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &Grafana{}).
		WithDefaulter(&GrafanaWebhook{}).
		WithValidator(&GrafanaWebhook{}).
		Complete()
}

type GrafanaWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-grafana,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=grafanas,verbs=create;update,versions=v1alpha1,name=mgrafana.kb.io,admissionReviewVersions=v1

func (h *GrafanaWebhook) Default(_ context.Context, obj *Grafana) error {
	grafanalog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-grafana,mutating=false,failurePolicy=fail,groups=aiven.io,resources=grafanas,versions=v1alpha1,name=vgrafana.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *GrafanaWebhook) ValidateCreate(_ context.Context, obj *Grafana) (admission.Warnings, error) {
	grafanalog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *GrafanaWebhook) ValidateUpdate(_ context.Context, _, newObj *Grafana) (admission.Warnings, error) {
	grafanalog.Info("validate update", "name", newObj.Name)

	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *GrafanaWebhook) ValidateDelete(_ context.Context, obj *Grafana) (admission.Warnings, error) {
	grafanalog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete Grafana service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
