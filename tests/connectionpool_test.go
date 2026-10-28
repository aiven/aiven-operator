//go:build postgresql

package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/postgresql"
	"github.com/aiven/go-client-codegen/handler/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
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
	s := NewSession(ctx, k8sClient)

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
	dbAvn, err := controllers.GetDatabaseByName(ctx, avnGen, cfg.Project, pgName, dbName)
	require.NoError(t, err)
	assert.Equal(t, dbName, db.GetName())
	assert.Equal(t, dbAvn.DatabaseName, db.GetName())

	// Validates ServiceUser
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, pgName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, user.GetName())
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, pgName, user.Spec.ServiceName)

	// Validates ConnectionPool
	poolAvn, err := getConnectionPoolByName(ctx, cfg.Project, pgName, poolName)
	require.NoError(t, err)
	assert.Equal(t, pgName, pool.Spec.ServiceName)
	assert.Equal(t, poolName, pool.GetName())
	assert.Equal(t, poolName, poolAvn.PoolName)
	assert.Equal(t, dbName, pool.Spec.DatabaseName)
	assert.Equal(t, dbName, poolAvn.Database)
	assert.Equal(t, userName, pool.Spec.Username)
	assert.Equal(t, userName, fromPtr(poolAvn.Username))
	assert.Equal(t, 25, pool.Spec.PoolSize)
	assert.Equal(t, 25, poolAvn.PoolSize)
	assert.EqualValues(t, "transaction", pool.Spec.PoolMode)
	assert.EqualValues(t, "transaction", poolAvn.PoolMode)

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

	// Validate connection pool specific secrets
	validateConnectionPoolSecret(t, secret, poolName)

	// Test connection to PostgreSQL through connection pool
	require.Eventually(t, func() bool {
		return testConnectionToDatabase(ctx, t, string(secret.Data["CONNECTIONPOOL_DATABASE_URI"]), string(secret.Data["CONNECTIONPOOL_USER"]))
	}, 3*time.Minute, 10*time.Second, "should be able to connect to PostgreSQL through connection pool")

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, pool is destroyed in Aiven. No service — no pool. No pool — no pool.
	// And we make sure that controller can delete db itself
	assert.NoError(t, s.Delete(pool, func() error {
		_, err = getConnectionPoolByName(ctx, cfg.Project, pgName, poolName)
		return err
	}))
}

func TestConnectionPoolAdoptionCredentials(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()
	pgName := sharedResources.AcquirePostgreSQL(t, shared).GetName()
	pg, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	dbName := pg.ServiceUriParams["dbname"]
	require.NotEmpty(t, dbName)
	require.Contains(t, pg.Databases, dbName)

	createUser := func(name string) *service.ServiceUserCreateOut {
		t.Helper()
		username := randName(name)
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
			defer cancel()
			if err := avnGen.ServiceUserDelete(cleanupCtx, cfg.Project, pgName, username); !avngen.IsNotFound(err) {
				assert.NoError(t, err)
			}
		})
		user, err := avnGen.ServiceUserCreate(ctx, cfg.Project, pgName, &service.ServiceUserCreateIn{Username: username})
		require.NoError(t, err)
		require.NotEmpty(t, user.Password)
		return user
	}
	observedUser := createUser("pool-observed-user")
	specUser := createUser("pool-spec-user")
	require.True(t, observedUser.Password != specUser.Password, "the users must have distinct passwords")

	for _, tc := range []struct {
		name         string
		policy       string
		expectedUser *service.ServiceUserCreateOut
	}{
		{name: "Full", expectedUser: specUser}, // Full is the default when the annotation is absent.
		{name: "CreateOnly", policy: "CreateOnly", expectedUser: observedUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			poolName := randName("connection-pool-adopt")
			pool := &v1alpha1.ConnectionPool{
				TypeMeta: metav1.TypeMeta{APIVersion: "aiven.io/v1alpha1", Kind: "ConnectionPool"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      poolName,
					Namespace: defaultNamespace,
				},
			}
			pool.Spec.Project = cfg.Project
			pool.Spec.ServiceName = pgName
			pool.Spec.AuthSecretRef = &v1alpha1.AuthSecretReference{Name: secretRefName, Key: secretRefKey}
			pool.Spec.DatabaseName = dbName
			pool.Spec.Username = specUser.Username
			pool.Spec.PoolMode = "transaction"
			pool.Spec.PoolSize = 25
			if tc.policy != "" {
				pool.Annotations = map[string]string{"controllers.aiven.io/management-policy": tc.policy}
			}

			// Register cleanup before creating the external pool, even if no CR is ever applied.
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
				defer cancel()
				// The finalizer may have removed the pool already; deleting it again returns 403.
				_, err := getConnectionPoolByName(cleanupCtx, cfg.Project, pgName, poolName)
				if avngen.IsNotFound(err) {
					return
				}
				if !assert.NoError(t, err) {
					return
				}
				if err := avnGen.ServicePGBouncerDelete(cleanupCtx, cfg.Project, pgName, poolName); !avngen.IsNotFound(err) {
					assert.NoError(t, err)
				}
			})
			require.NoError(t, avnGen.ServicePGBouncerCreate(ctx, cfg.Project, pgName, &postgresql.ServicePGBouncerCreateIn{
				Database: dbName,
				PoolName: poolName,
				PoolMode: pool.Spec.PoolMode,
				PoolSize: &pool.Spec.PoolSize,
				Username: &observedUser.Username,
			}))

			connect := func(rawURI, username, password, expectedUser string) bool {
				uri, err := url.Parse(rawURI)
				if err != nil {
					t.Log("connection pool URI is invalid")
					return false
				}
				// Aiven's URI may contain correct credentials even when the Secret fields do not.
				uri.User = url.UserPassword(username, password)
				attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
				return testConnectionToDatabase(attemptCtx, t, uri.String(), expectedUser)
			}

			// Prove the existing pool works before the operator sees the CR.
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				remote, err := getConnectionPoolByName(ctx, cfg.Project, pgName, poolName)
				if !assert.NoError(collect, err) {
					return
				}
				assert.Equal(collect, observedUser.Username, fromPtr(remote.Username))
				if !assert.NotEmpty(collect, remote.ConnectionUri) {
					return
				}
				assert.True(collect, connect(remote.ConnectionUri, observedUser.Username, observedUser.Password, observedUser.Username),
					"the original pool credentials must authenticate before adoption")
			}, 3*time.Minute, 10*time.Second)

			s := NewSession(ctx, k8sClient)
			defer s.Destroy(t)
			require.NoError(t, s.ApplyObjects(pool))
			require.NoError(t, s.GetRunning(pool, poolName))
			assert.Equal(t, specUser.Username, pool.Spec.Username)
			assert.Equal(t, tc.policy, pool.Annotations["controllers.aiven.io/management-policy"])

			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				remote, err := getConnectionPoolByName(ctx, cfg.Project, pgName, poolName)
				if !assert.NoError(collect, err) {
					return
				}
				assert.Equal(collect, tc.expectedUser.Username, fromPtr(remote.Username))
				assert.Equal(collect, dbName, remote.Database)
				assert.Equal(collect, pool.Spec.PoolSize, remote.PoolSize)
				assert.EqualValues(collect, pool.Spec.PoolMode, remote.PoolMode)

				secret, err := s.GetSecret(poolName)
				if !assert.NoError(collect, err) {
					return
				}
				assert.Equal(collect, tc.expectedUser.Username, string(secret.Data["CONNECTIONPOOL_USER"]))
				assert.Equal(collect, tc.expectedUser.Username, string(secret.Data["PGUSER"]))
				// Boolean assertions keep generated passwords out of failure output.
				assert.True(collect, string(secret.Data["CONNECTIONPOOL_PASSWORD"]) == tc.expectedUser.Password,
					"CONNECTIONPOOL_PASSWORD must belong to the observed pool user")
				assert.True(collect, string(secret.Data["PGPASSWORD"]) == tc.expectedUser.Password,
					"PGPASSWORD must belong to the observed pool user")
				if !assert.NotEmpty(collect, secret.Data["CONNECTIONPOOL_DATABASE_URI"]) {
					return
				}
				assert.True(collect, connect(string(secret.Data["CONNECTIONPOOL_DATABASE_URI"]),
					string(secret.Data["CONNECTIONPOOL_USER"]), string(secret.Data["CONNECTIONPOOL_PASSWORD"]), tc.expectedUser.Username),
					"published credentials must authenticate as the observed pool user")
			}, 3*time.Minute, 10*time.Second)
		})
	}
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
		dbName      = randName("database")
		poolName    = randName("connection-pool-inbound")
		userName    = randName("inbound-user") // Service user for testing "Reuse Inbound User" functionality

		s = NewSession(ctx, k8sClient)
	)

	defer func() {
		cancel()
		s.Destroy(t)
	}()

	pg := sharedResources.AcquirePostgreSQL(t, shared)

	pgName := pg.GetName()

	findPoolFunc := func(poolName string) *service.ConnectionPoolOut {
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

	dbObj := &v1alpha1.Database{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Database",
		},
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
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ConnectionPool",
		},
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

	// Create service user for testing "Reuse Inbound User" functionality
	userObj := &v1alpha1.ServiceUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ServiceUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: defaultNamespace,
		},
		Spec: v1alpha1.ServiceUserSpec{
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

	require.NoError(t, s.ApplyObjects(dbObj, poolObj, userObj))
	require.NoError(t, s.GetRunning(dbObj, dbName))
	require.NoError(t, s.GetRunning(poolObj, poolName))
	require.NoError(t, s.GetRunning(userObj, userName))

	avnPool := findPoolFunc(poolName)
	require.NotNil(t, avnPool, "connection pool should be created in Aiven")

	assert.Equal(t, pgName, poolObj.Spec.ServiceName)
	assert.Equal(t, poolName, poolObj.GetName())
	assert.Equal(t, poolName, avnPool.PoolName)
	assert.Equal(t, dbName, poolObj.Spec.DatabaseName)
	assert.Equal(t, dbName, avnPool.Database)
	assert.Empty(t, poolObj.Spec.Username) // username should be empty in the spec

	secret, err := s.GetSecret(poolObj.GetName())
	require.NoError(t, err)

	// Validate connection pool specific secrets
	validateConnectionPoolSecret(t, secret, poolName)

	// Get service user credentials to test inbound user feature
	userSecret, err := s.GetSecret(userObj.GetName())
	require.NoError(t, err, "Failed to get service user secret")

	// Test connecting as the inbound user through the connection pool
	// We'll create a custom URI that uses the connection pool host/port but the service user's credentials
	customURI := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=require",
		url.QueryEscape(string(userSecret.Data["SERVICEUSER_USERNAME"])),
		url.QueryEscape(string(userSecret.Data["SERVICEUSER_PASSWORD"])),
		string(secret.Data["CONNECTIONPOOL_HOST"]),
		string(secret.Data["CONNECTIONPOOL_PORT"]),
		string(secret.Data["CONNECTIONPOOL_NAME"]),
	)

	// Test connection to PostgreSQL through connection pool using the service user
	require.Eventually(t, func() bool {
		return testConnectionToDatabase(ctx, t, customURI, userName)
	}, 3*time.Minute, 10*time.Second, "should be able to connect to PostgreSQL through connection pool as inbound user")

	// Update the connection pool
	poolObj.Spec.PoolSize = 25
	poolObj.Spec.PoolMode = "session"
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
		updatedAvnPool := findPoolFunc(poolName)
		if updatedAvnPool == nil {
			return false
		}
		return updatedAvnPool.PoolSize == 25 &&
			updatedAvnPool.PoolMode == "session"
	}, 2*time.Minute, 5*time.Second, "connection pool changes should be propagated to Aiven")

	// Verify connection still works after update with the inbound user
	require.Eventually(t, func() bool {
		// Get updated connection details
		updatedSecret, err := s.GetSecret(poolObj.GetName())
		if err != nil {
			t.Logf("Failed to get updated secret: %v", err)
			return false
		}

		// Update the custom URI with new connection pool details
		updatedCustomURI := fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=require",
			url.QueryEscape(string(userSecret.Data["SERVICEUSER_USERNAME"])),
			url.QueryEscape(string(userSecret.Data["SERVICEUSER_PASSWORD"])),
			string(updatedSecret.Data["CONNECTIONPOOL_HOST"]),
			string(updatedSecret.Data["CONNECTIONPOOL_PORT"]),
			string(updatedSecret.Data["CONNECTIONPOOL_NAME"]),
		)

		return testConnectionToDatabase(ctx, t, updatedCustomURI, userName)
	}, 2*time.Minute, 10*time.Second, "should be able to connect to PostgreSQL after connection pool update")

	// Delete the connection pool
	assert.NoError(t, s.Delete(poolObj, func() error {
		p := findPoolFunc(poolName)
		if p != nil {
			return nil
		}

		return avngen.Error{Status: http.StatusNotFound}
	}))
}

func validateConnectionPoolSecret(t *testing.T, secret *corev1.Secret, poolName string) {
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
}

// TestConnectionToDatabase provides a simple test to verify database connection and user identity
func testConnectionToDatabase(ctx context.Context, t *testing.T, connectionURI string, expectedUser string) bool {
	// Try to connect to the database
	db, err := sql.Open("postgres", connectionURI)
	if err != nil {
		t.Logf("Failed to open connection: %v", err)
		return false
	}
	defer db.Close()

	// Ping the database to verify connection
	if err := db.PingContext(ctx); err != nil {
		t.Logf("Failed to ping database: %v", err)
		return false
	}

	// Verify the connected user
	var connectedUser string
	err = db.QueryRowContext(ctx, "SELECT current_user").Scan(&connectedUser)
	if err != nil {
		t.Logf("Failed to query current_user: %v", err)
		return false
	}

	if connectedUser != expectedUser {
		t.Logf("Connected user mismatch. Expected: %s, Got: %s", expectedUser, connectedUser)
		return false
	}

	t.Logf("Successfully connected as user: %s", connectedUser)
	return true
}

func getConnectionPoolByName(ctx context.Context, projectName, serviceName, poolName string) (*service.ConnectionPoolOut, error) {
	services, err := avnGen.ServiceGet(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	for _, p := range services.ConnectionPools {
		if p.PoolName == poolName {
			return &p, nil
		}
	}
	return nil, controllers.NewNotFound(fmt.Sprintf("connection pool %q not found", poolName))
}
