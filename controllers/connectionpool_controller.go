// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/url"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/postgresql"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ConnectionPoolReconciler reconciles a ConnectionPool object
type ConnectionPoolReconciler struct {
	Controller
}

func newConnectionPoolReconciler(c Controller) reconcilerType {
	return &ConnectionPoolReconciler{Controller: c}
}

// ConnectionPoolHandler handles an Aiven ConnectionPool
type ConnectionPoolHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools/finalizers,verbs=get;create;update

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ConnectionPoolHandler{}, &v1alpha1.ConnectionPool{})
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ConnectionPool{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ConnectionPoolHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) (bool, error) {
	connPool, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// check if the connection pool already exists
	s, err := avnGen.ServiceGet(ctx, connPool.Spec.Project, connPool.Spec.ServiceName)
	if err != nil {
		return false, fmt.Errorf("cannot get service: %w", err)
	}

	exists := false
	for _, connP := range s.ConnectionPools {
		if connP.PoolName == connPool.Name {
			exists = true
			break
		}
	}

	if !exists {
		req := postgresql.ServicePgbouncerCreateIn{
			Database: connPool.Spec.DatabaseName,
			PoolMode: connPool.Spec.PoolMode,
			PoolName: connPool.Name,
			PoolSize: NilIfZero(connPool.Spec.PoolSize),
			Username: NilIfZero(connPool.Spec.Username),
		}

		if err = avnGen.ServicePGBouncerCreate(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, &req); err != nil && !isAlreadyExists(err) {
			return false, fmt.Errorf("cannot create connection pool: %w", err)
		}
	} else {
		req := postgresql.ServicePgbouncerUpdateIn{
			Database: NilIfZero(connPool.Spec.DatabaseName),
			PoolMode: connPool.Spec.PoolMode,
			PoolSize: NilIfZero(connPool.Spec.PoolSize),
			Username: NilIfZero(connPool.Spec.Username),
		}

		if err = avnGen.ServicePGBouncerUpdate(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Name, &req); err != nil {
			return false, fmt.Errorf("cannot update connection pool: %w", err)
		}
	}

	return !exists, nil
}

func (h ConnectionPoolHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	cp, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if err = avnGen.ServicePGBouncerDelete(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name); err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ConnectionPoolHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	connPool, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	// Search the connection pool
	var cp *service.ConnectionPoolOut
	serviceList, err := avnGen.ServiceGet(ctx, connPool.Spec.Project, connPool.Spec.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("cannot get service: %w", err)
	}

	for _, connP := range serviceList.ConnectionPools {
		if connP.PoolName == connPool.Name {
			cp = &connP
		}
	}

	if cp == nil {
		return nil, fmt.Errorf("connection pool %q not found", connPool.Name)
	}

	cert, err := avnGen.ProjectKmsGetCA(ctx, connPool.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve project CA certificate: %w", err)
	}

	// The pool comes with its own port
	poolURI, err := url.Parse(cp.ConnectionUri)
	if err != nil {
		return nil, fmt.Errorf("can't parse ConnectionPool URI: %w", err)
	}

	s, err := avnGen.ServiceGet(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, service.ServiceGetIncludeSecrets(true))
	if err != nil {
		return nil, fmt.Errorf("cannot get service: %w", err)
	}

	metav1.SetMetaDataAnnotation(&connPool.ObjectMeta, instanceIsRunningAnnotation, "true")

	meta.SetStatusCondition(&connPool.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	if len(connPool.Spec.Username) == 0 {
		prefix := getSecretPrefix(connPool)
		stringData := map[string]string{
			prefix + "NAME":         connPool.Name,
			prefix + "HOST":         s.ServiceUriParams["host"],
			prefix + "PORT":         poolURI.Port(),
			prefix + "DATABASE":     cp.Database,
			prefix + "USER":         s.ServiceUriParams["user"],
			prefix + "PASSWORD":     s.ServiceUriParams["password"],
			prefix + "SSLMODE":      s.ServiceUriParams["sslmode"],
			prefix + "DATABASE_URI": cp.ConnectionUri,
			prefix + "CA_CERT":      cert,
			// todo: remove in future releases
			"PGHOST":       s.ServiceUriParams["host"],
			"PGPORT":       poolURI.Port(),
			"PGDATABASE":   cp.Database,
			"PGUSER":       s.ServiceUriParams["user"],
			"PGPASSWORD":   s.ServiceUriParams["password"],
			"PGSSLMODE":    s.ServiceUriParams["sslmode"],
			"DATABASE_URI": cp.ConnectionUri,
		}

		return newSecret(connPool, stringData, false), nil
	}

	u, err := avnGen.ServiceUserGet(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Spec.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	prefix := getSecretPrefix(connPool)
	stringData := map[string]string{
		prefix + "NAME":     connPool.Name,
		prefix + "HOST":     s.ServiceUriParams["host"],
		prefix + "PORT":     poolURI.Port(),
		prefix + "DATABASE": cp.Database,
		prefix + "USER": func() string {
			if cp.Username != nil {
				return *cp.Username
			}

			// this should never happen, but we have to handle this case anyway
			// this behaviour compatible with the previous implementation with aiven.Client
			return ""
		}(),
		prefix + "PASSWORD":     u.Password,
		prefix + "SSLMODE":      s.ServiceUriParams["sslmode"],
		prefix + "DATABASE_URI": cp.ConnectionUri,
		prefix + "CA_CERT":      cert,
		// todo: remove in future releases
		"PGHOST":     s.ServiceUriParams["host"],
		"PGPORT":     poolURI.Port(),
		"PGDATABASE": cp.Database,
		"PGUSER": func() string {
			if cp.Username != nil {
				return *cp.Username
			}

			// this should never happen, but we have to handle this case anyway
			// this behaviour compatible with the previous implementation with aiven.Client
			return ""
		}(),
		"PGPASSWORD":   u.Password,
		"PGSSLMODE":    s.ServiceUriParams["sslmode"],
		"DATABASE_URI": cp.ConnectionUri,
	}
	return newSecret(connPool, stringData, false), nil
}

func (h ConnectionPoolHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	cp, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&cp.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	isRunning, err := checkServiceIsOperational(ctx, avnGen, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	if !isRunning {
		return false, nil
	}

	// check if the database exists
	var exists bool
	dbList, err := avnGen.ServiceGet(ctx, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	for _, db := range dbList.Databases {
		if db == cp.Spec.DatabaseName {
			exists = true
			break
		}
	}

	if !exists {
		return false, nil
	}

	if cp.Spec.Username != "" {
		_, err = avnGen.ServiceUserGet(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if isNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (h ConnectionPoolHandler) convert(i client.Object) (*v1alpha1.ConnectionPool, error) {
	cp, ok := i.(*v1alpha1.ConnectionPool)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ConnectionPool")
	}

	return cp, nil
}
