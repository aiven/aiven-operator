// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/utils"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

// TODO: use oneOf in Grantee if https://github.com/kubernetes-sigs/controller-tools/issues/461 is resolved

// Grantee represents a user or a role to which privileges or roles are granted.
type Grantee struct {
	User string `json:"user,omitempty"`
	Role string `json:"role,omitempty"`
}

// PrivilegeGrant represents the privileges to be granted to users or roles.
// See https://clickhouse.com/docs/en/sql-reference/statements/grant#granting-privilege-syntax.
// +kubebuilder:validation:XValidation:rule="!has(self.columns) || (has(self.columns) && has(self.table))",message="`table` must be set if `columns` are set"
type PrivilegeGrant struct {
	// List of grantees (users or roles) to grant the privilege to.
	// +kubebuilder:validation:MinItems=1
	Grantees []Grantee `json:"grantees"`
	// The privileges to grant, i.e. `INSERT`, `SELECT`.
	// See https://clickhouse.com/docs/en/sql-reference/statements/grant#assigning-role-syntax.
	Privileges []string `json:"privileges"`
	// The database that the grant refers to.
	Database string `json:"database"`
	// The tables that the grant refers to. To grant a privilege on all tables in a database, omit this field instead of writing `table: "*"`.
	Table string `json:"table,omitempty"`
	// The column that the grant refers to.
	Columns []string `json:"columns,omitempty"`
	// If true, then the grantee (user or role) get the permission to execute the `GRANT` query.
	// Users can grant privileges of the same scope they have and less.
	// See https://clickhouse.com/docs/en/sql-reference/statements/grant#granting-privilege-syntax
	WithGrantOption bool `json:"withGrantOption,omitempty"`
}

// RoleGrant represents the roles to be assigned to users or roles.
// See https://clickhouse.com/docs/en/sql-reference/statements/grant#assigning-role-syntax.
type RoleGrant struct {
	// List of grantees (users or roles) to grant the privilege to.
	// +kubebuilder:validation:MinItems=1
	Grantees []Grantee `json:"grantees"`
	// List of roles to grant to the grantees.
	// +kubebuilder:validation:MinItems=1
	Roles []string `json:"roles"`
	// If true, the grant is executed with `ADMIN OPTION` privilege.
	// See https://clickhouse.com/docs/en/sql-reference/statements/grant#admin-option.
	WithAdminOption bool `json:"withAdminOption,omitempty"`
}

// ClickhouseGrantSpec defines the desired state of ClickhouseGrant
type ClickhouseGrantSpec struct {
	ServiceDependant `json:",inline,omitempty"`

	// Configuration to grant a privilege. Privileges not in the manifest are revoked. Existing privileges are retained; new ones are granted.
	PrivilegeGrants []PrivilegeGrant `json:"privilegeGrants,omitempty"`
	// Configuration to grant a role. Role grants not in the manifest are revoked. Existing role grants are retained; new ones are granted.
	RoleGrants []RoleGrant `json:"roleGrants,omitempty"`
}

// ClickhouseGrantStatus defines the observed state of ClickhouseGrant
type ClickhouseGrantStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClickhouseGrant is the Schema for the ClickhouseGrants API
// Warning "Ambiguity in the `GRANT` syntax": Due to [an ambiguity](https://github.com/aiven/ospo-tracker/issues/350) in the `GRANT` syntax in Clickhouse, you should not have users and roles with the same name. It is not clear if a grant refers to the user or the role.
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
type ClickhouseGrant struct {
	metav1.TypeMeta   `json:",inline,omitempty"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClickhouseGrantSpec   `json:"spec,omitempty"`
	Status ClickhouseGrantStatus `json:"status,omitempty"`
}

func (in ClickhouseGrantSpec) buildStatements(statementType chUtils.StatementType) []string {
	stmts := make([]string, 0, len(in.PrivilegeGrants)+len(in.RoleGrants))
	for _, g := range in.PrivilegeGrants {
		stmts = append(stmts, buildStatement(statementType, g))
	}
	for _, g := range in.RoleGrants {
		stmts = append(stmts, buildStatement(statementType, g))
	}
	return stmts
}

func (in ClickhouseGrantSpec) ExecuteStatements(ctx context.Context, avnGen avngen.Client, statementType chUtils.StatementType) (bool, error) {
	statements := in.buildStatements(statementType)
	for _, stmt := range statements {
		_, err := chUtils.ExecuteClickHouseQuery(ctx, avnGen, in.Project, in.ServiceName, stmt)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (in ClickhouseGrantSpec) CollectGrantees() []string {
	allGrantees := []string{}
	processGrantee := func(grantees []Grantee) {
		for _, grantee := range grantees {
			allGrantees = append(allGrantees, userOrRole(grantee))
		}
	}
	for _, grant := range in.PrivilegeGrants {
		processGrantee(grant.Grantees)
	}
	for _, grant := range in.RoleGrants {
		processGrantee(grant.Grantees)
	}

	return utils.UniqueSliceElements(allGrantees)
}

func (in ClickhouseGrantSpec) CollectDatabases() []string {
	allDatabases := []string{}
	for _, grant := range in.PrivilegeGrants {
		if grant.Database != "" {
			allDatabases = append(allDatabases, grant.Database)
		}
	}
	return utils.UniqueSliceElements(allDatabases)
}

func (in ClickhouseGrantSpec) CollectTables() []chUtils.DatabaseAndTable {
	allTables := []chUtils.DatabaseAndTable{}
	for _, grant := range in.PrivilegeGrants {
		if grant.Table != "" {
			allTables = append(allTables, chUtils.DatabaseAndTable{Database: grant.Database, Table: grant.Table})
		}
	}
	return utils.UniqueSliceElements(allTables)
}

func (in *ClickhouseGrant) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ClickhouseGrant) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ClickhouseGrant) NoSecret() bool {
	return true
}

var _ AivenManagedObject = &ClickhouseGrant{}

//+kubebuilder:object:root=true

// ClickhouseGrantList contains a list of ClickhouseGrant
type ClickhouseGrantList struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClickhouseGrant `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ClickhouseGrant{}, &ClickhouseGrantList{})
}

// Takes a slice of PrivilegeGrant and returns a new slice
// where each grantee has its own PrivilegeGrant entry.
func FlattenPrivilegeGrants(grants []PrivilegeGrant) []PrivilegeGrant {
	var flattened []PrivilegeGrant
	for _, grant := range grants {
		for _, grantee := range grant.Grantees {
			newGrant := PrivilegeGrant{
				Grantees:        []Grantee{grantee},
				Privileges:      grant.Privileges,
				Database:        grant.Database,
				Table:           grant.Table,
				Columns:         grant.Columns,
				WithGrantOption: grant.WithGrantOption,
			}
			flattened = append(flattened, newGrant)
		}
	}
	return flattened
}

// Takes a slice of RoleGrant and returns a new slice
// where each grantee has its own RoleGrant entry.
func FlattenRoleGrants(grants []RoleGrant) []RoleGrant {
	var flattened []RoleGrant
	for _, grant := range grants {
		for _, grantee := range grant.Grantees {
			newGrant := RoleGrant{
				Grantees:        []Grantee{grantee},
				Roles:           grant.Roles,
				WithAdminOption: grant.WithAdminOption,
			}
			flattened = append(flattened, newGrant)
		}
	}
	return flattened
}

func (g PrivilegeGrant) ConstructParts(t chUtils.StatementType) (string, string, string) {
	privilegesPart := constructPrivilegesPart(g)
	granteesPart := constructGranteePart(g.Grantees)
	options := ""
	if g.WithGrantOption && t == chUtils.GRANT {
		options = "WITH GRANT OPTION"
	}
	return privilegesPart, granteesPart, options
}

// Helper function to construct the privileges part of the statement
func constructPrivilegesPart(g PrivilegeGrant) string {
	privileges := make([]string, 0, len(g.Privileges))
	for _, privilege := range g.Privileges {
		if (privilege == "SELECT" || privilege == "INSERT") && len(g.Columns) > 0 {
			columnList := strings.Join(utils.MapSlice(g.Columns, escape), ", ")
			privileges = append(privileges, fmt.Sprintf("%s(%s)", privilege, columnList))
		} else {
			privileges = append(privileges, privilege)
		}
	}
	return strings.Join(privileges, ", ")
}

func (g RoleGrant) ConstructParts(t chUtils.StatementType) (string, string, string) {
	rolesPart := strings.Join(g.Roles, ", ")
	granteesPart := constructGranteePart(g.Grantees)
	options := ""
	if g.WithAdminOption && t == chUtils.GRANT {
		options = "WITH ADMIN OPTION"
	}
	return rolesPart, granteesPart, options
}

func ExecuteGrant[T chUtils.Grant](ctx context.Context, avnGen avngen.Client, t chUtils.StatementType, grant T, projectName string, serviceName string) error {
	stmt := buildStatement(t, grant)
	_, err := chUtils.ExecuteClickHouseQuery(ctx, avnGen, projectName, serviceName, stmt)
	if err != nil {
		return err
	}
	return nil
}

// Generates a ClickHouse GRANT or REVOKE statement for privilege or role grants. See https://clickhouse.com/docs/en/sql-reference/statements/grant.
func buildStatement[T chUtils.Grant](t chUtils.StatementType, grant T) string {
	mainPart, granteesPart, options := grant.ConstructParts(t)

	fmtString := ""
	switch t {
	case chUtils.GRANT:
		fmtString = "GRANT %s TO %s %s"
	case chUtils.REVOKE:
		fmtString = "REVOKE %s FROM %s %s"
	}

	// Adjust the format string based on the type of grant (PrivilegeGrant needs "ON %s" part)
	if p, ok := any(grant).(PrivilegeGrant); ok {
		// ON part is constructed only for PrivilegeGrant
		onPart := constructOnPart(p)
		if t == chUtils.GRANT {
			fmtString = strings.Replace(fmtString, "TO", "ON %s TO", 1)
		} else {
			fmtString = strings.Replace(fmtString, "FROM", "ON %s FROM", 1)
		}
		return fmt.Sprintf(fmtString, mainPart, onPart, granteesPart, options)
	}

	return fmt.Sprintf(fmtString, mainPart, granteesPart, options)
}

// Helper function to construct the ON part of the statement
func constructOnPart(grant PrivilegeGrant) string {
	switch {
	case grant.Table != "":
		return escape(grant.Database) + "." + escape(grant.Table)
	default:
		return escape(grant.Database) + ".*"
	}
}

// Helper function to construct the TO part of the statement
func constructGranteePart(grantees []Grantee) string {
	return strings.Join(utils.MapSlice(grantees, granteeToString), ", ")
}

// Converts a grantee to an escaped string
func granteeToString(grantee Grantee) string {
	return escape(userOrRole(grantee))
}

func userOrRole(grantee Grantee) string {
	if grantee.User != "" {
		return grantee.User
	}
	return grantee.Role
}

// Escapes database identifiers like table or column names
func escape(identifier string) string {
	// See https://github.com/ClickHouse/clickhouse-go/blob/8ad6ec6b95d8b0c96d00115bc2d69ff13083f94b/lib/column/column.go#L32
	replacer := strings.NewReplacer("`", "\\`", "\\", "\\\\")
	return "`" + replacer.Replace(identifier) + "`"
}
