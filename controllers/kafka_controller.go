// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaReconciler reconciles a Kafka object
type KafkaReconciler struct {
	Controller
}

// KafkaHandler handles an Aiven Kafka service
type KafkaHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkas/status,verbs=get;update;patch

func (r *KafkaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaHandler{}, &v1alpha1.Kafka{})
}

func (r *KafkaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Kafka{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h KafkaHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	kafka, err := h.convert(i)
	if err != nil {
		return err
	}

	exists, err := h.exists(avn, kafka)
	if err != nil {
		return err
	}

	var reason string
	if !exists {
		_, err = avn.Services.Create(kafka.Spec.Project, aiven.CreateServiceRequest{
			Cloud: kafka.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				kafka.Spec.MaintenanceWindowDow,
				kafka.Spec.MaintenanceWindowTime),
			Plan:                kafka.Spec.Plan,
			ProjectVPCID:        toOptionalStringPointer(kafka.Spec.ProjectVPCID),
			ServiceName:         kafka.Name,
			ServiceType:         "kafka",
			UserConfig:          UserConfigurationToAPI(kafka.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
			DiskSpaceMB:         v1alpha1.ConvertDiscSpace(kafka.Spec.DiskSpace),
		})
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}

		reason = "Created"
	} else {
		_, err := avn.Services.Update(kafka.Spec.Project, kafka.Name, aiven.UpdateServiceRequest{
			Cloud: kafka.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				kafka.Spec.MaintenanceWindowDow,
				kafka.Spec.MaintenanceWindowTime),
			Plan:         kafka.Spec.Plan,
			ProjectVPCID: toOptionalStringPointer(kafka.Spec.ProjectVPCID),
			UserConfig:   UserConfigurationToAPI(kafka.Spec.UserConfig).(map[string]interface{}),
			DiskSpaceMB:  v1alpha1.ConvertDiscSpace(kafka.Spec.DiskSpace),
			Powered:      true,
			Karapace:     kafka.Spec.Karapace,
		})
		if err != nil {
			return err
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
		processedGenerationAnnotation, strconv.FormatInt(kafka.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	if err := avn.Services.Delete(kafka.Spec.Project, kafka.Name); err != nil {
		if !aiven.IsNotFound(err) {
			return false, fmt.Errorf("aiven client delete Kafka error: %w", err)
		}
	}

	return true, nil
}

func (h KafkaHandler) exists(avn *aiven.Client, kafka *v1alpha1.Kafka) (bool, error) {
	s, err := avn.Services.Get(kafka.Spec.Project, kafka.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	kafka, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(kafka.Spec.Project, kafka.Name)
	if err != nil {
		return nil, err
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

		metav1.SetMetaDataAnnotation(&kafka.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	caCert, err := avn.CA.Get(kafka.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	return &corev1.Secret{
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

func (h KafkaHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
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
