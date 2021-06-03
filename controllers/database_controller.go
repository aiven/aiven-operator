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

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	Controller
}

// DatabaseHandler handles an Aiven Database
type DatabaseHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=databases/status,verbs=get;update;patch

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)
	log.Info("Reconciling Aiven Database")

	const dbFinalizer = "database-finalizer.k8s-operator.aiven.io"
	db := &k8soperatorv1alpha1.Database{}
	return r.reconcileInstance(&DatabaseHandler{}, ctx, log, req, db, dbFinalizer)
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Database{}).
		Complete(r)
}

func (h DatabaseHandler) create(log logr.Logger, i client.Object) (client.Object, error) {
	db, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating a new Database on Aiven side")

	database, err := aivenClient.Databases.Create(db.Spec.Project, db.Spec.ServiceName, aiven.CreateDatabaseRequest{
		Database:  db.Spec.DatabaseName,
		LcCollate: db.Spec.LcCollate,
		LcType:    db.Spec.LcType,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create database on Aiven side: %w", err)
	}

	h.setStatus(db, database)

	return db, nil
}

func (h DatabaseHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	db, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	err = aivenClient.Databases.Delete(
		db.Status.Project,
		db.Status.ServiceName,
		db.Status.DatabaseName)
	if !aiven.IsNotFound(err) {
		return nil, false, err
	}

	log.Info("Successfully finalized Database on Aiven side")

	return nil, true, nil
}

func (h DatabaseHandler) exists(log logr.Logger, i client.Object) (bool, error) {
	db, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("Checking if Database exists on Aiven side")

	d, err := aivenClient.Databases.Get(db.Spec.Project, db.Spec.ServiceName, db.Spec.DatabaseName)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return d != nil, nil
}

func (h DatabaseHandler) update(log logr.Logger, _ client.Object) (client.Object, error) {
	log.Info("Aiven Database cannot be updated, skipping update handler")
	return nil, nil
}

func (h DatabaseHandler) getSecret(log logr.Logger, _ client.Object) (*corev1.Secret, error) {
	log.Info("Aiven Database has no secrets, skipping this handler")
	return nil, nil
}

func (h DatabaseHandler) checkPreconditions(log logr.Logger, i client.Object) bool {
	db, err := h.convert(i)
	if err != nil {
		return false
	}

	log.Info("Checking Database preconditions")
	return checkServiceIsRunning(db.Spec.Project, db.Spec.ServiceName)
}

func (h DatabaseHandler) isActive(logr.Logger, client.Object) (bool, error) {
	return true, nil
}

func (h DatabaseHandler) setStatus(db *k8soperatorv1alpha1.Database, d *aiven.Database) {
	db.Status.DatabaseName = d.DatabaseName
	db.Status.LcCollate = d.LcCollate
	db.Status.LcType = d.LcType
	db.Status.Project = db.Spec.Project
	db.Status.ServiceName = db.Spec.ServiceName
}

func (h DatabaseHandler) convert(i client.Object) (*k8soperatorv1alpha1.Database, error) {
	db, ok := i.(*k8soperatorv1alpha1.Database)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Database")
	}

	return db, nil
}
