package controllers

import (
	"slices"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
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

const yamlKafkaTopic = `
apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: test-topic
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  partitions: 3
  replication: 2
  tags:
    - key: env
      value: test
  config:
    cleanup_policy: delete
    retention_bytes: 123
`

func TestKafkaTopicReconciler(t *testing.T) {
	t.Parallel()

	runScenario := func(t *testing.T, topic *v1alpha1.KafkaTopic, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.KafkaTopic], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{topic}, additionalObjects...)

		r := newKafkaTopicReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.KafkaTopic{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.KafkaTopic])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      topic.Name,
				Namespace: topic.Namespace,
			},
		})
		return r, res, err
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-preconditions"
		topic.Spec.ServiceName = "test-service-preconditions"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates KafkaTopic on Aiven", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-create"
		topic.Spec.ServiceName = "test-service-create"
		topic.Spec.Config = nil

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicCreate(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName, mock.MatchedBy(func(in *kafkatopic.ServiceKafkaTopicCreateIn) bool {
				return *in.Partitions == topic.Spec.Partitions &&
					*in.Replication == topic.Spec.Replication &&
					in.TopicName == topic.GetTopicName() &&
					slices.Equal(*in.Tags, []kafkatopic.TagIn{{Key: "env", Value: "test"}}) &&
					in.Config == nil
			})).Return(nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Requeues when KafkaTopic list returns server error after generation is processed", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-list-5xx"
		topic.Spec.ServiceName = "test-service-list-5xx"
		topic.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(nil, newAivenError(500, "server error")).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Updates status and requeues when KafkaTopic is configuring", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-configuring"
		topic.Spec.ServiceName = "test-service-configuring"
		topic.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{
				{TopicName: topic.GetTopicName(), State: kafkatopic.TopicStateTypeConfiguring},
			}, nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, kafkatopic.TopicStateTypeConfiguring, got.Status.State)
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Marks KafkaTopic running when it becomes ACTIVE", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-active"
		topic.Spec.ServiceName = "test-service-active"
		topic.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{
				{TopicName: topic.GetTopicName(), State: kafkatopic.TopicStateTypeActive},
			}, nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, kafkatopic.TopicStateTypeActive, got.Status.State)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Updates KafkaTopic on Aiven when generation changes", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 2
		topic.Spec.Project = "test-project-update"
		topic.Spec.ServiceName = "test-service-update"
		topic.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{
				{TopicName: topic.GetTopicName(), State: kafkatopic.TopicStateTypeActive},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicUpdate(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName(), mock.MatchedBy(func(in *kafkatopic.ServiceKafkaTopicUpdateIn) bool {
				return *in.Partitions == topic.Spec.Partitions &&
					*in.Replication == topic.Spec.Replication &&
					slices.Equal(*in.Tags, []kafkatopic.TagIn{{Key: "env", Value: "test"}}) &&
					*in.Config.RetentionBytes == 123
			})).Return(nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Equal(t, kafkatopic.TopicStateTypeActive, got.Status.State)
	})

	t.Run("Returns error when KafkaTopic isn't visible yet but API reports it already exists", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-not-visible"
		topic.Spec.ServiceName = "test-service-not-visible"
		topic.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicCreate(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName, mock.MatchedBy(func(in *kafkatopic.ServiceKafkaTopicCreateIn) bool {
				return *in.Partitions == topic.Spec.Partitions &&
					*in.Replication == topic.Spec.Replication &&
					in.TopicName == topic.GetTopicName()
			})).Return(newAivenError(409, "already exists")).Once()

		r, res, err := runScenario(t, topic, avn)
		require.EqualError(t, err, `unable to create or update instance at aiven: creating Kafka topic: [409 ]: already exists`)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Recreates KafkaTopic when it disappears after being ready", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Spec.Project = "test-project-drift"
		topic.Spec.ServiceName = "test-service-drift"
		topic.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return(&service.ServiceGetOut{
				NodeStates: []service.NodeStateOut{
					{State: service.NodeStateTypeRunning},
					{State: service.NodeStateTypeRunning},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicList(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName).
			Return([]kafkatopic.TopicOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaTopicCreate(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName, mock.MatchedBy(func(in *kafkatopic.ServiceKafkaTopicCreateIn) bool {
				return *in.Partitions == topic.Spec.Partitions &&
					*in.Replication == topic.Spec.Replication &&
					in.TopicName == topic.GetTopicName()
			})).Return(nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Blocks deletion when termination protection is on", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		topic.DeletionTimestamp = &now
		enabled := true
		topic.Spec.TerminationProtection = &enabled

		avn := avngen.NewMockClient(t)
		r, _, err := runScenario(t, topic, avn)
		require.EqualError(t, err, `unable to delete instance: termination protection is on`)

		got := &v1alpha1.KafkaTopic{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletes KafkaTopic and removes finalizer on deletion", func(t *testing.T) {
		topic := newObjectFromYAML[v1alpha1.KafkaTopic](t, yamlKafkaTopic)
		topic.Generation = 1
		topic.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		topic.DeletionTimestamp = &now
		disabled := false
		topic.Spec.TerminationProtection = &disabled

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaTopicDelete(mock.Anything, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName()).
			Return(nil).Once()

		r, res, err := runScenario(t, topic, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaTopic{}
		err = r.Get(t.Context(), types.NamespacedName{Name: topic.Name, Namespace: topic.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
