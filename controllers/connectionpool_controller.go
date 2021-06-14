// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConnectionPoolReconciler reconciles a ConnectionPool object
type ConnectionPoolReconciler struct {
	Controller
}

// ConnectionPoolHandler handles an Aiven ConnectionPool
type ConnectionPoolHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=connectionpools/status,verbs=get;update;patch

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("connectionpool", req.NamespacedName)
	log.Info("reconciling aiven connection pool")

	const finalizer = "connectionpool-finalizer.k8s-operator.aiven.io"
	cp := &k8soperatorv1alpha1.ConnectionPool{}
	return r.reconcileInstance(&ConnectionPoolHandler{}, ctx, log, req, cp, finalizer)
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ConnectionPool{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ConnectionPoolHandler) create(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("creating a connection pool on aiven side")

	conPool, err := c.ConnectionPools.Create(cp.Spec.Project, cp.Spec.ServiceName,
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

	h.setStatus(cp, conPool)

	return cp, nil
}

func (h ConnectionPoolHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("deleting a connection pool on aiven side")

	err = c.ConnectionPools.Delete(
		cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ConnectionPoolHandler) exists(c *aiven.Client, _ logr.Logger, i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	conPool, err := c.ConnectionPools.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil {
		if aiven.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return conPool != nil, nil
}

func (h ConnectionPoolHandler) update(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("updating a connection pool on aiven side")

	conPool, err := c.ConnectionPools.Update(cp.Spec.Project, cp.Spec.ServiceName, cp.Name,
		aiven.UpdateConnectionPoolRequest{
			Database: cp.Spec.DatabaseName,
			PoolMode: cp.Spec.PoolMode,
			PoolSize: cp.Spec.PoolSize,
			Username: cp.Spec.Username,
		})
	if err != nil {
		return nil, err
	}

	h.setStatus(cp, conPool)

	return cp, nil
}

func (h ConnectionPoolHandler) getSecret(c *aiven.Client, log logr.Logger, i client.Object) (*corev1.Secret, error) {
	connPool, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("getting connection pool secret")

	cp, err := c.ConnectionPools.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Name)
	if err != nil {
		return nil, fmt.Errorf("cannot get ConnectionPool: %w", err)
	}

	s, err := c.Services.Get(connPool.Spec.Project, connPool.Spec.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("cannot get service: %w", err)
	}

	u, err := c.ServiceUsers.Get(connPool.Spec.Project, connPool.Spec.ServiceName, connPool.Spec.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot get user: %w", err)
	}

	return &corev1.Secret{
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

func (h ConnectionPoolHandler) checkPreconditions(c *aiven.Client, log logr.Logger, i client.Object) bool {
	cp, err := h.convert(i)
	if err != nil {
		return false
	}

	log.Info("checking connection pool preconditions")

	if checkServiceIsRunning(c, cp.Spec.Project, cp.Spec.ServiceName) {
		log.Info("checking if database exists")
		db, err := c.Databases.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.DatabaseName)
		if err != nil {
			return false
		}

		user, err := c.ServiceUsers.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if err != nil {
			return false
		}

		return db != nil && user != nil
	}

	return false
}

func (h ConnectionPoolHandler) isActive(_ *aiven.Client, _ logr.Logger, _ client.Object) (bool, error) {
	return true, nil
}

func (h *ConnectionPoolHandler) convert(i client.Object) (*k8soperatorv1alpha1.ConnectionPool, error) {
	cp, ok := i.(*k8soperatorv1alpha1.ConnectionPool)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ConnectionPool")
	}

	return cp, nil
}

// updateCRStatus updates Kubernetes Custom Resource status
func (h ConnectionPoolHandler) setStatus(cp *k8soperatorv1alpha1.ConnectionPool, conPool *aiven.ConnectionPool) {
	cp.Status.Username = conPool.Username
	cp.Status.PoolMode = conPool.PoolMode
	cp.Status.DatabaseName = conPool.Database
	cp.Status.PoolSize = conPool.PoolSize
	cp.Status.ServiceName = cp.Spec.ServiceName
	cp.Status.Project = cp.Spec.Project
}

func (h ConnectionPoolHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	cp, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &cp.Spec.AuthSecretRef
}

func (h ConnectionPoolHandler) getSecretName(cp *k8soperatorv1alpha1.ConnectionPool) string {
	if cp.Spec.ConnInfoSecretTarget.Name != "" {
		return cp.Spec.ConnInfoSecretTarget.Name
	}
	return cp.Name
}
