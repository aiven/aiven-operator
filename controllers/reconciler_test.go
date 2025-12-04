package controllers

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/yaml"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func mockNewAivenGeneratedClient(m *mock.Mock) func(token, kubeVersion, operatorVersion string) (avngen.Client, error) {
	return func(token, kubeVersion, operatorVersion string) (avngen.Client, error) {
		args := m.MethodCalled("newAivenGeneratedClient", token, kubeVersion, operatorVersion)
		return args.Get(0).(avngen.Client), args.Error(1)
	}
}

func newObjectFromYAML[T any](t *testing.T, y string) *T {
	t.Helper()

	var obj T
	require.NoError(t, yaml.Unmarshal([]byte(y), &obj))

	return &obj
}

func normalizedConditions(conds []metav1.Condition) []metav1.Condition {
	out := make([]metav1.Condition, len(conds))
	for i, c := range conds {
		cc := c
		cc.LastTransitionTime = metav1.Time{}
		out[i] = cc
	}
	return out
}

func recorderEvents(recorder *record.FakeRecorder) []string {
	var events []string // nolint: prealloc
	for {
		select {
		case e := <-recorder.Events:
			events = append(events, e)
		default:
			return events
		}
	}
}

func newTestReconciler[T v1alpha1.AivenManagedObject](_ T, client crclient.Client, scheme *runtime.Scheme, recorder *record.FakeRecorder) *Reconciler[T] { //nolint:unparam
	return &Reconciler[T]{
		Controller: Controller{
			Client:   client,
			Scheme:   scheme,
			Recorder: recorder,
		},
		newSecret: newSecret,
	}
}

type logRecorderSink struct {
	logs []string
}

func (l *logRecorderSink) Init(logr.RuntimeInfo) {}

func (l *logRecorderSink) Enabled(_ int) bool { return true }

func (l *logRecorderSink) Info(_ int, msg string, _ ...any) {
	l.logs = append(l.logs, msg)
}

func (l *logRecorderSink) Error(_ error, msg string, _ ...any) {
	l.logs = append(l.logs, "ERROR: "+msg)
}

func (l *logRecorderSink) WithValues(_ ...any) logr.LogSink { return l }

func (l *logRecorderSink) WithName(_ string) logr.LogSink { return l }

func newAivenError(status int, msg string) error {
	return avngen.Error{Status: status, Message: msg}
}

const (
	// ClickhouseUser
	yamlClickhouseUser = `
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: test-user
  namespace: default
spec:
  project: test-project
  serviceName: test-service
`

	yamlClickhouseUserWithAuth = `
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: test-user
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  authSecretRef:
    name: aiven-token
    key: token
`

	// PostgreSQL
	yamlPostgres = `
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: pg-no-ref
  namespace: default
spec:
  project: test-project
  plan: hobby
`

	yamlPostgresWithRef = `
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: pg-with-ref
  namespace: default
spec:
  project: test-project
  plan: hobby
  projectVPCRef:
    name: test-vpc
`

	// ProjectVPC
	yamlProjectVPCNotReady = `
apiVersion: aiven.io/v1alpha1
kind: ProjectVPC
metadata:
  name: test-vpc
  namespace: default
spec:
  project: test-project
  cloudName: aws-eu-west-1
  networkCidr: 10.0.0.0/24
`

	yamlProjectVPCReady = `
apiVersion: aiven.io/v1alpha1
kind: ProjectVPC
metadata:
  name: test-vpc
  namespace: default
  generation: 1
  annotations:
    controllers.aiven.io/generation-was-processed: "1"
    controllers.aiven.io/instance-is-running: "true"
spec:
  project: test-project
  cloudName: aws-eu-west-1
  networkCidr: 10.0.0.0/24
`

	// Secret
	yamlAuthSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: aiven-token
  namespace: default
data:
  token: dGVzdC10b2tlbg== # gitleaks:allow
`
)

func TestReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Object not found is ignored", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: "test-user", Namespace: "default"},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)
	})

	t.Run("Returns error when Get returns non-not-found error", func(t *testing.T) {
		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(
				interceptor.Funcs{
					Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, o crclient.Object, opts ...crclient.GetOption) error {
						args := m.MethodCalled("Get", ctx, c, key, o, opts)
						return args.Error(0)
					},
				}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: "test-user", Namespace: "default"},
		})

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("Returns error when resolving refs fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				*args.Get(3).(*v1alpha1.PostgreSQL) = *obj.DeepCopy()
			}).
			Return(nil).Once()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		// Intentionally do NOT register v1alpha1 types to force Scheme.New to fail.

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, o crclient.Object, opts ...crclient.GetOption) error {
					args := m.MethodCalled("Get", ctx, c, key, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newObj: func() *v1alpha1.PostgreSQL { return &v1alpha1.PostgreSQL{} },
		}

		ctx := t.Context()
		nn := types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}

		res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: nn})

		require.Equal(t, ctrl.Result{}, res)
		require.Error(t, err)

		gvk := v1alpha1.GroupVersion.WithKind("ProjectVPC")
		require.Contains(t, err.Error(), "unable to resolve references:")
		require.Contains(t, err.Error(), "creating "+gvk.String())

		require.Equal(t, []string{
			"Warning UnableToWaitForPreconditions " + strings.TrimPrefix(err.Error(), "unable to resolve references: "),
		}, recorderEvents(recorder))
	})

	t.Run("Requests requeue when refs are not ready", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newObj: func() *v1alpha1.PostgreSQL { return &v1alpha1.PostgreSQL{} },
		}

		ctx := t.Context()
		nn := types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}

		res, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: nn})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)

		require.Equal(t, []string{
			"Normal WaitingForPreconditions waiting for referenced resources to be ready",
		}, recorderEvents(recorder))
	})

	t.Run("Propagates error when creating Aiven client", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return((*struct{ avngen.Client })(nil), assert.AnError).
			Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newObj:                  func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, fmt.Sprintf("cannot initialize aiven generated client: %s", assert.AnError.Error()))
		require.Equal(t, []string{
			"Warning UnableToCreateClient " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Calls finalize when object is marked for deletion", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.DeletionTimestamp = ptr(metav1.Now())
		obj.Finalizers = []string{instanceDeletionFinalizer}

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		ctx := t.Context()
		res, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)

		got := &v1alpha1.ClickhouseUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Normal SuccessfullyDeletedAtAiven instance is gone at aiven now",
		}, recorderEvents(recorder))
	})

	t.Run("Uses handleObserveError for observe errors", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			Return(Observation{}, fmt.Errorf("%w: some reason", errPreconditionNotMet)).
			Once()

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		ctx := t.Context()
		res, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			"Normal PreconditionsNotMet preconditions are not met, requeue",
		}, recorderEvents(recorder))
	})

	t.Run("Propagates error when publishing observe secret details fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once().
			On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return(avngen.NewMockClient(t), nil).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					args := m.MethodCalled("Create", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			Return(Observation{
				ResourceExists:   true,
				ResourceUpToDate: true,
				SecretDetails:    map[string]string{"FOO": "foo"},
			}, nil).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj:    func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
			newSecret: newSecret,
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.EqualError(t, err, fmt.Sprintf("unable to sync connection secret: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)
	})

	t.Run("Calls Create when resource does not exist", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			Return(Observation{
				ResourceExists:   false,
				ResourceUpToDate: false,
			}, nil).
			Once()
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(CreateResult{}, nil).
			Once()

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			"Normal PreconditionsAreMet preconditions are met, proceeding to create or update",
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Normal CreatedOrUpdatedAtAiven instance was created at aiven but may not be running yet",
		}, recorderEvents(recorder))
	})

	t.Run("Calls Update when resource is not up to date", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			Return(Observation{
				ResourceExists:   true,
				ResourceUpToDate: false,
			}, nil).
			Once()
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{}, nil).
			Once()

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			"Normal PreconditionsAreMet preconditions are met, proceeding to create or update",
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})

	t.Run("Requeues when resource is up to date", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		// Clear processedGenerationAnnotation to verify that a steady-state reconcile will mark the current generation as processed.
		delete(obj.GetAnnotations(), processedGenerationAnnotation)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			Return(Observation{
				ResourceExists:   true,
				ResourceUpToDate: true,
			}, nil).
			Once()

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
			newController: func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] {
				return c
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		res, err := r.Reconcile(t.Context(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.True(t, hasLatestGeneration(got))
	})
}

func TestReconciler_handleObserveError(t *testing.T) {
	t.Parallel()

	t.Run("Preconditions fail when service is powered off", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		obj := &v1alpha1.ClickhouseUser{}
		inner := fmt.Errorf("%w: %w", errPreconditionNotMet, fmt.Errorf("%w: project/service", errServicePoweredOff))
		res, err := r.handleObserveError(t.Context(), obj, inner)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)
		require.Equal(t, []string{"Warning UnableToWaitForPreconditions " + inner.Error()}, recorderEvents(recorder))
		require.ElementsMatch(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionPreconditions),
				Message: inner.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
	})

	t.Run("Requeues when preconditions are not met", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		obj := &v1alpha1.ClickhouseUser{}
		res, err := r.handleObserveError(t.Context(), obj, errPreconditionNotMet)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, []string{"Normal PreconditionsNotMet preconditions are not met, requeue"}, recorderEvents(recorder))
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
	})

	t.Run("Requeues on retryable Aiven error", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		obj := &v1alpha1.ClickhouseUser{}
		aivenErr := NewNotFound("not found")
		res, err := r.handleObserveError(t.Context(), obj, aivenErr)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, []string{"Warning UnableToWaitForPreconditions " + aivenErr.Error()}, recorderEvents(recorder))
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
	})

	t.Run("Returns error on non-retryable error", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		obj := &v1alpha1.ClickhouseUser{}
		origErr := fmt.Errorf("boom")
		res, err := r.handleObserveError(t.Context(), obj, origErr)

		require.EqualError(t, err, `cannot observe the resource: `+origErr.Error())
		require.ErrorIs(t, err, origErr)
		require.Equal(t, ctrl.Result{}, res)
		require.Equal(t, []string{"Warning UnableToWaitForPreconditions " + origErr.Error()}, recorderEvents(recorder))
		require.ElementsMatch(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionPreconditions),
				Message: origErr.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
	})
}

func TestReconciler_resolveK8sRefs(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("No requeue if an object has no refs", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		obj := &v1alpha1.ClickhouseUser{}
		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("No requeue if there are no dependencies", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.PostgreSQL]{}

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("Error if dependency GVK is unknown", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		// Intentionally do NOT register v1alpha1 types to force Scheme.New to fail.

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Scheme: scheme,
			},
		}

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		gvk := v1alpha1.GroupVersion.WithKind("ProjectVPC")
		_, underlying := scheme.New(gvk)
		require.ErrorContains(t, underlying, `no kind "ProjectVPC" is registered for version "aiven.io/v1alpha1" in scheme`)

		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.EqualError(t, err, fmt.Sprintf("creating %s: %s", gvk, underlying.Error()))
		require.False(t, requeue)
	})

	t.Run("Error if dependency type is not client.Object", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		gvk := v1alpha1.GroupVersion.WithKind("ProjectVPC")
		scheme.AddKnownTypeWithName(gvk, &struct{ runtime.Object }{})

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Scheme: scheme,
			},
		}

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.EqualError(t, err, fmt.Sprintf("gvk %s is not client.Object", gvk))
		require.False(t, requeue)
	})

	t.Run("Requeue if dependency is missing", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)
		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.NoError(t, err)
		require.True(t, requeue)
	})

	t.Run("No requeue if dependency is ready", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)
		vpc := newObjectFromYAML[v1alpha1.ProjectVPC](t, yamlProjectVPCReady)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(vpc).
			Build()

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.NoError(t, err)
		require.False(t, requeue)
	})

	t.Run("Requeue if dependency is not ready", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)
		vpc := newObjectFromYAML[v1alpha1.ProjectVPC](t, yamlProjectVPCNotReady)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(vpc).
			Build()

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.NoError(t, err)
		require.True(t, requeue)
	})
}

func TestReconciler_updateStatus(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("No-op when objects are equal", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Status.UUID = "uuid-1"
		orig := obj.DeepCopy()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		err := r.updateStatus(t.Context(), orig, obj)
		require.NoError(t, err)
	})

	t.Run("Returns error when Get fails", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		orig := obj.DeepCopy()
		obj.Status.UUID = "uuid-after"

		err := r.updateStatus(t.Context(), orig, obj)
		require.EqualError(t, err, `clickhouseusers.aiven.io "test-user" not found`)
	})

	t.Run("Retries on conflict error and eventually succeeds", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		conflictErr := apierrors.NewConflict(schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"}, obj.Name, fmt.Errorf("conflict"))
		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Twice().
			On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once().
			On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once().
			On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(nil).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, o crclient.Object, opts ...crclient.GetOption) error {
					args := m.MethodCalled("Get", ctx, c, key, o, opts)
					return args.Error(0)
				},
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					args := m.MethodCalled("Update", ctx, c, o, opts)
					return args.Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					args := m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts)
					return args.Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		orig := obj.DeepCopy()
		obj.Status.UUID = "uuid-after"

		err := r.updateStatus(t.Context(), orig, obj)
		require.NoError(t, err)
	})

	t.Run("Updates spec and status when objects differ", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Status.UUID = "old-uuid"
		obj.Status.Conditions = []metav1.Condition{
			{
				Type:   "Some",
				Status: metav1.ConditionFalse,
				Reason: "Old",
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		orig := obj.DeepCopy()

		// Mutate both spec and status to force an update
		obj.Spec.Project = "new-project"
		obj.Status.UUID = "new-uuid"
		obj.Status.Conditions = []metav1.Condition{
			{
				Type:   "Some",
				Status: metav1.ConditionTrue,
				Reason: "New",
			},
		}

		err := r.updateStatus(t.Context(), orig, obj)
		require.NoError(t, err)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))

		require.Equal(t, "new-project", got.Spec.Project)
		require.Equal(t, "new-uuid", got.Status.UUID)
		require.ElementsMatch(t, []metav1.Condition{
			{
				Type:   "Some",
				Status: metav1.ConditionTrue,
				Reason: "New",
			},
		}, normalizedConditions(got.Status.Conditions))
	})

	t.Run("Returns error when Update fails with non-conflict error", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					args := m.MethodCalled("Update", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		orig := obj.DeepCopy()
		obj.Status.UUID = "uuid-after"

		err := r.updateStatus(t.Context(), orig, obj)
		require.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("Returns error when Status update fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					args := m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts)
					return args.Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		orig := obj.DeepCopy()
		obj.Status.UUID = "uuid-after"

		err := r.updateStatus(t.Context(), orig, obj)
		require.EqualError(t, err, assert.AnError.Error())
	})
}

func TestReconciler_newAivenClient(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Propagates error from token resolution", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}
		obj := &v1alpha1.ClickhouseUser{}

		client, err := r.newAivenClient(t.Context(), obj)

		require.ErrorIs(t, err, errNoTokenProvided)
		require.Nil(t, client)
	})

	t.Run("Emits event and wraps error when client creation fails", func(t *testing.T) {
		recorder := record.NewFakeRecorder(10)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return((*struct{ avngen.Client })(nil), assert.AnError).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
		}

		obj := &v1alpha1.ClickhouseUser{}

		client, err := r.newAivenClient(t.Context(), obj)

		require.EqualError(t, err, fmt.Sprintf("cannot initialize aiven generated client: %s", assert.AnError.Error()))
		require.Nil(t, client)
		require.Equal(t, []string{
			"Warning UnableToCreateClient " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Returns client when token resolves and client creation succeeds", func(t *testing.T) {
		cl := avngen.NewMockClient(t)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return(cl, nil).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
			},
			newAivenGeneratedClient: mockNewAivenGeneratedClient(m),
		}

		obj := &v1alpha1.ClickhouseUser{}

		client, err := r.newAivenClient(t.Context(), obj)

		require.NoError(t, err)
		require.Equal(t, cl, client)
	})
}

func TestReconciler_resolveToken(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Uses default token when configured", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				DefaultToken: "default-token",
			},
		}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		token, err := r.resolveToken(t.Context(), obj)

		require.NoError(t, err)
		require.Equal(t, "default-token", token)
	})

	t.Run("Errors when no default token and authSecretRef is nil", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		token, err := r.resolveToken(t.Context(), obj)

		require.Empty(t, token)
		require.ErrorIs(t, err, errNoTokenProvided)
	})

	t.Run("Emits warning and wraps error when auth secret cannot be fetched", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Recorder: recorder,
			},
		}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		token, err := r.resolveToken(t.Context(), obj)

		errNotFound := apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "aiven-token")
		require.EqualError(t, err, fmt.Sprintf("cannot get secret %q: %s", "aiven-token", errNotFound.Error()))
		require.Empty(t, token)
		require.Equal(t, []string{
			fmt.Sprintf("Warning %s %s", eventUnableToGetAuthSecret, errNotFound.Error()),
		}, recorderEvents(recorder))
	})

	t.Run("Returns token from secret when authSecretRef is set", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)).
			Build()
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		token, err := r.resolveToken(t.Context(), obj)

		require.NoError(t, err)
		require.Equal(t, "test-token", token)
	})
}

func TestReconciler_createResource(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Sets error condition and returns error when creation fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Create(mock.Anything, mock.Anything).Return(CreateResult{}, assert.AnError).Once()

		res, err := r.createResource(t.Context(), c, obj)

		require.EqualError(t, err, fmt.Sprintf("unable to create or update instance at aiven: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)
		require.ElementsMatch(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionCreateOrUpdate),
				Message: assert.AnError.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Warning UnableToCreateOrUpdateAtAiven " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Returns error when publishing secret details fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					args := m.MethodCalled("Create", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(CreateResult{SecretDetails: map[string]string{"FOO": "foo"}}, nil).
			Once()

		res, err := r.createResource(t.Context(), c, obj)
		require.EqualError(t, err, fmt.Sprintf("unable to sync connection secret: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)

		secret := &corev1.Secret{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))

		require.Equal(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionConnInfoSecret),
				Message: assert.AnError.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Normal CreatedOrUpdatedAtAiven instance was created at aiven but may not be running yet",
			"Warning CannotPublishConnectionDetails " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Creates remote resource and publishes secrets", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(CreateResult{SecretDetails: map[string]string{"FOO": "foo"}}, nil).
			Once()

		res, err := r.createResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret))
		require.Equal(t, map[string][]byte{
			"FOO": []byte("foo"),
		}, secret.Data)
		require.True(t, hasLatestGeneration(obj))
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Normal CreatedOrUpdatedAtAiven instance was created at aiven but may not be running yet",
		}, recorderEvents(recorder))
	})
}

func TestReconciler_updateResource(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Requeues when remote resource is not found", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		errNotFound := NewNotFound("instance not found")

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, errNotFound).Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})

	t.Run("Returns error and emits warning when update fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, assert.AnError).Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.EqualError(t, err, fmt.Sprintf("unable to wait until instance is running: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
			"Warning UnableToWaitForInstanceToBeRunning " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Returns error when publishing secret details fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					args := m.MethodCalled("Create", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{SecretDetails: map[string]string{"BAR": "bar"}}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.EqualError(t, err, fmt.Sprintf("unable to sync connection secret: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)

		secret := &corev1.Secret{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))

		require.Equal(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionConnInfoSecret),
				Message: assert.AnError.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
			"Warning CannotPublishConnectionDetails " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Updates remote resource and publishes secrets", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{SecretDetails: map[string]string{"BAR": "bar"}}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: pollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret))
		require.Equal(t, map[string][]byte{
			"BAR": []byte("bar"),
		}, secret.Data)
		require.True(t, hasLatestGeneration(obj))
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})
}

type testManagedWithoutSecretTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status struct {
		Conditions []metav1.Condition `json:"conditions,omitempty"`
	}
}

var _ v1alpha1.AivenManagedObject = &testManagedWithoutSecretTarget{}

func (t *testManagedWithoutSecretTarget) DeepCopyObject() runtime.Object {
	panic("not implemented")
}

func (t *testManagedWithoutSecretTarget) AuthSecretRef() *v1alpha1.AuthSecretReference {
	panic("not implemented")
}

func (t *testManagedWithoutSecretTarget) Conditions() *[]metav1.Condition {
	panic("not implemented")
}

func (t *testManagedWithoutSecretTarget) GetObjectMeta() *metav1.ObjectMeta {
	panic("not implemented")
}

func (t *testManagedWithoutSecretTarget) NoSecret() bool {
	return false
}

func TestReconciler_publishSecretDetails(t *testing.T) {
	t.Parallel()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("No-op when details are empty", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}
		err := r.publishSecretDetails(t.Context(), nil, nil)
		require.NoError(t, err)
	})

	t.Run("Skips secret and emits event when NoSecret is true", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Spec.ConnInfoSecretTargetDisabled = ptr(true)

		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		err := r.publishSecretDetails(t.Context(), obj, map[string]string{"FOO": "foo"})

		require.NoError(t, err)
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal ConnInfoSecretCreationDisabled connection info secret creation is disabled for this resource, secret won't be changed (check spec.connInfoSecretTargetDisabled)",
		}, recorderEvents(recorder))
	})

	t.Run("Logs warning when object does not implement secret target", func(t *testing.T) {
		obj := &testManagedWithoutSecretTarget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-object",
				Namespace: "default",
			},
		}

		r := &Reconciler[*testManagedWithoutSecretTarget]{}

		sink := &logRecorderSink{}
		ctx := logr.NewContext(t.Context(), logr.New(sink))

		err := r.publishSecretDetails(ctx, obj, map[string]string{"FOO": "foo"})

		require.NoError(t, err)
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Contains(t, sink.logs, "object does not implement conn info secret target, skipping connection secret publish")
	})

	t.Run("Returns error and emits warning when secret sync fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					args := m.MethodCalled("Create", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		err := r.publishSecretDetails(t.Context(), obj, map[string]string{"FOO": "foo"})
		require.EqualError(t, err, fmt.Sprintf("unable to sync connection secret: %s", assert.AnError.Error()))

		secret := &corev1.Secret{}
		err = k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))

		require.Equal(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionConnInfoSecret),
				Message: assert.AnError.Error(),
			},
		}, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Warning CannotPublishConnectionDetails " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Creates connection secret with target metadata", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Spec.ConnInfoSecretTarget = v1alpha1.ConnInfoSecretTarget{
			Name: "custom-secret",
			Annotations: map[string]string{
				"foo": "bar",
			},
			Labels: map[string]string{
				"app": "test",
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
			newSecret: newSecret,
		}

		err := r.publishSecretDetails(t.Context(), obj, map[string]string{
			"HOST": "localhost",
			"PORT": "5432",
		})
		require.NoError(t, err)

		secret := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "custom-secret", Namespace: obj.Namespace}, secret))
		require.Equal(t, map[string][]byte{
			"HOST": []byte("localhost"),
			"PORT": []byte("5432"),
		}, secret.Data)
		require.Equal(t, map[string]string{"app": "test"}, secret.Labels)
		require.Equal(t, map[string]string{"foo": "bar"}, secret.Annotations)
		require.True(t, slices.ContainsFunc(secret.OwnerReferences, func(ref metav1.OwnerReference) bool {
			return ref.Name == obj.Name
		}), "secret should have owner reference to ClickhouseUser")

		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Empty(t, recorderEvents(recorder))
	})

	t.Run("Merges Data and StringData from goal secret", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
			newSecret: newSecret,
		}

		m := &mock.Mock{}
		t.Cleanup(func() {
			m.AssertExpectations(t)
		})

		r.newSecret = func(o objWithSecret, details map[string]string, addPrefix bool) *corev1.Secret {
			args := m.MethodCalled("newConnInfoSecret", o, details, addPrefix)
			return args.Get(0).(*corev1.Secret)
		}

		m.On("newConnInfoSecret", mock.Anything, mock.Anything, mock.Anything).Return(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      obj.Name,
				Namespace: obj.Namespace,
			},
			Data: map[string][]byte{
				"BIN": []byte("bin"),
			},
			StringData: map[string]string{
				"TXT": "txt",
			},
		}).Once()

		require.NoError(t, r.publishSecretDetails(t.Context(), obj, map[string]string{"IGNORED": "ignored"}))

		secret := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, secret))
		require.Equal(t, map[string][]byte{
			"BIN": []byte("bin"),
			"TXT": []byte("txt"),
		}, secret.Data)
	})

	t.Run("Known types with conn info secret target don't trigger fallback log", func(t *testing.T) {
		cases := []v1alpha1.AivenManagedObject{
			&v1alpha1.MySQL{},
			&v1alpha1.Project{},
			&v1alpha1.Clickhouse{},
			&v1alpha1.ConnectionPool{},
			&v1alpha1.Flink{},
			&v1alpha1.ServiceUser{},
			&v1alpha1.Cassandra{},
			&v1alpha1.OpenSearch{},
			&v1alpha1.Kafka{},
			&v1alpha1.Valkey{},
			&v1alpha1.Grafana{},
			&v1alpha1.PostgreSQL{},
			&v1alpha1.ClickhouseUser{},
			&v1alpha1.AlloyDBOmni{},
		}

		for _, obj := range cases {
			typeName := reflect.TypeOf(obj).Elem().Name()

			t.Run(typeName, func(t *testing.T) {
				obj.GetObjectMeta().Name = "test-" + typeName
				obj.GetObjectMeta().Namespace = "default"

				k8sClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(obj.(crclient.Object)).
					Build()
				events := record.NewFakeRecorder(10)

				sink := &logRecorderSink{}
				logger := logr.New(sink)
				ctx := logr.NewContext(t.Context(), logger)

				r := newTestReconciler(obj, k8sClient, scheme, events)
				require.NoError(t, r.publishSecretDetails(ctx, obj, map[string]string{"FOO": "foo"}))

				require.NotContains(t, sink.logs, "object does not implement conn info secret target, skipping connection secret publish")
			})
		}
	})
}

func TestReconciler_finalize(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("No-op when instance deletion finalizer is missing", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		res, err := r.finalize(t.Context(), nil, obj)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)
	})

	t.Run("Deletes remote resource and removes finalizer on success", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
			},
		}

		res, err := r.finalize(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Empty(t, got.Finalizers)
		require.Empty(t, normalizedConditions(got.Status.Conditions))
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Normal SuccessfullyDeletedAtAiven instance is gone at aiven now",
		}, recorderEvents(recorder))
	})

	t.Run("Removes finalizer on invalid token error", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		deleteErr := fmt.Errorf("Invalid token")

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(deleteErr).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Recorder: recorder,
			},
		}

		res, err := r.finalize(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Empty(t, got.Finalizers)
		require.Empty(t, normalizedConditions(obj.Status.Conditions))
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Normal SuccessfullyDeletedAtAiven instance is gone at aiven now",
		}, recorderEvents(recorder))
	})

	t.Run("Delegates delete errors to handleDeleteError", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Recorder: recorder,
			},
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(assert.AnError).Once()

		res, err := r.finalize(t.Context(), c, obj)

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, `unable to delete instance: `+assert.AnError.Error())
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Warning UnableToDelete " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})

	t.Run("Skips remote deletion when deletion policy is Orphan", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}
		obj.Annotations = map[string]string{
			deletionPolicyAnnotation: deletionPolicyOrphan,
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		res, err := r.finalize(t.Context(), nil, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Empty(t, got.Finalizers)
		require.Empty(t, normalizedConditions(got.Status.Conditions))
	})

	t.Run("Returns error when deletion policy is invalid", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}
		obj.Annotations = map[string]string{
			deletionPolicyAnnotation: "invalid",
		}

		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		res, err := r.finalize(t.Context(), nil, obj)
		require.EqualError(t, err, `unable to delete instance: invalid deletion policy "invalid", only "Orphan" is allowed`)
		require.Equal(t, ctrl.Result{}, res)
		require.ElementsMatch(t, []metav1.Condition{
			{
				Type:    ConditionTypeError,
				Status:  metav1.ConditionUnknown,
				Reason:  string(errConditionDelete),
				Message: `invalid deletion policy "invalid", only "Orphan" is allowed`,
			},
		}, normalizedConditions(obj.Status.Conditions))
	})

	t.Run("Returns error when removing finalizer fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		// Intercept Update calls to simulate a failure when removing the finalizer.
		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					args := m.MethodCalled("Update", ctx, c, o, opts)
					return args.Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Recorder: recorder,
			},
		}

		res, err := r.finalize(t.Context(), c, obj)
		require.EqualError(t, err, fmt.Sprintf("unable to remove finalizer: %s", assert.AnError.Error()))
		require.Equal(t, ctrl.Result{}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Equal(t, []string{instanceDeletionFinalizer}, got.Finalizers)
		require.Empty(t, normalizedConditions(got.Status.Conditions))
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Normal SuccessfullyDeletedAtAiven instance is gone at aiven now",
			"Warning UnableToDeleteFinalizer " + assert.AnError.Error(),
		}, recorderEvents(recorder))
	})
}

func TestReconciler_handleDeleteError(t *testing.T) {
	t.Parallel()

	errDeps := fmt.Errorf("%w: underlying error", v1alpha1.ErrDeleteDependencies)
	errNotFound := NewNotFound("instance not found")

	cases := []struct {
		deletionError error
		result        ctrl.Result
		err           error
		events        []string
	}{
		{
			deletionError: fmt.Errorf("%w: underlying error", errDeps),
			result:        ctrl.Result{RequeueAfter: requeueTimeout},
			err:           nil,
			events:        nil,
		},
		{
			deletionError: errNotFound,
			result:        ctrl.Result{},
			err:           fmt.Errorf("unable to delete instance at aiven: %w", errNotFound),
			events: []string{
				"Warning UnableToDeleteAtAiven " + errNotFound.Error(),
			},
		},
		{
			deletionError: newAivenError(500, "internal error"),
			result:        ctrl.Result{RequeueAfter: requeueTimeout},
			err:           nil,
			events:        nil,
		},
		{
			deletionError: assert.AnError,
			result:        ctrl.Result{},
			err:           fmt.Errorf("unable to delete instance: %w", assert.AnError),
			events:        []string{"Warning UnableToDelete " + assert.AnError.Error()},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("[%d] %s", i, c.deletionError.Error()), func(t *testing.T) {
			obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
			recorder := record.NewFakeRecorder(10)
			r := &Reconciler[*v1alpha1.ClickhouseUser]{
				Controller: Controller{
					Recorder: recorder,
				},
			}
			res, err := r.handleDeleteError(t.Context(), obj, c.deletionError)

			require.Equal(t, c.err, err)
			require.Equal(t, c.result, res)
			require.ElementsMatch(t, []metav1.Condition{
				{
					Type:    ConditionTypeError,
					Status:  metav1.ConditionUnknown,
					Reason:  string(errConditionDelete),
					Message: c.deletionError.Error(),
				},
			}, normalizedConditions(obj.Status.Conditions))
			require.Equal(t, c.events, recorderEvents(recorder))
		})
	}
}

func TestReconciler_SetupWithManager(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	restMapper := meta.NewDefaultRESTMapper(nil)
	restMapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
	restMapper.Add(v1alpha1.GroupVersion.WithKind("ClickhouseUser"), meta.RESTScopeNamespace)

	cfg := &rest.Config{
		Host: "https://127.0.0.1",
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
		PprofBindAddress:       "0",
		LeaderElection:         false,
		MapperProvider: func(*rest.Config, *http.Client) (meta.RESTMapper, error) {
			return restMapper, nil
		},
		NewClient: func(*rest.Config, crclient.Options) (crclient.Client, error) {
			return fake.NewClientBuilder().WithScheme(scheme).Build(), nil
		},
	})
	require.NoError(t, err)

	recorder := record.NewFakeRecorder(10)

	r := &Reconciler[*v1alpha1.ClickhouseUser]{
		Controller: Controller{
			Client:   mgr.GetClient(),
			Scheme:   mgr.GetScheme(),
			Recorder: recorder,
		},
		newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
	}

	err = r.SetupWithManager(mgr)
	require.NoError(t, err)
}
