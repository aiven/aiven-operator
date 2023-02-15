// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	Controller
}

type RedisHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=redis/status,verbs=get;update;patch

func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newRedisAdapter), &v1alpha1.Redis{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Redis{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newRedisAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	redis, ok := object.(*v1alpha1.Redis)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Redis")
	}
	return &redisAdapter{redis}, nil
}

// redisAdapter handles an Aiven Redis service
type redisAdapter struct {
	*v1alpha1.Redis
}

func (a *redisAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *redisAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *redisAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *redisAdapter) getUserConfig() any {
	return &a.Spec.UserConfig
}

func (a *redisAdapter) newSecret(s *aiven.Service) (*corev1.Secret, error) {
	name := a.Spec.ConnInfoSecretTarget.Name
	if name == "" {
		name = a.Name
	}

	stringData := map[string]string{
		"HOST":     s.URIParams["host"],
		"PASSWORD": s.URIParams["password"],
		"PORT":     s.URIParams["port"],
		"SSL":      s.URIParams["ssl"],
		"USER":     s.URIParams["user"],
	}

	// Removes empties
	for k, v := range stringData {
		if v == "" {
			delete(stringData, k)
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: a.Namespace},
		StringData: stringData,
	}, nil
}

func (a *redisAdapter) getServiceType() string {
	return "redis"
}

func (a *redisAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}
