// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	Controller
}

type RedisHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=redis/status,verbs=get;update;patch

func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, RedisHandler{}, &v1alpha1.Redis{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Redis{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h RedisHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	redis, err := h.convert(i)
	if err != nil {
		return err
	}

	var prVPCID *string
	if redis.Spec.ProjectVPCID != "" {
		prVPCID = &redis.Spec.ProjectVPCID
	}

	exists, err := h.exists(avn, redis)
	if err != nil {
		return err
	}

	var reason string
	if !exists {
		_, err := avn.Services.Create(redis.Spec.Project, aiven.CreateServiceRequest{
			Cloud: redis.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				redis.Spec.MaintenanceWindowDow,
				redis.Spec.MaintenanceWindowTime),
			Plan:                redis.Spec.Plan,
			ProjectVPCID:        prVPCID,
			ServiceName:         redis.Name,
			ServiceType:         "redis",
			UserConfig:          UserConfigurationToAPI(redis.Spec.UserConfig).(map[string]interface{}),
			ServiceIntegrations: nil,
		})
		if err != nil {
			return err
		}

		reason = "Created"
	} else {
		_, err := avn.Services.Update(redis.Spec.Project, redis.Name, aiven.UpdateServiceRequest{
			Cloud: redis.Spec.CloudName,
			MaintenanceWindow: getMaintenanceWindow(
				redis.Spec.MaintenanceWindowDow,
				redis.Spec.MaintenanceWindowTime),
			Plan:         redis.Spec.Plan,
			ProjectVPCID: prVPCID,
			UserConfig:   UserConfigurationToAPI(redis.Spec.UserConfig).(map[string]interface{}),
			Powered:      true,
		})
		if err != nil {
			return err
		}

		reason = "Updated"
	}

	meta.SetStatusCondition(&redis.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&redis.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&redis.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(redis.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h RedisHandler) convert(o client.Object) (*v1alpha1.Redis, error) {
	r, ok := o.(*v1alpha1.Redis)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Redis")
	}
	return r, nil
}

func (h RedisHandler) exists(avn *aiven.Client, r *v1alpha1.Redis) (bool, error) {
	s, err := avn.Services.Get(r.Spec.Project, r.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return s != nil, nil
}

func (h RedisHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	redis, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete Redis on Aiven side
	if err := avn.Services.Delete(redis.Spec.Project, redis.Name); err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete redis error: %w", err)
	}

	return true, nil
}

func (h RedisHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	redis, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(redis.Spec.Project, redis.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get redis: %w", err)
	}

	redis.Status.State = s.State

	if s.State == "RUNNING" {
		meta.SetStatusCondition(&redis.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&redis.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(redis),
			Namespace: redis.Namespace,
		},
		StringData: map[string]string{
			"HOST":     s.URIParams["host"],
			"PASSWORD": s.URIParams["password"],
			"PORT":     s.URIParams["port"],
			"SSL":      s.URIParams["ssl"],
			"USER":     s.URIParams["user"],
		},
	}, nil
}

func (h RedisHandler) getSecretName(redis *v1alpha1.Redis) string {
	if redis.Spec.ConnInfoSecretTarget.Name != "" {
		return redis.Spec.ConnInfoSecretTarget.Name
	}
	return redis.Name
}

func (h RedisHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}
