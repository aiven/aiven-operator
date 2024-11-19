package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	flinkuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/flink"
)

func getFlinkYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Flink
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: business-4

  tags:
    env: test
    instance: foo

  userConfig:
    number_of_task_slots: 10
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name, cloudName)
}

func TestFlink(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("flink")
	yml := getFlinkYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	flink := new(v1alpha1.Flink)
	require.NoError(t, s.GetRunning(flink, name))

	// THEN
	flinkAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, flinkAvn.ServiceName, flink.GetName())
	assert.Equal(t, serviceRunningState, flink.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, flinkAvn.State)
	assert.Equal(t, flinkAvn.Plan, flink.Spec.Plan)
	assert.Equal(t, flinkAvn.CloudName, flink.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, flink.Spec.Tags)
	flinkResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, flinkResp.Tags, flink.Spec.Tags)

	// UserConfig test
	assert.Equal(t, anyPointer(10), flink.Spec.UserConfig.NumberOfTaskSlots)

	// Validates ip filters
	require.Len(t, flink.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", flink.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *flink.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", flink.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, flink.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*flinkuserconfig.IpFilter
	require.NoError(t, castInterface(flinkAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, flink.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(flink.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["FLINK_HOST"])
	assert.NotEmpty(t, secret.Data["FLINK_USER"])
	assert.NotEmpty(t, secret.Data["FLINK_PASSWORD"])
	assert.NotEmpty(t, secret.Data["FLINK_URI"])
	assert.NotEmpty(t, secret.Data["FLINK_HOSTS"])
}
