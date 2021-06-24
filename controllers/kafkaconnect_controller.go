// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// KafkaConnectReconciler reconciles a KafkaConnect object
type KafkaConnectReconciler struct {
	Controller
}

type KafkaConnectHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaconnects/status,verbs=get;update;patch

func (r *KafkaConnectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kc := &k8soperatorv1alpha1.KafkaConnect{}
	err := r.Get(ctx, req.NamespacedName, kc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, kc.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &KafkaConnectHandler{
		client: c,
	}, kc)
}

func (r *KafkaConnectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaConnect{}).
		Complete(r)
}

func (h KafkaConnectHandler) exists(i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	s, err := h.client.Services.Get(kc.Spec.Project, kc.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h KafkaConnectHandler) createOrUpdate(i client.Object) (client.Object, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	var prVPCID *string
	if kc.Spec.ProjectVPCID != "" {
		prVPCID = &kc.Spec.ProjectVPCID
	}

	exits, err := h.exists(i)
	if err != nil {
		return nil, err
	}
	var reason string
	if !exits {
		_, err := h.client.Services.Create(kc.Spec.Project, aiven.CreateServiceRequest{
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

		reason = "Created"
	} else {
		_, err := h.client.Services.Update(kc.Spec.Project, kc.Name, aiven.UpdateServiceRequest{
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

		reason = "Updated"
	}

	meta.SetStatusCondition(&kc.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&kc.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&kc.ObjectMeta,
		processedGeneration, strconv.FormatInt(kc.GetGeneration(), 10))

	return kc, nil
}

func (h KafkaConnectHandler) delete(i client.Object) (bool, error) {
	kc, err := h.convert(i)
	if err != nil {
		return false, err
	}

	if err := h.client.Services.Delete(kc.Spec.Project, kc.Name); err != nil {
		if !aiven.IsNotFound(err) {
			return false, fmt.Errorf("aiven client delete KafkaConnect error: %w", err)
		}
	}

	return true, nil
}

func (h KafkaConnectHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	kc, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	if checkServiceIsRunning(h.client, kc.Spec.Project, kc.Name) {
		meta.SetStatusCondition(&kc.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "Get",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&kc.ObjectMeta, isRunning, "true")
	}

	return kc, nil, nil
}

func (h KafkaConnectHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaConnect, error) {
	kc, ok := i.(*k8soperatorv1alpha1.KafkaConnect)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaConnect")
	}

	return kc, nil
}

func (h KafkaConnectHandler) checkPreconditions(client.Object) bool {
	return true
}
