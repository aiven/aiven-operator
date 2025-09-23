package controllers

import (
	"context"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestStatusUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "creates status update with default values",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()
				assert.NotNil(t, statusUpdate)
				assert.NotNil(t, statusUpdate.conditions)
				assert.NotNil(t, statusUpdate.annotations)
				assert.NotNil(t, statusUpdate.statusFields)
				assert.False(t, statusUpdate.HasChanges())
			},
		},
		{
			name: "sets condition and tracks changes",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()
				condition := getRunningCondition(metav1.ConditionTrue, "TestReason", "Test message")

				statusUpdate.SetCondition(condition)

				assert.True(t, statusUpdate.HasChanges())
				conditions := statusUpdate.GetConditions()
				assert.Contains(t, conditions, conditionTypeRunning)
				assert.Equal(t, condition, conditions[conditionTypeRunning])
			},
		},
		{
			name: "sets annotation and tracks changes",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()

				statusUpdate.SetAnnotation("test-key", "test-value")

				assert.True(t, statusUpdate.HasChanges())
				annotations := statusUpdate.GetAnnotations()
				assert.Equal(t, "test-value", annotations["test-key"])
			},
		},
		{
			name: "sets processed generation",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()

				statusUpdate.SetProcessedGeneration(42)

				assert.True(t, statusUpdate.HasChanges())
				assert.Equal(t, "42", statusUpdate.processedGen)
				annotations := statusUpdate.GetAnnotations()
				assert.Equal(t, "42", annotations[processedGenerationAnnotation])
			},
		},
		{
			name: "sets running state",
			test: func(t *testing.T) {
				testCases := []struct {
					state    bool
					expected string
				}{
					{true, "true"},
					{false, "false"},
				}

				for _, tc := range testCases {
					statusUpdate := NewStatusUpdate()

					statusUpdate.SetRunningState(tc.state)

					assert.True(t, statusUpdate.HasChanges())
					assert.Equal(t, tc.expected, statusUpdate.runningState)
					annotations := statusUpdate.GetAnnotations()
					assert.Equal(t, tc.expected, annotations[instanceIsRunningAnnotation])
				}
			},
		},
		{
			name: "sets status field and tracks changes",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()

				statusUpdate.SetStatusField("test-field", "test-value")

				assert.True(t, statusUpdate.HasChanges())
				fields := statusUpdate.GetStatusFields()
				assert.Equal(t, "test-value", fields["test-field"])
			},
		},
		{
			name: "manages conditions correctly",
			test: func(t *testing.T) {
				statusUpdate := NewStatusUpdate()

				condition1 := getRunningCondition(metav1.ConditionFalse, "FirstReason", "First message")
				statusUpdate.SetCondition(condition1)

				condition2 := getRunningCondition(metav1.ConditionTrue, "SecondReason", "Second message")
				statusUpdate.SetCondition(condition2)

				conditions := statusUpdate.GetConditions()
				assert.Len(t, conditions, 1)
				assert.Equal(t, condition2, conditions[conditionTypeRunning])
				assert.Equal(t, metav1.ConditionTrue, conditions[conditionTypeRunning].Status)
				assert.Equal(t, "SecondReason", conditions[conditionTypeRunning].Reason)

				initCondition := getInitializedCondition("InitReason", "Init message")
				statusUpdate.SetCondition(initCondition)

				conditions = statusUpdate.GetConditions()
				assert.Len(t, conditions, 2)
				assert.Equal(t, condition2, conditions[conditionTypeRunning])
				assert.Equal(t, initCondition, conditions[conditionTypeInitialized])
			},
		},
		{
			name: "RemoveForceTriggerAnnotations detects and removes trigger annotations",
			test: func(t *testing.T) {
				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							ForceReconcileAnnotation: "test-value",
							"normal-annotation":      "keep-this",
						},
					},
				}

				statusUpdate := NewStatusUpdate()
				statusUpdate.RemoveForceTriggerAnnotations(kafka)

				annotations := statusUpdate.GetAnnotations()
				assert.Empty(t, annotations[ForceReconcileAnnotation])
				assert.NotContains(t, annotations, "normal-annotation") // Not touched
				assert.True(t, statusUpdate.HasChanges())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestCommitStatusUpdates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "nil status update",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				err := CommitStatusUpdates(ctx, fakeClient, kafka, nil)
				require.NoError(t, err)
			},
		},
		{
			name: "empty status update",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)
			},
		},
		{
			name: "commits conditions",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, "TestReason", "Test message"))

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				assert.Len(t, updated.Status.Conditions, 1)
				assert.Equal(t, conditionTypeRunning, updated.Status.Conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, updated.Status.Conditions[0].Status)
				assert.Equal(t, "TestReason", updated.Status.Conditions[0].Reason)
				assert.Equal(t, "Test message", updated.Status.Conditions[0].Message)
			},
		},
		{
			name: "commits annotations",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.SetAnnotation("test-key", "test-value")
				statusUpdate.SetProcessedGeneration(42)

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				annotations := updated.GetAnnotations()
				assert.Equal(t, "test-value", annotations["test-key"])
				assert.Equal(t, "42", annotations[processedGenerationAnnotation])
			},
		},
		{
			name: "commits kafka status fields",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.SetStatusField("state", service.ServiceStateTypeRunning)

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				assert.Equal(t, service.ServiceStateTypeRunning, updated.Status.State)
			},
		},
		{
			name: "removes force-reconcile annotation automatically",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
						Annotations: map[string]string{
							ForceReconcileAnnotation: "force-456",
							"normal-annotation":      "keep-this",
						},
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, "TestReason", "Test message"))

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				annotations := updated.GetAnnotations()
				assert.NotContains(t, annotations, ForceReconcileAnnotation)
				assert.Equal(t, "keep-this", annotations["normal-annotation"])
			},
		},
		{
			name: "commits updates",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-kafka",
						Namespace:  "default",
						Generation: 1,
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.SetCondition(getRunningCondition(metav1.ConditionTrue, "RunningReason", "Running message"))
				statusUpdate.SetCondition(getInitializedCondition("InitReason", "Init message"))
				statusUpdate.SetProcessedGeneration(1)
				statusUpdate.SetRunningState(true)
				statusUpdate.SetStatusField("state", service.ServiceStateTypeRunning)

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				assert.Len(t, updated.Status.Conditions, 2)
				conditionTypes := make(map[string]metav1.Condition)
				for _, cond := range updated.Status.Conditions {
					conditionTypes[cond.Type] = cond
				}

				runningCond, exists := conditionTypes[conditionTypeRunning]
				assert.True(t, exists)
				assert.Equal(t, metav1.ConditionTrue, runningCond.Status)
				assert.Equal(t, "RunningReason", runningCond.Reason)

				initCond, exists := conditionTypes[conditionTypeInitialized]
				assert.True(t, exists)
				assert.Equal(t, "InitReason", initCond.Reason)

				annotations := updated.GetAnnotations()
				assert.Equal(t, "1", annotations[processedGenerationAnnotation])
				assert.Equal(t, "true", annotations[instanceIsRunningAnnotation])

				assert.Equal(t, service.ServiceStateTypeRunning, updated.Status.State)
			},
		},
		{
			name: "removes annotations when value is empty",
			test: func(t *testing.T) {
				ctx := context.Background()
				scheme := runtime.NewScheme()
				require.NoError(t, v1alpha1.AddToScheme(scheme))

				kafka := &v1alpha1.Kafka{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-kafka",
						Namespace: "default",
						Annotations: map[string]string{
							"remove-me": "value",
							"keep-me":   "value",
						},
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(kafka).
					Build()

				statusUpdate := NewStatusUpdate()
				statusUpdate.RemoveAnnotation("remove-me")

				err := CommitStatusUpdates(ctx, fakeClient, kafka, statusUpdate)
				require.NoError(t, err)

				updated := &v1alpha1.Kafka{}
				err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-kafka", Namespace: "default"}, updated)
				require.NoError(t, err)

				annotations := updated.GetAnnotations()
				assert.NotContains(t, annotations, "remove-me")
				assert.Equal(t, "value", annotations["keep-me"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
