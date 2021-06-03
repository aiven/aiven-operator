// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

type ServiceIntegrationHandler struct {
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("serviceintegration", req.NamespacedName)
	log.Info("Reconciling Aiven ServiceIntegration")

	const finalizer = "serviceintegration-finalizer.k8s-operator.aiven.io"
	si := &k8soperatorv1alpha1.ServiceIntegration{}
	return r.reconcileInstance(&ServiceIntegrationHandler{}, ctx, log, req, si, finalizer)
}

func (h ServiceIntegrationHandler) create(_ logr.Logger, i client.Object) (client.Object, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	integration, err := aivenClient.ServiceIntegrations.Create(
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
		return nil, fmt.Errorf("cannot create service integration:%w", err)
	}

	h.setStatus(si, integration)

	return si, nil
}

func (h ServiceIntegrationHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	err = aivenClient.ServiceIntegrations.Delete(si.Spec.Project, si.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		log.Error(err, "Cannot delete Service Integration")
		return nil, false, fmt.Errorf("aiven client delete service ingtegration error: %w", err)
	}

	log.Info("Successfully finalized service integration")

	return nil, true, nil
}

func (h ServiceIntegrationHandler) exists(_ logr.Logger, i client.Object) (bool, error) {
	si, err := h.convert(i)
	if err != nil {
		return false, err
	}

	return si.Status.ID != "", nil
}

func (h ServiceIntegrationHandler) update(_ logr.Logger, i client.Object) (client.Object, error) {
	si, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	integration, err := aivenClient.ServiceIntegrations.Update(
		si.Spec.Project,
		si.Status.ID,
		aiven.UpdateServiceIntegrationRequest{
			UserConfig: h.getUserConfig(si),
		},
	)
	if err != nil {
		if strings.Contains(err.Error(), "User config not changed") {
			return nil, nil
		}
		return nil, err
	}

	h.setStatus(si, integration)

	return si, nil
}

func (h ServiceIntegrationHandler) getSecret(logr.Logger, client.Object) (secret *corev1.Secret, error error) {
	return nil, nil
}

func (h ServiceIntegrationHandler) checkPreconditions(_ logr.Logger, i client.Object) bool {
	si, err := h.convert(i)
	if err != nil {
		return false
	}

	if checkServiceIsRunning(si.Spec.Project, si.Spec.SourceServiceName) &&
		checkServiceIsRunning(si.Spec.Project, si.Spec.DestinationServiceName) {
		return true
	}

	return false
}

func (h ServiceIntegrationHandler) isActive(logr.Logger, client.Object) (bool, error) {
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

func (r *ServiceIntegrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceIntegration{}).
		Complete(r)
}
