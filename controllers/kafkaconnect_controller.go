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
	log.Info("reconciling aiven kafka connect")

	const finalizer = "kafkaconnect-service-finalizer.k8s-operator.aiven.io"
	kc := &k8soperatorv1alpha1.KafkaConnect{}
	return r.reconcileInstance(&KafkaConnectHandler{}, ctx, log, req, kc, finalizer)
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaConnect{}).
		Complete(r)
}

func (h KafkaConnectHandler) exists(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("checking if kafka connect service already exists")

	s, err := c.Services.Get(kc.Spec.Project, kc.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaConnectHandler) create(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("creating a new kafkaConnect service")

	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := c.Services.Create(kc.Spec.Project, aiven.CreateServiceRequest{
		Cloud: kc.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kc.Spec.MaintenanceWindowDow,
			kc.Spec.MaintenanceWindowTime),
		Plan:                kc.Spec.Plan,
		ProjectVPCID:        prVPCID,
		ServiceName:         kc.Name,
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

func (h KafkaConnectHandler) update(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("updating kafka connect service")

	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	s, err := c.Services.Update(kc.Spec.Project, kc.Name, aiven.UpdateServiceRequest{
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
	kc.Status.ProjectVPCID = prVPCID
	kc.Status.Plan = s.Plan
	kc.Status.MaintenanceWindowTime = s.MaintenanceWindow.TimeOfDay
	kc.Status.MaintenanceWindowDow = s.MaintenanceWindow.DayOfWeek
	kc.Status.CloudName = s.CloudName
}

func (h KafkaConnectHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	if err := c.Services.Delete(kc.Spec.Project, kc.Name); err != nil {
		if !aiven.IsNotFound(err) {
			log.Error(err, "cannot delete aiven kafka connect service")
			return nil, false, fmt.Errorf("aiven client delete KafkaConnect error: %w", err)
		}
	}

	log.Info("successfully finalized kafka connect service on aiven side")

	return nil, true, nil
}

func (h KafkaConnectHandler) getSecret(*aiven.Client, logr.Logger, client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h KafkaConnectHandler) isActive(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("Checking if KafkaConnect service is active")

	return checkServiceIsRunning(c, kc.Spec.Project, kc.Name), nil
}

func (h KafkaConnectHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaConnect, error) {
	kc, ok := i.(*k8soperatorv1alpha1.KafkaConnect)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnect")
	}

	return kc, nil
}

func (h KafkaConnectHandler) checkPreconditions(*aiven.Client, logr.Logger, client.Object) bool {
	return true
}

func (h KafkaConnectHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	kc, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &kc.Spec.AuthSecretRef
}
