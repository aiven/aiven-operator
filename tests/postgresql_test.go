package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getPgReadReplicaYaml(project, masterName, replicaName string) string {
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
  cloudName: google-europe-west1
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
  cloudName: google-europe-west1
  plan: startup-4

  serviceIntegrations:
    - integrationType: read_replica
      sourceServiceName: %[2]s

  tags:
    env: test
    instance: replica

  userConfig:
    pg_version: "14"
    public_access:
      pg: true
      prometheus: true

`, project, masterName, replicaName)
}

func TestPgReadReplica(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	masterName := randName("pg-master")
	replicaName := randName("pg-replica")
	yml := getPgReadReplicaYaml(testProject, masterName, replicaName)
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
	masterAvn, err := avnClient.Services.Get(testProject, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterAvn.Name, master.GetName())
	assert.Equal(t, "RUNNING", master.Status.State)
	assert.Equal(t, masterAvn.State, master.Status.State)
	assert.Equal(t, masterAvn.Plan, master.Spec.Plan)
	assert.Equal(t, masterAvn.CloudName, master.Spec.CloudName)
	assert.NotNil(t, masterAvn.UserConfig) // "Aiven instance has defaults set"
	assert.Nil(t, master.Spec.UserConfig)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, master.Spec.Tags)
	masterResp, err := avnClient.ServiceTags.Get(testProject, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterResp.Tags, master.Spec.Tags)

	replicaAvn, err := avnClient.Services.Get(testProject, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaAvn.Name, replica.GetName())
	assert.Equal(t, "RUNNING", replica.Status.State)
	assert.Equal(t, replicaAvn.State, replica.Status.State)
	assert.Equal(t, replicaAvn.Plan, replica.Spec.Plan)
	assert.Equal(t, replicaAvn.CloudName, replica.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "replica"}, replica.Spec.Tags)
	replicaResp, err := avnClient.ServiceTags.Get(testProject, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaResp.Tags, replica.Spec.Tags)

	// UserConfig test
	require.NotNil(t, replica.Spec.UserConfig)

	// Tests non-strict yaml. By sending string-integer we expect it's parsed as a string.
	// We don't set version number for master, we expect 14 to be a default value.
	// So this will fail when default version is changed.
	assert.Equal(t, "14", replicaAvn.UserConfig["pg_version"])
	assert.Equal(t, anyPointer("14"), replica.Spec.UserConfig.PgVersion)

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
