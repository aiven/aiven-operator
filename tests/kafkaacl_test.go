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

func getKafkaACLYaml(project, kafka, topic, acl, cloudName string) string {
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
  cloudName: %[5]s
  plan: startup-2

---

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
  topicName: %[3]s
  replication: 3
  partitions: 2

---

apiVersion: aiven.io/v1alpha1
kind: KafkaACL
metadata:
  name: %[4]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  topic: %[3]s
  username: my-user
  permission: admin
`, project, kafka, topic, acl, cloudName)
}

func TestKafkaACL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	kafkaName := randName("kafka-acl")
	topicName := randName("kafka-acl")
	aclName := randName("kafka-acl")
	yml := getKafkaACLYaml(testProject, kafkaName, topicName, aclName, testPrimaryCloudName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	topic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(topic, topicName))

	acl := new(v1alpha1.KafkaACL)
	require.NoError(t, s.GetRunning(acl, aclName))

	// THEN
	// Kafka
	kafkaAvn, err := avnClient.Services.Get(testProject, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.Name, kafka.GetName())
	assert.Equal(t, "RUNNING", kafka.Status.State)
	assert.Equal(t, kafkaAvn.State, kafka.Status.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// KafkaTopic
	topicAvn, err := avnClient.KafkaTopics.Get(testProject, kafkaName, topic.GetTopicName())
	require.NoError(t, err)
	assert.Equal(t, topicName, topic.GetName())
	assert.Equal(t, topicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.TopicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.State, topic.Status.State)
	assert.Equal(t, topicAvn.Replication, topic.Spec.Replication)
	assert.Len(t, topicAvn.Partitions, topic.Spec.Partitions)

	// KafkaACL
	aclAvn, err := avnClient.KafkaACLs.Get(testProject, kafkaName, acl.Status.ID)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(acl.Status.Conditions, "Running"))
	assert.Equal(t, "admin", acl.Spec.Permission)
	assert.Equal(t, aclAvn.Permission, acl.Spec.Permission)
	assert.Equal(t, "my-user", acl.Spec.Username)
	assert.Equal(t, aclAvn.Username, acl.Spec.Username)
	assert.Equal(t, topicName, acl.Spec.Topic)
	assert.Equal(t, aclAvn.Topic, acl.Spec.Topic)

	// KafkaACL Update
	// We check that update changes the ID
	ctx := context.Background()
	aclCopy := acl.DeepCopyObject().(*v1alpha1.KafkaACL)
	aclCopy.Spec.Permission = "write"
	require.NoError(t, k8sClient.Update(ctx, aclCopy))

	aclWrite := new(v1alpha1.KafkaACL)
	require.NoError(t, s.GetRunning(aclWrite, aclName))

	// The ID has changed
	assert.NotEqual(t, aclWrite.Status.ID, acl.Status.ID)

	// Permission has changed on Aiven side too
	aclWriteAvn, err := avnClient.KafkaACLs.Get(testProject, kafkaName, aclWrite.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "write", aclWrite.Spec.Permission)
	assert.Equal(t, aclWriteAvn.Permission, aclWrite.Spec.Permission)

	// Validate delete by new ID
	assert.NoError(t, s.Delete(aclWrite, func() error {
		_, err = avnClient.KafkaACLs.Get(testProject, kafkaName, aclWrite.Status.ID)
		return err
	}))
}
