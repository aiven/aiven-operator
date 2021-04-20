// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"k8s.io/apimachinery/pkg/api/errors"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ConnectionPoolReconciler reconciles a ConnectionPool object
type ConnectionPoolReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=connectionpools/status,verbs=get;update;patch

func (r *ConnectionPoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("connectionpool", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the ConnectionPool instance
	cp := &k8soperatorv1alpha1.ConnectionPool{}
	err := r.Get(ctx, req.NamespacedName, cp)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ConnectionPool resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ConnectionPool")
		return ctrl.Result{}, err
	}

	// Check if connection pool already exists on the Aiven side, create a
	// new one if connection pool is not found
	_, err = r.AivenClient.ConnectionPools.Get(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.PoolName)
	if err != nil {
		// Create a new connection pool if it does not exists and update CR status
		if aiven.IsNotFound(err) {
			_, err = r.createConPool(cp)
			if err != nil {
				log.Error(err, "Failed to create ConnectionPool")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Update connection poll via API and update CR status
	_, err = r.updateConPool(cp)
	if err != nil {
		log.Error(err, "Failed to update ConnectionPool")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createConPool creates a connection pool on Aiven side
func (r *ConnectionPoolReconciler) createConPool(cp *k8soperatorv1alpha1.ConnectionPool) (*aiven.ConnectionPool, error) {
	conPool, err := r.AivenClient.ConnectionPools.Create(cp.Spec.Project, cp.Spec.ServiceName,
		aiven.CreateConnectionPoolRequest{
			Database: cp.Spec.DatabaseName,
			PoolMode: cp.Spec.PoolMode,
			PoolName: cp.Spec.PoolName,
			PoolSize: cp.Spec.PoolSize,
			Username: cp.Spec.Username,
		})
	if err != nil {
		return nil, err
	}

	// Update connection pool custom resource status
	err = r.updateCRStatus(cp, conPool)
	if err != nil {
		return nil, fmt.Errorf("failed to update ConnectionPool status: %w", err)
	}

	return conPool, err
}

// updateConPool updates an existing connection pool on Aiven side
func (r *ConnectionPoolReconciler) updateConPool(cp *k8soperatorv1alpha1.ConnectionPool) (*aiven.ConnectionPool, error) {
	conPool, err := r.AivenClient.ConnectionPools.Update(cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.PoolName,
		aiven.UpdateConnectionPoolRequest{
			Database: cp.Spec.DatabaseName,
			PoolMode: cp.Spec.PoolMode,
			PoolSize: cp.Spec.PoolSize,
			Username: cp.Spec.Username,
		})
	if err != nil {
		return nil, err
	}

	// Update connection pool custom resource status
	err = r.updateCRStatus(cp, conPool)
	if err != nil {
		return nil, fmt.Errorf("failed to update ConnectionPool status: %w", err)
	}

	return conPool, err
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ConnectionPoolReconciler) updateCRStatus(cp *k8soperatorv1alpha1.ConnectionPool, conPool *aiven.ConnectionPool) error {
	cp.Status.Username = conPool.Username
	cp.Status.PoolName = conPool.PoolName
	cp.Status.PoolMode = conPool.PoolMode
	cp.Status.DatabaseName = conPool.Database
	cp.Status.PoolSize = conPool.PoolSize
	cp.Status.ConnectionURI = conPool.ConnectionURI

	err := r.Status().Update(context.Background(), cp)
	if err != nil {
		return err
	}

	return nil
}

func (r *ConnectionPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ConnectionPool{}).
		Complete(r)
}
