// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

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
)

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

type KafkaConnectHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaconnects/status,verbs=get;update;patch

func (r *KafkaConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaconnect", req.NamespacedName)
	log.Info("Reconciling Aiven Kafka Connect")

	const finalizer = "kafkaconnect-service-finalizer.k8s-operator.aiven.io"
	kc := &k8soperatorv1alpha1.KafkaConnect{}
	return r.reconcileInstance(&KafkaConnectHandler{}, ctx, log, req, kc, finalizer)
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaConnect{}).
		Complete(r)
}

func (h KafkaConnectHandler) exists(log logr.Logger, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("Checking if Kafka Connect service already exists")

	s, err := aivenClient.Services.Get(kc.Spec.Project, kc.Spec.ServiceName)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaConnectHandler) create(log logr.Logger, i client.Object) (client.Object, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating a new KafkaConnect service")

	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := aivenClient.Services.Create(kc.Spec.Project, aiven.CreateServiceRequest{
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

	h.setStatus(kc, s)

	return kc, err
}

func (h KafkaConnectHandler) update(log logr.Logger, i client.Object) (client.Object, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Updating KafkaConnect service")

	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := aivenClient.Services.Update(kc.Spec.Project, kc.Spec.ServiceName, aiven.UpdateServiceRequest{
		Cloud: kc.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kc.Spec.MaintenanceWindowDow,
			kc.Spec.MaintenanceWindowTime),
		Plan:         kc.Spec.Plan,
		ProjectVPCID: prVPCID,
		UserConfig:   UserConfigurationToAPI(kc.Spec.KafkaConnectUserConfig).(map[string]interface{}),
		Powered:      true,
	})
	if err != nil {
		return nil, err
	}

	h.setStatus(kc, s)

	return kc, nil
}

func (h KafkaConnectHandler) setStatus(kc *k8soperatorv1alpha1.KafkaConnect, s *aiven.Service) {
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
}

func (h KafkaConnectHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	if err := aivenClient.Services.Delete(kc.Spec.Project, kc.Spec.ServiceName); err != nil {
		if !aiven.IsNotFound(err) {
			log.Error(err, "Cannot delete Aiven KafkaConnect service")
			return nil, false, fmt.Errorf("aiven client delete KafkaConnect error: %w", err)
		}
	}

	log.Info("Successfully finalized KafkaConnect service on Aiven side")

	return nil, true, nil
}

func (h KafkaConnectHandler) getSecret(logr.Logger, client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h KafkaConnectHandler) isActive(log logr.Logger, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("Checking if KafkaConnect service is active")

	return checkServiceIsRunning(kc.Spec.Project, kc.Spec.ServiceName), nil
}

func (h KafkaConnectHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaConnect, error) {
	kc, ok := i.(*k8soperatorv1alpha1.KafkaConnect)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnect")
	}

	return kc, nil
}

func (h KafkaConnectHandler) checkPreconditions(logr.Logger, client.Object) bool {
	return true
}
