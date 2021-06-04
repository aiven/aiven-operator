// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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
	log.Info("Reconciling Aiven ConnectionPool")

	const finalizer = "connectionpool-finalizer.k8s-operator.aiven.io"
	cp := &k8soperatorv1alpha1.ConnectionPool{}
	return r.reconcileInstance(&ConnectionPoolHandler{}, ctx, log, req, cp, finalizer)
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ConnectionPool{}).
		Complete(r)
}

func (h ConnectionPoolHandler) create(log logr.Logger, i client.Object) (client.Object, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating a ConnectionPool on Aiven side")

	conPool, err := aivenClient.ConnectionPools.Create(cp.Spec.Project, cp.Spec.ServiceName,
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

func (h ConnectionPoolHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	log.Info("Deleting a ConnectionPool on Aiven side")

	err = aivenClient.ConnectionPools.Delete(
		cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if !aiven.IsNotFound(err) {
		return nil, false, err
	}

	return nil, true, nil
}

func (h ConnectionPoolHandler) exists(_ logr.Logger, i client.Object) (bool, error) {
	cp, err := h.convert(i)
	if err != nil {
		return false, err
	}

	conPool, err := aivenClient.ConnectionPools.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Name)
	if err != nil {
		if aiven.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return conPool != nil, nil
}

func (h ConnectionPoolHandler) update(log logr.Logger, i client.Object) (client.Object, error) {
	cp, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Updating a ConnectionPool on Aiven side")

	conPool, err := aivenClient.ConnectionPools.Update(cp.Spec.Project, cp.Spec.ServiceName, cp.Name,
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

func (h ConnectionPoolHandler) getSecret(_ logr.Logger, _ client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h ConnectionPoolHandler) checkPreconditions(log logr.Logger, i client.Object) bool {
	cp, err := h.convert(i)
	if err != nil {
		return false
	}

	log.Info("Checking ConnectionPool preconditions")

	if checkServiceIsRunning(cp.Spec.Project, cp.Spec.ServiceName) {
		log.Info("Checking if database exists")
		db, err := aivenClient.Databases.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.DatabaseName)
		if err != nil {
			return false
		}

		user, err := aivenClient.ServiceUsers.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if err != nil {
			return false
		}

		return db != nil && user != nil
	}

	return false
}

func (h ConnectionPoolHandler) isActive(_ logr.Logger, _ client.Object) (bool, error) {
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
func (h *ConnectionPoolHandler) setStatus(cp *k8soperatorv1alpha1.ConnectionPool, conPool *aiven.ConnectionPool) {
	cp.Status.Username = conPool.Username
	cp.Status.PoolMode = conPool.PoolMode
	cp.Status.DatabaseName = conPool.Database
	cp.Status.PoolSize = conPool.PoolSize
	cp.Status.ConnectionURI = conPool.ConnectionURI
	cp.Status.ServiceName = cp.Spec.ServiceName
	cp.Status.Project = cp.Spec.Project
}
