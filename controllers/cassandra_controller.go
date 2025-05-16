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

// CassandraReconciler reconciles a Cassandra object
type CassandraReconciler struct {
	Controller
}

func newCassandraReconciler(c Controller) reconcilerType {
	return &CassandraReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=cassandras,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=cassandras/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=cassandras/finalizers,verbs=get;create;update

func (r *CassandraReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newCassandraAdapter), &v1alpha1.Cassandra{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *CassandraReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Cassandra{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newCassandraAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	cassandra, ok := object.(*v1alpha1.Cassandra)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.Cassandra")
	}
	return &cassandraAdapter{cassandra}, nil
}

// cassandraAdapter handles an Aiven Cassandra service
type cassandraAdapter struct {
	*v1alpha1.Cassandra
}

func (a *cassandraAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *cassandraAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *cassandraAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *cassandraAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *cassandraAdapter) newSecret(_ context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":     s.ServiceUriParams["host"],
		"PORT":     s.ServiceUriParams["port"],
		"USER":     s.ServiceUriParams["user"],
		"PASSWORD": s.ServiceUriParams["password"],
		"URI":      s.ServiceUri,
		"HOSTS":    strings.Join(s.ConnectionInfo.Cassandra, ","),
	}

	return newSecret(a, stringData, true), nil
}

func (a *cassandraAdapter) getServiceType() string {
	return "cassandra"
}

func (a *cassandraAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *cassandraAdapter) performUpgradeTaskIfNeeded(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}

func (a *cassandraAdapter) createOrUpdateServiceSpecific(_ context.Context, _ avngen.Client, _ *service.ServiceGetOut) error {
	return nil
}
