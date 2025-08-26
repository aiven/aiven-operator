//go:build kafkaacl

package tests

import (
	"context"
	"fmt"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
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
	ctx, cancel := testCtx()
	defer cancel()

	kafkaName := randName("kafka-service")
	topicName := randName("kafka-topic")
	aclName := randName("kafka-acl")
	yml := getKafkaACLYaml(cfg.Project, kafkaName, topicName, aclName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafkaService := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafkaService, kafkaName))

	topic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(topic, topicName))

	acl := new(v1alpha1.KafkaACL)
	require.NoError(t, s.GetRunning(acl, aclName))

	// THEN
	// Kafka
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafkaService.GetName())
	assert.Equal(t, serviceRunningState, kafkaService.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)
	assert.Equal(t, kafkaAvn.Plan, kafkaService.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafkaService.Spec.CloudName)

	// KafkaTopic
	topicAvn, err := avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, kafkaName, topic.GetTopicName())
	require.NoError(t, err)
	assert.Equal(t, topicName, topic.GetName())
	assert.Equal(t, topicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.TopicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.State, topic.Status.State)
	assert.Equal(t, topicAvn.Replication, topic.Spec.Replication)
	assert.Len(t, topicAvn.Partitions, topic.Spec.Partitions)

	// KafkaACL
	aclAvn, err := getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, acl.Status.ID)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(acl.Status.Conditions, "Running"))
	assert.EqualValues(t, "admin", acl.Spec.Permission)
	assert.Equal(t, aclAvn.Permission, acl.Spec.Permission)
	assert.Equal(t, "my-user", acl.Spec.Username)
	assert.Equal(t, aclAvn.Username, acl.Spec.Username)
	assert.Equal(t, topicName, acl.Spec.Topic)
	assert.Equal(t, aclAvn.Topic, acl.Spec.Topic)

	// KafkaACL Update
	// We check that update changes the ID
	aclCopy := acl.DeepCopyObject().(*v1alpha1.KafkaACL)
	aclCopy.Spec.Permission = "write"
	require.NoError(t, k8sClient.Update(ctx, aclCopy))

	aclWrite := new(v1alpha1.KafkaACL)
	require.NoError(t, s.GetRunning(aclWrite, aclName))

	// The ID has changed
	assert.NotEqual(t, aclWrite.Status.ID, acl.Status.ID)

	// Permission has changed on Aiven side too
	aclWriteAvn, err := getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, aclWrite.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, kafka.PermissionTypeWrite, aclWrite.Spec.Permission)
	assert.Equal(t, aclWriteAvn.Permission, aclWrite.Spec.Permission)

	// Validate delete by new ID
	assert.NoError(t, s.Delete(aclWrite, func() error {
		_, err = getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, aclWrite.Status.ID)
		return err
	}))
}

func getKafkaACLbyID(ctx context.Context, avnGen avngen.Client, projectName, serviceName, aclID string) (*kafka.AclOut, error) {
	aclList, err := avnGen.ServiceKafkaAclList(ctx, projectName, serviceName)
	if err != nil {
		return nil, err
	}

	for _, acl := range aclList {
		if fromPtr(acl.Id) == aclID {
			return &acl, nil
		}
	}
	return nil, controllers.NewNotFound(fmt.Sprintf("Kafka ACL with ID %q not found", aclID))
}
