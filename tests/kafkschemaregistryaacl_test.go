package tests

import (
	"fmt"
	"testing"

	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func getKafkaSchemaRegistryACLYaml(project, cloudName, kafka, topic, acl string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[2]s
  plan: startup-2

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
  serviceName: %[3]s
  topicName: %[4]s
  replication: 3
  partitions: 2

---

apiVersion: aiven.io/v1alpha1
kind: KafkaSchemaRegistryACL
metadata:
  name: %[5]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[3]s
  resource: Subject:%[4]s
  username: my-user
  permission: schema_registry_read
`, project, cloudName, kafka, topic, acl)
}

func TestKafkaSchemaRegistryACL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafkaName := randName("kafka-service")
	topicName := randName("kafka-topic")
	aclName := randName("kafka-schema-registry-acl")
	yml := getKafkaSchemaRegistryACLYaml(cfg.Project, cfg.PrimaryCloudName, kafkaName, topicName, aclName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	// Cleans test afterward
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	topic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(topic, topicName))

	acl := new(v1alpha1.KafkaSchemaRegistryACL)
	require.NoError(t, s.GetRunning(acl, aclName))

	// THEN
	// Kafka
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafka.GetName())
	assert.Equal(t, serviceRunningState, kafka.Status.State)
	assert.EqualValues(t, kafkaAvn.State, kafka.Status.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// KafkaTopic
	// todo: replace with code-generated client, when the API schema is fixed:
	//  json: cannot unmarshal string into Go struct field SynonymOut.topic.config.cleanup_policy.synonyms.value of type bool
	topicAvn, err := avnClient.KafkaTopics.Get(ctx, cfg.Project, kafkaName, topic.GetTopicName())
	require.NoError(t, err)
	assert.Equal(t, topicName, topic.GetName())
	assert.Equal(t, topicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.TopicName, topic.GetTopicName())
	assert.Equal(t, topicAvn.State, topic.Status.State)
	assert.Equal(t, topicAvn.Replication, topic.Spec.Replication)
	assert.Len(t, topicAvn.Partitions, topic.Spec.Partitions)

	// KafkaSchemaRegistryACL
	aclListAvn, err := avnGen.ServiceSchemaRegistryAclList(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)

	var aclAvn *kafkaschemaregistry.AclOut
	for _, v := range aclListAvn {
		if v.Id != nil && *v.Id == acl.Status.ACLId {
			aclAvn = &v
			break
		}
	}

	require.NotNil(t, aclAvn)
	assert.True(t, meta.IsStatusConditionTrue(acl.Status.Conditions, "Running"))
	assert.Equal(t, "schema_registry_read", acl.Spec.Permission)
	assert.EqualValues(t, aclAvn.Permission, acl.Spec.Permission)
	assert.Equal(t, "my-user", acl.Spec.Username)
	assert.Equal(t, aclAvn.Username, acl.Spec.Username)
	assert.Equal(t, acl.Spec.Resource, "Subject:"+topicName)
	assert.Equal(t, aclAvn.Resource, "Subject:"+topicName)

	// Calls reconciler delete
	assert.NoError(t, s.Delete(acl, func() error {
		list, err := avnGen.ServiceSchemaRegistryAclList(ctx, cfg.Project, kafkaName)
		if err != nil {
			return err
		}

		for _, v := range list {
			if v.Id != nil && *v.Id == acl.Status.ACLId {
				return nil
			}
		}

		// There is no Get method for the ACL, so we emulate 404 for this
		return controllers.NewNotFound("KafkaSchemaRegistryAcl not found with id " + acl.Status.ACLId)
	}))
}
