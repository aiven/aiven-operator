// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ValkeyReconciler reconciles a Valkey object
type ValkeyReconciler struct {
	Controller
}

func newValkeyReconciler(c Controller) reconcilerType {
	return &ValkeyReconciler{Controller: c}
}

type ValkeyHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=valkeys,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=valkeys/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=valkeys/finalizers,verbs=get;create;update

func (r *ValkeyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newValkeyAdapter), &v1alpha1.Valkey{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ValkeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Valkey{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newValkeyAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	valkey, ok := object.(*v1alpha1.Valkey)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Valkey")
	}
	return &valkeyAdapter{valkey}, nil
}

// valkeyAdapter handles an Aiven Valkey service
type valkeyAdapter struct {
	*v1alpha1.Valkey
}

func (a *valkeyAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *valkeyAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *valkeyAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *valkeyAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *valkeyAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PASSWORD": s.ServiceUriParams["password"],
		prefix + "PORT":     s.ServiceUriParams["port"],
		prefix + "SSL":      s.ServiceUriParams["ssl"],
		prefix + "USER":     s.ServiceUriParams["user"],
	}

	return newSecret(a, stringData, false), nil
}

func (a *valkeyAdapter) getServiceType() string {
	return "valkey"
}

func (a *valkeyAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *valkeyAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn avngen.Client, old *service.ServiceGetOut) error {
	return nil
}
