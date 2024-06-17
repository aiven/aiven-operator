package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	clickhouseuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/clickhouse"
	"github.com/aiven/aiven-operator/controllers"
)

func TestClickhouse(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse")
	dbName1 := randName("database")
	dbName2 := randName("database")
	roleName1 := randName("role")
	roleName2 := randName("role")

	ymlClickhouse, err := loadExampleYaml("clickhouse.yaml", map[string]string{
		"google-europe-west1": cfg.PrimaryCloudName,
		"my-aiven-project":    cfg.Project,
		"my-clickhouse":       chName,
	})
	require.NoError(t, err)
	ymlDatabase1, err := loadExampleYaml("clickhousedatabase.yaml", map[string]string{
		"my-aiven-project": cfg.Project,
		"my-db":            dbName1,
		"my-clickhouse":    chName,
		// Remove 'databaseName' from the initial yaml
		"databaseName: example-db": "",
	})
	require.NoError(t, err)
	ymlDatabase2, err := loadExampleYaml("clickhousedatabase.yaml", map[string]string{
		"my-aiven-project": cfg.Project,
		"my-db":            dbName2,
		"my-clickhouse":    chName,
		// Remove 'databaseName' from the initial yaml
		"databaseName: example-db": "",
	})
	require.NoError(t, err)
	ymlRole1, err := loadExampleYaml("clickhouserole.yaml", map[string]string{
		"my-aiven-project": cfg.Project,
		"my-role":          roleName1,
		"my-clickhouse":    chName,
	})
	require.NoError(t, err)
	ymlRole2, err := loadExampleYaml("clickhouserole.yaml", map[string]string{
		"my-aiven-project": cfg.Project,
		"my-role":          roleName2,
		"my-clickhouse":    chName,
	})
	require.NoError(t, err)

	yml := fmt.Sprintf("%s---\n%s---\n%s---\n%s---\n%s", ymlClickhouse, ymlDatabase1, ymlDatabase2, ymlRole1, ymlRole2)
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

	// Validates ClickhouseDatabase
	db1 := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db1, dbName1))

	// Database exists
	dbAvn1, err := avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName1)
	require.NoError(t, err)

	// Gets name from `metadata.name` when `databaseName` is not set
	assert.Equal(t, dbName1, db1.ObjectMeta.Name)
	assert.Equal(t, dbAvn1.Name, db1.ObjectMeta.Name)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, db is destroyed in Aiven.
	db2 := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db2, dbName2))

	dbAvn2, err := avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName2)
	require.NoError(t, err)
	assert.Equal(t, dbName2, db2.ObjectMeta.Name)
	assert.Equal(t, dbAvn2.Name, db2.ObjectMeta.Name)

	// Calls reconciler delete
	assert.NoError(t, s.Delete(db2, func() error {
		_, err = avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName2)
		return err
	}))

	// Validates ClickhouseRole
	role1 := new(v1alpha1.ClickhouseRole)
	require.NoError(t, s.GetRunning(role1, roleName1))
	assert.Equal(t, roleName1, role1.Spec.Role)

	role2 := new(v1alpha1.ClickhouseRole)
	require.NoError(t, s.GetRunning(role2, roleName2))
	assert.Equal(t, roleName2, role2.Spec.Role)

	// Roles exist
	err = controllers.ClickhouseRoleExists(ctx, avnClient, role1)
	require.NoError(t, err)

	err = controllers.ClickhouseRoleExists(ctx, avnClient, role2)
	require.NoError(t, err)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, the role is destroyed in Aiven.
	assert.NoError(t, s.Delete(role2, func() error {
		return controllers.ClickhouseRoleExists(ctx, avnClient, role2)
	}))

	// Role 1 exists, role 2 is removed
	err = controllers.ClickhouseRoleExists(ctx, avnClient, role1)
	assert.NoError(t, err)

	err = controllers.ClickhouseRoleExists(ctx, avnClient, role2)
	assert.ErrorContains(t, err, fmt.Sprintf("ClickhouseRole %q not found", roleName2))

	// GIVEN
	// New manifest with 'databaseName' field set
	dbName3 := randName("database")
	ymlDatabase3, err := loadExampleYaml("clickhousedatabase.yaml", map[string]string{
		"name: my-db":              "name: metadata-name",
		"my-aiven-project":         cfg.Project,
		"my-clickhouse":            chName,
		"databaseName: example-db": fmt.Sprintf("databaseName: %s", dbName3),
	})

	// WHEN
	// Applies updated manifest
	require.NoError(t, s.Apply(ymlDatabase3))

	db3 := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db3, "metadata-name")) // GetRunning must be called with the metadata name

	dbAvn3, err := avnClient.ClickhouseDatabase.Get(ctx, cfg.Project, chName, dbName3)
	require.NoError(t, err)

	// THEN
	// 'databaseName' field is preferred over 'metadata.name'
	assert.NotEqual(t, dbName3, db3.ObjectMeta.Name)
	assert.NotEqual(t, dbAvn3.Name, db3.ObjectMeta.Name)
	assert.Equal(t, dbName3, db3.Spec.DatabaseName)
	assert.Equal(t, dbAvn3.Name, db3.Spec.DatabaseName)
}
