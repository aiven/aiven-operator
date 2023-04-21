package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/opensearch"
)

func getOpenSearchYaml(project, name string) string {
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
  cloudName: google-europe-west1
  plan: startup-4
  disk_space: 240Gib

  tags:
    env: test
    instance: foo

  userConfig:
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name)
}

func TestOpenSearch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	name := randName("opensearch")
	yml := getOpenSearchYaml(testProject, name)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	os := new(v1alpha1.OpenSearch)
	require.NoError(t, s.GetRunning(os, name))

	// THEN
	osAvn, err := avnClient.Services.Get(testProject, name)
	require.NoError(t, err)
	assert.Equal(t, osAvn.Name, os.GetName())
	assert.Equal(t, "RUNNING", os.Status.State)
	assert.Equal(t, osAvn.State, os.Status.State)
	assert.Equal(t, osAvn.Plan, os.Spec.Plan)
	assert.Equal(t, osAvn.CloudName, os.Spec.CloudName)
	assert.Equal(t, "240Gib", os.Spec.DiskSpace)
	assert.Equal(t, 245760, osAvn.DiskSpaceMB)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, os.Spec.Tags)

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
	assert.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
	assert.Equal(t, map[string]string{"baz": "egg"}, secret.Labels)
}
