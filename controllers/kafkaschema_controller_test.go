package controllers

import (
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
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
	"sigs.k8s.io/controller-runtime/pkg/event"

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

	// Soft-delete followed by hard-delete
	t.Run("Deletes KafkaSchema and removes finalizer on deletion", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		schema.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		// Soft-delete: no query options
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil).Once()
		// Hard-delete: must carry permanent=true
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				[][2]string{kafkaschemaregistry.ServiceSchemaRegistrySubjectDeletePermanent(true)}).
			Return(nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchema{}
		err = r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Resolves kafkaSchemaRef from referent spec and status", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "referent",
				Namespace:  "default",
				Generation: 1,
				Annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
			},
			Spec: func() v1alpha1.KafkaSchemaSpec {
				s := v1alpha1.KafkaSchemaSpec{
					SubjectName: "resolved-subject",
					SchemaType:  kafkaschemaregistry.SchemaTypeProtobuf,
					Schema:      "syntax = \"proto3\"; message X {}",
				}
				s.Project = "test-project"
				s.ServiceName = "test-service"
				return s
			}(),
			Status: v1alpha1.KafkaSchemaStatus{
				ID:      100,
				Version: 7,
			},
		}

		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
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
					return ref.Name == "common.proto" && ref.Subject == "resolved-subject" && ref.Version == 7
				}),
			).Return(11, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn, referent)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 11, got.Status.ID)
	})

	t.Run("Resolves explicit and kafkaSchemaRef entries in one list", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "referent",
				Namespace:  "default",
				Generation: 1,
				Annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
			},
			Spec: func() v1alpha1.KafkaSchemaSpec {
				s := v1alpha1.KafkaSchemaSpec{
					SubjectName: "shared-subject",
				}
				s.Project = "test-project"
				s.ServiceName = "test-service"
				return s
			}(),
			Status: v1alpha1.KafkaSchemaStatus{ID: 1, Version: 3},
		}

		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "legacy.proto", Subject: "legacy-subject", Version: 2},
			{Name: "shared.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
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
					if in.References == nil || len(*in.References) != 2 {
						return false
					}
					first := (*in.References)[0]
					second := (*in.References)[1]
					return first.Name == "legacy.proto" && first.Subject == "legacy-subject" && first.Version == 2 &&
						second.Name == "shared.proto" && second.Subject == "shared-subject" && second.Version == 3
				}),
			).Return(55, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn, referent)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 55, got.Status.ID)
	})

	t.Run("Requeues without calling Aiven when kafkaSchemaRef target is missing", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "not-there"}},
		}

		// the reconciler must block on the missing referent before call to Aiven
		avn := avngen.NewMockClient(t)

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	// resolveK8sRefs sees the referent as Ready, but its Status.Version is still 0.
	// The reconciler must soft-requeue.
	t.Run("Requeues from Create when referent is Ready but its Status.Version is still zero", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "referent",
				Namespace:  "default",
				Generation: 1,
				Annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
			},
			Spec: func() v1alpha1.KafkaSchemaSpec {
				s := v1alpha1.KafkaSchemaSpec{
					SubjectName: "resolved-subject",
				}
				s.Project = "test-project"
				s.ServiceName = "test-service"
				return s
			}(),
			// Status.Version intentionally zero: the referent is "Ready" by
			// annotations but the registry-assigned version hasn't landed yet.
			Status: v1alpha1.KafkaSchemaStatus{ID: 100, Version: 0},
		}

		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
		}

		avn := avngen.NewMockClient(t)
		// Service must be operational.
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// Subject is not registered yet → Create. The reconciler must
		// fail-fast on the version-zero precondition before reaching POST.
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(nil, newAivenError(404, "not found")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn, referent)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	// When a user applies a referent and a dependent in the same kubectl apply, the dependent's
	// first reconcile typically lands before the referent has finished its own.
	// The dependent must soft-requeue waiting for the referent to become Ready.
	t.Run("Multi-apply: dependent waits for referent", func(t *testing.T) {
		// Referent: exists in k8s but not yet processed.
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "referent",
				Namespace:  "default",
				Generation: 1,
			},
			Spec: func() v1alpha1.KafkaSchemaSpec {
				s := v1alpha1.KafkaSchemaSpec{
					SubjectName: "resolved-subject",
				}
				s.Project = "test-project"
				s.ServiceName = "test-service"
				return s
			}(),
		}

		dependent := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		dependent.Generation = 1
		dependent.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		dependent.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
		}

		// Any call during pass 1 fails the test (no expectation registered).
		avn := avngen.NewMockClient(t)

		r := setupKafkaSchemaReconciler(t, avn, dependent, referent)

		// dependent reconciles before the referent is Ready.
		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: dependent.Name, Namespace: dependent.Namespace},
		})
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res, "dependent must soft-requeue while referent is not Ready")

		afterPass1 := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: dependent.Name, Namespace: dependent.Namespace}, afterPass1))
		require.NotContains(t, afterPass1.Annotations, instanceIsRunningAnnotation, "dependent must not be marked Ready before its first apply")
		require.NotContains(t, afterPass1.Annotations, processedGenerationAnnotation, "dependent must not have processed any generation yet")
		require.Equal(t, 0, afterPass1.Status.ID, "dependent must not have a registry ID after pass 1")

		// Simulate the referent's successful reconcile by patching status and annotations directly.
		latestReferent := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: referent.Name, Namespace: referent.Namespace}, latestReferent))
		latestReferent.Status.Version = 7
		require.NoError(t, r.Status().Update(t.Context(), latestReferent))
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: referent.Name, Namespace: referent.Namespace}, latestReferent))
		latestReferent.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		require.NoError(t, r.Update(t.Context(), latestReferent))

		// Dependent reconciles after the referent advanced.
		avn.EXPECT().
			ServiceGet(mock.Anything, dependent.Spec.Project, dependent.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, dependent.Spec.Project, dependent.Spec.ServiceName, dependent.Spec.SubjectName).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, dependent.Spec.Project, dependent.Spec.ServiceName, dependent.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn) bool {
					if in.References == nil || len(*in.References) != 1 {
						return false
					}
					ref := (*in.References)[0]
					return ref.Name == "common.proto" &&
						ref.Subject == latestReferent.Spec.SubjectName &&
						ref.Version == 7
				}),
			).Return(99, nil).Once()

		res, err = r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: dependent.Name, Namespace: dependent.Namespace},
		})
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		afterPass2 := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: dependent.Name, Namespace: dependent.Namespace}, afterPass2))
		require.Equal(t, 99, afterPass2.Status.ID, "dependent must record the registry ID returned by the POST")
		require.Equal(t, "1", afterPass2.Annotations[processedGenerationAnnotation], "dependent must have processed its generation")
	})

	t.Run("Re-POSTs when a kafkaSchemaRef referent advances to a new version", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "referent",
				Namespace:  "default",
				Generation: 1,
				Annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
			},
			Spec: func() v1alpha1.KafkaSchemaSpec {
				s := v1alpha1.KafkaSchemaSpec{
					SubjectName: "resolved-subject",
				}
				s.Project = "test-project"
				s.ServiceName = "test-service"
				return s
			}(),
			// Referent has advanced to version 2.
			Status: v1alpha1.KafkaSchemaStatus{ID: 200, Version: 2},
		}

		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		schema.Spec.References = []v1alpha1.SchemaReference{
			{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
		}
		// Dependent's own Status.ID was assigned at version=1 of the referent.
		schema.Status.ID = 50
		schema.Status.Version = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// The registry currently has the dependent at version=1. Observe will see Status.ID match.
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return([]int{1}, nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, 1).
			Return(&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionGetOut{Id: 50, Version: 1}, nil).Once()

		// EXPECTED: a fresh POST that carries the referent's NEW version (2).
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn) bool {
					if in.References == nil || len(*in.References) != 1 {
						return false
					}
					ref := (*in.References)[0]
					return ref.Name == "common.proto" && ref.Subject == "resolved-subject" && ref.Version == 2
				}),
			).Return(60, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn, referent)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 60, got.Status.ID, "dependent must have re-POSTed and recorded the new ID")
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation, "Update must clear the running annotation when re-POSTing")
	})

	// Remove all references when Spec.References is empty, but
	// the registry currently serves the schema with reference still attached.
	t.Run("Re-POSTs without References when spec drops all references", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		schema.Spec.SchemaType = kafkaschemaregistry.SchemaTypeProtobuf
		// Spec carries no references.
		schema.Spec.References = nil
		schema.Status.ID = 77
		schema.Status.Version = 3

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionsGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return([]int{3}, nil).Once()
		// Registry reports one reference. Observe must treat this as stale.
		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionGet(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName, 3).
			Return(&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionGetOut{
				Id:      77,
				Version: 3,
				References: []kafkaschemaregistry.ReferenceOut{
					{Name: "stale.proto", Subject: "stale-subject", Version: 1},
				},
			}, nil).Once()

		avn.EXPECT().
			ServiceSchemaRegistrySubjectVersionPost(
				mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn) bool {
					return in.References == nil
				}),
			).Return(78, nil).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Equal(t, 78, got.Status.ID, "dependent must have re-POSTed and recorded the new ID")
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation, "Update must clear the running annotation when re-POSTing")
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
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName,
				[][2]string{kafkaschemaregistry.ServiceSchemaRegistrySubjectDeletePermanent(true)}).
			Return(newAivenError(404, "not found")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchema{}
		err = r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	// The schema registry rejects soft-delete of a subject that another
	// subject still references. On a transient/server error the
	// reconciler must keep the finalizer and requeue.
	t.Run("Keeps finalizer and requeues when delete fails with a server error", func(t *testing.T) {
		schema := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		schema.Generation = 1
		schema.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		schema.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceSchemaRegistrySubjectDelete(mock.Anything, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName).
			Return(newAivenError(500, "subject is referenced")).Once()

		r, res, err := runKafkaSchemaScenario(t, schema, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: schema.Name, Namespace: schema.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer, "finalizer must remain so deletion can retry")
	})

	// When in-namespace dependent still points at the referent via kafkaSchemaRef,
	// Delete must exit before any Aiven call and the reconciler must soft requeue.
	t.Run("Refuses delete and preserves Ready annotations when a kafkaSchemaRef dependent exists", func(t *testing.T) {
		referent := newObjectFromYAML[v1alpha1.KafkaSchema](t, yamlKafkaSchema)
		referent.Generation = 1
		referent.Finalizers = []string{instanceDeletionFinalizer}
		referent.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		now := metav1.Now()
		referent.DeletionTimestamp = &now

		dependent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dependent",
				Namespace: referent.Namespace,
			},
			Spec: v1alpha1.KafkaSchemaSpec{
				References: []v1alpha1.SchemaReference{
					{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: referent.Name}},
				},
			},
		}

		avn := avngen.NewMockClient(t)

		r, res, err := runKafkaSchemaScenario(t, referent, avn, dependent)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchema{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: referent.Name, Namespace: referent.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		// Both annotations must survive, resolveK8sRefs gates on that.
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation],
			"running annotation must survive the failed-delete pass so dependents can still proceed through resolveK8sRefs")
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation],
			"processed-generation annotation must survive so hasLatestGeneration stays true while the referent is Terminating")
	})
}

// runKafkaSchemaScenario builds a Reconciler seeded with schema +
// additionalObjects and reconciles schema.
func runKafkaSchemaScenario(
	t *testing.T,
	schema *v1alpha1.KafkaSchema,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.KafkaSchema], ctrlruntime.Result, error) {
	t.Helper()

	objects := append([]client.Object{schema}, additionalObjects...)
	r := setupKafkaSchemaReconciler(t, avn, objects...)
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

// setupKafkaSchemaReconciler builds a KafkaSchema Reconciler backed by a
// fake client seeded with the given objects. The fake client mirrors the
// runtime field indexer so Delete-time dependent lookups work.
func setupKafkaSchemaReconciler(
	t *testing.T,
	avn avngen.Client,
	objects ...client.Object,
) *Reconciler[*v1alpha1.KafkaSchema] {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	r := newKafkaSchemaReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.KafkaSchema{}).
			WithObjects(objects...).
			WithIndex(&v1alpha1.KafkaSchema{}, kafkaSchemaRefIndex, kafkaSchemaRefIndexValues).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.KafkaSchema])
	r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
		return avn, nil
	}
	return r
}

func TestFindKafkaSchemasReferencing(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	target := &v1alpha1.KafkaSchema{
		ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default"},
	}
	dependent := &v1alpha1.KafkaSchema{
		ObjectMeta: metav1.ObjectMeta{Name: "dependent", Namespace: "default"},
		Spec: v1alpha1.KafkaSchemaSpec{
			References: []v1alpha1.SchemaReference{
				{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "target"}},
			},
		},
	}
	unrelated := &v1alpha1.KafkaSchema{
		ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "default"},
		Spec: v1alpha1.KafkaSchemaSpec{
			References: []v1alpha1.SchemaReference{
				{Name: "other.proto", Subject: "other-subject", Version: 1},
			},
		},
	}
	// Same name in a different namespace must not be enqueued.
	otherNs := &v1alpha1.KafkaSchema{
		ObjectMeta: metav1.ObjectMeta{Name: "cross-ns-dependent", Namespace: "elsewhere"},
		Spec: v1alpha1.KafkaSchemaSpec{
			References: []v1alpha1.SchemaReference{
				{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "target"}},
			},
		},
	}

	// Register the same field indexer the controller installs at runtime.
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(target, dependent, unrelated, otherNs).
		WithIndex(&v1alpha1.KafkaSchema{}, kafkaSchemaRefIndex, kafkaSchemaRefIndexValues).
		Build()

	got := findKafkaSchemasReferencing(c)(t.Context(), target)
	require.Len(t, got, 1)
	require.Equal(t, "dependent", got[0].Name)
	require.Equal(t, "default", got[0].Namespace)
}

// TestKafkaSchemaRefIndexValues pins the indexer key extraction. Each
// kafkaSchemaRef contributes one key; explicit subject/version entries do
// not. A schema with two kafkaSchemaRef entries appears under both keys so
// the cache can match either referent name.
func TestKafkaSchemaRefIndexValues(t *testing.T) {
	t.Run("nil for non-KafkaSchema", func(t *testing.T) {
		require.Nil(t, kafkaSchemaRefIndexValues(&corev1.Secret{}))
	})

	t.Run("empty for schema with no references", func(t *testing.T) {
		s := &v1alpha1.KafkaSchema{}
		require.Empty(t, kafkaSchemaRefIndexValues(s))
	})

	t.Run("ignores explicit subject+version entries", func(t *testing.T) {
		s := &v1alpha1.KafkaSchema{Spec: v1alpha1.KafkaSchemaSpec{
			References: []v1alpha1.SchemaReference{
				{Name: "explicit.proto", Subject: "explicit-subject", Version: 1},
			},
		}}
		require.Empty(t, kafkaSchemaRefIndexValues(s))
	})

	t.Run("returns referent names from kafkaSchemaRef entries", func(t *testing.T) {
		s := &v1alpha1.KafkaSchema{Spec: v1alpha1.KafkaSchemaSpec{
			References: []v1alpha1.SchemaReference{
				{Name: "a.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "ref-a"}},
				{Name: "b.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "ref-b"}},
				{Name: "c.proto", Subject: "c-subject", Version: 2},
			},
		}}
		require.Equal(t, []string{"ref-a", "ref-b"}, kafkaSchemaRefIndexValues(s))
	})
}

// The schema parser identifies refs by name (the import path / $ref key), ordering guarantee
// is not documented. Reordered ref lists must compare equal.
func TestReferencesEqual(t *testing.T) {
	in := func(name, subject string, version int) kafkaschemaregistry.ReferenceIn {
		return kafkaschemaregistry.ReferenceIn{Name: name, Subject: subject, Version: version}
	}
	out := func(name, subject string, version int) kafkaschemaregistry.ReferenceOut {
		return kafkaschemaregistry.ReferenceOut{Name: name, Subject: subject, Version: version}
	}

	t.Run("both empty", func(t *testing.T) {
		require.True(t, referencesEqual(nil, nil))
	})

	t.Run("different lengths", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a", 1), out("b.proto", "b", 1)},
		))
	})

	t.Run("identical, same order", func(t *testing.T) {
		require.True(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1), in("b.proto", "b", 2)},
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a", 1), out("b.proto", "b", 2)},
		))
	})

	t.Run("identical, reordered", func(t *testing.T) {
		require.True(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1), in("b.proto", "b", 2)},
			[]kafkaschemaregistry.ReferenceOut{out("b.proto", "b", 2), out("a.proto", "a", 1)},
		))
	})

	t.Run("subject mismatch under same name", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a-other", 1)},
		))
	})

	t.Run("version mismatch under same name", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a", 2)},
		))
	})

	t.Run("name not present in got", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			[]kafkaschemaregistry.ReferenceOut{out("z.proto", "a", 1)},
		))
	})

	t.Run("desired empty, got non-empty", func(t *testing.T) {
		require.False(t, referencesEqual(
			nil,
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a", 1)},
		))
	})

	t.Run("desired non-empty, got empty", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			nil,
		))
	})

	// If the registry returns duplicate Names, the length check must reject the comparison.
	t.Run("duplicate names in got are not equal", func(t *testing.T) {
		require.False(t, referencesEqual(
			[]kafkaschemaregistry.ReferenceIn{in("a.proto", "a", 1)},
			[]kafkaschemaregistry.ReferenceOut{out("a.proto", "a", 1), out("a.proto", "a", 1)},
		))
	})
}

// A missing referent must surface as errPreconditionNotMet.
func TestResolveReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	dependent := func() *v1alpha1.KafkaSchema {
		return &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{Name: "dependent", Namespace: "default"},
			Spec: v1alpha1.KafkaSchemaSpec{
				References: []v1alpha1.SchemaReference{
					{Name: "common.proto", KafkaSchemaRef: &v1alpha1.LocalKafkaSchemaRef{Name: "referent"}},
				},
			},
		}
	}

	t.Run("missing referent surfaces as errPreconditionNotMet", func(t *testing.T) {
		c := &KafkaSchemaController{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(dependent()).Build(),
		}

		_, err := c.resolveReferences(t.Context(), dependent())
		require.Error(t, err)
		require.ErrorIs(t, err, errPreconditionNotMet,
			"missing referent must be a precondition miss so the reconciler soft-requeues")
	})

	t.Run("referent with zero Status.Version surfaces as errPreconditionNotMet", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{Name: "referent", Namespace: "default"},
			Spec:       v1alpha1.KafkaSchemaSpec{SubjectName: "resolved-subject"},
			// Status.Version intentionally zero.
		}
		c := &KafkaSchemaController{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(dependent(), referent).Build(),
		}

		_, err := c.resolveReferences(t.Context(), dependent())
		require.Error(t, err)
		require.ErrorIs(t, err, errPreconditionNotMet)
	})

	t.Run("resolved kafkaSchemaRef carries referent's spec subject and status version", func(t *testing.T) {
		referent := &v1alpha1.KafkaSchema{
			ObjectMeta: metav1.ObjectMeta{Name: "referent", Namespace: "default"},
			Spec:       v1alpha1.KafkaSchemaSpec{SubjectName: "resolved-subject"},
			Status:     v1alpha1.KafkaSchemaStatus{Version: 7},
		}
		c := &KafkaSchemaController{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(dependent(), referent).Build(),
		}

		got, err := c.resolveReferences(t.Context(), dependent())
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "common.proto", got[0].Name)
		require.Equal(t, "resolved-subject", got[0].Subject)
		require.Equal(t, 7, got[0].Version)
	})
}

func TestKafkaSchemaVersionChangedPredicate(t *testing.T) {
	pred := kafkaSchemaVersionChangedPredicate()

	base := &v1alpha1.KafkaSchema{
		ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default", Generation: 1},
		Status: v1alpha1.KafkaSchemaStatus{
			Version: 1,
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Unix(1_700_000_000, 0)),
					Reason:             "Stable",
					Message:            "running",
				},
			},
		},
	}

	// Unchanged -> no enqueue.
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: base.DeepCopy(), ObjectNew: base.DeepCopy()}))

	// Conditions-only change (timestamp / message) must not enqueue.
	condsTouched := base.DeepCopy()
	condsTouched.Status.Conditions[0].LastTransitionTime = metav1.NewTime(time.Unix(1_800_000_000, 0))
	condsTouched.Status.Conditions[0].Message = "still running"
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: base.DeepCopy(), ObjectNew: condsTouched}))

	// Status.Version bump -> enqueue.
	newer := base.DeepCopy()
	newer.Status.Version = 2
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: base.DeepCopy(), ObjectNew: newer}))

	// Generation bump -> enqueue.
	regen := base.DeepCopy()
	regen.Generation = 2
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: base.DeepCopy(), ObjectNew: regen}))

	// Deletes must not enqueue.
	require.False(t, pred.Delete(event.DeleteEvent{Object: base.DeepCopy()}))
}
