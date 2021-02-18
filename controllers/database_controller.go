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

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=databases/status,verbs=get;update;patch

func (r *DatabaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("database", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Database instance
	db := &k8soperatorv1alpha1.Database{}
	err := r.Get(ctx, req.NamespacedName, db)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Database resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// Check if database already exists on the Aiven side, create a
	// new one if database is not found
	_, err = r.AivenClient.Databases.Get(db.Spec.Project, db.Spec.ServiceName, db.Spec.DatabaseName)
	if err != nil {
		// Create a new database if it does not exists and update CR status
		if aiven.IsNotFound(err) {
			_, err = r.createDatabase(db)
			if err != nil {
				log.Error(err, "Failed to create Database")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) createDatabase(db *k8soperatorv1alpha1.Database) (*aiven.Database, error) {
	database, err := r.AivenClient.Databases.Create(db.Spec.Project, db.Spec.ServiceName, aiven.CreateDatabaseRequest{
		Database:  db.Spec.DatabaseName,
		LcCollate: db.Spec.LcCollate,
		LcType:    db.Spec.LcType,
	})
	if err != nil {
		return nil, err
	}

	// Update database custom resource status
	err = r.updateCRStatus(db, database)
	if err != nil {
		return nil, fmt.Errorf("failed to update database status: %w", err)
	}

	return database, err
}

func (r *DatabaseReconciler) updateCRStatus(db *k8soperatorv1alpha1.Database, d *aiven.Database) error {
	db.Status.DatabaseName = d.DatabaseName
	db.Status.LcCollate = d.LcCollate
	db.Status.LcType = d.LcType

	err := r.Status().Update(context.Background(), db)
	if err != nil {
		return err
	}

	return nil
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Database{}).
		Complete(r)
}
