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

// MySQLReconciler reconciles a MySQL object
type MySQLReconciler struct {
	Controller
}

func newMySQLReconciler(c Controller) reconcilerType {
	return &MySQLReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=mysqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=mysqls/finalizers,verbs=get;create;update

func (r *MySQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newMySQLAdapter), &v1alpha1.MySQL{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *MySQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MySQL{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newMySQLAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	mysql, ok := object.(*v1alpha1.MySQL)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.MySQL")
	}
	return &mySQLAdapter{mysql}, nil
}

// mySQLAdapter handles an Aiven MySQL service
type mySQLAdapter struct {
	*v1alpha1.MySQL
}

func (a *mySQLAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *mySQLAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *mySQLAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *mySQLAdapter) getUserConfig() any {
	return a.Spec.UserConfig
}

func (a *mySQLAdapter) newSecret(ctx context.Context, s *service.ServiceGetOut) (*corev1.Secret, error) {
	stringData := map[string]string{
		"HOST":        s.ServiceUriParams["host"],
		"PORT":        s.ServiceUriParams["port"],
		"DATABASE":    s.ServiceUriParams["dbname"],
		"USER":        s.ServiceUriParams["user"],
		"PASSWORD":    s.ServiceUriParams["password"],
		"SSL_MODE":    s.ServiceUriParams["ssl-mode"],
		"URI":         s.ServiceUri,
		"REPLICA_URI": *s.ConnectionInfo.MysqlReplicaUri,
	}

	return newSecret(a, stringData, true), nil
}

func (a *mySQLAdapter) getServiceType() string {
	return "mysql"
}

func (a *mySQLAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}

func (a *mySQLAdapter) performUpgradeTaskIfNeeded(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, old *service.ServiceGetOut) error {
	return nil
}
