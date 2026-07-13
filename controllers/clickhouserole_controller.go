// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// default database used for statements that do not target a particular database
// think CREATE ROLE / GRANT / etc...
const defaultDatabase = "system"

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseroles/finalizers,verbs=get;create;update

// ClickhouseRoleController reconciles a ClickhouseRole object.
type ClickhouseRoleController struct {
	client.Client
	avnGen avngen.Client
}

func newClickhouseRoleReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ClickhouseRole] {
			return &ClickhouseRoleController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *ClickhouseRoleController) Observe(ctx context.Context, role *v1alpha1.ClickhouseRole) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, role.Spec.Project, role.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	err := ClickhouseRoleExists(ctx, r.avnGen, role)
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("describing ClickHouse role: %w", err)
	}

	markInstanceRunning(role)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(role),
	}, nil
}

func (r *ClickhouseRoleController) Create(ctx context.Context, role *v1alpha1.ClickhouseRole) (CreateResult, error) {
	delete(role.GetAnnotations(), instanceIsRunningAnnotation)

	if err := runQuery(ctx, r.avnGen, role, "CREATE ROLE IF NOT EXISTS"); err != nil {
		// The service can report RUNNING via ServiceGet while its ClickHouse
		// engine is not ready yet, returning a transient 5xx.
		if isServerError(err) {
			return CreateResult{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
		}
		return CreateResult{}, fmt.Errorf("cannot create clickhouse role on Aiven side: %w", err)
	}

	const reason = "Created"
	meta.SetStatusCondition(&role.Status.Conditions, getInitializedCondition(reason, "Successfully created the instance in Aiven"))
	meta.SetStatusCondition(&role.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *ClickhouseRoleController) Update(_ context.Context, _ *v1alpha1.ClickhouseRole) (UpdateResult, error) {
	// ClickhouseRole spec fields are immutable.
	return UpdateResult{}, nil
}

func (r *ClickhouseRoleController) Delete(ctx context.Context, role *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, r.avnGen, role, "DROP ROLE IF EXISTS")
	if err != nil && !isUnknownRole(err) && !isNotFound(err) {
		return err
	}
	return nil
}

func ClickhouseRoleExists(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole) error {
	err := runQuery(ctx, avnGen, r, "SHOW CREATE ROLE")
	if isUnknownRole(err) {
		return NewNotFound(fmt.Sprintf("ClickhouseRole %q not found", r.Name))
	}
	return err
}

func runQuery(ctx context.Context, avnGen avngen.Client, r *v1alpha1.ClickhouseRole, query string) error {
	req := clickhouse.ServiceClickHouseQueryIn{
		Database: defaultDatabase,
		Query:    fmt.Sprintf("%s %s", query, escape(r.Spec.Role)),
	}
	_, err := avnGen.ServiceClickHouseQuery(ctx, r.Spec.Project, r.Spec.ServiceName, &req)
	return err
}

// Escapes database identifiers like table or column names
func escape(identifier string) string {
	// See https://github.com/ClickHouse/clickhouse-go/blob/8ad6ec6b95d8b0c96d00115bc2d69ff13083f94b/lib/column/column.go#L32
	replacer := strings.NewReplacer("`", "\\`", "\\", "\\\\")
	return "`" + replacer.Replace(identifier) + "`"
}

func isUnknownRole(err error) bool {
	var e avngen.Error
	return errors.As(err, &e) && strings.Contains(e.Message, "Code: 511")
}
