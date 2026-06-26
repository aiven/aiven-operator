package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
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

func newKafkaSchemaRegistryACL(t *testing.T) *v1alpha1.KafkaSchemaRegistryACL {
	t.Helper()
	acl := newObjectFromExampleYAML[v1alpha1.KafkaSchemaRegistryACL](t, "kafkaschemaregistryacl")
	acl.Namespace = "default"
	return acl
}

func runKafkaSchemaRegistryACLScenario(
	t *testing.T,
	acl *v1alpha1.KafkaSchemaRegistryACL,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.KafkaSchemaRegistryACL], ctrlruntime.Result, error) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	objects := append([]client.Object{acl}, additionalObjects...)

	r := newKafkaSchemaRegistryACLReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.KafkaSchemaRegistryACL{}).
			WithObjects(objects...).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.KafkaSchemaRegistryACL])
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

func TestKafkaSchemaRegistryACLReconciler(t *testing.T) {
	t.Parallel()

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates KafkaSchemaRegistryACL on Aiven when it does not exist", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistryAclAdd(
				mock.Anything, acl.Spec.Project, acl.Spec.ServiceName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistryAclAddIn) bool {
					return string(in.Permission) == acl.Spec.Permission &&
						in.Resource == acl.Spec.Resource &&
						in.Username == acl.Spec.Username
				}),
			).Return([]kafkaschemaregistry.AclOut{{
			Id:         new("acl-id"),
			Permission: kafkaschemaregistry.PermissionType(acl.Spec.Permission),
			Resource:   acl.Spec.Resource,
			Username:   acl.Spec.Username,
		}}, nil).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "acl-id", got.Status.ACLId)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Fails when the created ACL is absent from the Aiven response", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// The add succeeds but the returned list contains no matching entry,
		// so the ID cannot be resolved and the reconcile must fail.
		avn.EXPECT().
			ServiceSchemaRegistryAclAdd(
				mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything,
			).Return([]kafkaschemaregistry.AclOut{}, nil).Once()

		r, _, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.Error(t, err)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Empty(t, got.Status.ACLId)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Marks KafkaSchemaRegistryACL running when it already exists", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1
		acl.Annotations = map[string]string{processedGenerationAnnotation: "1"}
		acl.Status.ACLId = "acl-id"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceSchemaRegistryAclList(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName).
			Return([]kafkaschemaregistry.AclOut{{
				Id:         new("acl-id"),
				Permission: kafkaschemaregistry.PermissionType(acl.Spec.Permission),
				Resource:   acl.Spec.Resource,
				Username:   acl.Spec.Username,
			}}, nil).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "acl-id", got.Status.ACLId)
	})

	t.Run("Recreates ACL when Status.ACLId is no longer present on Aiven", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1
		acl.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		acl.Status.ACLId = "stale-id"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// Observe: the stored ID is no longer present, so the resource is recreated.
		avn.EXPECT().
			ServiceSchemaRegistryAclList(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName).
			Return(nil, nil).Once()
		// Create: adds the ACL and resolves the new ID.
		avn.EXPECT().
			ServiceSchemaRegistryAclAdd(
				mock.Anything, acl.Spec.Project, acl.Spec.ServiceName,
				mock.MatchedBy(func(in *kafkaschemaregistry.ServiceSchemaRegistryAclAddIn) bool {
					return string(in.Permission) == acl.Spec.Permission &&
						in.Resource == acl.Spec.Resource &&
						in.Username == acl.Spec.Username
				}),
			).Return([]kafkaschemaregistry.AclOut{{
			Id:         new("new-id"),
			Permission: kafkaschemaregistry.PermissionType(acl.Spec.Permission),
			Resource:   acl.Spec.Resource,
			Username:   acl.Spec.Username,
		}}, nil).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got))
		require.Equal(t, "new-id", got.Status.ACLId)
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Deletes KafkaSchemaRegistryACL and removes finalizer on deletion", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1
		acl.Status.ACLId = "acl-id"
		acl.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		acl.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceSchemaRegistryAclDelete(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "acl-id").
			Return(nil, nil).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		err = r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Treats 404 on delete as already deleted", func(t *testing.T) {
		acl := newKafkaSchemaRegistryACL(t)
		acl.Generation = 1
		acl.Status.ACLId = "acl-id"
		acl.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		acl.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceSchemaRegistryAclDelete(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, "acl-id").
			Return(nil, newAivenError(404, "not found")).Once()

		r, res, err := runKafkaSchemaRegistryACLScenario(t, acl, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaSchemaRegistryACL{}
		err = r.Get(t.Context(), types.NamespacedName{Name: acl.Name, Namespace: acl.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
