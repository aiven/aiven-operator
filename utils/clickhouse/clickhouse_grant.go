package chUtils

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
)

type StatementType string

const (
	GRANT  StatementType = "GRANT"
	REVOKE StatementType = "REVOKE"
)

// TODO: Move to clickhousegrant_types.go once the issue below is resolved
// See: https://github.com/kubernetes-sigs/controller-tools/issues/383
type Grant interface {
	// Returns the 1. main part (privileges or roles), 2. grantees part and 3. query options
	ConstructParts(t StatementType) (string, string, string)
}

var (
	// 'aiven', 'aiven_monitoring', and 'avnadmin' are Aiven-managed users
	InternalAivenRoles = []string{"aiven", "aiven_monitoring", "avnadmin"}
	// "OR user_name IS NULL" is needed to include roles in the result
	QueryNonAivenPrivileges = fmt.Sprintf("SELECT * FROM system.grants WHERE user_name NOT IN('%s') OR user_name IS NULL", strings.Join(InternalAivenRoles, "', '"))
	queryNonAivenUsers      = fmt.Sprintf("SELECT name FROM system.users WHERE name NOT IN('%s')", strings.Join(InternalAivenRoles, "', '"))

	internalDatabases = []string{"default", "INFORMATION_SCHEMA", "information_schema", "system"}
	queryAllDatabases = fmt.Sprintf("SELECT name FROM system.databases WHERE name NOT IN('%s')", strings.Join(internalDatabases, "', '"))
	queryAllTables    = fmt.Sprintf("SELECT database, name FROM system.tables WHERE database NOT IN('%s')", strings.Join(internalDatabases, "', '"))
)

const (
	rolesQuery      = "SELECT name FROM system.roles"
	RoleGrantsQuery = "SELECT * FROM system.role_grants"
)

func QueryPrivileges(ctx context.Context, avnGen avngen.Client, projectName, serviceName string) (*clickhouse.ServiceClickHouseQueryOut, error) {
	res, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, QueryNonAivenPrivileges)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func QueryRoleGrants(ctx context.Context, avnGen avngen.Client, projectName, serviceName string) (*clickhouse.ServiceClickHouseQueryOut, error) {
	res, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, RoleGrantsQuery)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func QueryGrantees(ctx context.Context, avnGen avngen.Client, projectName, serviceName string) ([]string, error) {
	resUsers, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, queryNonAivenUsers)
	if err != nil {
		return nil, err
	}
	resRoles, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, rolesQuery)
	if err != nil {
		return nil, err
	}

	users := extractColumnValues(resUsers.Data, 0)
	roles := extractColumnValues(resRoles.Data, 0)

	return append(users, roles...), nil
}

func QueryDatabases(ctx context.Context, avnGen avngen.Client, projectName, serviceName string) ([]string, error) {
	resDatabases, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, queryAllDatabases)
	if err != nil {
		return nil, err
	}

	databases := extractColumnValues(resDatabases.Data, 0)

	return databases, nil
}

type DatabaseAndTable struct {
	Database string
	Table    string
}

func QueryTables(ctx context.Context, avnGen avngen.Client, projectName, serviceName string) ([]DatabaseAndTable, error) {
	resDatabases, err := ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, queryAllTables)
	if err != nil {
		return nil, err
	}

	databases := extractColumnValues(resDatabases.Data, 0)
	tables := extractColumnValues(resDatabases.Data, 1)

	databaseAndTables := make([]DatabaseAndTable, 0, len(databases))
	for i := range len(databases) {
		databaseAndTables = append(databaseAndTables, DatabaseAndTable{
			Database: databases[i],
			Table:    tables[i],
		})
	}

	return databaseAndTables, nil
}

// Helper function to extract column values from a nested array
func extractColumnValues(data [][]interface{}, columnIndex int) []string {
	values := make([]string, 0, len(data))
	for _, row := range data {
		if value, ok := row[columnIndex].(string); ok && value != "" {
			values = append(values, value)
		}
	}
	return values
}
