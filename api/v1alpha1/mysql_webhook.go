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
var mysqllog = logf.Log.WithName("mysql-resource")

func (in *MySQL) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &MySQL{}).
		WithDefaulter(&MySQLWebhook{}).
		WithValidator(&MySQLWebhook{}).
		Complete()
}

type MySQLWebhook struct{}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-mysql,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=mysqls,verbs=create;update,versions=v1alpha1,name=mmysql.kb.io,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (h *MySQLWebhook) Default(_ context.Context, obj *MySQL) error {
	mysqllog.Info("default", "name", obj.Name)
	return nil
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-mysql,mutating=false,failurePolicy=fail,groups=aiven.io,resources=mysqls,versions=v1alpha1,name=vmysql.kb.io,sideEffects=none,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *MySQLWebhook) ValidateCreate(_ context.Context, obj *MySQL) (admission.Warnings, error) {
	mysqllog.Info("validate create", "name", obj.Name)

	return nil, obj.Spec.Validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (h *MySQLWebhook) ValidateUpdate(_ context.Context, _, newObj *MySQL) (admission.Warnings, error) {
	mysqllog.Info("validate update", "name", newObj.Name)

	return nil, newObj.Spec.Validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (h *MySQLWebhook) ValidateDelete(_ context.Context, obj *MySQL) (admission.Warnings, error) {
	mysqllog.Info("validate delete", "name", obj.Name)

	if obj.Spec.TerminationProtection != nil && *obj.Spec.TerminationProtection {
		return nil, errors.New("cannot delete MySQL service, termination protection is on")
	}

	if obj.Spec.ProjectVPCID != "" && obj.Spec.ProjectVPCRef != nil {
		return nil, errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil, nil
}
