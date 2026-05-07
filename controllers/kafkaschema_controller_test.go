package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
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

const yamlKafkaSchema = `
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: test-schema
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  subjectName: test-subject
  schemaType: AVRO
  schema: |
    {"type":"record","name":"Test","fields":[{"name":"id","type":"string"}]}
`

func runKafkaSchemaScenario(
	t *testing.T,
	schema *v1alpha1.KafkaSchema,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.KafkaSchema], ctrlruntime.Result, error) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	objects := append([]client.Object{schema}, additionalObjects...)

	r := newKafkaSchemaReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.KafkaSchema{}).
			WithObjects(objects...).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.KafkaSchema])
	r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
		return avn, nil
	}

	res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{
			Name:      schema.Name,
			Namespace: schema.Namespace,
		},
	})
	return r, res, err
}

func runningService() *service.ServiceGetOut {
	return &service.ServiceGetOut{
		State: service.ServiceStateTypeRunning,
		NodeStates: []service.NodeStateOut{
			{State: service.NodeStateTypeRunning},
		},
	}
}

func TestKafkaSchemaReconciler(t *testing.T) {
	t.Parallel()

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Requeues when schema registry returns server error", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil, newAivenError(500, "schema registry not ready")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates KafkaSchema on Aiven when subject does not exist", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn) bool {
					return in.Schema == schema.Spec.Schema &&
						in.SchemaType == schema.Spec.SchemaType &&
						in.References == nil
				}),
			).Return(42, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Equal(t, 42, got.Status.ID)
	})

	t.Run("Creates KafkaSchema with compatibility level", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.CompatibilityLevel = kafkaschemaregistry.CompatibilityTypeBackward

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, mock.Anything,
			).Return(7, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectConfigPut(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectConfigPutIn) bool {
					return in.Compatibility == kafkaschemaregistry.CompatibilityTypeBackward
				}),
			).Return(kafkaschemaregistry.CompatibilityTypeBackward, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 7, got.Status.ID)
	})

	t.Run("Creates KafkaSchema with references (PROTOBUF)", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", Subject: "common-subject", Version: 1},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn) bool {
					if in.References == nil || len(*in.References) != 1 {
						return false
					}
					ref := (*in.References)[0]
					return ref.Name == "common.proto" && ref.Subject == "common-subject" && ref.Version == 1
				}),
			).Return(11, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 11, got.Status.ID)
	})

	t.Run("Marks KafkaSchema running when tracked ID is visible", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Annotations = map[string]string{processedGenerationAnnotation: "1"}
		schema.Status.ID = 42

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return([]int{1}, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, 1).
			Return(&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionGetOut{Id: 42, Version: 1}, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 1, got.Status.Version)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Updates KafkaSchema when generation changes", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 2
		schema.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		schema.Status.ID = 42
		schema.Status.Version = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return([]int{1}, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, 1).
			Return(&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionGetOut{Id: 42, Version: 1}, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, mock.Anything,
			).Return(99, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Equal(t, 99, got.Status.ID)
	})

	t.Run("Requeues when tracked ID is not yet visible in the registry", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Annotations = map[string]string{processedGenerationAnnotation: "1"}
		schema.Status.ID = 42

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return([]int{1}, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, 1).
			Return(&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionGetOut{Id: 7, Version: 1}, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Deletes KafkaSchema and removes finalizer on deletion", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		schema.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchema{}
		err = r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Treats 404 on delete as already deleted", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		schema.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(newAivenError(404, "not found")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchema{}
		err = r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
