package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	redisuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/redis"
)

func getRedisYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Redis
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4

  tags:
    env: test
    instance: foo

  userConfig:
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName)
}

func TestRedis(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	name := randName("redis")
	yml := getRedisYaml(testProject, name, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	rs := new(v1alpha1.Redis)
	require.NoError(t, s.GetRunning(rs, name))

	// THEN
	rsAvn, err := avnClient.Services.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Equal(t, rsAvn.Name, rs.GetName())
	assert.Equal(t, "RUNNING", rs.Status.State)
	assert.Equal(t, rsAvn.State, rs.Status.State)
	assert.Equal(t, rsAvn.Plan, rs.Spec.Plan)
	assert.Equal(t, rsAvn.CloudName, rs.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, rs.Spec.Tags)
	rsResp, err := avnClient.ServiceTags.Get(ctx, testProject, name)
	require.NoError(t, err)
	assert.Equal(t, rsResp.Tags, rs.Spec.Tags)

	// UserConfig test
	require.NotNil(t, rs.Spec.UserConfig)

	// Validates ip filters
	require.Len(t, rs.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", rs.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *rs.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", rs.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, rs.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*redisuserconfig.IpFilter
	require.NoError(t, castInterface(rsAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, rs.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(rs.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USER"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])

	// New secrets
	assert.NotEmpty(t, secret.Data["REDIS_HOST"])
	assert.NotEmpty(t, secret.Data["REDIS_PORT"])
	assert.NotEmpty(t, secret.Data["REDIS_USER"])
	assert.NotEmpty(t, secret.Data["REDIS_PASSWORD"])
	assert.NotEmpty(t, secret.Data["REDIS_CA_CERT"])
}
