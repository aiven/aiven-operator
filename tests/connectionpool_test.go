package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestConnectionPool(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("pg")
	dbName := randName("database")
	userName := randName("service-user")
	poolName := randName("connection-pool")
	yml, err := loadExampleYaml("connectionpool.yaml", map[string]string{
		"doc[0].metadata.name":     poolName,
		"doc[0].spec.project":      cfg.Project,
		"doc[0].spec.serviceName":  pgName,
		"doc[0].spec.databaseName": dbName,
		"doc[0].spec.username":     userName,

		"doc[1].metadata.name":  pgName,
		"doc[1].spec.project":   cfg.Project,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,

		"doc[2].metadata.name":    dbName,
		"doc[2].spec.project":     cfg.Project,
		"doc[2].spec.serviceName": pgName,

		"doc[3].metadata.name":    userName,
		"doc[3].spec.project":     cfg.Project,
		"doc[3].spec.serviceName": pgName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	db := new(v1alpha1.Database)
	require.NoError(t, s.GetRunning(db, dbName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	pool := new(v1alpha1.ConnectionPool)
	require.NoError(t, s.GetRunning(pool, poolName))

	// THEN
	// Validates PostgreSQL
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)

	// Validates Database
	dbAvn, err := avnClient.Databases.Get(ctx, cfg.Project, pgName, dbName)
	require.NoError(t, err)
	assert.Equal(t, dbName, db.GetName())
	assert.Equal(t, dbAvn.DatabaseName, db.GetName())

	// Validates ServiceUser
	userAvn, err := avnClient.ServiceUsers.Get(ctx, cfg.Project, pgName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// Validates ConnectionPool
	poolAvn, err := avnClient.ConnectionPools.Get(ctx, cfg.Project, pgName, poolName)
	require.NoError(t, err)
	assert.Equal(t, pgName, pool.Spec.ServiceName)
	assert.Equal(t, poolName, pool.GetName())
	assert.Equal(t, poolName, poolAvn.PoolName)
	assert.Equal(t, dbName, pool.Spec.DatabaseName)
	assert.Equal(t, dbName, poolAvn.Database)
	assert.Equal(t, userName, pool.Spec.Username)
	assert.Equal(t, userName, poolAvn.Username)
	assert.Equal(t, 25, pool.Spec.PoolSize)
	assert.Equal(t, 25, poolAvn.PoolSize)
	assert.Equal(t, "transaction", pool.Spec.PoolMode)
	assert.Equal(t, "transaction", poolAvn.PoolMode)

	// Validates Secret
	secret, err := s.GetSecret(pool.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["PGHOST"])
	assert.NotEmpty(t, secret.Data["PGPORT"])
	assert.NotEmpty(t, secret.Data["PGDATABASE"])
	assert.NotEmpty(t, secret.Data["PGUSER"])
	assert.NotEmpty(t, secret.Data["PGPASSWORD"])
	assert.NotEmpty(t, secret.Data["PGSSLMODE"])
	assert.NotEmpty(t, secret.Data["DATABASE_URI"])

	// New secrets
	assert.Equal(t, poolName, string(secret.Data["CONNECTIONPOOL_NAME"]))
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_HOST"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_PORT"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_DATABASE"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_USER"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_PASSWORD"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_SSLMODE"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_DATABASE_URI"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_CA_CERT"])

	// URI contains valid values
	uri := string(secret.Data["CONNECTIONPOOL_DATABASE_URI"])
	assert.Contains(t, uri, string(secret.Data["CONNECTIONPOOL_HOST"]))
	assert.Contains(t, uri, string(secret.Data["CONNECTIONPOOL_PORT"]))

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that controller can delete db itself
	assert.NoError(t, s.Delete(pool, func() error {
		_, err = avnClient.ConnectionPools.Get(ctx, cfg.Project, pgName, poolName)
		return err
	}))
}

// TestConnectionPoolWithReuseInboundUser verifies that a ConnectionPool can be created without
// specifying a username, which enables the "Reuse Inbound User" feature. This allows the
// connection pool to use the credentials of whoever is connecting to the pool rather than
// a fixed service user.
func TestConnectionPoolWithReuseInboundUser(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	var (
		ctx, cancel = testCtx()
		pgName      = randName("pg")
		dbName      = randName("database")
		poolName    = randName("connection-pool-inbound")

		s = NewSession(ctx, k8sClient, cfg.Project)

		findPoolFunc = func(projectName, serviceName, poolName string) *service.ConnectionPoolOut {
			var avnPool *service.ConnectionPoolOut
			services, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
			if errors.IsNotFound(err) {
				return avnPool
			}
			require.NoError(t, err)

			for _, p := range services.ConnectionPools {
				if p.PoolName == poolName {
					avnPool = &p
					break
				}
			}

			return avnPool
		}
	)

	defer func() {
		cancel()
		s.Destroy(t)
	}()

	// Step 1: Create PostgreSQL service directly
	pgObj := &v1alpha1.PostgreSQL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pgName,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.PostgreSQLSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				BaseServiceFields: v1alpha1.BaseServiceFields{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: cfg.Project,
						},
						AuthSecretRefField: v1alpha1.AuthSecretRefField{
							AuthSecretRef: &v1alpha1.AuthSecretReference{
								Name: secretRefName,
								Key:  secretRefKey,
							},
						},
					},
					Plan:      "startup-4",
					CloudName: cfg.PrimaryCloudName,
				},
			},
		},
	}

	dbObj := &v1alpha1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbName,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.DatabaseSpec{
			ServiceDependant: v1alpha1.ServiceDependant{
				ProjectDependant: v1alpha1.ProjectDependant{
					ProjectField: v1alpha1.ProjectField{
						Project: cfg.Project,
					},
					AuthSecretRefField: v1alpha1.AuthSecretRefField{
						AuthSecretRef: &v1alpha1.AuthSecretReference{
							Name: secretRefName,
							Key:  secretRefKey,
						},
					},
				},
				ServiceField: v1alpha1.ServiceField{
					ServiceName: pgName,
				},
			},
		},
	}

	poolObj := &v1alpha1.ConnectionPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      poolName,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.ConnectionPoolSpec{
			ServiceDependant: v1alpha1.ServiceDependant{
				ProjectDependant: v1alpha1.ProjectDependant{
					ProjectField: v1alpha1.ProjectField{
						Project: cfg.Project,
					},
					AuthSecretRefField: v1alpha1.AuthSecretRefField{
						AuthSecretRef: &v1alpha1.AuthSecretReference{
							Name: secretRefName,
							Key:  secretRefKey,
						},
					},
				},
				ServiceField: v1alpha1.ServiceField{
					ServiceName: pgName,
				},
			},
			DatabaseName: dbName,
			// No username field
			PoolSize: 10,
			PoolMode: "transaction",
		},
	}
	require.NoError(t, s.ApplyObjects(pgObj, dbObj, poolObj))
	require.NoError(t, s.GetRunning(pgObj, pgName))
	require.NoError(t, s.GetRunning(dbObj, dbName))
	require.NoError(t, s.GetRunning(poolObj, poolName))

	avnPool := findPoolFunc(cfg.Project, pgName, poolName)
	require.NotNil(t, avnPool, "connection pool should be created in Aiven")

	assert.Equal(t, pgName, poolObj.Spec.ServiceName)
	assert.Equal(t, poolName, poolObj.GetName())
	assert.Equal(t, poolName, avnPool.PoolName)
	assert.Equal(t, dbName, poolObj.Spec.DatabaseName)
	assert.Equal(t, dbName, avnPool.Database)
	assert.Empty(t, poolObj.Spec.Username) // username should be empty in the spec

	secret, err := s.GetSecret(poolObj.GetName())
	require.NoError(t, err)

	// Check secret has necessary connection info
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_HOST"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_PORT"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_DATABASE"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_USER"]) // For inbound user, this is the default user
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_PASSWORD"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_SSLMODE"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_DATABASE_URI"])
	assert.NotEmpty(t, secret.Data["CONNECTIONPOOL_CA_CERT"])

	// Update the connection pool
	poolObj.Spec.PoolSize = 25
	poolObj.Spec.PoolMode = "session"
	require.NoError(t, err)
	require.NoError(t, s.ApplyObjects(poolObj))
	require.NoError(t, s.GetRunning(poolObj, poolName))

	// Verify that updating the connection pool works on Kubernetes
	require.Eventually(t, func() bool {
		updatedPoolObj := &v1alpha1.ConnectionPool{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      poolName,
			Namespace: defaultNamespace,
		}, updatedPoolObj)
		if err != nil {
			return false
		}
		return updatedPoolObj.Spec.PoolSize == 25 &&
			updatedPoolObj.Spec.PoolMode == "session"
	}, 1*time.Minute, 5*time.Second, "connection pool changes should be reflected in Kubernetes")

	// Verify that updating the connection pool works on Aiven
	require.Eventually(t, func() bool {
		updatedAvnPool := findPoolFunc(cfg.Project, pgName, poolName)
		if updatedAvnPool == nil {
			return false
		}
		return updatedAvnPool.PoolSize == 25 &&
			updatedAvnPool.PoolMode == "session"
	}, 2*time.Minute, 5*time.Second, "connection pool changes should be propagated to Aiven")

	// Delete the connection pool
	assert.NoError(t, s.Delete(poolObj, func() error {
		p := findPoolFunc(cfg.Project, pgName, poolName)
		if p != nil {
			return nil
		}

		return avngen.Error{Status: http.StatusNotFound}
	}))
}
