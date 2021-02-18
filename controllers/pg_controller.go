// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const pgServiceFinalizer = "pg-service-finalizer.k8s-operator.aiven.io"

// PGReconciler reconciles a PG object
type PGReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=pgs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=pgs/status,verbs=get;update;patch

func (r *PGReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("pg", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the PG instance
	pg := &k8soperatorv1alpha1.PG{}
	err := r.Get(ctx, req.NamespacedName, pg)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("PG resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get PG")
		return ctrl.Result{}, err
	}

	// Check if the PG instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isProjectMarkedToBeDeleted := pg.GetDeletionTimestamp() != nil
	if isProjectMarkedToBeDeleted {
		if contains(pg.GetFinalizers(), pgServiceFinalizer) {
			// Run finalization logic for projectFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizePGService(log, pg); err != nil {
				return reconcile.Result{}, err
			}

			// Remove projectFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(pg, pgServiceFinalizer)
			err := r.Client.Update(ctx, pg)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(pg.GetFinalizers(), pgServiceFinalizer) {
		if err := r.addFinalizer(log, pg); err != nil {
			return reconcile.Result{}, err
		}
	}

	var aivenPG *aiven.Service

	// Check if PG already exists on the Aiven side, create a
	// new one if PG is not found
	aivenPG, err = r.AivenClient.Services.Get(pg.Spec.Project, pg.Spec.ServiceName)
	if err != nil {
		// Create a new PG service if does not exists
		if aiven.IsNotFound(err) {
			aivenPG, err = r.createPGService(pg)
			if err != nil {
				log.Error(err, "Failed to create PG service")
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
	if aivenPG.State != "RUNNING" {
		log.Info("PG service state is " + aivenPG.State + ", waiting to become RUNNING")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	// Update PG service if it has differences between CR and Aiven representation
	if r.hasDifferences(pg, aivenPG) {
		aivenPG, err = r.updatePGService(pg)
		if err != nil {
			log.Error(err, "Failed to update PG service")
			return ctrl.Result{}, err
		}
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	err = r.updateCRStatus(pg, aivenPG)
	if err != nil {
		log.Error(err, "Failed to refresh PG service status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// hasDifferences defines if there is a different between CR speck and Aiven representation of
// PG service
func (r *PGReconciler) hasDifferences(pg *k8soperatorv1alpha1.PG, s *aiven.Service) bool {
	if pg.Spec.CloudName != s.CloudName {
		return true
	}
	if pg.Spec.Plan != s.Plan {
		return true
	}
	if pg.Spec.ServiceName != s.Name {
		return true
	}
	if pg.Spec.MaintenanceWindowDow != s.MaintenanceWindow.DayOfWeek {
		return true
	}
	if pg.Spec.MaintenanceWindowTime != s.MaintenanceWindow.TimeOfDay {
		return true
	}
	if s.ProjectVPCID != nil && pg.Spec.Project != *s.ProjectVPCID {
		return true
	}

	return false
}

// createPGService creates PG service and update CR status and creates secrets
func (r *PGReconciler) createPGService(pg *k8soperatorv1alpha1.PG) (*aiven.Service, error) {
	var prVPCID *string
	if pg.Spec.ProjectVPCID != "" {
		prVPCID = &pg.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Create(pg.Spec.Project, aiven.CreateServiceRequest{
		Cloud: pg.Spec.CloudName,
		MaintenanceWindow: &aiven.MaintenanceWindow{
			DayOfWeek: pg.Spec.MaintenanceWindowDow,
			TimeOfDay: pg.Spec.MaintenanceWindowTime,
		},
		Plan:                  pg.Spec.Plan,
		ProjectVPCID:          prVPCID,
		ServiceName:           pg.Spec.ServiceName,
		ServiceType:           "pg",
		TerminationProtection: false,
		UserConfig:            UserConfigurationToAPI(pg.Spec.PGUserConfig).(map[string]interface{}),
		ServiceIntegrations:   nil,
	})
	if err != nil {
		return nil, err
	}

	// Update project custom resource status
	err = r.updateCRStatus(pg, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update PG service status %w", err)
	}

	err = r.createPGSecret(pg, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create PG service secret %w", err)
	}

	return s, err
}

// updatePGService updates PG service and updates CR status
func (r *PGReconciler) updatePGService(pg *k8soperatorv1alpha1.PG) (*aiven.Service, error) {
	var prVPCID *string
	if pg.Spec.ProjectVPCID != "" {
		prVPCID = &pg.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Update(pg.Spec.Project, pg.Spec.ServiceName, aiven.UpdateServiceRequest{
		Cloud: pg.Spec.CloudName,
		MaintenanceWindow: &aiven.MaintenanceWindow{
			DayOfWeek: pg.Spec.MaintenanceWindowDow,
			TimeOfDay: pg.Spec.MaintenanceWindowTime,
		},
		Plan:                  pg.Spec.Plan,
		ProjectVPCID:          prVPCID,
		TerminationProtection: false,
		UserConfig:            UserConfigurationToAPI(pg.Spec.PGUserConfig).(map[string]interface{}),
		Powered:               true,
	})
	if err != nil {
		return nil, err
	}

	// Update project custom resource status
	err = r.updateCRStatus(pg, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update PG service status %w", err)
	}

	return s, err
}

// updateCRStatus updates PG CR status
func (r *PGReconciler) updateCRStatus(pg *k8soperatorv1alpha1.PG, s *aiven.Service) error {
	var prVPCID string

	if s.ProjectVPCID != nil {
		prVPCID = *s.ProjectVPCID
	}

	pg.Status.State = s.State
	pg.Status.ServiceName = s.Name
	pg.Status.ProjectVPCID = prVPCID
	pg.Status.Plan = s.Plan
	pg.Status.MaintenanceWindowTime = s.MaintenanceWindow.TimeOfDay
	pg.Status.MaintenanceWindowDow = s.MaintenanceWindow.DayOfWeek
	pg.Status.CloudName = s.CloudName

	err := r.Status().Update(context.Background(), pg)
	if err != nil {
		return err
	}

	return nil
}

// addFinalizer add finalizer to CR
func (r *PGReconciler) addFinalizer(reqLogger logr.Logger, p *k8soperatorv1alpha1.PG) error {
	reqLogger.Info("Adding Finalizer for the PG Service")
	controllerutil.AddFinalizer(p, pgServiceFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), p)
	if err != nil {
		reqLogger.Error(err, "Failed to update PG with finalizer")
		return err
	}
	return nil
}

// finalizeProject deletes Aiven PG service
func (r *PGReconciler) finalizePGService(log logr.Logger, p *k8soperatorv1alpha1.PG) error {
	// Delete project on Aiven side
	if err := r.AivenClient.Services.Delete(p.Spec.Project, p.Spec.ServiceName); err != nil {
		if !aiven.IsNotFound(err) {
			log.Error(err, "Cannot delete Aiven PG service")
			return fmt.Errorf("aiven client delete PG error: %w", err)
		}
	}

	// Check if secret exists and delete if it is
	secret := &corev1.Secret{}
	err := r.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("%s%s", p.Name, "-pg-secret"), Namespace: p.Namespace}, secret)
	if err == nil {
		err = r.Client.Delete(context.Background(), secret)
		if err != nil {
			return fmt.Errorf("delete project secret error: %w", err)
		}
	}

	log.Info("Successfully finalized PG service")
	return nil
}

// createPGSecret creates a PG service secret
func (r *PGReconciler) createPGSecret(pg *k8soperatorv1alpha1.PG, s *aiven.Service) error {
	params := s.URIParams
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", pg.Name, "-pg-secret"),
			Namespace: pg.Namespace,
			Labels: map[string]string{
				"app": pg.Name,
			},
		},
		StringData: map[string]string{
			"host":     params["host"],
			"port":     params["port"],
			"password": params["password"],
			"user":     params["user"],
		},
	}
	err := r.Client.Create(context.Background(), secret)
	if err != nil {
		return fmt.Errorf("k8s client create error %w", err)
	}

	// Set PG service instance as the owner and controller
	err = controllerutil.SetControllerReference(pg, secret, r.Scheme)
	if err != nil {
		return fmt.Errorf("k8s set controller error %w", err)
	}

	return nil
}

func (r *PGReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.PG{}).
		Complete(r)
}
