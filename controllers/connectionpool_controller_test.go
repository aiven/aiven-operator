package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/postgresql"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

const yamlConnectionPool = `
apiVersion: aiven.io/v1alpha1
kind: ConnectionPool
metadata:
  name: test-pool
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  databaseName: test-db
  poolMode: transaction
  poolSize: 25
`

func TestConnectionPoolReconciler(t *testing.T) {
	t.Parallel()

	runScenarioErr := func(t *testing.T, cp *v1alpha1.ConnectionPool, avn avngen.Client) (*Reconciler[*v1alpha1.ConnectionPool], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newConnectionPoolReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ConnectionPool{}).
				WithObjects([]client.Object{cp}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ConnectionPool])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, cp *v1alpha1.ConnectionPool, avn avngen.Client) (*Reconciler[*v1alpha1.ConnectionPool], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, cp, avn)
		require.NoError(t, err)
		return r, res
	}

	// serviceGetOut returns a running PG service exposing the db and (optionally) the pool.
	serviceGetOut := func(cp *v1alpha1.ConnectionPool, pools ...service.ConnectionPoolOut) *service.ServiceGetOut {
		return &service.ServiceGetOut{
			State:     service.ServiceStateTypeRunning,
			Databases: []string{cp.Spec.DatabaseName},
			ServiceUriParams: map[string]string{
				"host":     "pg.example.com",
				"user":     "avnadmin",
				"password": "svc-pass",
				"sslmode":  "require",
			},
			ConnectionPools: pools,
		}
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ConnectionPool{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues when target database isn't present yet", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).Once()

		_, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Creates connection pool and publishes secret", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1

		pool := service.ConnectionPoolOut{
			PoolName:      cp.Name,
			Database:      cp.Spec.DatabaseName,
			PoolMode:      service.PoolModeType(cp.Spec.PoolMode),
			PoolSize:      cp.Spec.PoolSize,
			ConnectionUri: "postgres://avnadmin:svc-pass@pg.example.com:24655/test-db?sslmode=require",
		}

		avn := avngen.NewMockClient(t)
		// Observe: pool doesn't exist yet -> Create.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp), nil).Once()
		avn.EXPECT().
			ServicePGBouncerCreate(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.MatchedBy(func(in *postgresql.ServicePGBouncerCreateIn) bool {
				return in.PoolName == cp.Name &&
					in.Database == cp.Spec.DatabaseName &&
					in.PoolSize != nil && *in.PoolSize == cp.Spec.PoolSize &&
					string(in.PoolMode) == string(cp.Spec.PoolMode)
			})).
			Return(nil).Once()
		// Create re-fetches the service to build connection details, now the pool is present.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, pool), nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, cp.Spec.Project).Return("ca-cert", nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ConnectionPool{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)

		// Secret is published in the same reconcile as create, not deferred to a later pass.
		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, secret))
		require.Equal(t, []byte("pg.example.com"), secret.Data["CONNECTIONPOOL_HOST"])
		require.Equal(t, []byte("svc-pass"), secret.Data["CONNECTIONPOOL_PASSWORD"])
		require.Equal(t, []byte("ca-cert"), secret.Data["CONNECTIONPOOL_CA_CERT"])
	})

	t.Run("Publishes secret and marks running when pool is up to date", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		pool := service.ConnectionPoolOut{
			PoolName:      cp.Name,
			Database:      cp.Spec.DatabaseName,
			PoolMode:      service.PoolModeType(cp.Spec.PoolMode),
			PoolSize:      cp.Spec.PoolSize,
			ConnectionUri: "postgres://avnadmin:svc-pass@pg.example.com:24655/test-db?sslmode=require",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, pool), nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, cp.Spec.Project).Return("ca-cert", nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ConnectionPool{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, secret))
		require.Equal(t, []byte("pg.example.com"), secret.Data["CONNECTIONPOOL_HOST"])
		require.Equal(t, []byte("24655"), secret.Data["CONNECTIONPOOL_PORT"])
		require.Equal(t, []byte("test-db"), secret.Data["CONNECTIONPOOL_DATABASE"])
		require.Equal(t, []byte("svc-pass"), secret.Data["CONNECTIONPOOL_PASSWORD"])
		require.Equal(t, []byte("ca-cert"), secret.Data["CONNECTIONPOOL_CA_CERT"])
		// Legacy unprefixed keys remain for backwards compatibility.
		require.Equal(t, []byte("pg.example.com"), secret.Data["PGHOST"])
	})

	t.Run("Publishes pool user credentials when spec sets a username", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Spec.Username = "pool-user"
		cp.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		poolUser := "pool-user"
		pool := service.ConnectionPoolOut{
			PoolName:      cp.Name,
			Database:      cp.Spec.DatabaseName,
			PoolMode:      service.PoolModeType(cp.Spec.PoolMode),
			PoolSize:      cp.Spec.PoolSize,
			Username:      &poolUser,
			ConnectionUri: "postgres://pool-user:user-pass@pg.example.com:24655/test-db?sslmode=require",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, pool), nil).Once()
		// Observe fetches the user once and threads it through to buildSecretDetails.
		avn.EXPECT().
			ServiceUserGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username).
			Return(&service.ServiceUserGetOut{Username: poolUser, Password: "user-pass"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, cp.Spec.Project).Return("ca-cert", nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, secret))
		// USER comes from the pool's username, PASSWORD from the service user lookup.
		require.Equal(t, []byte("pool-user"), secret.Data["CONNECTIONPOOL_USER"])
		require.Equal(t, []byte("user-pass"), secret.Data["CONNECTIONPOOL_PASSWORD"])
		require.Equal(t, []byte("pool-user"), secret.Data["PGUSER"])
		require.Equal(t, []byte("user-pass"), secret.Data["PGPASSWORD"])
		require.Equal(t, []byte("ca-cert"), secret.Data["CONNECTIONPOOL_CA_CERT"])
	})

	t.Run("Updates connection pool when spec drifts", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		// Remote pool has a different size than the spec, so Observe returns not-up-to-date.
		pool := service.ConnectionPoolOut{
			PoolName:      cp.Name,
			Database:      cp.Spec.DatabaseName,
			PoolMode:      service.PoolModeType(cp.Spec.PoolMode),
			PoolSize:      10,
			ConnectionUri: "postgres://avnadmin:svc-pass@pg.example.com:24655/test-db?sslmode=require",
		}

		// Pool with the spec's size, returned after the update so connection details can be built.
		updatedPool := pool
		updatedPool.PoolSize = cp.Spec.PoolSize

		avn := avngen.NewMockClient(t)
		// Observe sees drift (PoolSize 10 != 25) -> Update. No CA fetch on this branch.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, pool), nil).Once()
		avn.EXPECT().
			ServicePGBouncerUpdate(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Name, mock.MatchedBy(func(in *postgresql.ServicePGBouncerUpdateIn) bool {
				return in.PoolSize != nil && *in.PoolSize == cp.Spec.PoolSize
			})).
			Return(nil).Once()
		// Update re-fetches the service to build connection details.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, updatedPool), nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, cp.Spec.Project).Return("ca-cert", nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, secret))
		require.Equal(t, []byte("ca-cert"), secret.Data["CONNECTIONPOOL_CA_CERT"])
	})

	t.Run("Updates connection pool when pool mode drifts", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		// Remote pool has a different mode than the spec, so Observe returns not-up-to-date.
		pool := service.ConnectionPoolOut{
			PoolName:      cp.Name,
			Database:      cp.Spec.DatabaseName,
			PoolMode:      service.PoolModeTypeSession,
			PoolSize:      cp.Spec.PoolSize,
			ConnectionUri: "postgres://avnadmin:svc-pass@pg.example.com:24655/test-db?sslmode=require",
		}

		// Pool with the spec's mode, returned after the update so connection details can be built.
		updatedPool := pool
		updatedPool.PoolMode = service.PoolModeType(cp.Spec.PoolMode)

		avn := avngen.NewMockClient(t)
		// Observe sees drift (session != transaction) -> Update. No CA fetch on this branch.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, pool), nil).Once()
		avn.EXPECT().
			ServicePGBouncerUpdate(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Name, mock.MatchedBy(func(in *postgresql.ServicePGBouncerUpdateIn) bool {
				return string(in.PoolMode) == string(cp.Spec.PoolMode)
			})).
			Return(nil).Once()
		// Update re-fetches the service to build connection details.
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp, updatedPool), nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, cp.Spec.Project).Return("ca-cert", nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, secret))
		require.Equal(t, []byte("ca-cert"), secret.Data["CONNECTIONPOOL_CA_CERT"])
	})

	t.Run("Deletes connection pool and removes finalizer on deletion", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		cp.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServicePGBouncerDelete(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Name).
			Return(nil).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ConnectionPool{}
		err := r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Requeues when the pool's service user isn't present yet", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Spec.Username = "pool-user"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, mock.Anything).
			Return(serviceGetOut(cp), nil).Once()
		// The referenced service user doesn't exist yet -> soft requeue, not a hard error.
		avn.EXPECT().
			ServiceUserGet(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Spec.Username).
			Return(nil, newAivenError(404, "user not found")).Once()

		_, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Removes finalizer when the pool is already gone at Aiven", func(t *testing.T) {
		cp := newObjectFromYAML[v1alpha1.ConnectionPool](t, yamlConnectionPool)
		cp.Generation = 1
		cp.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		cp.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		// Delete tolerates a 404 from Aiven (pool already deleted) and still removes the finalizer.
		avn.EXPECT().
			ServicePGBouncerDelete(mock.Anything, cp.Spec.Project, cp.Spec.ServiceName, cp.Name).
			Return(newAivenError(404, "pool not found")).Once()

		r, res := runScenario(t, cp, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ConnectionPool{}
		err := r.Get(t.Context(), types.NamespacedName{Name: cp.Name, Namespace: cp.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
