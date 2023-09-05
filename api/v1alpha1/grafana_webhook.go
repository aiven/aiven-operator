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
var grafanalog = logf.Log.WithName("grafana-resource")

func (in *Grafana) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-grafana,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=grafanas,verbs=create;update,versions=v1alpha1,name=mgrafana.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Grafana{}

func (in *Grafana) Default() {
	grafanalog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-grafana,mutating=false,failurePolicy=fail,groups=aiven.io,resources=grafanas,versions=v1alpha1,name=vgrafana.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Grafana{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Grafana) ValidateCreate() error {
	grafanalog.Info("validate create", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Grafana) ValidateUpdate(old runtime.Object) error {
	grafanalog.Info("validate update", "name", in.Name)

	if in.Spec.Project != old.(*Grafana).Spec.Project {
		return errors.New("cannot update a Grafana service, project field is immutable and cannot be updated")
	}

	if in.Spec.ConnInfoSecretTarget.Name != old.(*Grafana).Spec.ConnInfoSecretTarget.Name {
		return errors.New("cannot update a Grafana service, connInfoSecretTarget.name field is immutable and cannot be updated")
	}

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Grafana) ValidateDelete() error {
	grafanalog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete Grafana service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
