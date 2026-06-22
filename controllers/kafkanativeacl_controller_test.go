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

	t.Run("Creates ACL on Aiven when status ID is empty", func(t *testing.T) {
		acl := newObjectFromYAML[v1alpha1.KafkaNativeACL](t, yamlKafkaNativeACL)
		acl.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, acl.Spec.Project, acl.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
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

	t.Run("Marks ACL running when it exists on Aiven", func(t *testing.T) {
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

	t.Run("Recreates ACL when status ID is stale (404 on get)", func(t *testing.T) {
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

	t.Run("Deletes ACL and removes finalizer on deletion", func(t *testing.T) {
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
}
