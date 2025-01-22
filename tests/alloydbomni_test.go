package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	alloydbomniuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/alloydbomni"
)

func TestAlloyDBOmni(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	name := randName("alloydbomni")
	yml, err := loadExampleYaml("alloydbomni.yaml", map[string]string{
		"google-europe-west1": cfg.PrimaryCloudName,
		"my-aiven-project":    cfg.Project,
		"my-alloydbomni":      name,
	})
	require.NoError(t, err)

	// WHEN
	err = s.Apply(yml)

	// THEN
	require.NoError(t, err)

	// Waits kube objects
	adbo := new(v1alpha1.AlloyDBOmni)
	require.NoError(t, s.GetRunning(adbo, name))

	// Validate the resource
	adboAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, adboAvn.ServiceName, adbo.GetName())
	assert.Equal(t, serviceRunningState, adbo.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, adboAvn.State)
	assert.Equal(t, adboAvn.Plan, adbo.Spec.Plan)
	assert.Equal(t, adboAvn.CloudName, adbo.Spec.CloudName)
	assert.Equal(t, "90GiB", adbo.Spec.DiskSpace)
	assert.Equal(t, int(92160), *adboAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, adbo.Spec.Tags)
	adboResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, adboResp.Tags, adbo.Spec.Tags)

	// UserConfig test
	require.NotNil(t, adbo.Spec.UserConfig)
	require.NotNil(t, adbo.Spec.UserConfig.ServiceLog)
	assert.Equal(t, anyPointer(true), adbo.Spec.UserConfig.ServiceLog)

	// Validates ip filters
	require.Len(t, adbo.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", adbo.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *adbo.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", adbo.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, adbo.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*alloydbomniuserconfig.IpFilter
	require.NoError(t, castInterface(adboAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, adbo.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(adbo.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_HOST"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PORT"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_USER"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_PASSWORD"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_SSLMODE"])
	assert.NotEmpty(t, secret.Data["ALLOYDBOMNI_DATABASE_URI"])
}
