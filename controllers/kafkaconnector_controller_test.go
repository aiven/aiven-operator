package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaconnect"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

func TestKafkaConnectorReconciler(t *testing.T) {
	t.Parallel()

	// The example ships the connector's secret-referenced value; this seeds a
	// matching Kubernetes secret so buildConnectorConfig's fromSecret lookup resolves.
	newConnectorSecret := func() *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "os-secret", Namespace: "default"},
			Data:       map[string][]byte{"OPENSEARCH_URI": []byte("https://example:9200")},
		}
	}

	newKafkaConnector := func(t *testing.T) *v1alpha1.KafkaConnector {
		t.Helper()
		conn := newObjectFromExampleYAMLByKind[v1alpha1.KafkaConnector](t, "kafkaconnector", "KafkaConnector")
		conn.Namespace = "default"
		return conn
	}

	runScenarioErr := func(
		t *testing.T,
		conn *v1alpha1.KafkaConnector,
		avn avngen.Client,
		additionalObjects ...client.Object,
	) (*Reconciler[*v1alpha1.KafkaConnector], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{conn}, additionalObjects...)

		r := newKafkaConnectorReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.KafkaConnector{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.KafkaConnector])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: conn.Name, Namespace: conn.Namespace},
		})
		return r, res, err
	}

	runScenario := func(
		t *testing.T,
		conn *v1alpha1.KafkaConnector,
		avn avngen.Client,
		additionalObjects ...client.Object,
	) (*Reconciler[*v1alpha1.KafkaConnector], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, conn, avn, additionalObjects...)
		require.NoError(t, err)
		return r, res
	}

	expectServiceRunning := func(avn *avngen.MockClient, conn *v1alpha1.KafkaConnector, times int) {
		avn.EXPECT().
			ServiceGet(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Times(times)
	}

	getConnector := func(t *testing.T, r *Reconciler[*v1alpha1.KafkaConnector], conn *v1alpha1.KafkaConnector) *v1alpha1.KafkaConnector {
		t.Helper()
		got := &v1alpha1.KafkaConnector{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: conn.Name, Namespace: conn.Namespace}, got))
		return got
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Requeues softly when a fromSecret-referenced secret is missing", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1

		avn := avngen.NewMockClient(t)
		// Service is operational, but the referenced secret is not seeded, so
		// buildConnectorConfig fails with errSecretFetch and requeues softly.
		expectServiceRunning(avn, conn, 1)

		r, res, err := runScenarioErr(t, conn, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Hard error when userConfig template is invalid", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Spec.UserConfig = map[string]string{"bad": "{{ .Broken "}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)

		_, _, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.Error(t, err)
		require.ErrorContains(t, err, "unable to parse template for key 'bad'")
	})

	t.Run("Hard error when a fromSecret key is missing from the secret", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Spec.UserConfig = map[string]string{"connection.url": `{{ fromSecret "os-secret" "MISSING_KEY" }}`}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)

		// The secret exists but lacks the referenced key, so reconcile must fail.
		_, _, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.Error(t, err)
		require.ErrorContains(t, err, "no such key in secret")
	})

	t.Run("Creates connector when absent", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectCreateConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, mock.MatchedBy(func(in *map[string]string) bool {
				cfg := *in
				return cfg["name"] == conn.Name &&
					cfg["connector.class"] == conn.Spec.ConnectorClass &&
					cfg["connection.url"] == "https://example:9200"
			})).
			Return(&kafkaconnect.ServiceKafkaConnectCreateConnectorOut{}, nil).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues on already-exists instead of editing from Create", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectCreateConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(409, "already exists")).Once()

		r, res, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		// Not processed: the next reconcile observes the existing connector and updates it.
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Requeues without hard error on transient error during create", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectCreateConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "connect not ready")).Once()

		r, res, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Updates connector on generation bump", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 2
		// Stale processed generation forces the Update path; the running annotation
		// simulates a connector that was RUNNING before the spec change.
		conn.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		// Mutate the mutable spec fields; the edited config must reflect them.
		conn.Spec.ConnectorClass = "io.aiven.updated.SinkConnector"
		conn.Spec.UserConfig = map[string]string{"connection.url": `{{ fromSecret "os-secret" "OPENSEARCH_URI" }}`}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{
				Connectors: []kafkaconnect.ConnectorOut{{Name: conn.Name}},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(&kafkaconnect.ServiceKafkaConnectGetConnectorStatusOut{
				State: kafkaconnect.ServiceKafkaConnectConnectorStateTypePaused,
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectEditConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, mock.MatchedBy(func(in *map[string]string) bool {
				cfg := *in
				return cfg["name"] == conn.Name &&
					cfg["connector.class"] == "io.aiven.updated.SinkConnector" &&
					cfg["connection.url"] == "https://example:9200"
			})).
			Return(&kafkaconnect.ServiceKafkaConnectEditConnectorOut{}, nil).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
		// The edit restarts the connector: readiness must be dropped until Observe
		// re-confirms the new config is RUNNING.
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		running := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
	})

	t.Run("Requeues softly on transient error during update", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			err  error
		}{
			{"not found", newAivenError(404, "connect not ready")},
			{"server error", newAivenError(500, "internal error")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				conn := newKafkaConnector(t)
				conn.Generation = 2
				conn.Annotations = map[string]string{processedGenerationAnnotation: "1"}

				avn := avngen.NewMockClient(t)
				expectServiceRunning(avn, conn, 1)
				avn.EXPECT().
					ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
					Return(&kafkaconnect.ServiceKafkaConnectListOut{
						Connectors: []kafkaconnect.ConnectorOut{{Name: conn.Name}},
					}, nil).Once()
				avn.EXPECT().
					ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
					Return(&kafkaconnect.ServiceKafkaConnectGetConnectorStatusOut{
						State: kafkaconnect.ServiceKafkaConnectConnectorStateTypePaused,
					}, nil).Once()
				avn.EXPECT().
					ServiceKafkaConnectEditConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, mock.Anything).
					Return(nil, tc.err).Once()

				r, res, err := runScenarioErr(t, conn, avn, newConnectorSecret())
				require.NoError(t, err)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

				got := getConnector(t, r, conn)
				// Transient failure: generation must not be marked processed.
				require.NotEqual(t, "2", got.Annotations[processedGenerationAnnotation])
			})
		}
	})

	t.Run("Hard error on non-transient error during update", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 2
		conn.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{
				Connectors: []kafkaconnect.ConnectorOut{{Name: conn.Name}},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(&kafkaconnect.ServiceKafkaConnectGetConnectorStatusOut{
				State: kafkaconnect.ServiceKafkaConnectConnectorStateTypePaused,
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectEditConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name, mock.Anything).
			Return(nil, newAivenError(400, "bad request")).Once()

		_, _, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.Error(t, err)
		require.ErrorContains(t, err, "cannot update kafka connector on Aiven side")
	})

	t.Run("Marks running and populates status when connector is RUNNING", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{
				Connectors: []kafkaconnect.ConnectorOut{{
					Name: conn.Name,
					Plugin: kafkaconnect.PluginOut{
						Author:  "aiven",
						Class:   conn.Spec.ConnectorClass,
						DocUrl:  "https://docs",
						Title:   "OpenSearch Sink",
						Type:    kafkaconnect.PluginTypeSink,
						Version: "1.2.3",
					},
				}},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(&kafkaconnect.ServiceKafkaConnectGetConnectorStatusOut{
				State: kafkaconnect.ServiceKafkaConnectConnectorStateTypeRunning,
				Tasks: []kafkaconnect.ServiceKafkaConnectGetConnectorStatusTaskOut{
					{Id: 0, State: kafkaconnect.TaskStateTypeRunning},
					{Id: 1, State: kafkaconnect.TaskStateTypePaused},
					{Id: 2, State: kafkaconnect.TaskStateTypeUnassigned},
					{Id: 3, State: kafkaconnect.TaskStateTypeFailed, Trace: "boom"},
					{Id: 4, State: kafkaconnect.TaskStateTypeStopped},
				},
			}, nil).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := getConnector(t, r, conn)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, kafkaconnect.ServiceKafkaConnectConnectorStateTypeRunning, got.Status.State)
		require.Equal(t, "aiven", got.Status.PluginStatus.Author)
		require.Equal(t, conn.Spec.ConnectorClass, got.Status.PluginStatus.Class)
		require.Equal(t, "https://docs", got.Status.PluginStatus.DocURL)
		require.Equal(t, "OpenSearch Sink", got.Status.PluginStatus.Title)
		require.Equal(t, "sink", got.Status.PluginStatus.Type)
		require.Equal(t, "1.2.3", got.Status.PluginStatus.Version)
		require.Equal(t, uint(5), got.Status.TasksStatus.Total)
		require.Equal(t, uint(1), got.Status.TasksStatus.Running)
		require.Equal(t, uint(1), got.Status.TasksStatus.Paused)
		require.Equal(t, uint(1), got.Status.TasksStatus.Unassigned)
		require.Equal(t, uint(1), got.Status.TasksStatus.Failed)
		require.Equal(t, uint(1), got.Status.TasksStatus.Unknown)
		require.Equal(t, "boom", got.Status.TasksStatus.StackTrace)
	})

	t.Run("Does not mark running when connector is not RUNNING", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{
				Connectors: []kafkaconnect.ConnectorOut{{Name: conn.Name}},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(&kafkaconnect.ServiceKafkaConnectGetConnectorStatusOut{
				State: kafkaconnect.ServiceKafkaConnectConnectorStateTypePaused,
			}, nil).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		// Up-to-date but not yet running: requeue soon rather than at the poll interval.
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getConnector(t, r, conn)
		require.NotEqual(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, kafkaconnect.ServiceKafkaConnectConnectorStateTypePaused, got.Status.State)
	})

	t.Run("Hard error when connector status lookup fails", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		expectServiceRunning(avn, conn, 1)
		avn.EXPECT().
			ServiceKafkaConnectList(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName).
			Return(&kafkaconnect.ServiceKafkaConnectListOut{
				Connectors: []kafkaconnect.ConnectorOut{{Name: conn.Name}},
			}, nil).Once()
		// A non-retryable (non-404/5xx) status error surfaces as a hard error;
		// retryable ones would be softly requeued by handleObserveError.
		avn.EXPECT().
			ServiceKafkaConnectGetConnectorStatus(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(nil, newAivenError(400, "bad request")).Once()

		_, _, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.Error(t, err)
		require.ErrorContains(t, err, "getting kafka connector status")
	})

	t.Run("Deletes connector and removes finalizer on deletion", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		conn.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaConnectDeleteConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(nil).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaConnector{}
		err := r.Get(t.Context(), types.NamespacedName{Name: conn.Name, Namespace: conn.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Ignores not found on deletion", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		conn.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaConnectDeleteConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(newAivenError(404, "not found")).Once()

		r, res := runScenario(t, conn, avn, newConnectorSecret())
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaConnector{}
		err := r.Get(t.Context(), types.NamespacedName{Name: conn.Name, Namespace: conn.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Hard error and keeps finalizer on non-404 delete failure", func(t *testing.T) {
		conn := newKafkaConnector(t)
		conn.Generation = 1
		conn.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		conn.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		// A non-404/non-5xx delete failure surfaces as a hard error (5xx would be
		// softly requeued by handleDeleteError, 404 is tolerated by Delete itself).
		avn.EXPECT().
			ServiceKafkaConnectDeleteConnector(mock.Anything, conn.Spec.Project, conn.Spec.ServiceName, conn.Name).
			Return(newAivenError(400, "bad request")).Once()

		r, _, err := runScenarioErr(t, conn, avn, newConnectorSecret())
		require.Error(t, err)
		require.ErrorContains(t, err, "unable to delete kafka connector")

		// The object (with finalizer) must still exist so deletion is retried.
		got := &v1alpha1.KafkaConnector{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: conn.Name, Namespace: conn.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})
}
