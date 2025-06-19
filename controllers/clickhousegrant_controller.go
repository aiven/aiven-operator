// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/utils"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

// ClickhouseGrantReconciler reconciles a ClickhouseGrant object
type ClickhouseGrantReconciler struct {
	Controller
}

func newClickhouseGrantReconciler(c Controller) reconcilerType {
	return &ClickhouseGrantReconciler{Controller: c}
}

// ClickhouseGrantHandler handles an Aiven ClickhouseGrant
type ClickhouseGrantHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/finalizers,verbs=get;create;update

func (r *ClickhouseGrantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &ClickhouseGrantHandler{}, &v1alpha1.ClickhouseGrant{})
}

func (r *ClickhouseGrantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseGrant{}).
		Complete(r)
}

func (h *ClickhouseGrantHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	g, err := h.convert(obj)
	if err != nil {
		return err
	}

	// Revokes previous grants
	if g.Status.State != nil {
		err = revokeGrants(ctx, avnGen, g, g.Status.State)
		if err != nil {
			return err
		}
	}

	// Grants new privileges
	err = grantSpecGrants(ctx, avnGen, g)
	if err != nil {
		return err
	}

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	metav1.SetMetaDataAnnotation(&g.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(g.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *ClickhouseGrantHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	g, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// Revokes latest grants.
	// Doesn't revoke previous grants (state) here.
	// If the object hasn't been committed yet, it might have an empty state.
	// Must revoke from the spec.
	err = revokeGrants(ctx, avnGen, g, &g.Spec.Grants)
	return err == nil, err
}

func (h *ClickhouseGrantHandler) get(_ context.Context, _ avngen.Client, obj client.Object) (*corev1.Secret, error) {
	g, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	// Stores previous state
	g.Status.State = g.Spec.Grants.DeepCopy()

	meta.SetStatusCondition(&g.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&g.ObjectMeta, instanceIsRunningAnnotation, "true")
	return nil, nil
}

func (h *ClickhouseGrantHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	/** Preconditions for ClickhouseGrant:
	 *
	 * 1. The service is running
	 * 2. All users and roles specified in spec exist
	 * 3. All databases specified in spec exist
	 */

	g, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	isOperational, err := checkServiceIsOperational(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if !isOperational || err != nil {
		return false, err
	}

	// Service is running, check users and roles specified in spec exist
	_, err = checkPrecondition(ctx, g, avnGen, g.Spec.CollectGrantees, chUtils.QueryGrantees, "missing users or roles defined in spec: %v")
	if err != nil {
		return false, err
	}

	// Check that databases specified in spec exist
	_, err = checkPrecondition(ctx, g, avnGen, g.Spec.CollectDatabases, chUtils.QueryDatabases, "missing databases defined in spec: %v")
	if err != nil {
		return false, err
	}

	// Remove previous error conditions
	meta.RemoveStatusCondition(&g.Status.Conditions, "Error")

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Preconditions", "Preconditions met"))

	return true, nil
}

func checkPrecondition[T comparable](ctx context.Context, g *v1alpha1.ClickhouseGrant, avnGen avngen.Client, collectFunc func() []T, queryFunc func(context.Context, avngen.Client, string, string) ([]T, error), errorMsgFormat string) (bool, error) {
	itemsInSpec := collectFunc()
	itemsInDB, err := queryFunc(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if err != nil {
		return false, err
	}
	missingItems := utils.CheckSliceContainment(itemsInSpec, itemsInDB)
	if len(missingItems) > 0 {
		err = fmt.Errorf(errorMsgFormat, missingItems)
		meta.SetStatusCondition(&g.Status.Conditions, getErrorCondition(errConditionPreconditions, err))
		return false, err
	}
	return true, nil
}

func (h *ClickhouseGrantHandler) convert(i client.Object) (*v1alpha1.ClickhouseGrant, error) {
	g, ok := i.(*v1alpha1.ClickhouseGrant)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseGrant")
	}

	return g, nil
}

func executeStatements(
	ctx context.Context,
	avnGen avngen.Client,
	project, serviceName string,
	statements []string,
) []error {
	errors := make([]error, 0)
	for _, stmt := range statements {
		_, err := chUtils.ExecuteClickHouseQuery(ctx, avnGen, project, serviceName, stmt)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func revokeGrants(ctx context.Context, avnGen avngen.Client, g *v1alpha1.ClickhouseGrant, grant *v1alpha1.Grants) error {
	statements := grant.BuildStatements(chUtils.REVOKE)
	if len(statements) == 0 {
		return nil
	}

	errors := executeStatements(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName, statements)
	for _, err := range errors {
		if isAivenError(err, http.StatusBadRequest) || isAivenError(err, http.StatusNotFound) {
			// "not found in user directories", "There is no role", "database does not exist" etc
			continue
		}
		return err
	}

	return nil
}

func grantSpecGrants(ctx context.Context, avnGen avngen.Client, g *v1alpha1.ClickhouseGrant) error {
	errors := executeStatements(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName, g.Spec.Grants.BuildStatements(chUtils.GRANT))
	var err error
	return multierror.Append(err, errors...).ErrorOrNil()
}
