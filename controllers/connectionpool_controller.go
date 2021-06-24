// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// ConnectionPoolReconciler reconciles a ConnectionPool object
type ConnectionPoolReconciler struct {
	Controller
}

// ConnectionPoolHandler handles an Aiven ConnectionPool
type ConnectionPoolHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=connectionpools/status,verbs=get;update;patch

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cp := &k8soperatorv1alpha1.ConnectionPool{}
	err := r.Get(ctx, req.NamespacedName, cp)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, cp.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &ConnectionPoolHandler{
		client: c,
	}, cp)
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ConnectionPool{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ConnectionPoolHandler) createOrUpdate(i client.Object) (client.Object, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	exists, err := h.exists(cp)
	if err != nil {
		return nil, err
	}
	var reason string
	if !exists {
		_, err := h.client.ConnectionPools.Create(cp.Spec.Project, cp.Spec.ServiceName,
			aiven.CreateConnectionPoolRequest{
				Database: cp.Spec.DatabaseName,
				PoolMode: cp.Spec.PoolMode,
				PoolName: cp.Name,
				PoolSize: cp.Spec.PoolSize,
				Username: cp.Spec.Username,
			})
		if err != nil && !aiven.IsAlreadyExists(err) {
			return nil, err
		}
		reason = "Created"
	} else {
		_, err := h.client.ConnectionPools.Update(cp.Spec.Project, cp.Spec.ServiceName, cp.Name,
			aiven.UpdateConnectionPoolRequest{
				Database: cp.Spec.DatabaseName,
				PoolMode: cp.Spec.PoolMode,
				PoolSize: cp.Spec.PoolSize,
				Username: cp.Spec.Username,
			})
		if err != nil {
			return nil, err
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
		processedGeneration, strconv.FormatInt(cp.GetGeneration(), 10))

	return cp, nil
}

func (h ConnectionPoolHandler) delete(i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = h.client.ConnectionPools.Delete(
		cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ConnectionPoolHandler) exists(cp *k8soperatorv1alpha1.ConnectionPool) (bool, error) {
	conPool, err := h.client.ConnectionPools.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil {
		if aiven.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return conPool != nil, nil
}

func (h ConnectionPoolHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	connPool, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	cp, err := h.client.ConnectionPools.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get ConnectionPool: %w", err)
	}

	s, err := h.client.Services.Get(connPool.Spec.Project, connPool.Spec.ServiceName)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get service: %w", err)
	}

	u, err := h.client.ServiceUsers.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Spec.Username)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get user: %w", err)
	}

	metav1.SetMetaDataAnnotation(&connPool.ObjectMeta, isRunning, "true")

	meta.SetStatusCondition(&connPool.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	return connPool, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(connPool),
			Namespace: connPool.Namespace,
			Labels: map[string]string{
				"app": connPool.Name,
			},
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

func (h ConnectionPoolHandler) checkPreconditions(i client.Object) bool {
	cp, err := h.convert(i)
	if err != nil {
		return false
	}

	if checkServiceIsRunning(h.client, cp.Spec.Project, cp.Spec.ServiceName) {
		db, err := h.client.Databases.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.DatabaseName)
		if err != nil {
			return false
		}

		user, err := h.client.ServiceUsers.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if err != nil {
			return false
		}

		return db != nil && user != nil
	}

	return false
}

func (h *ConnectionPoolHandler) convert(i client.Object) (*k8soperatorv1alpha1.ConnectionPool, error) {
	cp, ok := i.(*k8soperatorv1alpha1.ConnectionPool)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ConnectionPool")
	}

	return cp, nil
}

func (h ConnectionPoolHandler) getSecretName(cp *k8soperatorv1alpha1.ConnectionPool) string {
	if cp.Spec.ConnInfoSecretTarget.Name != "" {
		return cp.Spec.ConnInfoSecretTarget.Name
	}
	return cp.Name
}
