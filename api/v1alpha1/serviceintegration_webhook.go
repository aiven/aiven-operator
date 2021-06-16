// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var serviceintegrationlog = logf.Log.WithName("serviceintegration-resource")

func (r *ServiceIntegration) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-aiven-io-v1alpha1-serviceintegration,mutating=true,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,verbs=create;update,versions=v1alpha1,name=mserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Defaulter = &ServiceIntegration{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *ServiceIntegration) Default() {
	serviceintegrationlog.Info("default", "name", r.Name)
}

//+kubebuilder:webhook:verbs=create;update,path=/validate-aiven-io-v1alpha1-serviceintegration,mutating=false,failurePolicy=fail,groups=aiven.io,resources=serviceintegrations,versions=v1alpha1,name=vserviceintegration.kb.io,sideEffects=none,admissionReviewVersions=v1

var _ webhook.Validator = &ServiceIntegration{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceIntegration) ValidateCreate() error {
	serviceintegrationlog.Info("validate create", "name", r.Name)

	if (r.Spec.SourceServiceName == "" && r.Spec.DestinationServiceName == "") &&
		(r.Spec.SourceEndpointID == "" && r.Spec.DestinationEndpointID == "") {
		return errors.New("cannot create service integration when source and destination fields are empty")
	}

	if r.Spec.SourceServiceName != "" && r.Spec.DestinationServiceName == "" {
		return errors.New("destinationServiceName cannot be empty when sourceServiceName is set")
	}

	if r.Spec.SourceServiceName == "" && r.Spec.DestinationServiceName != "" {
		return errors.New("sourceServiceName cannot be empty when destinationServiceName is set")
	}

	if r.Spec.SourceEndpointID != "" && r.Spec.DestinationEndpointID == "" {
		return errors.New("sourceEndpointID cannot be empty when destinationEndpointID is set")
	}

	if r.Spec.SourceEndpointID == "" && r.Spec.DestinationEndpointID != "" {
		return errors.New("destinationEndpointID cannot be empty when sourceEndpointID is set")
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceIntegration) ValidateUpdate(old runtime.Object) error {
	serviceintegrationlog.Info("validate update", "name", r.Name)

	if r.Spec.Project != old.(*ServiceIntegration).Spec.Project {
		return errors.New("cannot update service integration, project field is idempotent")
	}

	if r.Spec.IntegrationType != old.(*ServiceIntegration).Spec.IntegrationType {
		return errors.New("cannot update service integration, integrationType field is idempotent")
	}

	if r.Spec.SourceEndpointID != old.(*ServiceIntegration).Spec.SourceEndpointID {
		return errors.New("cannot update service integration, sourceEndpointID field is idempotent")
	}

	if r.Spec.DestinationEndpointID != old.(*ServiceIntegration).Spec.DestinationEndpointID {
		return errors.New("cannot update service integration, destinationEndpointID field is idempotent")
	}

	if r.Spec.SourceServiceName != old.(*ServiceIntegration).Spec.SourceServiceName {
		return errors.New("cannot update service integration, sourceServiceName field is idempotent")
	}

	if r.Spec.DestinationServiceName != old.(*ServiceIntegration).Spec.DestinationServiceName {
		return errors.New("cannot update service integration, destinationServiceName field is idempotent")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *ServiceIntegration) ValidateDelete() error {
	serviceintegrationlog.Info("validate delete", "name", r.Name)

	return nil
}
