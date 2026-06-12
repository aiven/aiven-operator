// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousedatabases/finalizers,verbs=get;create;update

// ClickhouseDatabaseController reconciles a ClickhouseDatabase object.
type ClickhouseDatabaseController struct {
	client.Client
	avnGen avngen.Client
}

func newClickhouseDatabaseReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ClickhouseDatabase] {
			return &ClickhouseDatabaseController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *ClickhouseDatabaseController) Observe(ctx context.Context, db *v1alpha1.ClickhouseDatabase) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, db.Spec.Project, db.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	_, err := GetClickhouseDatabaseByName(ctx, r.avnGen, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("describing ClickHouse database: %w", err)
	}

	markInstanceRunning(db)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(db),
	}, nil
}

func (r *ClickhouseDatabaseController) Create(ctx context.Context, db *v1alpha1.ClickhouseDatabase) (CreateResult, error) {
	req := clickhouse.ServiceClickHouseDatabaseCreateIn{
		Database: db.GetDatabaseName(),
	}
	if err := r.avnGen.ServiceClickHouseDatabaseCreate(ctx, db.Spec.Project, db.Spec.ServiceName, &req); err != nil {
		// The service can report RUNNING via ServiceGet while its ClickHouse database
		// is not ready yet, returning a transient 5xx.
		if isServerError(err) {
			return CreateResult{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
		}
		return CreateResult{}, fmt.Errorf("cannot create clickhouse database on Aiven side: %w", err)
	}

	const reason = "Created"
	meta.SetStatusCondition(&db.Status.Conditions, getInitializedCondition(reason, "Successfully created the instance in Aiven"))
	meta.SetStatusCondition(&db.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *ClickhouseDatabaseController) Update(_ context.Context, _ *v1alpha1.ClickhouseDatabase) (UpdateResult, error) {
	// ClickHouse databases have no mutable fields, so there is nothing to update.
	return UpdateResult{}, nil
}

func (r *ClickhouseDatabaseController) Delete(ctx context.Context, db *v1alpha1.ClickhouseDatabase) error {
	err := r.avnGen.ServiceClickHouseDatabaseDelete(ctx, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
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
