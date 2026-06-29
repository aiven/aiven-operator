// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/http"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/utils"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/finalizers,verbs=get;create;update

// ClickhouseGrantController reconciles a ClickhouseGrant object.
type ClickhouseGrantController struct {
	client.Client
	avnGen avngen.Client
}

func newClickhouseGrantReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ClickhouseGrant] {
			return &ClickhouseGrantController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *ClickhouseGrantController) Observe(ctx context.Context, g *v1alpha1.ClickhouseGrant) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, g.Spec.Project, g.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	exists := wasEverApplied(g)
	if exists && hasLatestGeneration(g) {
		markInstanceRunning(g)
		return Observation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	if err := r.checkPreconditions(ctx, g); err != nil {
		return Observation{}, err
	}

	return Observation{ResourceExists: exists, ResourceUpToDate: false}, nil
}

func (r *ClickhouseGrantController) Create(ctx context.Context, g *v1alpha1.ClickhouseGrant) (CreateResult, error) {
	return CreateResult{}, r.applyGrants(ctx, g)
}

func (r *ClickhouseGrantController) Update(ctx context.Context, g *v1alpha1.ClickhouseGrant) (UpdateResult, error) {
	return UpdateResult{}, r.applyGrants(ctx, g)
}

func (r *ClickhouseGrantController) Delete(ctx context.Context, g *v1alpha1.ClickhouseGrant) error {
	// Revokes the latest grants from the spec.
	return revokeGrants(ctx, r.avnGen, g, &g.Spec.Grants)
}

// applyGrants revokes the previously applied grants (if any), then grants declared privileges in the spec.
func (r *ClickhouseGrantController) applyGrants(ctx context.Context, g *v1alpha1.ClickhouseGrant) error {
	delete(g.GetAnnotations(), instanceIsRunningAnnotation)

	// Revokes previous grants
	if g.Status.State != nil {
		if err := revokeGrants(ctx, r.avnGen, g, g.Status.State); err != nil {
			return err
		}
	}

	// Grants new privileges
	if err := grantSpecGrants(ctx, r.avnGen, g); err != nil {
		return err
	}

	// Stores the applied grants so the next reconciliation can revoke them before re-granting.
	g.Status.State = g.Spec.Grants.DeepCopy()
	markInstanceRunning(g)
	return nil
}

// checkPreconditions verifies that all users, roles and databases referenced in the spec already exist in ClickHouse.
// Missing references are reported as errPreconditionNotMet.
func (r *ClickhouseGrantController) checkPreconditions(ctx context.Context, g *v1alpha1.ClickhouseGrant) error {
	if err := checkPrecondition(ctx, g, r.avnGen, g.Spec.CollectGrantees, chUtils.QueryGrantees, "missing users or roles defined in spec: %v"); err != nil {
		return err
	}

	if err := checkPrecondition(ctx, g, r.avnGen, g.Spec.CollectDatabases, chUtils.QueryDatabases, "missing databases defined in spec: %v"); err != nil {
		return err
	}

	return nil
}

func checkPrecondition[T comparable](ctx context.Context, g *v1alpha1.ClickhouseGrant, avnGen avngen.Client, collectFunc func() []T, queryFunc func(context.Context, avngen.Client, string, string) ([]T, error), errorMsgFormat string) error {
	itemsInSpec := collectFunc()
	itemsInDB, err := queryFunc(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if err != nil {
		return err
	}
	missingItems := utils.CheckSliceContainment(itemsInSpec, itemsInDB)
	if len(missingItems) > 0 {
		err = fmt.Errorf(errorMsgFormat, missingItems)
		meta.SetStatusCondition(&g.Status.Conditions, getErrorCondition(errConditionPreconditions, err))
		return fmt.Errorf("%w: %w", errPreconditionNotMet, err)
	}
	return nil
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
