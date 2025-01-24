package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	valkeyuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/valkey"
)

func TestValkey(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("valkey")
	yml, err := loadExampleYaml("valkey.yaml", map[string]string{
		"metadata.name":                  name,
		"spec.project":                   cfg.Project,
		"spec.cloudName":                 cfg.PrimaryCloudName,
		"spec.connInfoSecretTarget.name": name,
	})
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	rs := new(v1alpha1.Valkey)
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
	rsResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
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
	var ipFilterAvn []*valkeyuserconfig.IpFilter
	require.NoError(t, castInterface(rsAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, rs.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(rs.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["VALKEY_HOST"])
	assert.NotEmpty(t, secret.Data["VALKEY_PORT"])
	assert.NotEmpty(t, secret.Data["VALKEY_USER"])
	assert.NotEmpty(t, secret.Data["VALKEY_PASSWORD"])
}
