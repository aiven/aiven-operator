//go:build redis

package tests

import (
	"fmt"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
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
	t.Skip("Redis is deprecated")

	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("redis")
	yml := getRedisYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	rs := new(v1alpha1.Redis)
	require.NoError(t, s.GetRunning(rs, name))

	// THEN
	rsAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, rsAvn.ServiceName, rs.GetName())
	assert.Equal(t, serviceRunningState, rs.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, rsAvn.State)
	assert.Equal(t, rsAvn.Plan, rs.Spec.Plan)
	assert.Equal(t, rsAvn.CloudName, rs.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, rs.Spec.Tags)
	rsTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, rsTags, rs.Spec.Tags)

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

	// Tests service power off functionality
	// Note: Power on testing is handled generically in generic_service_handler_test.go
	// since it's consistent across services. Power off testing is done here since
	// the flow can vary by service type and may require service-specific steps.
	poweredOff := rs.DeepCopy()
	poweredOff.Spec.Powered = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, poweredOff))
	require.NoError(t, s.GetRunning(poweredOff, name))

	poweredOffAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, poweredOffAvn.State)
}
