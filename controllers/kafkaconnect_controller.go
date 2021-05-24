// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

const kcServiceFinalizer = "kc-service-finalizer.k8s-operator.aiven.io"

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaconnects/status,verbs=get;update;patch

func (r *KafkaConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaconnect", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Kafka Connect instance
	kc := &k8soperatorv1alpha1.KafkaConnect{}
	err := r.Get(ctx, req.NamespacedName, kc)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Kafka Connect resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Kafka Connect")
		return ctrl.Result{}, err
	}

	// Check if the Kafka Konnect instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isProjectMarkedToBeDeleted := kc.GetDeletionTimestamp() != nil
	if isProjectMarkedToBeDeleted {
		if contains(kc.GetFinalizers(), kcServiceFinalizer) {
			// Run finalization logic for projectFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeService(log, kc); err != nil {
				return reconcile.Result{}, err
			}

			// Remove projectFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(kc, kcServiceFinalizer)
			err := r.Client.Update(ctx, kc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(kc.GetFinalizers(), kcServiceFinalizer) {
		if err := r.addFinalizer(log, kc); err != nil {
			return reconcile.Result{}, err
		}
	}

	var aivenKC *aiven.Service

	// Check if Kafka Connect already exists on the Aiven side, create a
	// new one if it is not found
	aivenKC, err = r.AivenClient.Services.Get(kc.Spec.Project, kc.Spec.ServiceName)
	if err != nil {
		// Create a new KafkaConnect service if does not exists
		if aiven.IsNotFound(err) {
			_, err = r.createService(kc)
			if err != nil {
				log.Error(err, "Failed to create KafkaConnect service")
				return ctrl.Result{}, err
			}

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 10 * time.Second,
			}, nil
		}
		return ctrl.Result{}, err
	}

	// Check service status and wait until it is RUNNING
	if aivenKC.State != "RUNNING" {
		log.Info("KafkaConnect service state is " + aivenKC.State + ", waiting to become RUNNING")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// Update KafkaConnect service if it has differences between CR and Aiven representation
	if r.hasDifferences(kc, aivenKC) {
		_, err = r.updateService(kc, aivenKC)
		if err != nil {
			log.Error(err, "Failed to update KafkaConnect service")
			return ctrl.Result{}, err
		}
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	err = r.updateCRStatus(kc, aivenKC)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// hasDifferences defines if there is a different between CR speck and Aiven representation of
// KafkaConnect service
func (r *KafkaConnectReconciler) hasDifferences(kc *k8soperatorv1alpha1.KafkaConnect, s *aiven.Service) bool {
	if kc.Spec.CloudName != s.CloudName {
		return true
	}
	if kc.Spec.Plan != s.Plan {
		return true
	}
	if kc.Spec.ServiceName != s.Name {
		return true
	}
	if kc.Spec.MaintenanceWindowDow != s.MaintenanceWindow.DayOfWeek {
		return true
	}
	if kc.Spec.MaintenanceWindowTime != s.MaintenanceWindow.TimeOfDay {
		return true
	}
	if s.ProjectVPCID != nil && kc.Spec.Project != *s.ProjectVPCID {
		return true
	}

	return false
}

// createService creates KafkaConnect service and update CR status and creates secrets
func (r *KafkaConnectReconciler) createService(kc *k8soperatorv1alpha1.KafkaConnect) (*aiven.Service, error) {
	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Create(kc.Spec.Project, aiven.CreateServiceRequest{
		Cloud: kc.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kc.Spec.MaintenanceWindowDow,
			kc.Spec.MaintenanceWindowTime),
		Plan:                kc.Spec.Plan,
		ProjectVPCID:        prVPCID,
		ServiceName:         kc.Spec.ServiceName,
		ServiceType:         "kafka_connect",
		UserConfig:          UserConfigurationToAPI(kc.Spec.KafkaConnectUserConfig).(map[string]interface{}),
		ServiceIntegrations: nil,
	})
	if err != nil {
		return nil, err
	}

	// Update project custom resource status
	err = r.updateCRStatus(kc, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update KafkaConect service status %w", err)
	}

	return s, err
}

// updateService updates Kafka Connect service and updates CR status
func (r *KafkaConnectReconciler) updateService(kc *k8soperatorv1alpha1.KafkaConnect, s *aiven.Service) (*aiven.Service, error) {
	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Update(kc.Spec.Project, kc.Spec.ServiceName, aiven.UpdateServiceRequest{
		Cloud: kc.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kc.Spec.MaintenanceWindowDow,
			kc.Spec.MaintenanceWindowTime),
		Plan:                  kc.Spec.Plan,
		ProjectVPCID:          prVPCID,
		TerminationProtection: s.TerminationProtection,
		UserConfig:            UserConfigurationToAPI(kc.Spec.KafkaConnectUserConfig).(map[string]interface{}),
		Powered:               true,
	})
	if err != nil {
		return nil, err
	}

	// Update project custom resource status
	err = r.updateCRStatus(kc, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update KafkaConnect service status %w", err)
	}

	return s, err
}

// updateCRStatus updates KafkaConnect CR status
func (r *KafkaConnectReconciler) updateCRStatus(kc *k8soperatorv1alpha1.KafkaConnect, s *aiven.Service) error {
	var prVPCID string

	if s.ProjectVPCID != nil {
		prVPCID = *s.ProjectVPCID
	}

	kc.Status.State = s.State
	kc.Status.ServiceName = s.Name
	kc.Status.ProjectVPCID = prVPCID
	kc.Status.Plan = s.Plan
	kc.Status.MaintenanceWindowTime = s.MaintenanceWindow.TimeOfDay
	kc.Status.MaintenanceWindowDow = s.MaintenanceWindow.DayOfWeek
	kc.Status.CloudName = s.CloudName

	err := r.Status().Update(context.Background(), kc)
	if err != nil {
		return fmt.Errorf("cannot update CR status: %w", err)
	}

	return nil
}

// addFinalizer add finalizer to CR
func (r *KafkaConnectReconciler) addFinalizer(reqLogger logr.Logger, p *k8soperatorv1alpha1.KafkaConnect) error {
	reqLogger.Info("Adding Finalizer for the KafkaKonnect Service")
	controllerutil.AddFinalizer(p, kcServiceFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), p)
	if err != nil {
		reqLogger.Error(err, "Failed to update KafkaConnect with finalizer")
		return err
	}
	return nil
}

// finalizeProject deletes Aiven KafkaConnect service
func (r *KafkaConnectReconciler) finalizeService(log logr.Logger, kc *k8soperatorv1alpha1.KafkaConnect) error {
	// Delete project on Aiven side
	if err := r.AivenClient.Services.Delete(kc.Spec.Project, kc.Spec.ServiceName); err != nil {
		if !aiven.IsNotFound(err) {
			log.Error(err, "Cannot delete Aiven KafkaConnect service")
			return fmt.Errorf("aiven client delete KafkaConnect error: %w", err)
		}
	}

	log.Info("Successfully finalized KafkaConnect service")
	return nil
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaConnect{}).
		Complete(r)
}
