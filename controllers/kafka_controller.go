// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkas/status,verbs=get;update;patch

func (r *KafkaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("kafka", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Kafka instance
	kafka := &k8soperatorv1alpha1.Kafka{}
	err := r.Get(ctx, req.NamespacedName, kafka)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Kafka resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Kafka")
		return ctrl.Result{}, err
	}

	var aivenKafka *aiven.Service

	// Check if Kafka already exists on the Aiven side, create a
	// new one if it is not found
	aivenKafka, err = r.AivenClient.Services.Get(kafka.Spec.Project, kafka.Spec.ServiceName)
	if err != nil {
		// Create a new PG service if does not exists
		if aiven.IsNotFound(err) {
			_, err = r.createKafkaService(kafka)
			if err != nil {
				log.Error(err, "Failed to create Kafka service")
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
	if aivenKafka.State != "RUNNING" {
		log.Info("Kafka service state is " + aivenKafka.State + ", waiting to become RUNNING")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	_, err = r.updateKafkaService(kafka, aivenKafka)
	if err != nil {
		log.Error(err, "Failed to update Kafka service")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createKafkaService creates Kafka service and update CR status and creates secrets
func (r *KafkaReconciler) createKafkaService(kafka *k8soperatorv1alpha1.Kafka) (*aiven.Service, error) {
	var prVPCID *string
	if kafka.Spec.ProjectVPCID != "" {
		prVPCID = &kafka.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Create(kafka.Spec.Project, aiven.CreateServiceRequest{
		Cloud: kafka.Spec.CloudName,
		MaintenanceWindow: &aiven.MaintenanceWindow{
			DayOfWeek: kafka.Spec.MaintenanceWindowDow,
			TimeOfDay: kafka.Spec.MaintenanceWindowTime,
		},
		Plan:                kafka.Spec.Plan,
		ProjectVPCID:        prVPCID,
		ServiceName:         kafka.Spec.ServiceName,
		ServiceType:         "kafka",
		UserConfig:          UserConfigurationToAPI(kafka.Spec.KafkaUserConfig).(map[string]interface{}),
		ServiceIntegrations: nil,
	})
	if err != nil {
		return nil, err
	}

	// Update Kafka custom resource status
	err = r.updateCRStatus(kafka, s)
	if err != nil {
		return nil, fmt.Errorf("failed to update Kafka service status %w", err)
	}

	err = r.createSecret(kafka, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka service secret %w", err)
	}

	return s, nil
}

// updateCRStatus updates Kafka CR status
func (r *KafkaReconciler) updateCRStatus(kafka *k8soperatorv1alpha1.Kafka, s *aiven.Service) error {
	var prVPCID string

	if s.ProjectVPCID != nil {
		prVPCID = *s.ProjectVPCID
	}

	kafka.Status.State = s.State
	kafka.Status.ServiceName = s.Name
	kafka.Status.ProjectVPCID = prVPCID
	kafka.Status.Plan = s.Plan
	kafka.Status.MaintenanceWindowTime = s.MaintenanceWindow.TimeOfDay
	kafka.Status.MaintenanceWindowDow = s.MaintenanceWindow.DayOfWeek
	kafka.Status.CloudName = s.CloudName

	return r.Status().Update(context.Background(), kafka)
}

// createSecret creates a Kafka service secret
func (r *KafkaReconciler) createSecret(kafka *k8soperatorv1alpha1.Kafka, s *aiven.Service) error {
	params := s.URIParams
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", kafka.Name, "-kafka-secret"),
			Namespace: kafka.Namespace,
			Labels: map[string]string{
				"app": kafka.Name,
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

	// Set Kafka service instance as the owner and controller
	err = controllerutil.SetControllerReference(kafka, secret, r.Scheme)
	if err != nil {
		return fmt.Errorf("k8s set controller error %w", err)
	}

	return nil
}

// updatePGService updates Kafka service and updates CR status
func (r *KafkaReconciler) updateKafkaService(kafka *k8soperatorv1alpha1.Kafka, s *aiven.Service) (*aiven.Service, error) {
	var prVPCID *string
	if kafka.Spec.ProjectVPCID != "" {
		prVPCID = &kafka.Spec.ProjectVPCID
	}

	s, err := r.AivenClient.Services.Update(kafka.Spec.Project, kafka.Spec.ServiceName, aiven.UpdateServiceRequest{
		Cloud: kafka.Spec.CloudName,
		MaintenanceWindow: &aiven.MaintenanceWindow{
			DayOfWeek: kafka.Spec.MaintenanceWindowDow,
			TimeOfDay: kafka.Spec.MaintenanceWindowTime,
		},
		Plan:                  kafka.Spec.Plan,
		ProjectVPCID:          prVPCID,
		TerminationProtection: s.TerminationProtection,
		UserConfig:            UserConfigurationToAPI(kafka.Spec.KafkaUserConfig).(map[string]interface{}),
		Powered:               true,
	})
	if err != nil {
		return nil, err
	}

	err = r.updateCRStatus(kafka, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Kafka{}).
		Complete(r)
}
