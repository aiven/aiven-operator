// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/utils"
	chUtils "github.com/aiven/aiven-operator/utils/clickhouse"
)

// ClickhouseGrantReconciler reconciles a ClickhouseGrant object
type ClickhouseGrantReconciler struct {
	Controller
}

func newClickhouseGrantReconciler(c Controller) reconcilerType {
	return &ClickhouseGrantReconciler{Controller: c}
}

// ClickhouseGrantHandler handles an Aiven ClickhouseGrant
type ClickhouseGrantHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=clickhousegrants/finalizers,verbs=get;create;update

func (r *ClickhouseGrantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &ClickhouseGrantHandler{}, &v1alpha1.ClickhouseGrant{})
}

func (r *ClickhouseGrantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseGrant{}).
		Complete(r)
}

func (h *ClickhouseGrantHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	g, err := h.convert(obj)
	if err != nil {
		return err
	}

	flatPrivilegeGrants := v1alpha1.FlattenPrivilegeGrants(g.Spec.PrivilegeGrants)
	flatRoleGrants := v1alpha1.FlattenRoleGrants(g.Spec.RoleGrants)
	apiPrivilegeGrants, apiRoleGrants, err := getGrantsFromApi(ctx, avnGen, g)
	if err != nil {
		return err
	}

	privilegeGrantsToRevoke, _, roleGrantsToRevoke, _ := diffClickhouseGrantSpecToApi(flatPrivilegeGrants, apiPrivilegeGrants, flatRoleGrants, apiRoleGrants)

	// Issue revoke grant statements for privilege and role grants not found in the spec
	projectName := g.Spec.Project
	serviceName := g.Spec.ServiceName
	for _, grantToRevoke := range privilegeGrantsToRevoke {
		v1alpha1.ExecuteGrant(ctx, avnGen, chUtils.REVOKE, grantToRevoke, projectName, serviceName) //nolint:errcheck
	}
	for _, grantToRevoke := range roleGrantsToRevoke {
		v1alpha1.ExecuteGrant(ctx, avnGen, chUtils.REVOKE, grantToRevoke, projectName, serviceName) //nolint:errcheck
	}

	_, err = g.Spec.ExecuteStatements(ctx, avnGen, chUtils.GRANT)
	if err != nil {
		return err
	}

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	metav1.SetMetaDataAnnotation(&g.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(g.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *ClickhouseGrantHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	g, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	_, err = g.Spec.ExecuteStatements(ctx, avnGen, chUtils.REVOKE)
	if isAivenError(err, http.StatusBadRequest) {
		// "not found in user directories", "There is no role", etc
		return true, nil
	}

	return isDeleted(err)
}

func (h *ClickhouseGrantHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	g, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	flatPrivilegeGrants := v1alpha1.FlattenPrivilegeGrants(g.Spec.PrivilegeGrants)
	flatRoleGrants := v1alpha1.FlattenRoleGrants(g.Spec.RoleGrants)
	apiPrivilegeGrants, apiRoleGrants, err := getGrantsFromApi(ctx, avnGen, g)
	if err != nil {
		return nil, err
	}

	_, privilegeGrantsToAd, _, roleGrantsToAdd := diffClickhouseGrantSpecToApi(flatPrivilegeGrants, apiPrivilegeGrants, flatRoleGrants, apiRoleGrants)

	if len(privilegeGrantsToAd) > 0 || len(roleGrantsToAdd) > 0 {
		return nil, fmt.Errorf("missing grants defined in spec: %+v (privilege grants) %+v (role grants)", privilegeGrantsToAd, roleGrantsToAdd)
	}

	meta.SetStatusCondition(&g.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&g.ObjectMeta, instanceIsRunningAnnotation, "true")
	return nil, nil
}

func getGrantsFromApi(ctx context.Context, avnGen avngen.Client, g *v1alpha1.ClickhouseGrant) ([]v1alpha1.PrivilegeGrant, []v1alpha1.RoleGrant, error) {
	privileges, err := chUtils.QueryPrivileges(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if err != nil {
		return nil, nil, err
	}
	roleGrants, err := chUtils.QueryRoleGrants(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if err != nil {
		return nil, nil, err
	}

	apiPrivilegeGrants, err := processGrantsFromApiResponse(privileges, PrivilegeGrantType, convertPrivilegeGrantFromApiStruct)
	if err != nil {
		return nil, nil, err
	}
	apiRoleGrants, err := processGrantsFromApiResponse(roleGrants, RoleGrantType, convertRoleGrantFromApiStruct)
	if err != nil {
		return nil, nil, err
	}
	return apiPrivilegeGrants, apiRoleGrants, nil
}

func diffClickhouseGrantSpecToApi(specPrivilegeGrants []v1alpha1.PrivilegeGrant, apiPrivilegeGrants []v1alpha1.PrivilegeGrant, specRoleGrants []v1alpha1.RoleGrant, apiRoleGrants []v1alpha1.RoleGrant) ([]v1alpha1.PrivilegeGrant, []v1alpha1.PrivilegeGrant, []v1alpha1.RoleGrant, []v1alpha1.RoleGrant) {
	var privilegeGrantsToRevoke, privilegeGrantsToAdd []v1alpha1.PrivilegeGrant
	var roleGrantsToRevoke, roleGrantsToAdd []v1alpha1.RoleGrant

	for _, apiGrant := range apiPrivilegeGrants {
		if !containsPrivilegeGrant(specPrivilegeGrants, apiGrant) {
			privilegeGrantsToRevoke = append(privilegeGrantsToRevoke, apiGrant)
		}
	}

	for _, specGrant := range specPrivilegeGrants {
		if !containsPrivilegeGrant(apiPrivilegeGrants, specGrant) {
			privilegeGrantsToAdd = append(privilegeGrantsToAdd, specGrant)
		}
	}

	for _, apiGrant := range apiRoleGrants {
		if !containsRoleGrant(specRoleGrants, apiGrant) {
			roleGrantsToRevoke = append(roleGrantsToRevoke, apiGrant)
		}
	}

	for _, specGrant := range specRoleGrants {
		if !containsRoleGrant(apiRoleGrants, specGrant) {
			roleGrantsToAdd = append(roleGrantsToAdd, specGrant)
		}
	}

	return privilegeGrantsToRevoke, privilegeGrantsToAdd, roleGrantsToRevoke, roleGrantsToAdd
}

func containsPrivilegeGrant(grants []v1alpha1.PrivilegeGrant, grant chUtils.Grant) bool {
	for _, g := range grants {
		if cmp.Equal(g, grant) {
			return true
		}
	}
	return false
}

func containsRoleGrant(grants []v1alpha1.RoleGrant, grant v1alpha1.RoleGrant) bool {
	for _, g := range grants {
		if cmp.Equal(g, grant) {
			return true
		}
	}
	return false
}

func (h *ClickhouseGrantHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	/** Preconditions for ClickhouseGrant:
	 *
	 * 1. The service is running
	 * 2. All users and roles specified in spec exist
	 * 3. All databases specified in spec exist
	 * 4. All tables specified in spec exist
	 */

	g, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	serviceIsRunning, err := checkServiceIsRunning(ctx, avn, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if !serviceIsRunning || err != nil {
		return false, err
	}

	// Service is running, check users and roles specified in spec exist
	_, err = checkPrecondition(ctx, g, avnGen, g.Spec.CollectGrantees, chUtils.QueryGrantees, "missing users or roles defined in spec: %v")
	if err != nil {
		return false, err
	}

	// Check that databases specified in spec exist
	_, err = checkPrecondition(ctx, g, avnGen, g.Spec.CollectDatabases, chUtils.QueryDatabases, "missing databases defined in spec: %v")
	if err != nil {
		return false, err
	}

	// Check that tables specified in spec exist
	_, err = checkPrecondition(ctx, g, avnGen, g.Spec.CollectTables, chUtils.QueryTables, "missing tables defined in spec: %+v")
	if err != nil {
		return false, err
	}

	// Remove previous error conditions
	meta.RemoveStatusCondition(&g.Status.Conditions, "Error")

	meta.SetStatusCondition(&g.Status.Conditions,
		getInitializedCondition("Preconditions", "Preconditions met"))

	return true, nil
}

func checkPrecondition[T comparable](ctx context.Context, g *v1alpha1.ClickhouseGrant, avnGen avngen.Client, collectFunc func() []T, queryFunc func(context.Context, avngen.Client, string, string) ([]T, error), errorMsgFormat string) (bool, error) {
	itemsInSpec := collectFunc()
	itemsInDb, err := queryFunc(ctx, avnGen, g.Spec.Project, g.Spec.ServiceName)
	if err != nil {
		return false, err
	}
	missingItems := utils.CheckSliceContainment(itemsInSpec, itemsInDb)
	if len(missingItems) > 0 {
		err = fmt.Errorf(errorMsgFormat, missingItems)
		meta.SetStatusCondition(&g.Status.Conditions, getErrorCondition(errConditionPreconditions, err))
		return false, err
	}
	return true, nil
}

func (h *ClickhouseGrantHandler) convert(i client.Object) (*v1alpha1.ClickhouseGrant, error) {
	g, ok := i.(*v1alpha1.ClickhouseGrant)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ClickhouseGrant")
	}

	return g, nil
}

type ApiPrivilegeGrant struct {
	Grantee         v1alpha1.Grantee
	Privilege       string
	Database        string
	Table           string
	Column          string
	WithGrantOption bool
}

type ApiRoleGrant struct {
	Grantee         v1alpha1.Grantee
	Role            string
	WithAdminOption bool
}

type GrantType string

const (
	PrivilegeGrantType GrantType = "PrivilegeGrant"
	RoleGrantType      GrantType = "RoleGrant"
)

// GrantColumns defines the columns required for each grant type
var GrantColumns = map[GrantType][]string{
	PrivilegeGrantType: {"user_name", "role_name", "access_type", "database", "table", "column", "is_partial_revoke", "grant_option"},
	RoleGrantType:      {"user_name", "role_name", "granted_role_name", "with_admin_option"},
}

// Process either privilege or role grants from the API response
func processGrantsFromApiResponse[U any, T any](r *clickhouse.ServiceClickHouseQueryOut, grantType GrantType, processFn func([]U) []T) ([]T, error) {
	requiredColumns := GrantColumns[grantType]
	columnNameMap, err := validateColumns(r.Meta, requiredColumns)
	if err != nil {
		return nil, err
	}

	var grants []U
	for _, dataRow := range r.Data {
		grant, err := extractGrant(dataRow, columnNameMap, grantType)
		if err != nil {
			return nil, err
		}
		g, ok := grant.(U)
		if !ok {
			return nil, fmt.Errorf("failed to convert grant to type %T", grant)
		}
		grants = append(grants, g)
	}

	return processFn(grants), nil
}

// validateColumns checks if all required columns are present in the metadata
func validateColumns(meta []clickhouse.MetaOut, requiredColumns []string) (map[string]int, error) {
	columnNameMap := make(map[string]int)
	for i, md := range meta {
		columnNameMap[md.Name] = i
	}
	for _, columnName := range requiredColumns {
		if _, ok := columnNameMap[columnName]; !ok {
			return nil, fmt.Errorf("'system.grants' metadata is missing the '%s' column", columnName)
		}
	}
	return columnNameMap, nil
}

func extractGrant(dataRow []interface{}, columnNameMap map[string]int, grantType GrantType) (interface{}, error) {
	switch grantType {
	case PrivilegeGrantType:
		grant := ApiPrivilegeGrant{
			Grantee: v1alpha1.Grantee{
				User: getString(dataRow, columnNameMap, "user_name"),
				Role: getString(dataRow, columnNameMap, "role_name"),
			},
			Privilege:       getString(dataRow, columnNameMap, "access_type"),
			Database:        getString(dataRow, columnNameMap, "database"),
			Table:           getString(dataRow, columnNameMap, "table"),
			Column:          getString(dataRow, columnNameMap, "column"),
			WithGrantOption: getBool(dataRow, columnNameMap, "grant_option"),
		}
		return grant, nil
	case RoleGrantType:
		grant := ApiRoleGrant{
			Grantee: v1alpha1.Grantee{
				User: getString(dataRow, columnNameMap, "user_name"),
				Role: getString(dataRow, columnNameMap, "role_name"),
			},
			Role:            getString(dataRow, columnNameMap, "granted_role_name"),
			WithAdminOption: getBool(dataRow, columnNameMap, "with_admin_option"),
		}
		return grant, nil
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", grantType)
	}
}

// getString and getBool are helper functions to extract string and boolean values from dataRow based on columnNameMap
func getString(dataRow []interface{}, columnNameMap map[string]int, columnName string) string {
	if index, ok := columnNameMap[columnName]; ok && index < len(dataRow) {
		if value, ok := dataRow[index].(string); ok {
			return value
		}
	}
	return ""
}

func getBool(dataRow []interface{}, columnNameMap map[string]int, columnName string) bool {
	if index, ok := columnNameMap[columnName]; ok && index < len(dataRow) {
		return dataRow[index] != float64(0)
	}
	return false
}

func convertPrivilegeGrantFromApiStruct(clickhouseGrants []ApiPrivilegeGrant) []v1alpha1.PrivilegeGrant {
	grantMap := make(map[string]*v1alpha1.PrivilegeGrant)
	for _, chGrant := range clickhouseGrants {
		key := chGrant.Grantee.User + chGrant.Grantee.Role + chGrant.Database + chGrant.Table
		if grant, exists := grantMap[key]; exists {
			// If the grant already exists, append the privilege and column if not empty.
			grant.Privileges = appendUnique(grant.Privileges, chGrant.Privilege)
			if chGrant.Column != "" {
				grant.Columns = appendUnique(grant.Columns, chGrant.Column)
			}
		} else {
			// Create a new PrivilegeGrant if it does not exist.
			newGrant := v1alpha1.PrivilegeGrant{
				Grantees:        []v1alpha1.Grantee{{User: chGrant.Grantee.User, Role: chGrant.Grantee.Role}},
				Privileges:      []string{chGrant.Privilege},
				Database:        chGrant.Database,
				Table:           chGrant.Table,
				Columns:         nil,
				WithGrantOption: chGrant.WithGrantOption,
			}
			if chGrant.Column != "" {
				newGrant.Columns = append(newGrant.Columns, chGrant.Column)
			}
			grantMap[key] = &newGrant
		}
	}

	// Extract the values from the map to a slice
	var privilegeGrants []v1alpha1.PrivilegeGrant
	for _, grant := range grantMap {
		privilegeGrants = append(privilegeGrants, *grant)
	}

	return privilegeGrants
}

func appendUnique(slice []string, item string) []string {
	for _, elem := range slice {
		if elem == item {
			return slice
		}
	}
	return append(slice, item)
}

func convertRoleGrantFromApiStruct(clickhouseRoleGrants []ApiRoleGrant) []v1alpha1.RoleGrant {
	grantMap := make(map[string]*v1alpha1.RoleGrant)

	for _, chRoleGrant := range clickhouseRoleGrants {
		key := chRoleGrant.Grantee.User + chRoleGrant.Grantee.Role
		if grant, exists := grantMap[key]; exists {
			if !slices.Contains(grant.Roles, chRoleGrant.Role) {
				grant.Roles = append(grant.Roles, chRoleGrant.Role)
			}
		} else {
			// Create a new RoleGrant and add it to the map
			grantMap[key] = &v1alpha1.RoleGrant{
				Grantees: []v1alpha1.Grantee{{
					User: chRoleGrant.Grantee.User,
					Role: chRoleGrant.Grantee.Role,
				}},
				Roles:           []string{chRoleGrant.Role},
				WithAdminOption: chRoleGrant.WithAdminOption,
			}
		}
	}

	var roleGrants []v1alpha1.RoleGrant
	for _, grant := range grantMap {
		roleGrants = append(roleGrants, *grant)
	}

	return roleGrants
}
