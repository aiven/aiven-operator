// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

type ServiceIntegrationHandler struct {
}

// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("serviceintegration", req.NamespacedName)
	log.Info("reconciling aiven service integration")

	const finalizer = "serviceintegration-finalizer.aiven.io"
	si := &k8soperatorv1alpha1.ServiceIntegration{}
	return r.reconcileInstance(&ServiceIntegrationHandler{}, ctx, log, req, si, finalizer)
}

func (r *ServiceIntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceIntegration{}).
		Complete(r)
}

func (h ServiceIntegrationHandler) create(c *aiven.Client, _ logr.Logger, i client.Object) (client.Object, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	integration, err := c.ServiceIntegrations.Create(
		si.Spec.Project,
		aiven.CreateServiceIntegrationRequest{
			DestinationEndpointID: toOptionalStringPointer(si.Spec.DestinationEndpointID),
			DestinationService:    toOptionalStringPointer(si.Spec.DestinationServiceName),
			IntegrationType:       si.Spec.IntegrationType,
			SourceEndpointID:      toOptionalStringPointer(si.Spec.SourceEndpointID),
			SourceService:         toOptionalStringPointer(si.Spec.SourceServiceName),
			UserConfig:            h.getUserConfig(si),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create service integration: %w", err)
	}

	h.setStatus(si, integration)

	return si, nil
}

func (h ServiceIntegrationHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = c.ServiceIntegrations.Delete(si.Spec.Project, si.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		log.Error(err, "cannot delete service integration")
		return false, fmt.Errorf("aiven client delete service ingtegration error: %w", err)
	}

	log.Info("successfully finalized service integration")

	return true, nil
}

func (h ServiceIntegrationHandler) exists(_ *aiven.Client, _ logr.Logger, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	return si.Status.ID != "", nil
}

func (h ServiceIntegrationHandler) update(c *aiven.Client, _ logr.Logger, i client.Object) (client.Object, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	integration, err := c.ServiceIntegrations.Update(
		si.Spec.Project,
		si.Status.ID,
		aiven.UpdateServiceIntegrationRequest{
			UserConfig: h.getUserConfig(si),
		},
	)
	if err != nil {
		if strings.Contains(err.Error(), "user config not changed") {
			return nil, nil
		}
		return nil, err
	}

	h.setStatus(si, integration)

	return si, nil
}

func (h ServiceIntegrationHandler) getSecret(*aiven.Client, logr.Logger, client.Object) (secret *corev1.Secret, error error) {
	return nil, nil
}

func (h ServiceIntegrationHandler) checkPreconditions(c *aiven.Client, _ logr.Logger, i client.Object) bool {
	si, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(c, si.Spec.Project, si.Spec.SourceServiceName) &&
		checkServiceIsRunning(c, si.Spec.Project, si.Spec.DestinationServiceName)
}

func (h ServiceIntegrationHandler) isActive(*aiven.Client, logr.Logger, client.Object) (bool, error) {
	return true, nil
}

func (h ServiceIntegrationHandler) convert(i client.Object) (*k8soperatorv1alpha1.ServiceIntegration, error) {
	si, ok := i.(*k8soperatorv1alpha1.ServiceIntegration)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceIntegration")
	}

	return si, nil
}

func (h ServiceIntegrationHandler) setStatus(int *k8soperatorv1alpha1.ServiceIntegration, i *aiven.ServiceIntegration) {
	int.Status.Project = int.Spec.Project
	int.Status.IntegrationType = i.IntegrationType
	int.Status.SourceServiceName = stringPointerToString(i.SourceService)
	int.Status.DestinationServiceName = stringPointerToString(i.DestinationService)
	int.Status.DestinationEndpointID = stringPointerToString(i.DestinationEndpointID)
	int.Status.SourceEndpointID = stringPointerToString(i.SourceEndpointID)
	int.Status.ID = i.ServiceIntegrationID
}

func (h ServiceIntegrationHandler) getUserConfig(int *k8soperatorv1alpha1.ServiceIntegration) map[string]interface{} {
	if int.Spec.IntegrationType == "datadog" {
		return UserConfigurationToAPI(int.Spec.DatadogUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "kafka_connect" {
		return UserConfigurationToAPI(int.Spec.KafkaConnectUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "kafka_logs" {
		return UserConfigurationToAPI(int.Spec.KafkaLogsUserConfig).(map[string]interface{})
	}
	if int.Spec.IntegrationType == "metrics" {
		return UserConfigurationToAPI(int.Spec.MetricsUserConfig).(map[string]interface{})
	}

	return nil
}

func (h ServiceIntegrationHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	si, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &si.Spec.AuthSecretRef
}
