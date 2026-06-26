package controllers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestInstanceReconcilerHelper_reconcile(t *testing.T) {
	t.Run("Marks resource not ready before dependency gates when connection Secret must be published", func(t *testing.T) {
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.UID = types.UID("pg-uid")
		pg.Generation = 1
		metav1.SetMetaDataAnnotation(&pg.ObjectMeta, processedGenerationAnnotation, "1")
		metav1.SetMetaDataAnnotation(&pg.ObjectMeta, instanceIsRunningAnnotation, "true")
		meta.SetStatusCondition(&pg.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
		pg.Finalizers = []string{instanceDeletionFinalizer}
		pg.Spec.ProjectVPCRef = &v1alpha1.ResourceReference{Name: "missing-vpc"}

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&v1alpha1.PostgreSQL{}).
			WithObjects(pg.DeepCopy()).
			Build()

		helper := &instanceReconcilerHelper{
			k8s: k8sClient,
			h:   NewMockHandlers(t),
			rec: record.NewFakeRecorder(10),
		}

		requeue, err := helper.reconcile(t.Context(), pg)
		require.NoError(t, err)
		require.True(t, requeue)

		require.False(t, hasIsRunningAnnotation(pg))
		running := meta.FindStatusCondition(pg.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
		require.Equal(t, string(errConditionConnInfoSecret), running.Reason)
		require.Nil(t, meta.FindStatusCondition(pg.Status.Conditions, ConditionTypeError))

		got := &v1alpha1.PostgreSQL{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: pg.Name, Namespace: pg.Namespace}, got))

		require.False(t, hasIsRunningAnnotation(got))
		running = meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
		require.Equal(t, string(errConditionConnInfoSecret), running.Reason)
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
	})
}
