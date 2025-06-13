// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// FlinkReconciler reconciles a Flink object
type FlinkReconciler struct {
	Controller
}

func newFlinkReconciler(c Controller) reconcilerType {
	return &FlinkReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=flinks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=flinks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=flinks/finalizers,verbs=get;create;update

func (r *FlinkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newFlinkAdapter, r.Log), &v1alpha1.Flink{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlinkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Flink{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newFlinkAdapter(object client.Object) (serviceAdapter, error) {
	flink, ok := object.(*v1alpha1.Flink)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Flink")
	}
	return &flinkAdapter{flink}, nil
}

// flinkAdapter handles an Aiven Flink service
type flinkAdapter struct {
	*v1alpha1.Flink
}

func (a *flinkAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *flinkAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *flinkAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *flinkAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *flinkAdapter) newSecret(_ context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":     s.ServiceUriParams["host"],
		"USER":     s.ServiceUriParams["user"],
		"PASSWORD": s.ServiceUriParams["password"],
		"URI":      s.ServiceUri,
		"HOSTS":    strings.Join(s.ConnectionInfo.Flink, ","),
	}

	return newSecret(a, stringData, true), nil
}

func (a *flinkAdapter) getServiceType() serviceType {
	return serviceTypeFlink
}

func (a *flinkAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *flinkAdapter) performUpgradeTaskIfNeeded(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *flinkAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}
