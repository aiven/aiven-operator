//go:build postgresql

package tests

import (
	"fmt"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

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
	ctx, cancel := testCtx()
	defer cancel()

	masterName := randName("pg-master")
	replicaName := randName("pg-replica")
	yml := getPgReadReplicaYaml(cfg.Project, masterName, replicaName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

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
	masterAvn, err := avnGen.ServiceGet(ctx, cfg.Project, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterAvn.ServiceName, master.GetName())
	assert.Equal(t, serviceRunningState, master.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, masterAvn.State)
	assert.Equal(t, masterAvn.Plan, master.Spec.Plan)
	assert.Equal(t, masterAvn.CloudName, master.Spec.CloudName)
	assert.NotNil(t, masterAvn.UserConfig) // "Aiven instance has defaults set"
	assert.Nil(t, master.Spec.UserConfig)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "master"}, master.Spec.Tags)
	masterTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, masterName)
	require.NoError(t, err)
	assert.Equal(t, masterTags, master.Spec.Tags)

	replicaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaAvn.ServiceName, replica.GetName())
	assert.Equal(t, serviceRunningState, replica.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, replicaAvn.State)
	assert.Equal(t, replicaAvn.Plan, replica.Spec.Plan)
	assert.Equal(t, replicaAvn.CloudName, replica.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "replica"}, replica.Spec.Tags)
	replicaTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, replicaName)
	require.NoError(t, err)
	assert.Equal(t, replicaTags, replica.Spec.Tags)

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
		assert.NotEmpty(t, secret.Data["POSTGRESQL_CA_CERT"])
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
	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("secret-prefix")
	yml := getPgCustomPrefixYaml(cfg.Project, pgName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	// THEN
	// Validates instance
	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, pgAvn.ServiceName, pg.GetName())
	assert.Equal(t, serviceRunningState, pg.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, pgAvn.State)
	assert.Equal(t, pgAvn.Plan, pg.Spec.Plan)
	assert.Equal(t, pgAvn.CloudName, pg.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "prod", "instance": "pg"}, pg.Spec.Tags)
	masterTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, masterTags, pg.Spec.Tags)

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
	assert.NotEmpty(t, secret.Data["MY_PG_CA_CERT"])

	// Tests service power off functionality
	// Note: Power on testing is handled generically in generic_service_handler_test.go
	// since it's consistent across services. Power off testing is done here since
	// the flow can vary by service type and may require service-specific steps.
	poweredOff := pg.DeepCopy()
	poweredOff.Spec.Powered = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, poweredOff))
	require.NoError(t, s.GetRunning(poweredOff, pgName))

	poweredOffAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, poweredOffAvn.State)
}

func getPgUpgradeVersionYaml(project, pgName, cloudName, version string) string {
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
    userConfig:
      pg_version: "%[4]s"
`, project, pgName, cloudName, version)
}

func TestPgUpgradeVersion(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	pgVersions := service.TargetVersionTypeChoices()
	// The latest reported version from the upgrade check task may not be available in the operator yet.
	// Therefore, we set targetVersion to the second to last version.
	startingVersion := pgVersions[len(pgVersions)-3] // third to last
	targetVersion := pgVersions[len(pgVersions)-2]   // second to last

	ctx, cancel := testCtx()
	defer cancel()

	pgName := randName("upgrade-test")
	yaml := getPgUpgradeVersionYaml(cfg.Project, pgName, cfg.PrimaryCloudName, startingVersion)
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	require.NoError(t, s.Apply(yaml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, pgName))

	pgAvn, err := avnGen.ServiceGet(ctx, cfg.Project, pgName)
	require.NoError(t, err)
	assert.Equal(t, startingVersion, pgAvn.UserConfig["pg_version"])
	assert.Equal(t, anyPointer(startingVersion), pg.Spec.UserConfig.PgVersion)

	require.NotNil(t, pg.Spec.UserConfig)
	assert.NotNil(t, pgAvn.UserConfig)

	updatedYaml := getPgUpgradeVersionYaml(cfg.Project, pgName, cfg.PrimaryCloudName, targetVersion)
	require.NoError(t, s.Apply(updatedYaml))

	// Verify that the service was upgraded successfully
	var pgAvnUpd *service.ServiceGetOut
	require.NoError(t, retryForever(ctx, "check that PG version was upgraded", func() (bool, error) {
		pgAvnUpd, err = avnGen.ServiceGet(ctx, cfg.Project, pgName)
		if err != nil {
			return false, err
		}

		return pgAvnUpd.UserConfig["pg_version"] != startingVersion, nil
	}))

	pgUpd := new(v1alpha1.PostgreSQL)
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: pgName, Namespace: "default"}, pgUpd))
	assert.Equal(t, targetVersion, *pgUpd.Spec.UserConfig.PgVersion)
}
