package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	grafanauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/grafana"
)

func getGrafanaYaml(project, name string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Grafana
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: google-europe-west1
  plan: startup-1

  tags:
    env: test
    instance: foo

  userConfig:
    alerting_enabled: true
    public_access:
      grafana: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

`, project, name)
}

func TestGrafana(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	name := randName("grafana")
	yml := getGrafanaYaml(testProject, name)
	s, err := NewSession(k8sClient, avnClient, testProject, yml)
	require.NoError(t, err)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply())

	// Waits kube objects
	grafana := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(grafana, name))

	// THEN
	grafanaAvn, err := avnClient.Services.Get(testProject, name)
	require.NoError(t, err)
	assert.Equal(t, grafanaAvn.Name, grafana.GetName())
	assert.Equal(t, "RUNNING", grafana.Status.State)
	assert.Equal(t, grafanaAvn.State, grafana.Status.State)
	assert.Equal(t, grafanaAvn.Plan, grafana.Spec.Plan)
	assert.Equal(t, grafanaAvn.CloudName, grafana.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, grafana.Spec.Tags)
	grafanaResp, err := avnClient.ServiceTags.Get(testProject, name)
	require.NoError(t, err)
	assert.Equal(t, grafanaResp.Tags, grafana.Spec.Tags)

	// UserConfig test
	require.NotNil(t, grafana.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), grafana.Spec.UserConfig.AlertingEnabled)

	// UserConfig nested options test
	require.NotNil(t, grafana.Spec.UserConfig.PublicAccess)
	assert.Equal(t, anyPointer(true), grafana.Spec.UserConfig.PublicAccess.Grafana)

	// Validates ip filters
	require.Len(t, grafana.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", grafana.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *grafana.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", grafana.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, grafana.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*grafanauserconfig.IpFilter
	require.NoError(t, castInterface(grafanaAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, grafana.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(grafana.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["GRAFANA_HOST"])
	assert.NotEmpty(t, secret.Data["GRAFANA_PORT"])
	assert.NotEmpty(t, secret.Data["GRAFANA_USER"])
	assert.NotEmpty(t, secret.Data["GRAFANA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["GRAFANA_URI"])
	assert.NotEmpty(t, secret.Data["GRAFANA_HOSTS"])
}
