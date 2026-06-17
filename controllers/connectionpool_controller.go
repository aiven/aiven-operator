// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/postgresql"
	"github.com/aiven/go-client-codegen/handler/service"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=connectionpools/finalizers,verbs=get;create;update

// ConnectionPoolController reconciles a ConnectionPool object.
type ConnectionPoolController struct {
	client.Client
	avnGen avngen.Client
}

func newConnectionPoolReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.ConnectionPool] {
			return &ConnectionPoolController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *ConnectionPoolController) Observe(ctx context.Context, cp *v1alpha1.ConnectionPool) (Observation, error) {
	svc, err := getServiceIfOperational(ctx, r.avnGen, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return Observation{}, err
	}

	if !slices.Contains(svc.Databases, cp.Spec.DatabaseName) {
		return Observation{}, fmt.Errorf("%w: database %q not found", errPreconditionNotMet, cp.Spec.DatabaseName)
	}

	var poolUser *service.ServiceUserGetOut
	if cp.Spec.Username != "" {
		u, err := r.avnGen.ServiceUserGet(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if isNotFound(err) {
			return Observation{}, fmt.Errorf("%w: service user %q not found", errPreconditionNotMet, cp.Spec.Username)
		}
		if err != nil {
			return Observation{}, fmt.Errorf("cannot get user: %w", err)
		}
		poolUser = u
	}

	pool := findConnectionPool(svc, cp.Name)
	if pool == nil {
		return Observation{ResourceExists: false}, nil
	}

	if !hasLatestGeneration(cp) || !poolMatchesSpec(pool, cp) {
		return Observation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	details, err := r.buildSecretDetails(ctx, cp, svc, pool, poolUser)
	if err != nil {
		return Observation{}, err
	}

	markInstanceRunning(cp)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		SecretDetails:    details,
	}, nil
}

func (r *ConnectionPoolController) Create(ctx context.Context, cp *v1alpha1.ConnectionPool) (CreateResult, error) {
	delete(cp.GetAnnotations(), instanceIsRunningAnnotation)

	req := postgresql.ServicePGBouncerCreateIn{
		Database: cp.Spec.DatabaseName,
		PoolMode: cp.Spec.PoolMode,
		PoolName: cp.Name,
		PoolSize: NilIfZero(cp.Spec.PoolSize),
		Username: NilIfZero(cp.Spec.Username),
	}
	if err := r.avnGen.ServicePGBouncerCreate(ctx, cp.Spec.Project, cp.Spec.ServiceName, &req); err != nil && !isAlreadyExists(err) {
		return CreateResult{}, fmt.Errorf("cannot create connection pool: %w", err)
	}

	details, err := r.connectionDetails(ctx, cp)
	if err != nil {
		return CreateResult{}, err
	}

	markInstanceRunning(cp)

	return CreateResult{SecretDetails: details}, nil
}

func (r *ConnectionPoolController) Update(ctx context.Context, cp *v1alpha1.ConnectionPool) (UpdateResult, error) {
	delete(cp.GetAnnotations(), instanceIsRunningAnnotation)

	req := postgresql.ServicePGBouncerUpdateIn{
		Database: NilIfZero(cp.Spec.DatabaseName),
		PoolMode: cp.Spec.PoolMode,
		PoolSize: NilIfZero(cp.Spec.PoolSize),
		Username: NilIfZero(cp.Spec.Username),
	}
	if err := r.avnGen.ServicePGBouncerUpdate(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name, &req); err != nil {
		return UpdateResult{}, fmt.Errorf("cannot update connection pool: %w", err)
	}

	details, err := r.connectionDetails(ctx, cp)
	if err != nil {
		return UpdateResult{}, err
	}

	markInstanceRunning(cp)

	return UpdateResult{SecretDetails: details}, nil
}

func (r *ConnectionPoolController) Delete(ctx context.Context, cp *v1alpha1.ConnectionPool) error {
	if err := r.avnGen.ServicePGBouncerDelete(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Name); err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

// connectionDetails fetches the service and builds the connection secret for the pool.
func (r *ConnectionPoolController) connectionDetails(ctx context.Context, cp *v1alpha1.ConnectionPool) (SecretDetails, error) {
	svc, err := getServiceIfOperational(ctx, r.avnGen, cp.Spec.Project, cp.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	pool := findConnectionPool(svc, cp.Name)
	if pool == nil {
		return nil, fmt.Errorf("%w: connection pool %q not yet available", errPreconditionNotMet, cp.Name)
	}

	var poolUser *service.ServiceUserGetOut
	if cp.Spec.Username != "" {
		u, err := r.avnGen.ServiceUserGet(ctx, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username)
		if err != nil {
			return nil, fmt.Errorf("cannot get user: %w", err)
		}
		poolUser = u
	}

	return r.buildSecretDetails(ctx, cp, svc, pool, poolUser)
}

// buildSecretDetails assembles the connection secret. poolUser must be non-nil when spec.Username is set.
func (r *ConnectionPoolController) buildSecretDetails(ctx context.Context, cp *v1alpha1.ConnectionPool, svc *service.ServiceGetOut, pool *service.ConnectionPoolOut, poolUser *service.ServiceUserGetOut) (SecretDetails, error) {
	cert, err := r.avnGen.ProjectKmsGetCA(ctx, cp.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve project CA certificate: %w", err)
	}

	// The pool comes with its own port.
	poolURI, err := url.Parse(pool.ConnectionUri)
	if err != nil {
		return nil, fmt.Errorf("can't parse ConnectionPool URI: %w", err)
	}

	user := svc.ServiceUriParams["user"]
	password := svc.ServiceUriParams["password"]
	if cp.Spec.Username != "" && poolUser != nil {
		user = fromAnyPointer(pool.Username)
		password = poolUser.Password
	}

	prefix := getSecretPrefix(cp)
	return SecretDetails{
		prefix + "NAME":         cp.Name,
		prefix + "HOST":         svc.ServiceUriParams["host"],
		prefix + "PORT":         poolURI.Port(),
		prefix + "DATABASE":     pool.Database,
		prefix + "USER":         user,
		prefix + "PASSWORD":     password,
		prefix + "SSLMODE":      svc.ServiceUriParams["sslmode"],
		prefix + "DATABASE_URI": pool.ConnectionUri,
		prefix + "CA_CERT":      cert,
		// todo: remove in future releases
		"PGHOST":       svc.ServiceUriParams["host"],
		"PGPORT":       poolURI.Port(),
		"PGDATABASE":   pool.Database,
		"PGUSER":       user,
		"PGPASSWORD":   password,
		"PGSSLMODE":    svc.ServiceUriParams["sslmode"],
		"DATABASE_URI": pool.ConnectionUri,
	}, nil
}

func findConnectionPool(svc *service.ServiceGetOut, name string) *service.ConnectionPoolOut {
	for i := range svc.ConnectionPools {
		if svc.ConnectionPools[i].PoolName == name {
			return &svc.ConnectionPools[i]
		}
	}
	return nil
}

// poolMatchesSpec reports whether the remote pool matches the mutable spec fields.
func poolMatchesSpec(remote *service.ConnectionPoolOut, cp *v1alpha1.ConnectionPool) bool {
	if cp.Spec.PoolMode != "" && string(remote.PoolMode) != string(cp.Spec.PoolMode) {
		return false
	}
	if cp.Spec.PoolSize != 0 && remote.PoolSize != cp.Spec.PoolSize {
		return false
	}
	return true
}
