// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	Controller
}

func newDatabaseReconciler(c Controller) reconcilerType {
	return &DatabaseReconciler{Controller: c}
}

// DatabaseHandler handles an Aiven Database
type DatabaseHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=databases/finalizers,verbs=get;create;update

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, DatabaseHandler{}, &v1alpha1.Database{})
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Database{}).
		Complete(r)
}

func (h DatabaseHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	db, err := h.convert(obj)
	if err != nil {
		return err
	}

	exists, err := h.exists(ctx, avnGen, db)
	if err != nil {
		return err
	}

	if !exists {
		err = avnGen.ServiceDatabaseCreate(ctx, db.Spec.Project, db.Spec.ServiceName, &service.ServiceDatabaseCreateIn{
			Database:  db.GetDatabaseName(),
			LcCollate: NilIfZero(db.Spec.LcCollate),
			LcCtype:   NilIfZero(db.Spec.LcCtype),
		})
		if err != nil {
			return fmt.Errorf("cannot create database on Aiven side: %w", err)
		}
	}

	return nil
}

func (h DatabaseHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	db, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if fromAnyPointer(db.Spec.TerminationProtection) {
		return false, errTerminationProtectionOn
	}

	err = avnGen.ServiceDatabaseDelete(
		ctx,
		db.Spec.Project,
		db.Spec.ServiceName,
		db.GetDatabaseName())
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h DatabaseHandler) exists(ctx context.Context, avnGen avngen.Client, db *v1alpha1.Database) (bool, error) {
	d, err := GetDatabaseByName(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	if isNotFound(err) {
		return false, nil
	}

	return d != nil, nil
}

func (h DatabaseHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	db, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	_, err = GetDatabaseByName(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&db.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h DatabaseHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	db, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName)
}

func (h DatabaseHandler) convert(i client.Object) (*v1alpha1.Database, error) {
	db, ok := i.(*v1alpha1.Database)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to Database")
	}

	return db, nil
}

func GetDatabaseByName(ctx context.Context, avnGen avngen.Client, projectName, serviceName, dbName string) (*service.DatabaseOut, error) {
	list, err := avnGen.ServiceDatabaseList(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	for _, db := range list {
		if db.DatabaseName == dbName {
			return &db, nil
		}
	}
	return nil, NewNotFound(fmt.Sprintf("Database with name %q not found", dbName))
}
