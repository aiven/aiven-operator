package controllers

import (
	"context"
	"strings"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const yamlClickhouseGrant = `
apiVersion: aiven.io/v1alpha1
kind: ClickhouseGrant
metadata:
  name: test-grant
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  privilegeGrants:
    - grantees:
        - user: test-user
      privileges:
        - SELECT
      database: test-db
  roleGrants:
    - grantees:
        - user: test-user
      roles:
        - test-role
`

func newRunningService() *service.ServiceGetOut {
	return &service.ServiceGetOut{State: service.ServiceStateTypeRunning}
}

func Test_newClickhouseGrantReconciler(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := newClickhouseGrantReconciler(Controller{Client: k8sClient})

	rec, ok := r.(*Reconciler[*v1alpha1.ClickhouseGrant])
	require.True(t, ok)

	require.IsType(t, &v1alpha1.ClickhouseGrant{}, rec.newObj())

	ctrl, ok := rec.newController(nil).(*ClickhouseGrantController)
	require.True(t, ok)
	require.Equal(t, k8sClient, ctrl.Client)
}

func TestClickhouseGrantController_Observe(t *testing.T) {
	t.Parallel()

	t.Run("Returns error when service is not operational", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).
			Once()

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Observe(t.Context(), grant)
		require.EqualError(t, err, "preconditions are not met: [404 ]: service not found")
	})

	t.Run("Marks up to date without precondition queries when already applied", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)
		grant.SetGeneration(1)
		grant.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(newRunningService(), nil).
			Once()

		ctrl := &ClickhouseGrantController{avnGen: avn}

		obs, err := ctrl.Observe(t.Context(), grant)
		require.NoError(t, err)
		require.Equal(t, Observation{ResourceExists: true, ResourceUpToDate: true}, obs)
		// The running condition/marker is refreshed on the up-to-date path.
		require.Equal(t, "true", grant.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Reports resource missing when preconditions are met and never applied", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		router := &chQueryRouter{users: []string{"test-user"}, databases: []string{"test-db"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(newRunningService(), nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		obs, err := ctrl.Observe(t.Context(), grant)
		require.NoError(t, err)
		require.Equal(t, Observation{ResourceExists: false, ResourceUpToDate: false}, obs)
	})

	t.Run("Returns errPreconditionNotMet when a grantee is missing", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		// No users/roles -> the spec's grantee "test-user" is missing.
		router := &chQueryRouter{databases: []string{"test-db"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(newRunningService(), nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Observe(t.Context(), grant)
		require.ErrorIs(t, err, errPreconditionNotMet)
		require.Contains(t, err.Error(), "missing users or roles defined in spec: [test-user]")
	})

	t.Run("Returns errPreconditionNotMet when a database is missing", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		// Grantee exists but the referenced database does not.
		router := &chQueryRouter{users: []string{"test-user"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(newRunningService(), nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Observe(t.Context(), grant)
		require.ErrorIs(t, err, errPreconditionNotMet)
		require.Contains(t, err.Error(), "missing databases defined in spec: [test-db]")
	})
}

func TestClickhouseGrantController_Create(t *testing.T) {
	t.Parallel()

	t.Run("Grants spec privileges and records state without prior revoke", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		router := &chQueryRouter{}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Create(t.Context(), grant)
		require.NoError(t, err)

		// No prior state, so only GRANT statements are executed.
		require.Len(t, router.executed, 2)
		for _, stmt := range router.executed {
			require.True(t, strings.HasPrefix(stmt, "GRANT "), "expected only GRANT statements, got %q", stmt)
		}

		// State is recorded so the next reconcile can revoke it, and the running marker is set.
		require.NotNil(t, grant.Status.State)
		require.Equal(t, grant.Spec.Grants, *grant.Status.State)
		require.Equal(t, "true", grant.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Wraps error from a GRANT statement", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)
		// Stale running marker from a previous reconcile must be cleared on failure.
		grant.Annotations = map[string]string{instanceIsRunningAnnotation: "true"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(nil, assert.AnError)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Create(t.Context(), grant)
		require.ErrorIs(t, err, assert.AnError)
		// State must not advance when granting fails.
		require.Nil(t, grant.Status.State)
		// The resource must not keep advertising itself as running after a failed apply.
		require.NotContains(t, grant.Annotations, instanceIsRunningAnnotation)
	})
}

func TestClickhouseGrantController_Update(t *testing.T) {
	t.Parallel()

	t.Run("Revokes previous state before granting spec privileges", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)
		// A previously applied state that must be revoked first.
		grant.Status.State = &v1alpha1.Grants{
			PrivilegeGrants: []v1alpha1.PrivilegeGrant{
				{
					Grantees:   []v1alpha1.Grantee{{User: "test-user"}},
					Privileges: []string{"INSERT"},
					Database:   "old-db",
				},
			},
		}

		router := &chQueryRouter{}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		_, err := ctrl.Update(t.Context(), grant)
		require.NoError(t, err)

		// 1 REVOKE (old state) followed by 2 GRANTs (spec privilege + role grant).
		require.Len(t, router.executed, 3)
		require.True(t, strings.HasPrefix(router.executed[0], "REVOKE "), "first statement must revoke previous state, got %q", router.executed[0])
		require.True(t, strings.HasPrefix(router.executed[1], "GRANT "))
		require.True(t, strings.HasPrefix(router.executed[2], "GRANT "))

		require.Equal(t, grant.Spec.Grants, *grant.Status.State)
		require.Equal(t, "true", grant.Annotations[instanceIsRunningAnnotation])
	})
}

func TestClickhouseGrantController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Revokes spec grants", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		router := &chQueryRouter{}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			RunAndReturn(router.handle)

		ctrl := &ClickhouseGrantController{avnGen: avn}

		require.NoError(t, ctrl.Delete(t.Context(), grant))

		require.Len(t, router.executed, 2)
		for _, stmt := range router.executed {
			require.True(t, strings.HasPrefix(stmt, "REVOKE "), "expected only REVOKE statements, got %q", stmt)
		}
	})

	t.Run("Tolerates 400 and 404 errors during revoke", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "there is no role")).
			Twice()

		ctrl := &ClickhouseGrantController{avnGen: avn}

		require.NoError(t, ctrl.Delete(t.Context(), grant))
	})

	t.Run("Propagates unexpected revoke errors", func(t *testing.T) {
		grant := newObjectFromYAML[v1alpha1.ClickhouseGrant](t, yamlClickhouseGrant)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, grant.Spec.Project, grant.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(500, "boom"))

		ctrl := &ClickhouseGrantController{avnGen: avn}

		err := ctrl.Delete(t.Context(), grant)
		require.Error(t, err)
		require.Contains(t, err.Error(), "boom")
	})
}

// chQueryRouter dispatches ServiceClickHouseQuery calls.
// Precondition queries, GRANT/REVOKE statements are recorded in order.
type chQueryRouter struct {
	users     []string
	roles     []string
	databases []string

	executed []string
}

func (r *chQueryRouter) handle(_ context.Context, _, _ string, in *clickhouse.ServiceClickHouseQueryIn) (*clickhouse.ServiceClickHouseQueryOut, error) {
	q := in.Query
	switch {
	case strings.HasPrefix(q, "GRANT "), strings.HasPrefix(q, "REVOKE "):
		r.executed = append(r.executed, q)
		return &clickhouse.ServiceClickHouseQueryOut{}, nil
	case strings.Contains(q, "system.users"):
		return &clickhouse.ServiceClickHouseQueryOut{Data: toRows(r.users)}, nil
	case strings.Contains(q, "system.roles"):
		return &clickhouse.ServiceClickHouseQueryOut{Data: toRows(r.roles)}, nil
	case strings.Contains(q, "system.databases"):
		return &clickhouse.ServiceClickHouseQueryOut{Data: toRows(r.databases)}, nil
	default:
		return &clickhouse.ServiceClickHouseQueryOut{}, nil
	}
}

func toRows(values []string) [][]any {
	rows := make([][]any, 0, len(values))
	for _, v := range values {
		rows = append(rows, []any{v})
	}
	return rows
}
