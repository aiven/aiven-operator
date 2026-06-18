package controllers

import (
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

func TestClickhouseDatabaseReconciler(t *testing.T) {
	t.Parallel()

	newClickhouseDatabase := func(t *testing.T) *v1alpha1.ClickhouseDatabase {
		t.Helper()
		db := newObjectFromExampleYAML[v1alpha1.ClickhouseDatabase](t, "clickhousedatabase")
		db.Namespace = "default"
		return db
	}

	runScenarioErr := func(t *testing.T, db *v1alpha1.ClickhouseDatabase, avn avngen.Client) (*Reconciler[*v1alpha1.ClickhouseDatabase], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newClickhouseDatabaseReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ClickhouseDatabase{}).
				WithObjects([]client.Object{db}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ClickhouseDatabase])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: db.Name, Namespace: db.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, db *v1alpha1.ClickhouseDatabase, avn avngen.Client) (*Reconciler[*v1alpha1.ClickhouseDatabase], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, db, avn)
		require.NoError(t, err)
		return r, res
	}

	expectServiceRunning := func(avn *avngen.MockClient, db *v1alpha1.ClickhouseDatabase, times int) {
		avn.EXPECT().
			ServiceGet(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Times(times)
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Creates database on Aiven when it doesn't exist", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceClickHouseDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceClickHouseDatabaseCreate(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseDatabaseCreateIn) bool {
				return in.Database == db.GetDatabaseName()
			})).
			Return(nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues without hard error on transient server error during create", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceClickHouseDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceClickHouseDatabaseCreate(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(newAivenError(501, "Not Implemented")).Once()

		r, res, err := runScenarioErr(t, db, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Marks running and requeues when database already exists", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceClickHouseDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return([]clickhouse.DatabaseOut{{Name: db.GetDatabaseName()}}, nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Deletes database and removes finalizer on deletion", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1
		db.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		db.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseDatabaseDelete(mock.Anything, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName()).
			Return(nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		err := r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Ignores not found on deletion", func(t *testing.T) {
		db := newClickhouseDatabase(t)
		db.Generation = 1
		db.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		db.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseDatabaseDelete(mock.Anything, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName()).
			Return(newAivenError(404, "not found")).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ClickhouseDatabase{}
		err := r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
