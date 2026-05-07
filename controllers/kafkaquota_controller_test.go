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

const yamlKafkaQuota = `
apiVersion: aiven.io/v1alpha1
kind: KafkaQuota
metadata:
  name: test-quota
  namespace: default
spec:
  project: test-project
  serviceName: test-service
  user: test-user
  clientId: test-client
  consumerByteRate: 1000
  producerByteRate: 2000
  requestPercentage: 50
`

func runKafkaQuotaScenario(
	t *testing.T,
	quota *v1alpha1.KafkaQuota,
	avn avngen.Client,
	additionalObjects ...client.Object,
) (*Reconciler[*v1alpha1.KafkaQuota], ctrlruntime.Result, error) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	objects := append([]client.Object{quota}, additionalObjects...)

	r := newKafkaQuotaReconciler(Controller{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.KafkaQuota{}).
			WithObjects(objects...).
			Build(),
		Scheme:       scheme,
		Recorder:     record.NewFakeRecorder(10),
		DefaultToken: "test-token",
		PollInterval: testPollInterval,
	}).(*Reconciler[*v1alpha1.KafkaQuota])
	r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
		return avn, nil
	}

	res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{
			Name:      quota.Name,
			Namespace: quota.Namespace,
		},
	})
	return r, res, err
}

func TestQuotaSelector(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		user     string
		clientID string
		expected [][2]string
	}{
		"both set": {
			user:     "alice",
			clientID: "app-1",
			expected: [][2]string{
				kafka.ServiceKafkaQuotaDescribeUser("alice"),
				kafka.ServiceKafkaQuotaDescribeClientId("app-1"),
			},
		},
		"only user": {
			user:     "alice",
			expected: [][2]string{kafka.ServiceKafkaQuotaDescribeUser("alice")},
		},
		"only client-id": {
			clientID: "app-1",
			expected: [][2]string{kafka.ServiceKafkaQuotaDescribeClientId("app-1")},
		},
		"neither set": {
			expected: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			q := &v1alpha1.KafkaQuota{
				Spec: v1alpha1.KafkaQuotaSpec{User: tc.user, ClientID: tc.clientID},
			}
			require.Equal(t, tc.expected, quotaSelector(q))
		})
	}
}

func TestQuotaMatchesSpec(t *testing.T) {
	t.Parallel()

	spec := v1alpha1.KafkaQuotaSpec{
		ConsumerByteRate:  new(int64(1000)),
		ProducerByteRate:  new(int64(2000)),
		RequestPercentage: new(float64(50)),
	}

	cases := map[string]struct {
		remote   *kafka.ServiceKafkaQuotaDescribeOut
		spec     v1alpha1.KafkaQuotaSpec
		expected bool
	}{
		"all values match": {
			remote: &kafka.ServiceKafkaQuotaDescribeOut{
				ConsumerByteRate:  new(float64(1000)),
				ProducerByteRate:  new(float64(2000)),
				RequestPercentage: new(float64(50)),
			},
			spec:     spec,
			expected: true,
		},
		"consumer byte rate drifts": {
			remote: &kafka.ServiceKafkaQuotaDescribeOut{
				ConsumerByteRate:  new(float64(999)),
				ProducerByteRate:  new(float64(2000)),
				RequestPercentage: new(float64(50)),
			},
			spec:     spec,
			expected: false,
		},
		"remote unset but spec set": {
			remote:   &kafka.ServiceKafkaQuotaDescribeOut{},
			spec:     spec,
			expected: false,
		},
		"both unset": {
			remote:   &kafka.ServiceKafkaQuotaDescribeOut{},
			spec:     v1alpha1.KafkaQuotaSpec{},
			expected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			q := &v1alpha1.KafkaQuota{Spec: tc.spec}
			require.Equal(t, tc.expected, quotaMatchesSpec(tc.remote, q))
		})
	}
}

func TestInt64ToFloatPtr(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, int64ToFloatPtr(nil))
	})

	t.Run("non-nil input converts value", func(t *testing.T) {
		t.Parallel()
		got := int64ToFloatPtr(new(int64(1073741824)))
		require.NotNil(t, got)
		require.InDelta(t, float64(1073741824), *got, 0)
	})
}

func TestKafkaQuotaReconciler(t *testing.T) {
	t.Parallel()

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, quota.Spec.Project, quota.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaQuota{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
	})

	t.Run("Creates KafkaQuota on Aiven when quota does not exist", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, quota.Spec.Project, quota.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceKafkaQuotaDescribe(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.Anything, mock.Anything,
			).Return(&kafka.ServiceKafkaQuotaDescribeOut{}, nil).Once()
		avn.EXPECT().
			ServiceKafkaQuotaCreate(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.MatchedBy(func(in *kafka.ServiceKafkaQuotaCreateIn) bool {
					return in.User != nil && *in.User == "test-user" &&
						in.ClientId != nil && *in.ClientId == "test-client" &&
						in.ConsumerByteRate != nil && *in.ConsumerByteRate == 1000 &&
						in.ProducerByteRate != nil && *in.ProducerByteRate == 2000 &&
						in.RequestPercentage != nil && *in.RequestPercentage == 50
				}),
			).Return(nil).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaQuota{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Marks KafkaQuota running when remote quota matches spec", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1
		quota.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, quota.Spec.Project, quota.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		avn.EXPECT().
			ServiceKafkaQuotaDescribe(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.Anything, mock.Anything,
			).Return(&kafka.ServiceKafkaQuotaDescribeOut{
			User:              new("test-user"),
			ClientId:          new("test-client"),
			ConsumerByteRate:  new(float64(1000)),
			ProducerByteRate:  new(float64(2000)),
			RequestPercentage: new(float64(50)),
		}, nil).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.KafkaQuota{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Updates KafkaQuota when remote values drift from spec", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1
		quota.Annotations = map[string]string{processedGenerationAnnotation: "1"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, quota.Spec.Project, quota.Spec.ServiceName, mock.Anything).
			Return(runningService(), nil).Once()
		// Remote has a different consumer byte rate: triggers Update.
		avn.EXPECT().
			ServiceKafkaQuotaDescribe(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.Anything, mock.Anything,
			).Return(&kafka.ServiceKafkaQuotaDescribeOut{
			User:              new("test-user"),
			ClientId:          new("test-client"),
			ConsumerByteRate:  new(float64(999)),
			ProducerByteRate:  new(float64(2000)),
			RequestPercentage: new(float64(50)),
		}, nil).Once()
		avn.EXPECT().
			ServiceKafkaQuotaCreate(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.MatchedBy(func(in *kafka.ServiceKafkaQuotaCreateIn) bool {
					return in.ConsumerByteRate != nil && *in.ConsumerByteRate == 1000
				}),
			).Return(nil).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.KafkaQuota{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
	})

	t.Run("Deletes KafkaQuota and removes finalizer on deletion", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1
		quota.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		quota.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaQuotaDelete(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.Anything, mock.Anything,
			).Return(nil).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaQuota{}
		err = r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Treats 404 on delete as already deleted", func(t *testing.T) {
		quota := newObjectFromYAML[v1alpha1.KafkaQuota](t, yamlKafkaQuota)
		quota.Generation = 1
		quota.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		quota.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceKafkaQuotaDelete(
				mock.Anything, quota.Spec.Project, quota.Spec.ServiceName,
				mock.Anything, mock.Anything,
			).Return(newAivenError(404, "not found")).Once()

		r, res, err := runKafkaQuotaScenario(t, quota, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.KafkaQuota{}
		err = r.Get(t.Context(), types.NamespacedName{Name: quota.Name, Namespace: quota.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
