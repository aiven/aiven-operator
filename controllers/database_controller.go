// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
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

// +kubebuilder:rbac:groups=aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=databases/status,verbs=get;update;patch

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)
	log.Info("reconciling aiven database")

	const dbFinalizer = "database-finalizer.aiven.io"
	db := &k8soperatorv1alpha1.Database{}
	return r.reconcileInstance(&DatabaseHandler{}, ctx, log, req, db, dbFinalizer)
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Database{}).
		Complete(r)
}

func (h DatabaseHandler) create(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	db, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("creating a new database on aiven side")

	database, err := c.Databases.Create(db.Spec.Project, db.Spec.ServiceName, aiven.CreateDatabaseRequest{
		Database:  db.Name,
		LcCollate: db.Spec.LcCollate,
		LcType:    db.Spec.LcType,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create database on Aiven side: %w", err)
	}

	h.setStatus(db, database)

	return db, nil
}

func (h DatabaseHandler) delete(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	db, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = c.Databases.Delete(
		db.Status.Project,
		db.Status.ServiceName,
		db.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	log.Info("successfully finalized database on aiven side")
	return true, nil
}

func (h DatabaseHandler) exists(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	db, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("checking if database exists on aiven side")

	d, err := c.Databases.Get(db.Spec.Project, db.Spec.ServiceName, db.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return d != nil, nil
}

func (h DatabaseHandler) update(_ *aiven.Client, log logr.Logger, _ client.Object) (client.Object, error) {
	log.Info("aiven database cannot be updated, skipping update handler")
	return nil, nil
}

func (h DatabaseHandler) getSecret(_ *aiven.Client, log logr.Logger, _ client.Object) (*corev1.Secret, error) {
	log.Info("aiven database has no secrets, skipping this handler")
	return nil, nil
}

func (h DatabaseHandler) checkPreconditions(c *aiven.Client, log logr.Logger, i client.Object) bool {
	db, err := h.convert(i)
	if err != nil {
		return false
	}

	log.Info("checking database preconditions")
	return checkServiceIsRunning(c, db.Spec.Project, db.Spec.ServiceName)
}

func (h DatabaseHandler) isActive(*aiven.Client, logr.Logger, client.Object) (bool, error) {
	return true, nil
}

func (h *DatabaseHandler) setStatus(db *k8soperatorv1alpha1.Database, d *aiven.Database) {
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

func (h DatabaseHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	cp, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &cp.Spec.AuthSecretRef
}
