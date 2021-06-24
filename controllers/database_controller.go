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

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	Controller
}

// DatabaseHandler handles an Aiven Database
type DatabaseHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=databases/status,verbs=get;update;patch

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	db := &k8soperatorv1alpha1.Database{}
	err := r.Get(ctx, req.NamespacedName, db)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, db.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &DatabaseHandler{
		client: c,
	}, db)
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Database{}).
		Complete(r)
}

func (h DatabaseHandler) createOrUpdate(i client.Object) (client.Object, error) {
	db, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	exists, err := h.exists(db)

	if err != nil {
		return nil, err
	}

	if !exists {
		_, err := h.client.Databases.Create(db.Spec.Project, db.Spec.ServiceName, aiven.CreateDatabaseRequest{
			Database:  db.Name,
			LcCollate: db.Spec.LcCollate,
			LcType:    db.Spec.LcCtype,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot create database on Aiven side: %w", err)
		}
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getInitializedCondition("Created",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&db.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&db.ObjectMeta,
		processedGeneration, strconv.FormatInt(db.GetGeneration(), 10))

	return db, nil
}

func (h DatabaseHandler) delete(i client.Object) (bool, error) {
	db, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = h.client.Databases.Delete(
		db.Spec.Project,
		db.Spec.ServiceName,
		db.Name)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h DatabaseHandler) exists(db *k8soperatorv1alpha1.Database) (bool, error) {
	d, err := h.client.Databases.Get(db.Spec.Project, db.Spec.ServiceName, db.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return d != nil, nil
}

func (h DatabaseHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	db, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	_, err = h.client.Databases.Get(db.Spec.Project, db.Spec.ServiceName, db.Name)
	if err != nil {
		return nil, nil, err
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "Get",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&db.ObjectMeta, isRunning, "true")

	return db, nil, nil
}

func (h DatabaseHandler) checkPreconditions(i client.Object) bool {
	db, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(h.client, db.Spec.Project, db.Spec.ServiceName)
}

func (h DatabaseHandler) convert(i client.Object) (*k8soperatorv1alpha1.Database, error) {
	db, ok := i.(*k8soperatorv1alpha1.Database)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Database")
	}

	return db, nil
}
