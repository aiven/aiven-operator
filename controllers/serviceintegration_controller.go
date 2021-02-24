// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"github.com/aiven/aiven-go-client"
	"k8s.io/apimachinery/pkg/api/errors"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("serviceintegration", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Service Integration instance
	serviceInt := &k8soperatorv1alpha1.ServiceIntegration{}
	err := r.Get(ctx, req.NamespacedName, serviceInt)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Service Integration resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ServiceIntegration")
		return ctrl.Result{}, err
	}

	// Check if service integration already exists on the Aiven side, create a
	// new one if it is not found
	isServiceIntegrationExists, err := r.exists(serviceInt)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !isServiceIntegrationExists {
		_, err = r.createServiceIntegration(serviceInt)
		if err != nil {
			log.Error(err, "Failed to create Service Integration")
			return ctrl.Result{}, err
		}
	}

	_, err = r.updateServiceIntegration(serviceInt)
	if err != nil {
		log.Error(err, "Failed to update Service Integration")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceIntegrationReconciler) exists(int *k8soperatorv1alpha1.ServiceIntegration) (bool, error) {
	integrations, err := r.AivenClient.ServiceIntegrations.List(int.Spec.Project, int.Spec.SourceServiceName)
	if err != nil {
		return false, err
	}

	for _, i := range integrations {
		if i.SourceService == nil || i.DestinationService == nil {
			continue
		}

		if i.IntegrationType == int.Spec.IntegrationType &&
			*i.SourceService == int.Spec.SourceServiceName &&
			*i.DestinationService == int.Spec.DestinationServiceName {
			return true, nil
		}
	}

	return false, nil
}

func (r *ServiceIntegrationReconciler) createServiceIntegration(int *k8soperatorv1alpha1.ServiceIntegration) (*aiven.ServiceIntegration, error) {
	i, err := r.AivenClient.ServiceIntegrations.Create(
		int.Spec.Project,
		aiven.CreateServiceIntegrationRequest{
			DestinationEndpointID: &int.Spec.DestinationEndpointID,
			DestinationService:    &int.Spec.DestinationServiceName,
			IntegrationType:       int.Spec.IntegrationType,
			SourceEndpointID:      &int.Spec.SourceEndpointID,
			SourceService:         &int.Spec.SourceServiceName,
			UserConfig:            r.GetUserConfig(int),
		},
	)
	if err != nil {
		return nil, err
	}

	err = r.updateCRStatus(int, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (r *ServiceIntegrationReconciler) updateServiceIntegration(int *k8soperatorv1alpha1.ServiceIntegration) (*aiven.ServiceIntegration, error) {
	i, err := r.AivenClient.ServiceIntegrations.Update(
		int.Spec.Project,
		int.Status.ID,
		aiven.UpdateServiceIntegrationRequest{
			UserConfig: r.GetUserConfig(int),
		},
	)
	if err != nil {
		return nil, err
	}

	err = r.updateCRStatus(int, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ServiceIntegrationReconciler) updateCRStatus(int *k8soperatorv1alpha1.ServiceIntegration, i *aiven.ServiceIntegration) error {
	int.Status.Project = int.Spec.Project
	int.Status.IntegrationType = i.IntegrationType
	int.Status.SourceServiceName = *i.SourceService
	int.Status.DestinationServiceName = *i.DestinationService
	int.Status.DestinationEndpointID = *i.DestinationEndpointID
	int.Status.SourceEndpointID = *i.SourceEndpointID
	int.Status.ID = i.ServiceIntegrationID

	return r.Status().Update(context.Background(), int)
}

func (r *ServiceIntegrationReconciler) GetUserConfig(int *k8soperatorv1alpha1.ServiceIntegration) map[string]interface{} {
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
