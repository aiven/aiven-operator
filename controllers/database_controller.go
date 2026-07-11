// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=databases/finalizers,verbs=get;create;update

// DatabaseController reconciles a Database object.
type DatabaseController struct {
	client.Client
	avnGen avngen.Client
}

func newDatabaseReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.Database] {
			return &DatabaseController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *DatabaseController) Observe(ctx context.Context, db *v1alpha1.Database) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, db.Spec.Project, db.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	_, err := GetDatabaseByName(ctx, r.avnGen, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("describing database: %w", err)
	}

	markInstanceRunning(db)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(db),
	}, nil
}

func (r *DatabaseController) Create(ctx context.Context, db *v1alpha1.Database) (CreateResult, error) {
	delete(db.GetAnnotations(), instanceIsRunningAnnotation)

	req := service.ServiceDatabaseCreateIn{
		Database:  db.GetDatabaseName(),
		LcCollate: NilIfZero(db.Spec.LcCollate),
		LcCtype:   NilIfZero(db.Spec.LcCtype),
	}
	if err := r.avnGen.ServiceDatabaseCreate(ctx, db.Spec.Project, db.Spec.ServiceName, &req); err != nil {
		// The service can report RUNNING via ServiceGet while it is not ready
		// to accept database creation yet, returning a transient 5xx.
		if isServerError(err) {
			return CreateResult{}, fmt.Errorf("%w: %w", errPreconditionNotMet, err)
		}
		return CreateResult{}, fmt.Errorf("cannot create database on Aiven side: %w", err)
	}

	const reason = "Created"
	meta.SetStatusCondition(&db.Status.Conditions, getInitializedCondition(reason, "Successfully created the instance in Aiven"))
	meta.SetStatusCondition(&db.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *DatabaseController) Update(_ context.Context, _ *v1alpha1.Database) (UpdateResult, error) {
	// Databases have no mutable fields.
	return UpdateResult{}, nil
}

func (r *DatabaseController) Delete(ctx context.Context, db *v1alpha1.Database) error {
	if fromAnyPointer(db.Spec.TerminationProtection) {
		return errTerminationProtectionOn
	}

	err := r.avnGen.ServiceDatabaseDelete(ctx, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName())
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// maxDatabaseListPages is an upper bound on the number of pages GetDatabaseByName will fetch.
const maxDatabaseListPages = 100

func GetDatabaseByName(
	ctx context.Context,
	avnGen avngen.Client,
	projectName, serviceName, dbName string,
) (*service.DatabaseOut, error) {
	var after string
	for range maxDatabaseListPages {
		var query [][2]string
		if after != "" {
			query = append(query, service.ServiceDatabaseListAfter(after))
		}

		list, err := avnGen.ServiceDatabaseList(ctx, projectName, serviceName, query...)
		if err != nil {
			return nil, err
		}

		for i := range list.Databases {
			if list.Databases[i].DatabaseName == dbName {
				return &list.Databases[i], nil
			}
		}

		if list.Next == nil || *list.Next == "" {
			return nil, NewNotFound(fmt.Sprintf("Database with name %q not found", dbName))
		}
		after = *list.Next
	}

	return nil, fmt.Errorf(
		"GetDatabaseByName: exceeded %d pages searching for database %q",
		maxDatabaseListPages,
		dbName,
	)
}
