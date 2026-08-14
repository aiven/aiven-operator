package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
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

const yamlKafkaNativeACL = `
apiVersion: aiven.io/v1alpha1
kind: KafkaNativeACL
metadata:
  name: test-native-acl
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  host: "*"
  operation: Read
  patternType: LITERAL
  permissionType: ALLOW
  principal: "User:alice"
  resourceName: test-topic
  resourceType: Topic
`

func runKafkaNativeACLScenario(
	t *testing.T,
	acl *v1alpha1.KafkaNativeACL,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.KafkaNativeACL], ctrlruntime.Result, error) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	objects := append([]client.Object{acl}, additionalObjects...)

	r := newKafkaNativeACLReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.KafkaNativeACL{}).
			WithObjects(objects...).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.KafkaNativeACL])
	r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
		return avn, nil
	}

	res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{
			Name:      acl.Name,
			Namespace: acl.Namespace,
		},
	})
	return r, res, err
}

func TestKafkaNativeACLReconciler(t *testing.T) {
	t.Parallel()

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaNativeACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates KafkaNativeACL on Aiven when it does not exist", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// New path: list is checked first; empty list means no orphan to adopt.
		avn.EXPECT().
			ServiceKafkaNativeAclList(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName).
			Return(&kafka.ServiceKafkaNativeAclListOut{KafkaAcl: nil}, nil).Once()
		avn.EXPECT().
			ServiceKafkaNativeAclAdd(
				mock.Anything, acl.Spec.Project, acl.Spec.ServiceName,
				mock.MatchedBy(func(in *kafka.ServiceKafkaNativeAclAddIn) bool {
					return in.Host != nil && *in.Host == "*" &&
						in.Operation == acl.Spec.Operation &&
						in.PatternType == acl.Spec.PatternType &&
						in.PermissionType == acl.Spec.PermissionType &&
						in.Principal == acl.Spec.Principal &&
						in.ResourceName == acl.Spec.ResourceName &&
						in.ResourceType == acl.Spec.ResourceType
				}),
			).Return(&kafka.ServiceKafkaNativeAclAddOut{Id: "acl-123"}, nil).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaNativeACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "acl-123", got.Status.ID)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Marks KafkaNativeACL running when it already exists", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1
		acl.Status.ID = "acl-123"
		acl.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceKafkaNativeAclGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "acl-123").
			Return(&kafka.ServiceKafkaNativeAclGetOut{Id: "acl-123"}, nil).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaNativeACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Recreates KafkaNativeACL when status ID is stale (404 on get)", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1
		acl.Status.ID = "stale-id"
		acl.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceKafkaNativeAclGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "stale-id").
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceKafkaNativeAclAdd(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(&kafka.ServiceKafkaNativeAclAddOut{Id: "acl-456"}, nil).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaNativeACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "acl-456", got.Status.ID)
	})

	t.Run("Deletes KafkaNativeACL and removes finalizer on deletion", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1
		acl.Status.ID = "acl-123"
		acl.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		acl.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaNativeAclDelete(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "acl-123").
			Return(nil).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaNativeACL{}
		err = r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Treats 404 on delete as already deleted", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1
		acl.Status.ID = "acl-123"
		acl.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		acl.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaNativeAclDelete(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "acl-123").
			Return(newAivenError(404, "not found")).Once()

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaNativeACL{}
		err = r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Adopts orphaned KafkaNativeACL when status ID is empty but spec matches existing entry", func(t *testing.T) {
		// Simulates: CR deleted with deletionPolicy:Orphan, then re-applied.
		// The operator must adopt the existing ACL instead of trying to create a duplicate.
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1
		// No Status.ID — as if this is a freshly applied CR after an Orphan deletion.

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceKafkaNativeAclList(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName).
			Return(&kafka.ServiceKafkaNativeAclListOut{
				KafkaAcl: []kafka.KafkaAclOut{
					{
						Id:             "orphaned-acl-id",
						Principal:      acl.Spec.Principal,
						ResourceName:   acl.Spec.ResourceName,
						Operation:      acl.Spec.Operation,
						PatternType:    acl.Spec.PatternType,
						PermissionType: kafka.KafkaAclPermissionType(acl.Spec.PermissionType),
						ResourceType:   acl.Spec.ResourceType,
						Host:           acl.Spec.Host,
					},
				},
			}, nil).Once()
		// ServiceKafkaNativeAclAdd must NOT be called — adoption avoids the 409.

		r, res, err := runKafkaNativeACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaNativeACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "orphaned-acl-id", got.Status.ID)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})
}

func TestNativeSpecMatches(t *testing.T) {
	t.Parallel()

	spec := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL).Spec
	remote := kafka.KafkaAclOut{
		Id:             "acl-1",
		Host:           spec.Host,
		Operation:      spec.Operation,
		PatternType:    spec.PatternType,
		PermissionType: kafka.KafkaAclPermissionType(spec.PermissionType),
		Principal:      spec.Principal,
		ResourceName:   spec.ResourceName,
		ResourceType:   spec.ResourceType,
	}
	require.True(t, nativeSpecMatches(spec, remote))

	// Every field below is part of the Kafka ACL identity tuple, so a difference in any
	// single one of them must prevent adoption.
	mismatches := map[string]func(*kafka.KafkaAclOut){
		"host":           func(o *kafka.KafkaAclOut) { o.Host = "10.0.0.1" },
		"principal":      func(o *kafka.KafkaAclOut) { o.Principal = "User:bob" },
		"resourceName":   func(o *kafka.KafkaAclOut) { o.ResourceName = "other-topic" },
		"operation":      func(o *kafka.KafkaAclOut) { o.Operation = kafka.OperationTypeWrite },
		"patternType":    func(o *kafka.KafkaAclOut) { o.PatternType = kafka.PatternTypePrefixed },
		"permissionType": func(o *kafka.KafkaAclOut) { o.PermissionType = kafka.KafkaAclPermissionTypeDeny },
		"resourceType":   func(o *kafka.KafkaAclOut) { o.ResourceType = kafka.ResourceTypeGroup },
	}

	for name, mutate := range mismatches {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out := remote
			mutate(&out)
			require.False(t, nativeSpecMatches(spec, out))
		})
	}
}
