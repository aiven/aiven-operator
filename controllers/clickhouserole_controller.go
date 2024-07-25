// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
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

func (h *clickhouseRoleHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	role, err := h.convert(obj)
	if err != nil {
		return err
	}

	err = clickhouseRoleCreate(ctx, avn, role)
	if err != nil {
		return err
	}

	meta.SetStatusCondition(&role.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	metav1.SetMetaDataAnnotation(&role.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(role.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *clickhouseRoleHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	role, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = clickhouseRoleDelete(ctx, avn, role)
	return isDeleted(err)
}

func (h *clickhouseRoleHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	role, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	err = ClickhouseRoleExists(ctx, avn, role)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&role.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&role.ObjectMeta, instanceIsRunningAnnotation, "true")
	return nil, nil
}

func (h *clickhouseRoleHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
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

func runQuery(ctx context.Context, avn *aiven.Client, r *v1alpha1.ClickhouseRole, query string) error {
	q := fmt.Sprintf("%s %s", query, escape(r.Spec.Role))
	_, err := avn.ClickHouseQuery.Query(ctx, r.Spec.Project, r.Spec.ServiceName, defaultDatabase, q)
	return err
}

func ClickhouseRoleExists(ctx context.Context, avn *aiven.Client, r *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, avn, r, "SHOW CREATE ROLE")
	if isUnknownRole(err) {
		return NewNotFound(fmt.Sprintf("ClickhouseRole %q not found", r.Name))
	}
	return err
}

func clickhouseRoleCreate(ctx context.Context, avn *aiven.Client, r *v1alpha1.ClickhouseRole) error {
	return runQuery(ctx, avn, r, "CREATE ROLE IF NOT EXISTS")
}

func clickhouseRoleDelete(ctx context.Context, avn *aiven.Client, r *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, avn, r, "DROP ROLE IF EXISTS")
	if isUnknownRole(err) {
		return nil
	}
	return err
}

func isUnknownRole(err error) bool {
	var e aiven.Error
	return errors.As(err, &e) && strings.Contains(e.Message, "Code: 511")
}
