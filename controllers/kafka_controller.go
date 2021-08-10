// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	v1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	Controller
}

// KafkaHandler handles an Aiven Kafka service
type KafkaHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkas/status,verbs=get;update;patch

func (r *KafkaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var kafka *v1alpha1.Kafka
	if err := r.Get(ctx, req.NamespacedName, kafka); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("unable to get kafka resource: %w", err)
	}

	c, err := r.InitAivenClient(ctx, req, kafka.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to create aiven client: %w", err)
	}

	return r.reconcileInstance(ctx, &KafkaHandler{client: c}, kafka)
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kafka{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h *KafkaHandler) createOrUpdate(i client.Object) (client.Object, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	var prVPCID *string
	if kafka.Spec.ProjectVPCID != "" {
		prVPCID = &kafka.Spec.ProjectVPCID
	}

	exists, err := h.exists(kafka)
	if err != nil {
		return nil, err
	}

	var reason string
	if !exists {
		_, err = h.client.Services.Create(kafka.Spec.Project, aiven.CreateServiceRequest{
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

		reason = "Created"
	} else {
		_, err := h.client.Services.Update(kafka.Spec.Project, kafka.Name, aiven.UpdateServiceRequest{
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

		reason = "Updated"
	}

	meta.SetStatusCondition(&kafka.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&kafka.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&kafka.ObjectMeta,
		processedGeneration, strconv.FormatInt(kafka.GetGeneration(), formatIntBaseDecimal))

	return kafka, nil
}

func (h KafkaHandler) delete(i client.Object) (bool, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	if err := h.client.Services.Delete(kafka.Spec.Project, kafka.Name); err != nil {
		if !aiven.IsNotFound(err) {
			return false, fmt.Errorf("aiven client delete Kafka error: %w", err)
		}
	}

	return true, nil
}

func (h KafkaHandler) exists(kafka *v1alpha1.Kafka) (bool, error) {
	s, err := h.client.Services.Get(kafka.Spec.Project, kafka.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	s, err := h.client.Services.Get(kafka.Spec.Project, kafka.Name)
	if err != nil {
		return nil, nil, err
	}

	var userName, password string
	if len(s.Users) > 0 {
		userName = s.Users[0].Username
		password = s.Users[0].Password
	}

	params := s.URIParams
	kafka.Status.State = s.State

	if s.State == "RUNNING" {
		meta.SetStatusCondition(&kafka.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&kafka.ObjectMeta, isRunning, "true")
	}

	caCert, err := h.client.CA.Get(kafka.Spec.Project)
	if err != nil {
		return nil, nil, fmt.Errorf("aiven client error %w", err)
	}

	return kafka, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(kafka),
			Namespace: kafka.Namespace,
		},
		StringData: map[string]string{
			"HOST":        params["host"],
			"PORT":        params["port"],
			"PASSWORD":    password,
			"USERNAME":    userName,
			"ACCESS_CERT": s.ConnectionInfo.KafkaAccessCert,
			"ACCESS_KEY":  s.ConnectionInfo.KafkaAccessKey,
			"CA_CERT":     caCert,
		},
	}, nil
}

func (h KafkaHandler) checkPreconditions(_ client.Object) bool {
	return true
}

func (h KafkaHandler) convert(i client.Object) (*v1alpha1.Kafka, error) {
	kafka, ok := i.(*v1alpha1.Kafka)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Kafka")
	}

	return kafka, nil
}

func (h KafkaHandler) getSecretName(kafka *v1alpha1.Kafka) string {
	if kafka.Spec.ConnInfoSecretTarget.Name != "" {
		return kafka.Spec.ConnInfoSecretTarget.Name
	}
	return kafka.Name
}
