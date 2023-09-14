package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getKafkaTopicNameYaml(project, ksName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-2

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: foo-topic
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  replication: 2
  partitions: 1
  config:
    min_cleanable_dirty_ratio: 0.2

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: bar-topic
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  topicName: bar_topic_name_with_underscores
  replication: 3
  partitions: 2
`, project, ksName, cloudName)
}

// TestKafkaTopic creates two topics: one with metadata.name, another one with spec.topicName
// Also validates kafka topic controller checkPreconditions(), because kafka and topic are applied simultaneously
func TestKafkaTopic(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx := context.Background()
	ksName := randName("kafka-topic")
	yml := getKafkaTopicNameYaml(testProject, ksName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, ksName))

	// Finds topics by metadata.Name, because you don't store it by spec.topicName
	fooTopic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(fooTopic, "foo-topic"))

	barTopic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(barTopic, "bar-topic"))

	// THEN
	// Validates Kafka
	ksAvn, err := avnClient.Services.Get(ctx, testProject, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.Name, ks.GetName())
	assert.Equal(t, ksAvn.State, ks.Status.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)

	// Validates KafkaTopics
	assert.True(t, meta.IsStatusConditionTrue(fooTopic.Status.Conditions, "Running"))
	assert.True(t, meta.IsStatusConditionTrue(barTopic.Status.Conditions, "Running"))

	// KafkaTopic with name `foo-topic`
	fooAvn, err := avnClient.KafkaTopics.Get(ctx, testProject, ksName, fooTopic.GetTopicName())
	require.NoError(t, err)
	assert.Equal(t, "foo-topic", fooTopic.GetName())
	assert.Equal(t, "foo-topic", fooTopic.GetTopicName())
	assert.Equal(t, fooAvn.TopicName, fooTopic.GetTopicName())
	assert.Equal(t, "ACTIVE", fooTopic.Status.State)
	assert.Equal(t, fooAvn.State, fooTopic.Status.State)
	assert.Equal(t, fooAvn.Replication, fooTopic.Spec.Replication)
	assert.Len(t, fooAvn.Partitions, fooTopic.Spec.Partitions)

	// Validates MinCleanableDirtyRatio
	require.Equal(t, anyPointer(0.2), fooTopic.Spec.Config.MinCleanableDirtyRatio)

	// KafkaTopic with name `bar_topic_name_with_underscores`
	barAvn, err := avnClient.KafkaTopics.Get(ctx, testProject, ksName, barTopic.GetTopicName())
	require.NoError(t, err)
	assert.Equal(t, "bar-topic", barTopic.GetName())
	assert.Equal(t, "bar_topic_name_with_underscores", barTopic.GetTopicName())
	assert.Equal(t, barAvn.TopicName, barTopic.GetTopicName())
	assert.Equal(t, "ACTIVE", barTopic.Status.State)
	assert.Equal(t, barAvn.State, barTopic.Status.State)
	assert.Equal(t, barAvn.Replication, barTopic.Spec.Replication)
	assert.Len(t, barAvn.Partitions, barTopic.Spec.Partitions)

	// Validates MinCleanableDirtyRatio (not set)
	assert.Nil(t, barTopic.Spec.Config.MinCleanableDirtyRatio)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, topic is destroyed in Aiven. No service — no topic. No topic — no topic.
	// And we make sure that controller can delete topic itself
	assert.NoError(t, s.Delete(fooTopic, func() error {
		_, err = avnClient.KafkaTopics.Get(ctx, testProject, ksName, fooTopic.Name)
		return err
	}))

	assert.NoError(t, s.Delete(barTopic, func() error {
		_, err = avnClient.KafkaTopics.Get(ctx, testProject, ksName, "bar_topic_name_with_underscores")
		return err
	}))
}
