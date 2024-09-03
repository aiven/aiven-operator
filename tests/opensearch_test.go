package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/opensearch"
)

func getOpenSearchYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: OpenSearch
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-os-secret
    annotations:
      foo: bar
    labels:
      baz: egg

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4
  disk_space: 240GiB

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

func TestOpenSearch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("opensearch")
	yml := getOpenSearchYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	os := new(v1alpha1.OpenSearch)
	require.NoError(t, s.GetRunning(os, name))

	// THEN
	osAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, osAvn.ServiceName, os.GetName())
	assert.Equal(t, serviceRunningState, os.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, osAvn.State)
	assert.Equal(t, osAvn.Plan, os.Spec.Plan)
	assert.Equal(t, osAvn.CloudName, os.Spec.CloudName)
	assert.Equal(t, "240GiB", os.Spec.DiskSpace)
	assert.Equal(t, 245760, osAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, os.Spec.Tags)
	osResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, osResp.Tags, os.Spec.Tags)

	// UserConfig test
	require.NotNil(t, os.Spec.UserConfig)

	// Validates ip filters
	require.Len(t, os.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", os.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *os.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", os.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, os.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*opensearchuserconfig.IpFilter
	require.NoError(t, castInterface(osAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, os.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(os.Spec.ConnInfoSecretTarget.Name)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USER"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["OPENSEARCH_HOST"])
	assert.NotEmpty(t, secret.Data["OPENSEARCH_PORT"])
	assert.NotEmpty(t, secret.Data["OPENSEARCH_USER"])
	assert.NotEmpty(t, secret.Data["OPENSEARCH_PASSWORD"])
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)
}
