package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/clickhouse"
)

func getClickhouseYaml(project, cloudName, chName, dbName1, dbName2 string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[2]s
  plan: startup-16

  tags:
    env: test
    instance: foo

  userConfig:
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

---

apiVersion: aiven.io/v1alpha1
kind: ClickhouseDatabase
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName:  %[3]s

---

apiVersion: aiven.io/v1alpha1
kind: ClickhouseDatabase
metadata:
  name: %[5]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName:  %[3]s
`, project, cloudName, chName, dbName1, dbName2)
}

func TestClickhouse(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse")
	dbName1 := randName("clickhouse")
	dbName2 := randName("clickhouse")
	yml := getClickhouseYaml(cfg.Project, cfg.PrimaryCloudName, chName, dbName1, dbName2)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	// THEN
	chAvn, err := avnClient.Services.Get(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chAvn.Name, ch.GetName())
	assert.Equal(t, "RUNNING", ch.Status.State)
	assert.Equal(t, chAvn.State, ch.Status.State)
	assert.Equal(t, chAvn.Plan, ch.Spec.Plan)
	assert.Equal(t, chAvn.CloudName, ch.Spec.CloudName)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ch.Spec.Tags)
	chResp, err := avnClient.ServiceTags.Get(ctx, cfg.Project, chName)
	require.NoError(t, err)
	assert.Equal(t, chResp.Tags, ch.Spec.Tags)

	// UserConfig test
	require.NotNil(t, ch.Spec.UserConfig)

	// Validates ip filters
	require.Len(t, ch.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", ch.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *ch.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", ch.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, ch.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*clickhouseuserconfig.IpFilter
	require.NoError(t, castInterface(chAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, ch.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(ch.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USER"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])

	// New secrets
	assert.NotEmpty(t, secret.Data["CLICKHOUSE_HOST"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSE_PORT"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSE_USER"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSE_PASSWORD"])
	assert.NotEmpty(t, secret.Data["CLICKHOUSE_CA_CERT"])

	// Validates ClickhouseDatabase
	db1 := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db1, dbName1))

	// Database exists
	dbAvn1, err := avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName1)
	require.NoError(t, err)
	assert.Equal(t, dbName1, db1.GetName())
	assert.Equal(t, dbAvn1.Name, db1.GetName())

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, db is destroyed in Aiven.
	db2 := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db2, dbName2))

	dbAvn2, err := avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName2)
	require.NoError(t, err)
	assert.Equal(t, dbName2, db2.GetName())
	assert.Equal(t, dbAvn2.Name, db2.GetName())

	// Calls reconciler delete
	assert.NoError(t, s.Delete(db2, func() error {
		_, err = avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName2)
		return err
	}))
}
