// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiven/aiven-go-client"
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

// +kubebuilder:rbac:groups=aiven.io,resources=grafanas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=grafanas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aiven.io,resources=grafanas/finalizers,verbs=update

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
	return &a.Spec.UserConfig
}

func (a *grafanaAdapter) newSecret(s *aiven.Service) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":     s.URIParams["host"],
		"PORT":     s.URIParams["port"],
		"USER":     s.URIParams["user"],
		"PASSWORD": s.URIParams["password"],
		"URI":      s.URI,
		"HOSTS":    strings.Join(s.ConnectionInfo.GrafanaURIs, ","),
	}

	return newSecret(a, stringData, true), nil
}

func (a *grafanaAdapter) getServiceType() string {
	return "grafana"
}

func (a *grafanaAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}
