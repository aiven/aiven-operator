// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
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

// ConnectionPoolHandler handles an Aiven ConnectionPool
type ConnectionPoolHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools/finalizers,verbs=update

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ConnectionPoolHandler{}, &v1alpha1.ConnectionPool{})
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ConnectionPool{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ConnectionPoolHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
	cp, err := h.convert(i)
	if err != nil {
		return err
	}

	exists, err := h.exists(avn, cp)
	if err != nil {
		return err
	}
	var reason string
	if !exists {
		_, err := avn.ConnectionPools.Create(cp.Spec.Project, cp.Spec.ServiceName,
			aiven.CreateConnectionPoolRequest{
				Database: cp.Spec.DatabaseName,
				PoolMode: cp.Spec.PoolMode,
				PoolName: cp.Name,
				PoolSize: cp.Spec.PoolSize,
				Username: optionalStringPointer(cp.Spec.Username),
			})
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}
		reason = "Created"
	} else {
		_, err := avn.ConnectionPools.Update(cp.Spec.Project, cp.Spec.ServiceName, cp.Name,
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

func (h ConnectionPoolHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = avn.ConnectionPools.Delete(
		cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ConnectionPoolHandler) exists(avn *aiven.Client, cp *v1alpha1.ConnectionPool) (bool, error) {
	conPool, err := avn.ConnectionPools.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil {
		if aiven.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return conPool != nil, nil
}

func (h ConnectionPoolHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	connPool, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	cp, err := avn.ConnectionPools.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get ConnectionPool: %w", err)
	}

	s, err := avn.Services.Get(connPool.Spec.Project, connPool.Spec.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("cannot get service: %w", err)
	}

	metav1.SetMetaDataAnnotation(&connPool.ObjectMeta, instanceIsRunningAnnotation, "true")

	meta.SetStatusCondition(&connPool.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	if len(connPool.Spec.Username) == 0 {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      h.getSecretName(connPool),
				Namespace: connPool.Namespace,
			},
			StringData: map[string]string{
				"PGHOST":       s.URIParams["host"],
				"PGPORT":       s.URIParams["port"],
				"PGDATABASE":   cp.Database,
				"PGUSER":       s.URIParams["user"],
				"PGPASSWORD":   s.URIParams["password"],
				"PGSSLMODE":    s.URIParams["sslmode"],
				"DATABASE_URI": cp.ConnectionURI,
			},
		}, nil
	}

	u, err := avn.ServiceUsers.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Spec.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(connPool),
			Namespace: connPool.Namespace,
		},
		StringData: map[string]string{
			"PGHOST":       s.URIParams["host"],
			"PGPORT":       s.URIParams["port"],
			"PGDATABASE":   cp.Database,
			"PGUSER":       cp.Username,
			"PGPASSWORD":   u.Password,
			"PGSSLMODE":    s.URIParams["sslmode"],
			"DATABASE_URI": cp.ConnectionURI,
		},
	}, nil
}

func (h ConnectionPoolHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&cp.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	check, err := checkServiceIsRunning(avn, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	if check {
		db, err := avn.Databases.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.DatabaseName)
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

func (h ConnectionPoolHandler) getSecretName(cp *v1alpha1.ConnectionPool) string {
	if cp.Spec.ConnInfoSecretTarget.Name != "" {
		return cp.Spec.ConnInfoSecretTarget.Name
	}
	return cp.Name
}
