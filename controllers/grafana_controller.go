// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// GrafanaReconciler reconciles a Grafana object
type GrafanaReconciler struct {
	Controller
}

func newGrafanaReconciler(c Controller) reconcilerType {
	return &GrafanaReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=grafanas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=grafanas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=grafanas/finalizers,verbs=get;create;update

func (r *GrafanaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newGrafanaAdapter), &v1alpha1.Grafana{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *GrafanaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Grafana{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newGrafanaAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	grafana, ok := object.(*v1alpha1.Grafana)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Grafana")
	}
	return &grafanaAdapter{grafana}, nil
}

// grafanaAdapter handles an Aiven Grafana service
type grafanaAdapter struct {
	*v1alpha1.Grafana
}

func (a *grafanaAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *grafanaAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *grafanaAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *grafanaAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *grafanaAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":     s.ServiceUriParams["host"],
		"PORT":     s.ServiceUriParams["port"],
		"USER":     s.ServiceUriParams["user"],
		"PASSWORD": s.ServiceUriParams["password"],
		"URI":      s.ServiceUri,
		"HOSTS":    strings.Join(s.ConnectionInfo.Grafana, ","),
	}

	return newSecret(a, stringData, true), nil
}

func (a *grafanaAdapter) getServiceType() string {
	return "grafana"
}

func (a *grafanaAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *grafanaAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, old *service.ServiceGetOut) error {
	return nil
}
