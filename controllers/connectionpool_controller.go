// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
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

// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools/finalizers,verbs=get;list;watch;create;update;patch;delete

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ConnectionPoolHandler{}, &v1alpha1.ConnectionPool{})
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ConnectionPool{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ConnectionPoolHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	cp, err := h.convert(obj)
	if err != nil {
		return err
	}

	exists, err := h.exists(ctx, avn, cp)
	if err != nil {
		return err
	}
	var reason string
	if !exists {
		_, err := avn.ConnectionPools.Create(ctx, cp.Spec.Project, cp.Spec.ServiceName,
			aiven.CreateConnectionPoolRequest{
				Database: cp.Spec.DatabaseName,
				PoolMode: cp.Spec.PoolMode,
				PoolName: cp.Name,
				PoolSize: cp.Spec.PoolSize,
				Username: optionalStringPointer(cp.Spec.Username),
			})
		if err != nil && !isAlreadyExists(err) {
			return err
		}
		reason = "Created"
	} else {
		_, err := avn.ConnectionPools.Update(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name,
			aiven.UpdateConnectionPoolRequest{
				Database: cp.Spec.DatabaseName,
				PoolMode: cp.Spec.PoolMode,
				PoolSize: cp.Spec.PoolSize,
				Username: optionalStringPointer(cp.Spec.Username),
			})
		if err != nil {
			return err
		}
		reason = "Updated"
	}

	meta.SetStatusCondition(&cp.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&cp.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&cp.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(cp.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ConnectionPoolHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	cp, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = avn.ConnectionPools.Delete(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ConnectionPoolHandler) exists(ctx context.Context, avn *aiven.Client, cp *v1alpha1.ConnectionPool) (bool, error) {
	conPool, err := avn.ConnectionPools.Get(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return conPool != nil, nil
}

func (h ConnectionPoolHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	connPool, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	cp, err := avn.ConnectionPools.Get(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get ConnectionPool: %w", err)
	}

	cert, err := avnGen.ProjectKmsGetCA(ctx, connPool.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve project CA certificate: %w", err)
	}

	// The pool comes with its own port
	poolURI, err := url.Parse(cp.ConnectionURI)
	if err != nil {
		return nil, fmt.Errorf("can't parse ConnectionPool URI: %w", err)
	}

	s, err := avn.Services.Get(ctx, connPool.Spec.Project, connPool.Spec.ServiceName)
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
			prefix + "HOST":         s.URIParams["host"],
			prefix + "PORT":         poolURI.Port(),
			prefix + "DATABASE":     cp.Database,
			prefix + "USER":         s.URIParams["user"],
			prefix + "PASSWORD":     s.URIParams["password"],
			prefix + "SSLMODE":      s.URIParams["sslmode"],
			prefix + "DATABASE_URI": cp.ConnectionURI,
			prefix + "CA_CERT":      cert,
			// todo: remove in future releases
			"PGHOST":       s.URIParams["host"],
			"PGPORT":       poolURI.Port(),
			"PGDATABASE":   cp.Database,
			"PGUSER":       s.URIParams["user"],
			"PGPASSWORD":   s.URIParams["password"],
			"PGSSLMODE":    s.URIParams["sslmode"],
			"DATABASE_URI": cp.ConnectionURI,
		}

		return newSecret(connPool, stringData, false), nil
	}

	u, err := avn.ServiceUsers.Get(ctx, connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Spec.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	prefix := getSecretPrefix(connPool)
	stringData := map[string]string{
		prefix + "NAME":         connPool.Name,
		prefix + "HOST":         s.URIParams["host"],
		prefix + "PORT":         poolURI.Port(),
		prefix + "DATABASE":     cp.Database,
		prefix + "USER":         cp.Username,
		prefix + "PASSWORD":     u.Password,
		prefix + "SSLMODE":      s.URIParams["sslmode"],
		prefix + "DATABASE_URI": cp.ConnectionURI,
		prefix + "CA_CERT":      cert,
		// todo: remove in future releases
		"PGHOST":       s.URIParams["host"],
		"PGPORT":       poolURI.Port(),
		"PGDATABASE":   cp.Database,
		"PGUSER":       cp.Username,
		"PGPASSWORD":   u.Password,
		"PGSSLMODE":    s.URIParams["sslmode"],
		"DATABASE_URI": cp.ConnectionURI,
	}
	return newSecret(connPool, stringData, false), nil
}

func (h ConnectionPoolHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	cp, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&cp.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	check, err := checkServiceIsRunning(ctx, avn, avnGen, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	if check {
		db, err := avn.Databases.Get(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.DatabaseName)
		if err != nil {
			return false, err
		}

		return db != nil, nil
	}

	return false, nil
}

func (h ConnectionPoolHandler) convert(i client.Object) (*v1alpha1.ConnectionPool, error) {
	cp, ok := i.(*v1alpha1.ConnectionPool)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ConnectionPool")
	}

	return cp, nil
}
