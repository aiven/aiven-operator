package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"slices"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

func chConnFromSecret(ctx context.Context, secret *corev1.Secret) (clickhouse.Conn, error) {
	c, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", string(secret.Data["HOST"]), string(secret.Data["PORT"]))},
		Auth: clickhouse.Auth{
			Username: string(secret.Data["CLICKHOUSE_USER"]),
			Password: string(secret.Data["CLICKHOUSE_PASSWORD"]),
		},
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	return c, err
}

func getClickhouseGrantYaml(project, chName, cloudName, dbName, userName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-16
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
  serviceName: %[2]s
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %[5]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseRole
metadata:
  name: writer
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  role: writer
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseGrant
metadata:
  name: test-grant
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s

  privilegeGrants:
    - grantees:
        - user: %[5]s
        - role: writer
      privileges:
        - SELECT
        - INSERT
      database: %[4]s
      table: example-table
      columns:
        - col1
      withGrantOption: true

  roleGrants:
    - roles:
        - writer
      grantees:
        - user: %[5]s
      withAdminOption: true
`, project, chName, cloudName, dbName, userName)
}

func TestClickhouseGrant(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse")
	userName := "clickhouse-user"
	dbName := "clickhouse-db"

	yml := getClickhouseGrantYaml(cfg.Project, chName, cfg.PrimaryCloudName, dbName, userName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	// user := new(v1alpha1.ClickhouseUser)
	db := new(v1alpha1.ClickhouseDatabase)
	grant := new(v1alpha1.ClickhouseGrant)

	require.NoError(t, s.GetRunning(ch, chName))
	// require.NoError(t, s.GetRunning(user, userName))
	require.NoError(t, s.GetRunning(db, dbName))

	// Constructs connection to ClickHouse from service secret
	secret, err := s.GetSecret(ch.GetName())
	require.NoError(t, err, "Failed to get secret")
	conn, err := chConnFromSecret(ctx, secret)
	require.NoError(t, err, "failed to connect to ClickHouse")

	// THEN
	// Initially grant isn't created because the table that it is targeting doesn't exist
	err = s.GetRunning(grant, "test-grant")
	require.ErrorContains(t, err, "unable to wait for preconditions: missing tables defined in spec: [{Database:clickhouse-db Table:example-table}]")

	// Create example-table in clickhouse-db
	createTableQuery := fmt.Sprintf("CREATE TABLE `%s`.`example-table` (col1 String, col2 Int32) ENGINE = MergeTree() ORDER BY col1", dbName)
	_, err = conn.Query(ctx, createTableQuery)
	require.NoError(t, err)

	// Clear conditions to stop erroring out in GetRunning. The condition is also removed in ClickhouseGrant
	// checkPreconditions but due to timing issues the tests may fail if we don't remove it here.
	meta.RemoveStatusCondition(grant.Conditions(), "Error")
	errStatus := k8sClient.Status().Update(ctx, grant)
	require.NoError(t, errStatus)

	grant = new(v1alpha1.ClickhouseGrant)
	// ... and wait for the grant to be created and running
	require.NoError(t, s.GetRunning(grant, "test-grant"))

	// THEN
	// Query and collect ClickhouseGrant results
	results, err := queryAndCollectResults[ClickhouseGrant](ctx, conn, chUtils.QueryNonAivenPrivileges)
	require.NoError(t, err)

	filteredResults := filterPrivilegeGrantResults(results)
	assert.Len(t, filteredResults, 4)
	assert.ElementsMatch(t, filteredResults, expectedPrivilegeGrants)

	// Query and collect ClickhouseRoleGrant results
	roleGrantResults, err := queryAndCollectResults[ClickhouseRoleGrant](ctx, conn, chUtils.RoleGrantsQuery)
	require.NoError(t, err)

	// Override GrantedRoleId to nil, it changes between test runs
	for i := range roleGrantResults {
		roleGrantResults[i].GrantedRoleId = nil
	}

	assert.Len(t, roleGrantResults, 1)
	assert.ElementsMatch(t, roleGrantResults, expectedRoleGrants)
}

func queryAndCollectResults[T any](ctx context.Context, conn clickhouse.Conn, query string) ([]T, error) {
	var results []T
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result T
		err := rows.ScanStruct(&result)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// Removes Aiven roles from the results
func filterPrivilegeGrantResults(results []ClickhouseGrant) []ClickhouseGrant {
	var filteredResults []ClickhouseGrant
	for _, r := range results {
		isAivenUser := r.UserName != nil && slices.Contains(chUtils.InternalAivenRoles, *r.UserName)
		isRole := r.UserName == nil && r.RoleName != nil
		if isRole || !isAivenUser {
			filteredResults = append(filteredResults, r)
		}
	}
	return filteredResults
}

type ClickhouseGrant struct {
	UserName        *string `ch:"user_name"`
	RoleName        *string `ch:"role_name"`
	AccessType      string  `ch:"access_type"`
	Database        *string `ch:"database"`
	Table           *string `ch:"table"`
	Column          *string `ch:"column"`
	IsPartialRevoke bool    `ch:"is_partial_revoke"`
	GrantOption     bool    `ch:"grant_option"`
}

/**
 * Expected privilege grants are constructed from this part of the manifest:
 *
 * privilegeGrants:
 *   - grantees:
 *       - user: %[5]s
 *       - role: writer
 *     privileges:
 *       - SELECT
 *       - INSERT
 *     database: %[4]s
 *     table: example-table
 *     columns:
 *       - col1
 *     withGrantOption: true
 */
var expectedPrivilegeGrants = []ClickhouseGrant{
	{
		UserName:        ptr("clickhouse-user"),
		RoleName:        nil,
		AccessType:      "SELECT",
		Database:        ptr("clickhouse-db"),
		Table:           ptr("example-table"),
		Column:          ptr("col1"),
		IsPartialRevoke: false,
		GrantOption:     true,
	},
	{
		UserName:        ptr("clickhouse-user"),
		RoleName:        nil,
		AccessType:      "INSERT",
		Database:        ptr("clickhouse-db"),
		Table:           ptr("example-table"),
		Column:          ptr("col1"),
		IsPartialRevoke: false,
		GrantOption:     true,
	},
	{
		UserName:        nil,
		RoleName:        ptr("writer"),
		AccessType:      "SELECT",
		Database:        ptr("clickhouse-db"),
		Table:           ptr("example-table"),
		Column:          ptr("col1"),
		IsPartialRevoke: false,
		GrantOption:     true,
	},
	{
		UserName:        nil,
		RoleName:        ptr("writer"),
		AccessType:      "INSERT",
		Database:        ptr("clickhouse-db"),
		Table:           ptr("example-table"),
		Column:          ptr("col1"),
		IsPartialRevoke: false,
		GrantOption:     true,
	},
}

type ClickhouseRoleGrant struct {
	UserName             *string `ch:"user_name"`
	RoleName             *string `ch:"role_name"`
	GrantedRoleName      *string `ch:"granted_role_name"`
	GrantedRoleId        *string `ch:"granted_role_id"`
	GrantedRoleIsDefault bool    `ch:"granted_role_is_default"`
	WithAdminOption      bool    `ch:"with_admin_option"`
}

/**
 * Expected role grants are constructed from this part of the manifest:
 *
 * roleGrants:
 *   - roles:
 *       - writer
 *     grantees:
 *       - user: %[5]s
 *     withAdminOption: true
 */
var expectedRoleGrants = []ClickhouseRoleGrant{
	{
		UserName:             ptr("clickhouse-user"),
		RoleName:             nil,
		GrantedRoleName:      ptr("writer"),
		GrantedRoleId:        nil, // Not actually nil, changes between test runs. We override this in the test.
		GrantedRoleIsDefault: true,
		WithAdminOption:      true,
	},
}

func ptr(s string) *string { return &s }
