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
var cassandralog = logf.Log.WithName("cassandra-resource")

func (in *Cassandra) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-cassandra,mutating=true,failurePolicy=fail,sideEffects=None,groups=aiven.io,resources=cassandras,verbs=create;update,versions=v1alpha1,name=mcassandra.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Cassandra{}

func (in *Cassandra) Default() {
	cassandralog.Info("default", "name", in.Name)
}

//+kubebuilder:webhook:verbs=create;update;delete,path=/validate-aiven-io-v1alpha1-cassandra,mutating=false,failurePolicy=fail,groups=aiven.io,resources=cassandras,versions=v1alpha1,name=vcassandra.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &Cassandra{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *Cassandra) ValidateCreate() error {
	cassandralog.Info("validate create", "name", in.Name)
	return in.Spec.Validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *Cassandra) ValidateUpdate(old runtime.Object) error {
	cassandralog.Info("validate update", "name", in.Name)

	return in.Spec.Validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *Cassandra) ValidateDelete() error {
	cassandralog.Info("validate delete", "name", in.Name)

	if in.Spec.TerminationProtection != nil && *in.Spec.TerminationProtection {
		return errors.New("cannot delete Cassandra service, termination protection is on")
	}

	if in.Spec.ProjectVPCID != "" && in.Spec.ProjectVPCRef != nil {
		return errors.New("cannot use both projectVpcId and projectVPCRef")
	}

	return nil
}
