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

	"github.com/aiven/aiven-operator/api/v1alpha1"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

func chConnFromSecret(secret *corev1.Secret) (clickhouse.Conn, error) {
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
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ch := new(v1alpha1.Clickhouse)
	db := new(v1alpha1.ClickhouseDatabase)

	require.NoError(t, s.GetRunning(ch, chName))
	require.NoError(t, s.GetRunning(db, dbName))

	// Constructs connection to ClickHouse from service secret
	secret, err := s.GetSecret(ch.GetName())
	require.NoError(t, err, "Failed to get secret")
	conn, err := chConnFromSecret(secret)
	require.NoError(t, err, "failed to connect to ClickHouse")

	// THEN
	grant := new(v1alpha1.ClickhouseGrant)
	require.NoError(t, s.GetRunning(grant, "test-grant"))

	// Query and collect ClickhouseGrant results
	results, err := queryAndCollectResults[ClickhouseGrant](ctx, conn, chUtils.QueryNonAivenPrivileges)
	require.NoError(t, err)

	filteredResults := filterPrivilegeGrantResults(results)
	assert.Len(t, filteredResults, 4)
	assert.ElementsMatch(t, filteredResults, expectedPrivilegeGrants)

	// Query and collect ClickhouseRoleGrant results
	roleGrantResults, err := queryAndCollectResults[ClickhouseRoleGrant](ctx, conn, chUtils.RoleGrantsQuery)
	require.NoError(t, err)

	// Override GrantedRoleID to nil, it changes between test runs
	for i := range roleGrantResults {
		roleGrantResults[i].GrantedRoleID = nil
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
	GrantedRoleID        *string `ch:"granted_role_id"`
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
		GrantedRoleID:        nil, // Not actually nil, changes between test runs. We override this in the test.
		GrantedRoleIsDefault: true,
		WithAdminOption:      true,
	},
}

func ptr(s string) *string { return &s }

func fromPtr[T any](v *T) T {
	if v == nil {
		var empty T
		return empty
	}
	return *v
}

func clickhouseGrantExampleExtra(project, serviceName, db, user, role, grant string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
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
kind: ClickhouseRole
metadata:
  name: writer
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  role: %[5]s
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseGrant
metadata:
  name: %[6]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  privilegeGrants:
    - grantees:
        - user: %[4]s
        - role: %[5]s
      privileges:
        - SELECT
        - INSERT
      database: %[3]s
      withGrantOption: true
  roleGrants:
    - roles:
        - %[5]s
      grantees:
        - user: %[4]s
      withAdminOption: true
`, project, serviceName, db, user, role, grant)
}

func TestClickhouseGrantExample(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	chName := randName("clickhouse-service")
	dbName := randName("clickhouse-db")
	userName := randName("clickhouse-user")
	roleName := randName("clickhouse-role")
	grantName := randName("clickhouse-grant")

	yml, err := loadExampleYaml("clickhousegrant.yaml", map[string]string{
		"doc[0].metadata.name":                            grantName,
		"doc[0].spec.project":                             cfg.Project,
		"doc[0].spec.serviceName":                         chName,
		"doc[0].spec.privilegeGrants[0].database":         dbName,
		"doc[0].spec.roleGrants[0].grantees[0].user":      userName,
		"doc[0].spec.privilegeGrants[0].grantees[0].role": roleName,
		"doc[0].spec.roleGrants[0].roles[0]":              roleName,

		"doc[1].metadata.name":  chName,
		"doc[1].spec.cloudName": cfg.PrimaryCloudName,
		"doc[1].spec.project":   cfg.Project,

		"doc[2].metadata.name":    dbName,
		"doc[2].spec.project":     cfg.Project,
		"doc[2].spec.serviceName": chName,

		"doc[3].metadata.name":    userName,
		"doc[3].spec.project":     cfg.Project,
		"doc[3].spec.serviceName": chName,

		"doc[4].metadata.name":    roleName,
		"doc[4].spec.project":     cfg.Project,
		"doc[4].spec.role":        roleName,
		"doc[4].spec.serviceName": chName,
	})
	require.NoError(t, err)

	extraUserName := randName("clickhouse-extra-user")
	extraRoleName := randName("clickhouse-extra-role")
	extraGrantName := randName("clickhouse-extra-grant")
	extraYml := clickhouseGrantExampleExtra(cfg.Project, chName, dbName, extraUserName, extraRoleName, extraGrantName)

	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	allYml := fmt.Sprintf("%s\n---\n%s", yml, extraYml)
	require.NoError(t, s.Apply(allYml))

	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, chName))

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	db := new(v1alpha1.ClickhouseDatabase)
	require.NoError(t, s.GetRunning(db, dbName))

	role := new(v1alpha1.ClickhouseRole)
	require.NoError(t, s.GetRunning(role, roleName))

	grant := new(v1alpha1.ClickhouseGrant)
	require.NoError(t, s.GetRunning(grant, grantName))

	// Creates connection
	secret, err := s.GetSecret(ch.GetName())
	require.NoError(t, err, "Failed to get secret")

	conn, err := chConnFromSecret(secret)
	require.NoError(t, err, "failed to connect to ClickHouse")

	results, err := queryAndCollectResults[ClickhouseGrant](ctx, conn, chUtils.QueryNonAivenPrivileges)
	require.NoError(t, err)

	// Validates the state
	assert.Equal(t, grant.Spec.Grants, *grant.Status.State)

	// Privileges validation
	expected := map[string]bool{
		fmt.Sprintf("%s/%s/INSERT", dbName, roleName):       true,
		fmt.Sprintf("%s/%s/SELECT", dbName, roleName):       true,
		fmt.Sprintf("%s/%s/CREATE TABLE", dbName, roleName): true,
		fmt.Sprintf("%s/%s/CREATE VIEW", dbName, roleName):  true,
	}

	// Finds and removes grants from the expected list
	for _, r := range results {
		key := fmt.Sprintf("%s/%s/%s", fromPtr(r.Database), fromPtr(r.RoleName), r.AccessType)
		delete(expected, key)
	}

	// Nothing left == all found
	assert.Empty(t, expected)

	// Validates concurrency
	extraGrant := new(v1alpha1.ClickhouseGrant)
	require.NoError(t, s.GetRunning(extraGrant, grantName))
}
