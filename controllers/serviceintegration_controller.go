// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

// ServiceIntegrationReconciler reconciles a ServiceIntegration object
type ServiceIntegrationReconciler struct {
	Controller
}

const serviceIntegrationFinalizer = "serviceintegration-finalizer.k8s-operator.aiven.io"

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceintegrations/status,verbs=get;update;patch

func (r *ServiceIntegrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("serviceintegration", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Service Integration instance
	serviceInt := &k8soperatorv1alpha1.ServiceIntegration{}
	err := r.Get(ctx, req.NamespacedName, serviceInt)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Service Integration resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ServiceIntegration")
		return ctrl.Result{}, err
	}

	// Check if the ServiceIntegration instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isServiceIntegrationMarkedToBeDeleted := serviceInt.GetDeletionTimestamp() != nil
	if isServiceIntegrationMarkedToBeDeleted {
		if contains(serviceInt.GetFinalizers(), serviceIntegrationFinalizer) {
			// Run finalization logic for serviceIntegrationFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeServiceIntegration(log, serviceInt); err != nil {
				return reconcile.Result{}, err
			}

			// Remove serviceIntegrationFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(serviceInt, serviceIntegrationFinalizer)
			err := r.Client.Update(ctx, serviceInt)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(serviceInt.GetFinalizers(), serviceIntegrationFinalizer) {
		if err := r.addFinalizer(ctx, log, serviceInt); err != nil {
			return reconcile.Result{}, err
		}
	}

	if serviceInt.Status.ID == "" {
		log.Info("Creating a new Service integration")
		_, err = r.createServiceIntegration(ctx, serviceInt)
		if err != nil {
			log.Error(err, "Failed to create Service Integration")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	_, err = r.updateServiceIntegration(ctx, serviceInt)
	if err != nil {
		log.Error(err, "Failed to update Service Integration")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// addFinalizer add finalizer to CR
func (r *ServiceIntegrationReconciler) addFinalizer(ctx context.Context, reqLogger logr.Logger, i *k8soperatorv1alpha1.ServiceIntegration) error {
	reqLogger.Info("Adding Finalizer for the Service Integration")
	controllerutil.AddFinalizer(i, serviceIntegrationFinalizer)

	// Update CR
	err := r.Client.Update(ctx, i)
	if err != nil {
		reqLogger.Error(err, "Failed to update Service Integration with finalizer")
		return err
	}

	return nil
}

// finalizeServiceIntegration deletes Service Integration on Aiven side
func (r *ServiceIntegrationReconciler) finalizeServiceIntegration(log logr.Logger, i *k8soperatorv1alpha1.ServiceIntegration) error {
	// Delete service integration on Aiven side
	err := r.AivenClient.ServiceIntegrations.Delete(i.Spec.Project, i.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		log.Error(err, "Cannot delete Service Integration")
		return fmt.Errorf("aiven client delete service ingtegration error: %w", err)
	}

	log.Info("Successfully finalized service integration")
	return nil
}

func (r *ServiceIntegrationReconciler) createServiceIntegration(ctx context.Context, int *k8soperatorv1alpha1.ServiceIntegration) (*aiven.ServiceIntegration, error) {
	i, err := r.AivenClient.ServiceIntegrations.Create(
		int.Spec.Project,
		aiven.CreateServiceIntegrationRequest{
			DestinationEndpointID: toOptionalStringPointer(int.Spec.DestinationEndpointID),
			DestinationService:    toOptionalStringPointer(int.Spec.DestinationServiceName),
			IntegrationType:       int.Spec.IntegrationType,
			SourceEndpointID:      toOptionalStringPointer(int.Spec.SourceEndpointID),
			SourceService:         toOptionalStringPointer(int.Spec.SourceServiceName),
			UserConfig:            r.GetUserConfig(int),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot create service integration:%w", err)
	}

	err = r.updateCRStatus(ctx, int, i)
	if err != nil {
		return nil, fmt.Errorf("cannot update CR status: %w", err)
	}

	return i, nil
}

func (r *ServiceIntegrationReconciler) updateServiceIntegration(ctx context.Context, int *k8soperatorv1alpha1.ServiceIntegration) (*aiven.ServiceIntegration, error) {
	i, err := r.AivenClient.ServiceIntegrations.Update(
		int.Spec.Project,
		int.Status.ID,
		aiven.UpdateServiceIntegrationRequest{
			UserConfig: r.GetUserConfig(int),
		},
	)
	if err != nil {
		if strings.Contains(err.Error(), "User config not changed") {
			return nil, nil
		}
		return nil, err
	}

	err = r.updateCRStatus(ctx, int, i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ServiceIntegrationReconciler) updateCRStatus(ctx context.Context, int *k8soperatorv1alpha1.ServiceIntegration, i *aiven.ServiceIntegration) error {
	int.Status.Project = int.Spec.Project
	int.Status.IntegrationType = i.IntegrationType
	int.Status.SourceServiceName = stringPointerToString(i.SourceService)
	int.Status.DestinationServiceName = stringPointerToString(i.DestinationService)
	int.Status.DestinationEndpointID = stringPointerToString(i.DestinationEndpointID)
	int.Status.SourceEndpointID = stringPointerToString(i.SourceEndpointID)
	int.Status.ID = i.ServiceIntegrationID

	return r.Status().Update(ctx, int)
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
