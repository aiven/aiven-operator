package controllers

import (
	"fmt"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
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

func TestDatabaseReconciler(t *testing.T) {
	t.Parallel()

	newDatabase := func(t *testing.T) *v1alpha1.Database {
		t.Helper()
		db := newObjectFromExampleYAML[v1alpha1.Database](t, "database")
		db.Namespace = "default"
		return db
	}

	runScenarioErr := func(t *testing.T, db *v1alpha1.Database, avn avngen.Client) (*Reconciler[*v1alpha1.Database], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newDatabaseReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.Database{}).
				WithObjects([]client.Object{db}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.Database])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: db.Name, Namespace: db.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, db *v1alpha1.Database, avn avngen.Client) (*Reconciler[*v1alpha1.Database], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, db, avn)
		require.NoError(t, err)
		return r, res
	}

	expectServiceRunning := func(avn *avngen.MockClient, db *v1alpha1.Database, times int) {
		avn.EXPECT().
			ServiceGet(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Times(times)
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Database{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Creates database on Aiven when it doesn't exist", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return(&service.ServiceDatabaseListOut{}, nil).Once()
		avn.EXPECT().
			ServiceDatabaseCreate(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceDatabaseCreateIn) bool {
				return in.Database == db.GetDatabaseName()
			})).
			Return(nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Database{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues without hard error on transient server error during create", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return(&service.ServiceDatabaseListOut{}, nil).Once()
		avn.EXPECT().
			ServiceDatabaseCreate(mock.Anything, db.Spec.Project, db.Spec.ServiceName, mock.Anything).
			Return(newAivenError(501, "Not Implemented")).Once()

		r, res, err := runScenarioErr(t, db, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Database{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Marks running and requeues when database already exists", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, db, 1)
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, db.Spec.Project, db.Spec.ServiceName).
			Return(&service.ServiceDatabaseListOut{Databases: []service.DatabaseOut{{DatabaseName: db.GetDatabaseName()}}}, nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.Database{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Deletes database and removes finalizer on deletion", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1
		db.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		db.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceDatabaseDelete(mock.Anything, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName()).
			Return(nil).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.Database{}
		err := r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Ignores not found on deletion", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1
		db.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		db.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceDatabaseDelete(mock.Anything, db.Spec.Project, db.Spec.ServiceName, db.GetDatabaseName()).
			Return(newAivenError(404, "not found")).Once()

		r, res := runScenario(t, db, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.Database{}
		err := r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Blocks deletion when termination protection is enabled", func(t *testing.T) {
		db := newDatabase(t)
		db.Generation = 1
		db.Finalizers = []string{instanceDeletionFinalizer}
		enabled := true
		db.Spec.TerminationProtection = &enabled
		now := metav1.Now()
		db.DeletionTimestamp = &now

		// No ServiceDatabaseDelete call is expected: termination protection short-circuits deletion.
		avn := avngen.NewMockClient(t)

		r, _, err := runScenarioErr(t, db, avn)
		require.Error(t, err)

		// The finalizer must remain so deletion is retried once protection is lifted.
		got := &v1alpha1.Database{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})
}

func TestGetDatabaseByName(t *testing.T) {
	t.Parallel()

	const (
		project = "test-project"
		svc     = "test-service"
	)

	t.Run("Finds database on a later page by following the Next cursor", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		// First page: does not contain the target, but advertises a next page.
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc).
			Return(&service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "other-a"}, {DatabaseName: "other-b"}},
				Next:      toOptionalStringPointer("cursor-1"),
			}, nil).Once()
		// Second page: fetched using the cursor from the first page, contains the target.
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc, [][2]string{service.ServiceDatabaseListAfter("cursor-1")}).
			Return(&service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "target"}},
			}, nil).Once()

		got, err := GetDatabaseByName(t.Context(), avn, project, svc, "target")
		require.NoError(t, err)
		require.Equal(t, "target", got.DatabaseName)
	})

	t.Run("Stops paging as soon as the database is found", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		// The first page contains the target. Even though Next is set, no second page must be requested.
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc).
			Return(&service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "target"}},
				Next:      toOptionalStringPointer("cursor-1"),
			}, nil).Once()

		got, err := GetDatabaseByName(t.Context(), avn, project, svc, "target")
		require.NoError(t, err)
		require.Equal(t, "target", got.DatabaseName)
	})

	t.Run("Returns not found after exhausting all pages", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc).
			Return(&service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "other-a"}},
				Next:      toOptionalStringPointer("cursor-1"),
			}, nil).Once()
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc, [][2]string{service.ServiceDatabaseListAfter("cursor-1")}).
			Return(&service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "other-b"}},
			}, nil).Once()

		_, err := GetDatabaseByName(t.Context(), avn, project, svc, "target")
		require.True(t, isNotFound(err))
	})

	t.Run("Returns error after exceeding the page cap", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		// Each page reports a Next cursor so the loop never terminates naturally.
		// The cap must kick in after maxDatabaseListPages iterations.
		out := func(i int) *service.ServiceDatabaseListOut {
			return &service.ServiceDatabaseListOut{
				Databases: []service.DatabaseOut{{DatabaseName: "other"}},
				Next:      toOptionalStringPointer(fmt.Sprintf("cursor-%d", i+1)),
			}
		}
		avn.EXPECT().
			ServiceDatabaseList(mock.Anything, project, svc).
			Return(out(0), nil).Once()
		for i := 1; i < maxDatabaseListPages; i++ {
			avn.EXPECT().
				ServiceDatabaseList(mock.Anything, project, svc, [][2]string{service.ServiceDatabaseListAfter(fmt.Sprintf("cursor-%d", i))}).
				Return(out(i), nil).Once()
		}

		_, err := GetDatabaseByName(t.Context(), avn, project, svc, "target")
		require.Error(t, err)
		require.False(t, isNotFound(err))
		require.ErrorContains(t, err, "exceeded")
	})
}
