// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// default database used for statements that do not target a particular database
// think CREATE ROLE / GRANT / etc...
const defaultDatabase = "system"

func newClickhouseRoleReconciler(c Controller) reconcilerType {
	return &ClickhouseRoleReconciler{Controller: c}
}

// ClickhouseRoleReconciler reconciles a ClickhouseRole object
type ClickhouseRoleReconciler struct {
	Controller
}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles/finalizers,verbs=get;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ClickhouseRoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &clickhouseRoleHandler{}, &v1alpha1.ClickhouseRole{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClickhouseRoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseRole{}).
		Complete(r)
}

type clickhouseRoleHandler struct{}

func (h *clickhouseRoleHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	role, err := h.convert(obj)
	if err != nil {
		return err
	}

	return clickhouseRoleCreate(ctx, avnGen, role)
}

func (h *clickhouseRoleHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	role, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = clickhouseRoleDelete(ctx, avnGen, role)
	return isDeleted(err)
}

func (h *clickhouseRoleHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	role, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	err = ClickhouseRoleExists(ctx, avnGen, role)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&role.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&role.ObjectMeta, instanceIsRunningAnnotation, "true")
	return nil, nil
}

func (h *clickhouseRoleHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	role, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&role.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, role.Spec.Project, role.Spec.ServiceName)
}

func (h *clickhouseRoleHandler) convert(i client.Object) (*v1alpha1.ClickhouseRole, error) {
	role, ok := i.(*v1alpha1.ClickhouseRole)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseRole")
	}
	return role, nil
}

// Escapes database identifiers like table or column names
func escape(identifier string) string {
	// See https://github.com/ClickHouse/clickhouse-go/blob/8ad6ec6b95d8b0c96d00115bc2d69ff13083f94b/lib/column/column.go#L32
	replacer := strings.NewReplacer("`", "\\`", "\\", "\\\\")
	return "`" + replacer.Replace(identifier) + "`"
}

func runQuery(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole, query string) error {
	req := clickhouse.ServiceClickHouseQueryIn{
		Database: defaultDatabase,
		Query:    fmt.Sprintf("%s %s", query, escape(r.Spec.Role)),
	}
	_, err := avnGen.ServiceClickHouseQuery(ctx, r.Spec.Project, r.Spec.ServiceName, &req)
	return err
}

func ClickhouseRoleExists(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, avnGen, r, "SHOW CREATE ROLE")
	if isUnknownRole(err) {
		return NewNotFound(fmt.Sprintf("ClickhouseRole %q not found", r.Name))
	}
	return err
}

func clickhouseRoleCreate(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole) error {
	return runQuery(ctx, avnGen, r, "CREATE ROLE IF NOT EXISTS")
}

func clickhouseRoleDelete(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, avnGen, r, "DROP ROLE IF EXISTS")
	if isUnknownRole(err) {
		return nil
	}
	return err
}

func isUnknownRole(err error) bool {
	var e avngen.Error
	return errors.As(err, &e) && strings.Contains(e.Message, "Code: 511")
}
