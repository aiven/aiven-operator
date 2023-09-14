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

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	Controller
}

//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=postgresqls/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *PostgreSQLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, newGenericServiceHandler(newPostgresSQLAdapter), &v1alpha1.PostgreSQL{})
}

func (r *PostgreSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.PostgreSQL{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func newPostgresSQLAdapter(_ *aiven.Client, object client.Object) (serviceAdapter, error) {
	pg, ok := object.(*v1alpha1.PostgreSQL)
	if !ok {
		return nil, fmt.Errorf("object is not of type v1alpha1.PostgresSQL")
	}
	return &postgresSQLAdapter{pg}, nil
}

// postgresSQLAdapter handles an Aiven PostgresSQL service
type postgresSQLAdapter struct {
	*v1alpha1.PostgreSQL
}

func (a *postgresSQLAdapter) getObjectMeta() *metav1.ObjectMeta {
	return &a.ObjectMeta
}

func (a *postgresSQLAdapter) getServiceStatus() *v1alpha1.ServiceStatus {
	return &a.Status
}

func (a *postgresSQLAdapter) getServiceCommonSpec() *v1alpha1.ServiceCommonSpec {
	return &a.Spec.ServiceCommonSpec
}

func (a *postgresSQLAdapter) getUserConfig() any {
	return &a.Spec.UserConfig
}

func (a *postgresSQLAdapter) newSecret(ctx context.Context, s *aiven.Service) (*corev1.Secret, error) {
	prefix := getSecretPrefix(a)
	stringData := map[string]string{
		prefix + "HOST":         s.URIParams["host"],
		prefix + "PORT":         s.URIParams["port"],
		prefix + "DATABASE":     s.URIParams["dbname"],
		prefix + "USER":         s.URIParams["user"],
		prefix + "PASSWORD":     s.URIParams["password"],
		prefix + "SSLMODE":      s.URIParams["sslmode"],
		prefix + "DATABASE_URI": s.URI,
		// todo: remove in future releases
		"PGHOST":       s.URIParams["host"],
		"PGPORT":       s.URIParams["port"],
		"PGDATABASE":   s.URIParams["dbname"],
		"PGUSER":       s.URIParams["user"],
		"PGPASSWORD":   s.URIParams["password"],
		"PGSSLMODE":    s.URIParams["sslmode"],
		"DATABASE_URI": s.URI,
	}

	return newSecret(a, stringData, false), nil
}

func (a *postgresSQLAdapter) getServiceType() string {
	return "pg"
}

func (a *postgresSQLAdapter) getDiskSpace() string {
	return a.Spec.DiskSpace
}
