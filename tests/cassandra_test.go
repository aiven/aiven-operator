package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	cassandrauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/cassandra"
)

func getCassandraYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Cassandra
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4
  disk_space: 450GiB

  tags:
    env: test
    instance: foo

  userConfig:
    migrate_sstableloader: true
    public_access:
      prometheus: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName)
}

func TestCassandra(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("cassandra")
	yml := getCassandraYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	cs := new(v1alpha1.Cassandra)
	require.NoError(t, s.GetRunning(cs, name))

	// THEN
	csAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, csAvn.ServiceName, cs.GetName())
	assert.Equal(t, serviceRunningState, cs.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, csAvn.State)
	assert.Equal(t, csAvn.Plan, cs.Spec.Plan)
	assert.Equal(t, csAvn.CloudName, cs.Spec.CloudName)
	assert.Equal(t, "450GiB", cs.Spec.DiskSpace)
	assert.Equal(t, int(460800), *csAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, cs.Spec.Tags)
	csTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, csTags, cs.Spec.Tags)

	// UserConfig test
	require.NotNil(t, cs.Spec.UserConfig)
	require.NotNil(t, cs.Spec.UserConfig.PublicAccess)
	assert.Equal(t, anyPointer(true), cs.Spec.UserConfig.PublicAccess.Prometheus)
	assert.Equal(t, anyPointer(true), cs.Spec.UserConfig.MigrateSstableloader)

	// Validates ip filters
	require.Len(t, cs.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", cs.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *cs.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", cs.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, cs.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*cassandrauserconfig.IpFilter
	require.NoError(t, castInterface(csAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, cs.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(cs.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["CASSANDRA_HOST"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_PORT"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_USER"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_URI"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_HOSTS"])
	assert.NotEmpty(t, secret.Data["CASSANDRA_CA_CERT"])
}
