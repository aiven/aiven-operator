package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getPgReadReplicaYaml(project, masterName, replicaName, cloudName string) string {
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
  cloudName: %[4]s
  plan: startup-4

  tags:
    env: prod
    instance: master

---

apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  serviceIntegrations:
    - integrationType: read_replica
      sourceServiceName: %[2]s

  tags:
    env: test
    instance: replica

  userConfig:
    public_access:
      pg: true
      prometheus: true

`, project, masterName, replicaName, cloudName)
}

func TestPgReadReplica(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	masterName := randName("pg-master")
	replicaName := randName("pg-replica")
	yml := getPgReadReplicaYaml(testProject, masterName, replicaName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	master := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(master, masterName))

	replica := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(replica, replicaName))

	// THEN
	// Validates instances
	masterAvn, err := avnClient.Services.Get(ctx, testProject, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterAvn.Name, master.GetName())
	assert.Equal(t, "RUNNING", master.Status.State)
	assert.Equal(t, masterAvn.State, master.Status.State)
	assert.Equal(t, masterAvn.Plan, master.Spec.Plan)
	assert.Equal(t, masterAvn.CloudName, master.Spec.CloudName)
	assert.NotNil(t, masterAvn.UserConfig) // "Aiven instance has defaults set"
	assert.Nil(t, master.Spec.UserConfig)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, master.Spec.Tags)
	masterResp, err := avnClient.ServiceTags.Get(ctx, testProject, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterResp.Tags, master.Spec.Tags)

	replicaAvn, err := avnClient.Services.Get(ctx, testProject, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaAvn.Name, replica.GetName())
	assert.Equal(t, "RUNNING", replica.Status.State)
	assert.Equal(t, replicaAvn.State, replica.Status.State)
	assert.Equal(t, replicaAvn.Plan, replica.Spec.Plan)
	assert.Equal(t, replicaAvn.CloudName, replica.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "replica"}, replica.Spec.Tags)
	replicaResp, err := avnClient.ServiceTags.Get(ctx, testProject, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaResp.Tags, replica.Spec.Tags)

	// UserConfig test
	require.NotNil(t, replica.Spec.UserConfig)

	// UserConfig nested options test
	require.NotNil(t, replica.Spec.UserConfig.PublicAccess)
	assert.Equal(t, anyPointer(true), replica.Spec.UserConfig.PublicAccess.Prometheus)
	assert.Equal(t, anyPointer(true), replica.Spec.UserConfig.PublicAccess.Pg)

	// Secrets test
	for _, o := range []*v1alpha1.PostgreSQL{master, replica} {
		secret, err := s.GetSecret(o.GetName())
		require.NoError(t, err)
		assert.NotEmpty(t, secret.Data["PGHOST"])
		assert.NotEmpty(t, secret.Data["PGPORT"])
		assert.NotEmpty(t, secret.Data["PGDATABASE"])
		assert.NotEmpty(t, secret.Data["PGUSER"])
		assert.NotEmpty(t, secret.Data["PGPASSWORD"])
		assert.NotEmpty(t, secret.Data["PGSSLMODE"])
		assert.NotEmpty(t, secret.Data["DATABASE_URI"])

		// New secrets
		assert.NotEmpty(t, secret.Data["POSTGRESQL_HOST"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_PORT"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_DATABASE"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_USER"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_PASSWORD"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_SSLMODE"])
		assert.NotEmpty(t, secret.Data["POSTGRESQL_DATABASE_URI"])
	}
}

func getPgCustomPrefixYaml(project, pgName, cloudName string) string {
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
  cloudName: %[3]s
  plan: startup-4

  connInfoSecretTarget:
    name: postgresql-secret
    prefix: MY_PG_
    annotations:
      foo: bar
    labels:
      baz: egg

  tags:
    env: prod
    instance: pg
  
  userConfig:
    pg_version: "14"
`, project, pgName, cloudName)
}

func TestPgCustomPrefix(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	pgName := randName("secret-prefix")
	yml := getPgCustomPrefixYaml(testProject, pgName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	// THEN
	// Validates instance
	pgAvn, err := avnClient.Services.Get(ctx, testProject, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.Name, pg.GetName())
	assert.Equal(t, "RUNNING", pg.Status.State)
	assert.Equal(t, pgAvn.State, pg.Status.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "pg"}, pg.Spec.Tags)
	masterResp, err := avnClient.ServiceTags.Get(ctx, testProject, pgName)
	require.NoError(t, err)
	assert.Equal(t, masterResp.Tags, pg.Spec.Tags)

	// UserConfig test
	require.NotNil(t, pg.Spec.UserConfig)
	assert.NotNil(t, pgAvn.UserConfig) // "Aiven instance has defaults set"

	// Tests non-strict yaml. By sending string-integer we expect it's parsed as a string.
	// Default version is 15, we get 14, as we set it.
	assert.Equal(t, "14", pgAvn.UserConfig["pg_version"])
	assert.Equal(t, anyPointer("14"), pg.Spec.UserConfig.PgVersion)

	// Validates secret
	secret, err := s.GetSecret("postgresql-secret")
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)

	// Legacy secrets
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["PGHOST"])
	assert.NotEmpty(t, secret.Data["PGPORT"])
	assert.NotEmpty(t, secret.Data["PGDATABASE"])
	assert.NotEmpty(t, secret.Data["PGUSER"])
	assert.NotEmpty(t, secret.Data["PGPASSWORD"])
	assert.NotEmpty(t, secret.Data["PGSSLMODE"])
	assert.NotEmpty(t, secret.Data["DATABASE_URI"])

	// New secrets
	assert.NotEmpty(t, secret.Data["MY_PG_HOST"])
	assert.NotEmpty(t, secret.Data["MY_PG_PORT"])
	assert.NotEmpty(t, secret.Data["MY_PG_DATABASE"])
	assert.NotEmpty(t, secret.Data["MY_PG_USER"])
	assert.NotEmpty(t, secret.Data["MY_PG_PASSWORD"])
	assert.NotEmpty(t, secret.Data["MY_PG_SSLMODE"])
	assert.NotEmpty(t, secret.Data["MY_PG_DATABASE_URI"])
}
