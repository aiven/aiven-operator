package controllers

import (
	"strings"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestClickhouseRoleReconciler(t *testing.T) {
	t.Parallel()

	newClickhouseRole := func(t *testing.T) *v1alpha1.ClickhouseRole {
		t.Helper()
		role := newObjectFromExampleYAML[v1alpha1.ClickhouseRole](t, "clickhouserole")
		role.Namespace = "default"
		return role
	}

	runScenarioErr := func(t *testing.T, role *v1alpha1.ClickhouseRole, avn avngen.Client) (*Reconciler[*v1alpha1.ClickhouseRole], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newClickhouseRoleReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ClickhouseRole{}).
				WithObjects([]client.Object{role}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ClickhouseRole])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: role.Name, Namespace: role.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, role *v1alpha1.ClickhouseRole, avn avngen.Client) (*Reconciler[*v1alpha1.ClickhouseRole], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, role, avn)
		require.NoError(t, err)
		return r, res
	}

	expectServiceRunning := func(avn *avngen.MockClient, role *v1alpha1.ClickhouseRole, times int) {
		avn.EXPECT().
			ServiceGet(mock.Anything, role.Spec.Project, role.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Times(times)
	}

	unknownRoleErr := newAivenError(511, "Code: 511. DB::Exception: Role not found")

	matchQuery := func(prefix string, role *v1alpha1.ClickhouseRole) any {
		expected := prefix + " " + escape(role.Spec.Role)
		return mock.MatchedBy(func(in *clickhouse.ServiceClickHouseQueryIn) bool {
			return in.Database == defaultDatabase && in.Query == expected
		})
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, role.Spec.Project, role.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates role on Aiven when it doesn't exist", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(nil, unknownRoleErr).Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("CREATE ROLE IF NOT EXISTS", role)).
			Return(&clickhouse.ServiceClickHouseQueryOut{}, nil).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues without hard error on transient server error during create", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(nil, unknownRoleErr).Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("CREATE ROLE IF NOT EXISTS", role)).
			Return(nil, newAivenError(501, "Not Implemented")).Once()

		r, res, err := runScenarioErr(t, role, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Returns hard error on non-transient failure during create", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(nil, unknownRoleErr).Once()
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("CREATE ROLE IF NOT EXISTS", role)).
			Return(nil, newAivenError(400, "Bad Request")).Once()

		_, _, err := runScenarioErr(t, role, avn)
		require.Error(t, err)
		require.ErrorContains(t, err, "cannot create clickhouse role on Aiven side")
	})

	t.Run("Returns hard error when Observe fails with a non-404, non-511 error", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(nil, newAivenError(400, "Bad Request")).Once()

		_, _, err := runScenarioErr(t, role, avn)
		require.Error(t, err)
		require.ErrorContains(t, err, "describing ClickHouse role")
	})

	t.Run("Marks running and requeues when role already exists", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(&clickhouse.ServiceClickHouseQueryOut{}, nil).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Steady-state re-reconcile when role already exists and generation is processed", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1
		role.Finalizers = []string{instanceDeletionFinalizer}
		role.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, role, 1)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("SHOW CREATE ROLE", role)).
			Return(&clickhouse.ServiceClickHouseQueryOut{}, nil).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletes role and removes finalizer on deletion", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1
		role.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		role.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("DROP ROLE IF EXISTS", role)).
			Return(&clickhouse.ServiceClickHouseQueryOut{}, nil).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ClickhouseRole{}
		err := r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Ignores unknown role (Code: 511) on deletion", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1
		role.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		role.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("DROP ROLE IF EXISTS", role)).
			Return(nil, unknownRoleErr).Once()

		r, res := runScenario(t, role, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ClickhouseRole{}
		err := r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Propagates hard error when deletion genuinely fails", func(t *testing.T) {
		role := newClickhouseRole(t)
		role.Generation = 1
		role.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		role.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseQuery(mock.Anything, role.Spec.Project, role.Spec.ServiceName, matchQuery("DROP ROLE IF EXISTS", role)).
			Return(nil, newAivenError(400, "Bad Request")).Once()

		r, _, err := runScenarioErr(t, role, avn)
		require.Error(t, err)

		got := &v1alpha1.ClickhouseRole{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})
}

func Test_escape(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "plain identifier", input: "my-role", expected: "`my-role`"},
		{name: "backtick is escaped", input: "role`with`backtick", expected: "`role\\`with\\`backtick`"},
		{name: "backslash is escaped", input: `role\with\backslash`, expected: "`role\\\\with\\\\backslash`"},
		{name: "empty identifier", input: "", expected: "``"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, escape(tc.input))
			// Whatever escaping happens, the result must remain wrapped in backticks.
			require.True(t, strings.HasPrefix(escape(tc.input), "`"))
			require.True(t, strings.HasSuffix(escape(tc.input), "`"))
		})
	}
}
