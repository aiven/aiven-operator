// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ClickhouseDatabaseReconciler reconciles a ClickhouseDatabase object
type ClickhouseDatabaseReconciler struct {
	Controller
}

func newClickhouseDatabaseReconciler(c Controller) reconcilerType {
	return &ClickhouseDatabaseReconciler{Controller: c}
}

// ClickhouseDatabaseHandler handles an Aiven ClickhouseDatabase
type ClickhouseDatabaseHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases/finalizers,verbs=get;create;update

func (r *ClickhouseDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &ClickhouseDatabaseHandler{}, &v1alpha1.ClickhouseDatabase{})
}

func (r *ClickhouseDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseDatabase{}).
		Complete(r)
}

func (h *ClickhouseDatabaseHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	db, err := h.convert(obj)
	if err != nil {
		return err
	}

	dbName := db.GetDatabaseName()
	_, err = GetClickhouseDatabaseByName(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName, dbName)
	switch {
	case isNotFound(err):
		req := clickhouse.ServiceClickHouseDatabaseCreateIn{
			Database: dbName,
		}
		err = avnGen.ServiceClickHouseDatabaseCreate(ctx, db.Spec.Project, db.Spec.ServiceName, &req)
		if err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("cannot create clickhouse database on Aiven side: %w", err)
	}

	return nil
}

func (h *ClickhouseDatabaseHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	db, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	dbName := db.GetDatabaseName()
	err = avnGen.ServiceClickHouseDatabaseDelete(ctx, db.Spec.Project, db.Spec.ServiceName, dbName)
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h *ClickhouseDatabaseHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	db, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	dbName := db.GetDatabaseName()
	_, err = GetClickhouseDatabaseByName(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName, dbName)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&db.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h *ClickhouseDatabaseHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	db, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&db.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, db.Spec.Project, db.Spec.ServiceName)
}

func (h *ClickhouseDatabaseHandler) convert(i client.Object) (*v1alpha1.ClickhouseDatabase, error) {
	db, ok := i.(*v1alpha1.ClickhouseDatabase)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseDatabase")
	}

	return db, nil
}

func GetClickhouseDatabaseByName(ctx context.Context, avnGen avngen.Client, project, service, name string) (*clickhouse.DatabaseOut, error) {
	list, err := avnGen.ServiceClickHouseDatabaseList(ctx, project, service)
	if err != nil {
		return nil, err
	}
	for _, db := range list {
		if db.Name == name {
			return &db, nil
		}
	}
	return nil, NewNotFound(fmt.Sprintf("database %q not found", name))
}
