// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ClickhouseReconciler reconciles a Clickhouse object
type ClickhouseReconciler struct {
	Controller
}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouses/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *ClickhouseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newClickhouseAdapter), &v1alpha1.Clickhouse{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClickhouseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Clickhouse{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newClickhouseAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	clickhouse, ok := object.(*v1alpha1.Clickhouse)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Clickhouse")
	}
	return &clickhouseAdapter{clickhouse}, nil
}

// clickhouseAdapter handles an Aiven Clickhouse service
type clickhouseAdapter struct {
	*v1alpha1.Clickhouse
}

func (a *clickhouseAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *clickhouseAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *clickhouseAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *clickhouseAdapter) getUserConfig() any {
	return &a.Spec.UserConfig
}

func (a *clickhouseAdapter) newSecret(ctx context.Context, s *aiven.Service) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":     s.URIParams["host"],
		prefix + "PASSWORD": s.URIParams["password"],
		prefix + "PORT":     s.URIParams["port"],
		prefix + "USER":     s.URIParams["user"],
		// todo: remove in future releases
		"HOST":     s.URIParams["host"],
		"PASSWORD": s.URIParams["password"],
		"PORT":     s.URIParams["port"],
		"USER":     s.URIParams["user"],
	}

	return newSecret(a, stringData, false), nil
}

func (a *clickhouseAdapter) getServiceType() string {
	return "clickhouse"
}

func (a *clickhouseAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}
