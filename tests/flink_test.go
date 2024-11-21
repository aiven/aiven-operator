package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	flinkuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/flink"
)

func TestFlink(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("flink")
	yml, err := loadExampleYaml("flink.yaml", map[string]string{
		"google-europe-west1": cfg.PrimaryCloudName,
		"my-aiven-project":    cfg.Project,
		"my-flink":            name,
	})
	require.NoError(t, err)

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

	// UserConfig test
	assert.Equal(t, anyPointer(10), flink.Spec.UserConfig.NumberOfTaskSlots)

	// Validates ip filters
	require.Len(t, flink.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", flink.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "whatever", *flink.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", flink.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, flink.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*flinkuserconfig.IpFilter
	require.NoError(t, castInterface(flinkAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, flink.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(flink.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["FLINK_HOST"])
	assert.NotEmpty(t, secret.Data["FLINK_USER"])
	assert.NotEmpty(t, secret.Data["FLINK_PASSWORD"])
	assert.NotEmpty(t, secret.Data["FLINK_URI"])
	assert.NotEmpty(t, secret.Data["FLINK_HOSTS"])
}
