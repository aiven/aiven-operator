// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

type KafkaConnectHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/status,verbs=get;update;patch

func (r *KafkaConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaConnectHandler{}, &v1alpha1.KafkaConnect{})
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaConnect{}).
		Complete(r)
}

func (h KafkaConnectHandler) exists(avn *aiven.Client, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	s, err := avn.Services.Get(kc.Spec.Project, kc.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaConnectHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
	kc, err := h.convert(i)
	if err != nil {
		return err
	}

	projectVPCID := kc.Spec.ProjectVPCID
	if projectVPCID == "" {
		if p := v1alpha1.FindProjectVPC(refs); p != nil {
			projectVPCID = p.Status.ID
		}
	}

	exits, err := h.exists(avn, i)
	if err != nil {
		return err
	}
	var reason string
	if !exits {
		_, err := avn.Services.Create(kc.Spec.Project, aiven.CreateServiceRequest{
			Cloud: kc.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				kc.Spec.MaintenanceWindowDow,
				kc.Spec.MaintenanceWindowTime),
			Plan:                kc.Spec.Plan,
			ProjectVPCID:        toOptionalStringPointer(projectVPCID),
			ServiceName:         kc.Name,
			ServiceType:         "kafka_connect",
			UserConfig:          UserConfigurationToAPI(kc.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
		})
		if err != nil {
			return err
		}

		reason = "Created"
	} else {
		_, err := avn.Services.Update(kc.Spec.Project, kc.Name, aiven.UpdateServiceRequest{
			Cloud: kc.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				kc.Spec.MaintenanceWindowDow,
				kc.Spec.MaintenanceWindowTime),
			Plan:         kc.Spec.Plan,
			ProjectVPCID: toOptionalStringPointer(projectVPCID),
			UserConfig:   UserConfigurationToAPI(kc.Spec.UserConfig).(map[string]interface{}),
			Powered:      true,
		})
		if err != nil {
			return err
		}

		reason = "Updated"
	}

	meta.SetStatusCondition(&kc.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&kc.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&kc.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(kc.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaConnectHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	if err := avn.Services.Delete(kc.Spec.Project, kc.Name); err != nil {
		if !aiven.IsNotFound(err) {
			return false, fmt.Errorf("aiven client delete KafkaConnect error: %w", err)
		}
	}

	return true, nil
}

func (h KafkaConnectHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(kc.Spec.Project, kc.Name)
	if err != nil {
		return nil, err
	}

	kc.Status.State = s.State

	if s.State == "RUNNING" {
		meta.SetStatusCondition(&kc.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&kc.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return nil, nil
}

func (h KafkaConnectHandler) convert(i client.Object) (*v1alpha1.KafkaConnect, error) {
	kc, ok := i.(*v1alpha1.KafkaConnect)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnect")
	}

	return kc, nil
}

func (h KafkaConnectHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}
