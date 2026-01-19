//go:build kafka

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// getKafkaTopicNameYaml creates two KafkaTopic resources
func getKafkaTopicYaml(project, ksName, fooTopicName, barTopicName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: %[3]s
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
    retention_bytes: 2048
---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  topicName: bar_topic_name_with_underscores
  replication: 2
  partitions: 2
`, project, ksName, fooTopicName, barTopicName)
}

// TestKafkaTopic creates two topics: one with metadata.name, another one with spec.topicName
func TestKafkaTopic(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	ks, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	ksName := ks.GetName()
	fooTopicName := randName("foo-topic")
	barTopicName := randName("bar-topic")

	yml := getKafkaTopicYaml(cfg.Project, ksName, fooTopicName, barTopicName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	fooTopic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(fooTopic, fooTopicName))

	barTopic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(barTopic, barTopicName))

	// THEN
	ksAvn, err := avnGen.ServiceGet(ctx, cfg.Project, ksName)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.ServiceName, ks.GetName())
	assert.Contains(t, serviceRunningStatesAiven, ksAvn.State)

	// Validates KafkaTopics
	assert.True(t, meta.IsStatusConditionTrue(fooTopic.Status.Conditions, "Running"))
	assert.True(t, meta.IsStatusConditionTrue(barTopic.Status.Conditions, "Running"))

	// KafkaTopic with dynamic name
	var fooAvn *kafkatopic.ServiceKafkaTopicGetOut
	// Kafka topics are eventually consistent in Aiven API, so we poll until they become readable
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		fooAvn, err = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, ksName, fooTopic.GetTopicName())
		assert.NoError(collect, err)
	}, 2*time.Minute, 10*time.Second)

	assert.Equal(t, fooTopicName, fooTopic.GetName())
	assert.Equal(t, fooTopicName, fooTopic.GetTopicName())
	assert.Equal(t, fooAvn.TopicName, fooTopic.GetTopicName())
	assert.Equal(t, fooAvn.State, fooTopic.Status.State)
	assert.Equal(t, fooAvn.State, fooTopic.Status.State)
	assert.Equal(t, fooAvn.Replication, fooTopic.Spec.Replication)
	assert.Len(t, fooAvn.Partitions, fooTopic.Spec.Partitions)

	// Validates MinCleanableDirtyRatio
	require.Equal(t, anyPointer(0.2), fooTopic.Spec.Config.MinCleanableDirtyRatio)
	require.Equal(t, anyPointer(2048), fooTopic.Spec.Config.RetentionBytes)

	// Validates MaxMessageBytes (not set)
	assert.Nil(t, fooTopic.Spec.Config.MaxMessageBytes)

	// KafkaTopic with name `bar_topic_name_with_underscores`
	var barAvn *kafkatopic.ServiceKafkaTopicGetOut
	// Kafka topics are eventually consistent in Aiven API, so we poll until they become readable
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		barAvn, err = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, ksName, barTopic.GetTopicName())
		assert.NoError(collect, err)
	}, 2*time.Minute, 10*time.Second)

	assert.Equal(t, barTopicName, barTopic.GetName())
	assert.Equal(t, "bar_topic_name_with_underscores", barTopic.GetTopicName())
	assert.Equal(t, barAvn.TopicName, barTopic.GetTopicName())
	assert.Equal(t, barAvn.State, barTopic.Status.State)
	assert.Equal(t, barAvn.State, barTopic.Status.State)
	assert.Equal(t, barAvn.Replication, barTopic.Spec.Replication)
	assert.Len(t, barAvn.Partitions, barTopic.Spec.Partitions)
	assert.Nil(t, barTopic.Spec.Config)

	// We need to validate deletion,
	// because we can get false positive here:
	// if service is deleted, topic is destroyed in Aiven. No service — no topic. No topic — no topic.
	// And we make sure that controller can delete topic itself
	assert.NoError(t, s.Delete(fooTopic, func() error {
		_, err = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, ksName, fooTopic.Name)
		return err
	}))

	assert.NoError(t, s.Delete(barTopic, func() error {
		_, err = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, ksName, "bar_topic_name_with_underscores")
		return err
	}))
}
