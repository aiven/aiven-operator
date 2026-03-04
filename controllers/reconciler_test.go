package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

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
	ctrlcontroller "sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
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
			Client:       client,
			Scheme:       scheme,
			Recorder:     recorder,
			PollInterval: testPollInterval,
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
	testPollInterval = time.Minute

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

	t.Run("No-op if managed resource is not found", func(t *testing.T) {
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

	t.Run("Requeues when resolving refs fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				*args.Get(3).(*v1alpha1.PostgreSQL) = *obj.DeepCopy()
			}).
			Return(nil).Once()
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(nil).Once()

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
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
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

		gvk := v1alpha1.GroupVersion.WithKind("ProjectVPC")
		_, underlying := scheme.New(gvk)
		require.Error(t, underlying)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Warning UnableToWaitForPreconditions " + fmt.Sprintf("creating %s: %s", gvk, underlying.Error()),
		}, recorderEvents(recorder))
	})

	t.Run("Requests requeue when refs are not ready", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.PostgreSQL{}).
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

		got := &v1alpha1.PostgreSQL{}
		require.NoError(t, k8sClient.Get(ctx, nn, got))
		synced := meta.FindStatusCondition(got.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, "waiting for referenced resources to be ready", synced.Message)
	})

	t.Run("Requeues when creating Aiven client fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return((*struct{ avngen.Client })(nil), assert.AnError).
			Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
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

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Warning UnableToCreateClient " + assert.AnError.Error(),
		}, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		synced := meta.FindStatusCondition(got.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, fmt.Sprintf("cannot initialize aiven generated client: %s", assert.AnError.Error()), synced.Message)
	})

	t.Run("Returns error when persisting instance finalizer fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once().
			On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return(avngen.NewMockClient(t), nil).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
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

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, assert.AnError.Error())
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
		}, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.NotContains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Silently requeues when instance finalizer update conflicts", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once().
			On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return(avngen.NewMockClient(t), nil).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
		}, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.NotContains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Empty(t, normalizedConditions(got.Status.Conditions))
	})

	t.Run("Requeues with synced error when protecting auth secret fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj, secret).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		ctx := t.Context()
		res, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Empty(t, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		synced := meta.FindStatusCondition(got.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, fmt.Sprintf("unable to add finalizer to secret: %s", assert.AnError.Error()), synced.Message)
	})

	t.Run("Silently requeues when auth secret protection conflicts", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "", Resource: "secrets"},
			secret.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		// addFinalizer uses RetryOnConflict, so Update may be called multiple times.
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Times(5)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj, secret).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		ctx := t.Context()
		res, err := r.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace},
		})

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Empty(t, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Empty(t, normalizedConditions(got.Status.Conditions))
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
				PollInterval:    testPollInterval,
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

	t.Run("Requeues with Observe condition on observe error", func(t *testing.T) {
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
			Return(Observation{}, fmt.Errorf("boom")).
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
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			"Warning UnableToObserve boom",
		}, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		cond := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError)
		require.NotNil(t, cond)
		require.Equal(t, string(errConditionObserve), cond.Reason)
		require.Equal(t, "boom", cond.Message)
	})

	t.Run("Requeues when publishing observe secret details fails", func(t *testing.T) {
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
				PollInterval:    testPollInterval,
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

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			fmt.Sprintf("Warning CannotPublishConnectionDetails %s", assert.AnError.Error()),
		}, recorderEvents(recorder))

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		cond := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError)
		require.NotNil(t, cond)
		require.Equal(t, string(errConditionConnInfoSecret), cond.Reason)
		require.Equal(t, assert.AnError.Error(), cond.Message)
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
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
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
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, []string{
			"Normal InstanceFinalizerAdded instance finalizer added",
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})

	t.Run("Requeues when resource is up to date", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		// Clear processedGenerationAnnotation to verify that steady-state reconcile doesn't write object metadata.
		delete(obj.GetAnnotations(), processedGenerationAnnotation)
		obj.Annotations = map[string]string{instanceIsRunningAnnotation: "true"}

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
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Empty(t, got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Persists observe annotation changes when resource is up to date", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, o *v1alpha1.ClickhouseUser) (Observation, error) {
				metav1.SetMetaDataAnnotation(o.GetObjectMeta(), "test.aiven.io/observed", "true")
				return Observation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				}, nil
			}).
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
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Equal(t, "true", got.GetAnnotations()["test.aiven.io/observed"])
	})

	t.Run("Silently requeues when persisting observe annotation changes conflicts", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once().
			On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").Return(avngen.NewMockClient(t), nil).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, o *v1alpha1.ClickhouseUser) (Observation, error) {
				metav1.SetMetaDataAnnotation(o.GetObjectMeta(), "test.aiven.io/observed", "true")
				return Observation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				}, nil
			}).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
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
		require.Equal(t, ctrl.Result{Requeue: true}, res)
	})

	t.Run("Returns error when persisting observe annotation changes fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()
		m.On("newAivenGeneratedClient", "default-token", "v1.30.0", "v0.0.0-test").
			Return(avngen.NewMockClient(t), nil).
			Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Observe(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, o *v1alpha1.ClickhouseUser) (Observation, error) {
				metav1.SetMetaDataAnnotation(o.GetObjectMeta(), "test.aiven.io/observed", "true")
				return Observation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				}, nil
			}).
			Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:          k8sClient,
				Scheme:          scheme,
				Recorder:        recorder,
				DefaultToken:    "default-token",
				KubeVersion:     "v1.30.0",
				OperatorVersion: "v0.0.0-test",
				PollInterval:    testPollInterval,
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

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("Silently requeues on conflict errors", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				*args.Get(3).(*v1alpha1.PostgreSQL) = *obj.DeepCopy()
			}).
			Return(nil).Once()

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "postgresqls"},
			obj.Name,
			fmt.Errorf("conflict"),
		)
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(conflictErr).Once()

		// Intentionally do NOT register v1alpha1 types to force Scheme.New to fail.
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, c crclient.WithWatch, key crclient.ObjectKey, o crclient.Object, opts ...crclient.GetOption) error {
					args := m.MethodCalled("Get", ctx, c, key, o, opts)
					return args.Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					args := m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts)
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

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
	})
}

func TestReconciler_handleObserveError(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Propagates status update error for ErrRequeueNeeded", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		sentinelErr := errors.New("sentinel status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Recorder: recorder,
			},
		}

		origErr := errPreconditionNotMet
		res, err := r.handleObserveError(t.Context(), obj, ErrRequeueNeeded{OriginalError: origErr})

		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal WaitingForPreconditions " + origErr.Error(),
		}, recorderEvents(recorder))

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, origErr.Error(), synced.Message)
	})

	t.Run("Propagates status update error for generic observe error", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		sentinelErr := errors.New("sentinel status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Recorder: recorder,
			},
		}

		origErr := errors.New("boom")
		res, err := r.handleObserveError(t.Context(), obj, origErr)

		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Warning UnableToObserve " + origErr.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, metav1.ConditionUnknown, observeErr.Status)
		require.Equal(t, string(errConditionObserve), observeErr.Reason)
		require.Equal(t, origErr.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, origErr.Error(), synced.Message)
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

	t.Run("Error if dependency Get returns non-not-found error", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgresWithRef)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(context.Context, crclient.WithWatch, crclient.ObjectKey, crclient.Object, ...crclient.GetOption) error {
					return errors.New("boom")
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.PostgreSQL]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		requeue, err := r.resolveK8sRefs(t.Context(), obj)

		require.EqualError(t, err, `getting referenced resource aiven.io/v1alpha1, Kind=ProjectVPC default/test-vpc: boom`)
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

	t.Run("Returns nil when status update succeeds", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Status.UUID = "uuid-after"

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

		err := r.updateStatus(t.Context(), obj)
		require.NoError(t, err)

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))
		require.Equal(t, "uuid-after", got.Status.UUID)
	})

	t.Run("Returns nil on not found when object is being deleted", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.DeletionTimestamp = ptr(metav1.Now())
		obj.Status.UUID = "uuid-after"

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateStatus(t.Context(), obj)
		require.NoError(t, err)
	})

	t.Run("Returns not found on not deleting object", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Status.UUID = "uuid-after"

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateStatus(t.Context(), obj)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Propagates conflict error from status update", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Status.UUID = "uuid-after"

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(conflictErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateStatus(t.Context(), obj)
		require.ErrorIs(t, err, conflictErr)
	})
}

func TestReconciler_updateObject(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("Returns nil and updates resourceVersion on success", func(t *testing.T) {
		stored := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(stored).
			Build()

		obj := &v1alpha1.ClickhouseUser{}
		key := types.NamespacedName{Name: stored.Name, Namespace: stored.Namespace}
		require.NoError(t, k8sClient.Get(t.Context(), key, obj))

		origRV := obj.GetResourceVersion()
		metav1.SetMetaDataAnnotation(obj.GetObjectMeta(), "test.aiven.io/updated", "true")

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateObject(t.Context(), obj)
		require.NoError(t, err)
		require.NotEmpty(t, obj.GetResourceVersion())
		require.NotEqual(t, origRV, obj.GetResourceVersion())

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), key, got))
		require.Equal(t, "true", got.GetAnnotations()["test.aiven.io/updated"])
	})

	t.Run("Returns nil on not found when object is being deleted", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.DeletionTimestamp = ptr(metav1.Now())

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateObject(t.Context(), obj)
		require.NoError(t, err)
	})

	t.Run("Returns not found on not deleting object", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateObject(t.Context(), obj)
		require.Error(t, err)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Propagates conflict error from object update", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()

		got := &v1alpha1.ClickhouseUser{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, got))

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}

		err := r.updateObject(t.Context(), got)
		require.ErrorIs(t, err, conflictErr)
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
		createErr := errors.New("create failed")
		sentinelErr := errors.New("test status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Create(mock.Anything, mock.Anything).Return(CreateResult{}, createErr).Once()

		res, err := r.createResource(t.Context(), c, obj)

		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Warning UnableToCreateOrUpdateAtAiven " + createErr.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, string(errConditionCreateOrUpdate), observeErr.Reason)
		require.Equal(t, createErr.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, createErr.Error(), synced.Message)
	})

	t.Run("Returns error when publishing secret details fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		sentinelErr := errors.New("test status update error")
		publishError := errors.New("secret create failed")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(publishError).Once()
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					return m.MethodCalled("Create", ctx, c, o, opts).Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(CreateResult{SecretDetails: map[string]string{"FOO": "foo"}}, nil).
			Once()

		res, err := r.createResource(t.Context(), c, obj)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Warning CannotPublishConnectionDetails " + publishError.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, string(errConditionConnInfoSecret), observeErr.Reason)
		require.Equal(t, publishError.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, fmt.Sprintf("unable to sync connection secret: %s", publishError.Error()), synced.Message)
	})

	t.Run("Requeues when persisting annotations after create conflicts", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, o *v1alpha1.ClickhouseUser) (CreateResult, error) {
				metav1.SetMetaDataAnnotation(o.GetObjectMeta(), "test.aiven.io/create-mutated", "true")
				return CreateResult{}, nil
			}).
			Once()

		res, err := r.createResource(t.Context(), c, obj)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
		}, recorderEvents(recorder))
	})

	t.Run("Propagates status update error when persisting annotations after create fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		sentinelErr := errors.New("test status update error")
		persistErr := errors.New("persist annotations failed")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(persistErr).Once()
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			RunAndReturn(func(_ context.Context, o *v1alpha1.ClickhouseUser) (CreateResult, error) {
				metav1.SetMetaDataAnnotation(o.GetObjectMeta(), "test.aiven.io/create-mutated", "true")
				return CreateResult{}, nil
			}).
			Once()

		res, err := r.createResource(t.Context(), c, obj)

		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal CreateOrUpdatedAtAiven about to create instance at aiven",
			"Warning UnableToCreateOrUpdateAtAiven " + persistErr.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, string(errConditionCreateOrUpdate), observeErr.Reason)
		require.Equal(t, persistErr.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, persistErr.Error(), synced.Message)
	})

	t.Run("Requeues and sets synced success when create succeeds", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Create(mock.Anything, mock.Anything).
			Return(CreateResult{}, nil).
			Once()

		res, err := r.createResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionTrue, synced.Status)
		require.Equal(t, reconcileReasonSuccess, synced.Reason)
		require.Equal(t, "Successfully reconciled resource", synced.Message)
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError))

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

	t.Run("Returns update error when update fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		updateErr := errors.New("update failed")
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, updateErr).Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, "unable to wait until instance is running: update failed")
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
			"Warning UnableToWaitForInstanceToBeRunning " + updateErr.Error(),
		}, recorderEvents(recorder))

		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError))
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced))
	})

	t.Run("Does not persist status when update fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		updateErr := errors.New("update failed")
		statusUpdated := false

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(context.Context, crclient.Client, string, crclient.Object, ...crclient.SubResourceUpdateOption) error {
					statusUpdated = true
					return nil
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, updateErr).Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, "unable to wait until instance is running: update failed")
		require.False(t, statusUpdated)
	})

	t.Run("Soft requeues when update returns not found", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		updateErr := newAivenError(404, "not found")

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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, updateErr).Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError))
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced))
	})

	t.Run("Does not persist status when update returns not found", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		updateErr := newAivenError(404, "not found")
		statusUpdated := false

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(context.Context, crclient.Client, string, crclient.Object, ...crclient.SubResourceUpdateOption) error {
					statusUpdated = true
					return nil
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
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Update(mock.Anything, mock.Anything).Return(UpdateResult{}, updateErr).Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.False(t, statusUpdated)
	})

	t.Run("Propagates status update error when persisting updated object fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Generation = 1
		persistErr := errors.New("persist failed")
		sentinelErr := errors.New("test status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(persistErr).Once()
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
			"Warning UnableToCreateOrUpdateAtAiven " + persistErr.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, string(errConditionCreateOrUpdate), observeErr.Reason)
		require.Equal(t, persistErr.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, persistErr.Error(), synced.Message)
	})

	t.Run("Propagates status update error when publishing secret details fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		sentinelErr := errors.New("test status update error")
		publishError := errors.New("secret create failed")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(publishError).Once()
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Create: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.CreateOption) error {
					return m.MethodCalled("Create", ctx, c, o, opts).Error(0)
				},
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{SecretDetails: map[string]string{"FOO": "foo"}}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
			"Warning CannotPublishConnectionDetails " + publishError.Error(),
		}, recorderEvents(recorder))

		observeErr := meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError)
		require.NotNil(t, observeErr)
		require.Equal(t, string(errConditionConnInfoSecret), observeErr.Reason)
		require.Equal(t, publishError.Error(), observeErr.Message)

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionFalse, synced.Status)
		require.Equal(t, reconcileReasonError, synced.Reason)
		require.Equal(t, fmt.Sprintf("unable to sync connection secret: %s", publishError.Error()), synced.Message)
	})

	t.Run("Requeues when persisting processed generation conflicts", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Generation = 1
		conflictErr := apierrors.NewConflict(
			schema.GroupResource{Group: "aiven.io", Resource: "clickhouseusers"},
			obj.Name,
			fmt.Errorf("conflict"),
		)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(conflictErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)

		require.NoError(t, err)
		require.Equal(t, ctrl.Result{Requeue: true}, res)
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})

	t.Run("Requeues with poll interval when update succeeds and resource is ready", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Generation = 1
		obj.Annotations = map[string]string{
			instanceIsRunningAnnotation: "true",
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: testPollInterval}, res)
		require.Equal(t, "1", obj.GetAnnotations()[processedGenerationAnnotation])

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionTrue, synced.Status)
		require.Equal(t, reconcileReasonSuccess, synced.Reason)
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})

	t.Run("Requeues soon when update succeeds and resource is not ready", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Generation = 1

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				PollInterval: testPollInterval,
			},
			newSecret: newSecret,
		}

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().
			Update(mock.Anything, mock.Anything).
			Return(UpdateResult{}, nil).
			Once()

		res, err := r.updateResource(t.Context(), c, obj)
		require.NoError(t, err)
		require.Equal(t, ctrl.Result{RequeueAfter: requeueTimeout}, res)
		require.Equal(t, "1", obj.GetAnnotations()[processedGenerationAnnotation])

		synced := meta.FindStatusCondition(obj.Status.Conditions, conditionTypeSynced)
		require.NotNil(t, synced)
		require.Equal(t, metav1.ConditionTrue, synced.Status)
		require.Equal(t, reconcileReasonSuccess, synced.Reason)
		require.Nil(t, meta.FindStatusCondition(obj.Status.Conditions, ConditionTypeError))
		require.Equal(t, []string{
			"Normal WaitingForInstanceToBeRunning waiting for the instance to be running",
		}, recorderEvents(recorder))
	})
}

func TestReconciler_ensureAuthSecretFinalizer(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("No-op when default token is configured", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				DefaultToken: "default-token",
			},
		}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))
	})

	t.Run("No-op when auth secret reference is not configured", func(t *testing.T) {
		r := &Reconciler[*v1alpha1.ClickhouseUser]{}

		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))
	})

	t.Run("No-op when auth secret already has protection finalizer", func(t *testing.T) {
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)
		secret.Finalizers = []string{secretProtectionFinalizer}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))

		got := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "aiven-token", Namespace: "default"}, got))
		require.Equal(t, []string{secretProtectionFinalizer}, got.Finalizers)
	})

	t.Run("Adds protection finalizer to auth secret", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))

		got := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "aiven-token", Namespace: "default"}, got))
		require.Contains(t, got.Finalizers, secretProtectionFinalizer)
	})

	t.Run("Deletion path ignores auth secret protection update errors", func(t *testing.T) {
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		obj.DeletionTimestamp = ptr(metav1.Now())

		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))

		got := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "aiven-token", Namespace: "default"}, got))
		require.NotContains(t, got.Finalizers, secretProtectionFinalizer)
	})

	t.Run("Returns wrapped error when adding auth secret finalizer fails", func(t *testing.T) {
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)

		err := r.ensureAuthSecretFinalizer(t.Context(), obj)
		require.EqualError(t, err, fmt.Sprintf("unable to add finalizer to secret: %s", assert.AnError.Error()))

		got := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "aiven-token", Namespace: "default"}, got))
		require.NotContains(t, got.Finalizers, secretProtectionFinalizer)
	})

	t.Run("Deletion path is best-effort when auth secret is terminating", func(t *testing.T) {
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)
		secret.DeletionTimestamp = ptr(metav1.Now())
		secret.Finalizers = []string{"example.com/existing"}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(secret).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
			},
		}
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		obj.DeletionTimestamp = ptr(metav1.Now())

		require.NoError(t, r.ensureAuthSecretFinalizer(t.Context(), obj))

		got := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: "aiven-token", Namespace: "default"}, got))
		require.NotContains(t, got.Finalizers, secretProtectionFinalizer)
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

		res, err := r.finalize(t.Context(), obj)

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
				// Avoid auth secret resolution in tests.
				DefaultToken: "default-token",
			},
			newAivenGeneratedClient: func(string, string, string) (avngen.Client, error) { return nil, nil },
			newController:           func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] { return c },
		}

		res, err := r.finalize(t.Context(), obj)
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
				// Avoid auth secret resolution in tests.
				DefaultToken: "default-token",
			},
			newAivenGeneratedClient: func(string, string, string) (avngen.Client, error) { return nil, nil },
			newController:           func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] { return c },
		}

		res, err := r.finalize(t.Context(), obj)
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

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(assert.AnError).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: recorder,
				// Avoid auth secret resolution in tests.
				DefaultToken: "default-token",
			},
			newAivenGeneratedClient: func(string, string, string) (avngen.Client, error) { return nil, nil },
			newController:           func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] { return c },
		}

		// Mimic real reconcile - ensure resourceVersion is set.
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj))

		res, err := r.finalize(t.Context(), obj)

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

		res, err := r.finalize(t.Context(), obj)
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

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		// Mimic real reconcile - ensure resourceVersion is set.
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj))

		res, err := r.finalize(t.Context(), obj)
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

	t.Run("Returns status update error when persisting invalid deletion policy condition fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}
		obj.Annotations = map[string]string{
			deletionPolicyAnnotation: "invalid",
		}
		sentinelErr := errors.New("test status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		res, err := r.finalize(t.Context(), obj)
		require.Equal(t, ctrl.Result{}, res)
		require.ErrorIs(t, err, sentinelErr)
	})

	t.Run("Returns error when ensuring auth secret finalizer fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
		obj.Finalizers = []string{instanceDeletionFinalizer}
		secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj, secret).
			WithInterceptorFuncs(interceptor.Funcs{
				Update: func(ctx context.Context, c crclient.WithWatch, o crclient.Object, opts ...crclient.UpdateOption) error {
					return m.MethodCalled("Update", ctx, c, o, opts).Error(0)
				},
			}).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		res, err := r.finalize(t.Context(), obj)
		require.Equal(t, ctrl.Result{}, res)
		require.EqualError(t, err, fmt.Sprintf("unable to add finalizer to secret: %s", assert.AnError.Error()))
	})

	t.Run("Returns error when creating Aiven client fails during finalize", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(obj).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client: k8sClient,
				Scheme: scheme,
			},
		}

		res, err := r.finalize(t.Context(), obj)
		require.Equal(t, ctrl.Result{}, res)
		require.ErrorIs(t, err, errNoTokenProvided)
	})

	t.Run("Returns status update error when persisting delete failure condition fails", func(t *testing.T) {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Finalizers = []string{instanceDeletionFinalizer}
		sentinelErr := errors.New("test status update error")

		m := &mock.Mock{}
		t.Cleanup(func() { m.AssertExpectations(t) })
		m.On("SubResourceUpdate", mock.Anything, mock.Anything, "status", mock.Anything, mock.Anything).Return(sentinelErr).Once()

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.ClickhouseUser{}).
			WithObjects(obj).
			WithInterceptorFuncs(interceptor.Funcs{
				SubResourceUpdate: func(ctx context.Context, c crclient.Client, subResourceName string, o crclient.Object, opts ...crclient.SubResourceUpdateOption) error {
					return m.MethodCalled("SubResourceUpdate", ctx, c, subResourceName, o, opts).Error(0)
				},
			}).
			Build()
		recorder := record.NewFakeRecorder(10)

		c := NewMockAivenController[*v1alpha1.ClickhouseUser](t)
		c.EXPECT().Delete(mock.Anything, mock.Anything).Return(assert.AnError).Once()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:       k8sClient,
				Scheme:       scheme,
				Recorder:     recorder,
				DefaultToken: "default-token",
			},
			newAivenGeneratedClient: func(string, string, string) (avngen.Client, error) { return nil, nil },
			newController:           func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] { return c },
		}

		res, err := r.finalize(t.Context(), obj)
		require.Equal(t, ctrl.Result{}, res)
		require.ErrorIs(t, err, sentinelErr)
		require.Equal(t, []string{
			"Normal TryingToDeleteAtAiven trying to delete instance at aiven",
			"Warning UnableToDelete " + assert.AnError.Error(),
		}, recorderEvents(recorder))
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
				// Avoid auth secret resolution in tests.
				DefaultToken: "default-token",
			},
			newAivenGeneratedClient: func(string, string, string) (avngen.Client, error) { return nil, nil },
			newController:           func(avngen.Client) AivenController[*v1alpha1.ClickhouseUser] { return c },
		}

		res, err := r.finalize(t.Context(), obj)
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

func TestDeletionTimestampChanged(t *testing.T) {
	t.Parallel()

	makeObj := func(withDeletionTimestamp bool, ts time.Time) *v1alpha1.ClickhouseUser {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		if withDeletionTimestamp {
			obj.DeletionTimestamp = ptr(metav1.NewTime(ts))
		}
		return obj
	}

	ts1 := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 3, 4, 10, 1, 0, 0, time.UTC)

	tests := []struct {
		oldObj crclient.Object
		newObj crclient.Object
		want   bool
	}{
		{oldObj: nil, newObj: makeObj(false, ts1), want: false},
		{oldObj: makeObj(false, ts1), newObj: nil, want: false},
		{oldObj: makeObj(false, ts1), newObj: makeObj(false, ts1), want: false},
		{oldObj: makeObj(false, ts1), newObj: makeObj(true, ts1), want: true},
		{oldObj: makeObj(true, ts1), newObj: makeObj(false, ts1), want: true},
		{oldObj: makeObj(true, ts1), newObj: makeObj(true, ts1), want: false},
		{oldObj: makeObj(true, ts1), newObj: makeObj(true, ts2), want: true},
	}

	p := deletionTimestampChanged()
	for i, tt := range tests {
		t.Run(fmt.Sprintf("[%d]", i), func(t *testing.T) {
			got := p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObj,
				ObjectNew: tt.newObj,
			})
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAnnotationsChangedExcluding(t *testing.T) {
	t.Parallel()

	makeObj := func(annotations map[string]string) *v1alpha1.ClickhouseUser {
		obj := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		obj.Annotations = annotations
		return obj
	}

	tests := []struct {
		oldObj crclient.Object
		newObj crclient.Object
		keys   []string
		want   bool
	}{
		{oldObj: nil, newObj: makeObj(map[string]string{"foo": "bar"}), keys: []string{"ignored"}, want: false},
		{oldObj: makeObj(map[string]string{"foo": "bar"}), newObj: nil, keys: []string{"ignored"}, want: false},
		{oldObj: makeObj(map[string]string{"foo": "bar"}), newObj: makeObj(map[string]string{"foo": "bar"}), keys: []string{"ignored"}, want: false},
		{oldObj: makeObj(map[string]string{"ignored": "old", "foo": "bar"}), newObj: makeObj(map[string]string{"ignored": "new", "foo": "bar"}), keys: []string{"ignored"}, want: false},
		{oldObj: makeObj(map[string]string{"foo": "old"}), newObj: makeObj(map[string]string{"foo": "new"}), keys: []string{"ignored"}, want: true},
		{oldObj: makeObj(map[string]string{"ignored": "same"}), newObj: makeObj(map[string]string{"ignored": "same", "foo": "bar"}), keys: []string{"ignored"}, want: true},
		{oldObj: makeObj(map[string]string{"ignored": "same", "foo": "bar"}), newObj: makeObj(map[string]string{"ignored": "same"}), keys: []string{"ignored"}, want: true},
		{oldObj: makeObj(nil), newObj: makeObj(map[string]string{"ignored": "new"}), keys: []string{"ignored"}, want: false},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("[%d]", i), func(t *testing.T) {
			p := annotationsChangedExcluding(tt.keys...)
			got := p.Update(event.UpdateEvent{
				ObjectOld: tt.oldObj,
				ObjectNew: tt.newObj,
			})
			require.Equal(t, tt.want, got)
		})
	}
}

func TestReconciler_SetupWithManager(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	cfg := &rest.Config{
		Host: "https://127.0.0.1",
	}

	newManager := func(t *testing.T, restMapper meta.RESTMapper) ctrl.Manager {
		t.Helper()

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

		return mgr
	}

	t.Run("Owns Secret for resources with secret target", func(t *testing.T) {
		restMapper := meta.NewDefaultRESTMapper(nil)
		restMapper.Add(corev1.SchemeGroupVersion.WithKind("Secret"), meta.RESTScopeNamespace)
		restMapper.Add(v1alpha1.GroupVersion.WithKind("ClickhouseUser"), meta.RESTScopeNamespace)

		mgr := newManager(t, restMapper)

		recorder := record.NewFakeRecorder(10)
		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   mgr.GetClient(),
				Scheme:   mgr.GetScheme(),
				Recorder: recorder,
			},
			newObj: func() *v1alpha1.ClickhouseUser { return &v1alpha1.ClickhouseUser{} },
		}

		err := r.SetupWithManager(mgr)
		require.NoError(t, err)
	})

	t.Run("Doesn't Own Secret for resources without secret target", func(t *testing.T) {
		restMapper := meta.NewDefaultRESTMapper(nil)
		restMapper.Add(v1alpha1.GroupVersion.WithKind("KafkaTopic"), meta.RESTScopeNamespace)

		mgr := newManager(t, restMapper)

		r := &Reconciler[*v1alpha1.KafkaTopic]{
			Controller: Controller{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			},
			newObj: func() *v1alpha1.KafkaTopic { return &v1alpha1.KafkaTopic{} },
		}

		err := r.SetupWithManager(mgr)
		require.NoError(t, err)
	})

	t.Run("Applies controller options when configured", func(t *testing.T) {
		restMapper := meta.NewDefaultRESTMapper(nil)
		restMapper.Add(v1alpha1.GroupVersion.WithKind("KafkaTopic"), meta.RESTScopeNamespace)

		mgr := newManager(t, restMapper)

		r := &Reconciler[*v1alpha1.KafkaTopic]{
			Controller: Controller{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			},
			newObj: func() *v1alpha1.KafkaTopic { return &v1alpha1.KafkaTopic{} },
			options: &ctrlcontroller.Options{
				MaxConcurrentReconciles: 2,
			},
		}

		err := r.SetupWithManager(mgr)
		require.NoError(t, err)
	})
}
