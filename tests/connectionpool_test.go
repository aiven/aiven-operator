package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getConnectionPoolYaml(project, pgName, dbName, userName, poolName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[6]s
  plan: startup-4

---

apiVersion: aiven.io/v1alpha1
kind: Database
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s

---

apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s

---

apiVersion: aiven.io/v1alpha1
kind: ConnectionPool
metadata:
  name: %[5]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  databaseName: %[3]s
  username: %[4]s
  poolMode: transaction
  poolSize: 25
`, project, pgName, dbName, userName, poolName, cloudName)
}

func TestConnectionPool(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("connection-pool")
	dbName := randName("connection-pool")
	userName := randName("connection-pool")
	poolName := randName("connection-pool")
	yml := getConnectionPoolYaml(cfg.Project, pgName, dbName, userName, poolName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

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
	pgAvn, err := avnClient.Services.Get(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.Name, pg.GetName())
	assert.Equal(t, "RUNNING", pg.Status.State)
	assert.Equal(t, pgAvn.State, pg.Status.State)
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
