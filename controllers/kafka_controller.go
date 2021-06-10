// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	Controller
}

// KafkaHandler handles an Aiven Kafka service
type KafkaHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkas/status,verbs=get;update;patch

func (r *KafkaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafka", req.NamespacedName)
	log.Info("reconciling aiven kafka")

	const finalizer = "kafka-service-finalizer.k8s-operator.aiven.io"
	kafka := &k8soperatorv1alpha1.Kafka{}
	return r.reconcileInstance(&KafkaHandler{}, ctx, log, req, kafka, finalizer)
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Kafka{}).
		Complete(r)
}

func (h *KafkaHandler) create(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("creating kafka service")

	var prVPCID *string
	if kafka.Spec.ProjectVPCID != "" {
		prVPCID = &kafka.Spec.ProjectVPCID
	}

	s, err := c.Services.Create(kafka.Spec.Project, aiven.CreateServiceRequest{
		Cloud: kafka.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kafka.Spec.MaintenanceWindowDow,
			kafka.Spec.MaintenanceWindowTime),
		Plan:                kafka.Spec.Plan,
		ProjectVPCID:        prVPCID,
		ServiceName:         kafka.Name,
		ServiceType:         "kafka",
		UserConfig:          UserConfigurationToAPI(kafka.Spec.KafkaUserConfig).(map[string]interface{}),
		ServiceIntegrations: nil,
	})
	if err != nil && !aiven.IsAlreadyExists(err) {
		return nil, err
	}

	h.setStatus(kafka, s)

	return kafka, nil
}

func (h KafkaHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, bool, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	// Delete project on Aiven side
	if err := c.Services.Delete(kafka.Spec.Project, kafka.Name); err != nil {
		if !aiven.IsNotFound(err) {
			log.Error(err, "cannot delete aiven kafka service")
			return nil, false, fmt.Errorf("aiven client delete Kafka error: %w", err)
		}
	}

	log.Info("successfully finalized kafka service on aiven side")

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", kafka.Name, "-kafka-secret"),
			Namespace: kafka.Namespace,
		},
	}, true, nil
}

func (h KafkaHandler) exists(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("checking if kafka service already exists")

	s, err := c.Services.Get(kafka.Spec.Project, kafka.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaHandler) update(c *aiven.Client, _ logr.Logger, i client.Object) (client.Object, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	var prVPCID *string
	if kafka.Spec.ProjectVPCID != "" {
		prVPCID = &kafka.Spec.ProjectVPCID
	}

	s, err := c.Services.Update(kafka.Spec.Project, kafka.Name, aiven.UpdateServiceRequest{
		Cloud: kafka.Spec.CloudName,
		MaintenanceWindow: getMaintenanceWindow(
			kafka.Spec.MaintenanceWindowDow,
			kafka.Spec.MaintenanceWindowTime),
		Plan:         kafka.Spec.Plan,
		ProjectVPCID: prVPCID,
		UserConfig:   UserConfigurationToAPI(kafka.Spec.KafkaUserConfig).(map[string]interface{}),
		Powered:      true,
	})
	if err != nil {
		return nil, err
	}

	h.setStatus(kafka, s)

	return kafka, nil
}

func (h KafkaHandler) getSecret(c *aiven.Client, _ logr.Logger, i client.Object) (*corev1.Secret, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := c.Services.Get(kafka.Spec.Project, kafka.Name)
	if err != nil {
		return nil, err
	}

	params := s.URIParams
	return &corev1.Secret{
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
	}, nil
}

func (h KafkaHandler) checkPreconditions(_ *aiven.Client, _ logr.Logger, _ client.Object) bool {
	return true
}

func (h KafkaHandler) isActive(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("checking if kafka service is active")

	return checkServiceIsRunning(c, kafka.Spec.Project, kafka.Name), nil
}

func (h KafkaHandler) convert(i client.Object) (*k8soperatorv1alpha1.Kafka, error) {
	kafka, ok := i.(*k8soperatorv1alpha1.Kafka)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Kafka")
	}

	return kafka, nil
}

func (h KafkaHandler) setStatus(kafka *k8soperatorv1alpha1.Kafka, s *aiven.Service) {
	var prVPCID string

	if s.ProjectVPCID != nil {
		prVPCID = *s.ProjectVPCID
	}

	kafka.Status.State = s.State
	kafka.Status.ProjectVPCID = prVPCID
	kafka.Status.Plan = s.Plan
	kafka.Status.MaintenanceWindowTime = s.MaintenanceWindow.TimeOfDay
	kafka.Status.MaintenanceWindowDow = s.MaintenanceWindow.DayOfWeek
	kafka.Status.CloudName = s.CloudName
}

func (h KafkaHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	kafka, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &kafka.Spec.AuthSecretRef
}
