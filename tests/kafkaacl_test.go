//go:build kafka

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

// getKafkaACLComponentsYaml creates the ACL-related components
func getKafkaACLComponentsYaml(project, kafka, topic, acl string) string {
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
  username: my-user
  topic: %[3]s
  permission: admin
`, project, kafka, topic, acl)
}

func TestKafkaACL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafkaService := sharedResources.AcquireKafka(t, shared)

	kafkaName := kafkaService.GetName()
	topicName := randName("kafka-topic")
	aclName := randName("kafka-acl")

	yml := getKafkaACLComponentsYaml(cfg.Project, kafkaName, topicName, aclName)
	s := NewSession(ctx, k8sClient)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	topic := new(v1alpha1.KafkaTopic)
	require.NoError(t, s.GetRunning(topic, topicName))

	acl := new(v1alpha1.KafkaACL)
	require.NoError(t, s.GetRunning(acl, aclName))

	// THEN
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafkaService.GetName())
	assert.Equal(t, serviceRunningState, kafkaService.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)

	// KafkaTopic
	var topicAvn *kafkatopic.ServiceKafkaTopicGetOut
	// Kafka topics are eventually consistent in Aiven API, so we poll until they become readable
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		topicAvn, err = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, kafkaName, topic.GetTopicName())
		assert.NoError(collect, err)
	}, 2*time.Minute, 10*time.Second)

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

func TestKafkaACLRecreatesDeletedACL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	kafkaName := sharedResources.AcquireKafka(t, shared).GetName()

	for _, tc := range []struct {
		name   string
		policy string
	}{
		{name: "Full"}, // Full is the default when the annotation is absent.
		{name: "CreateOnly", policy: "CreateOnly"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := testCtx()
			defer cancel()

			topicName := randName("kafka-topic")
			aclName := randName("kafka-acl-recreate")
			objs, err := parseObjs(getKafkaACLComponentsYaml(cfg.Project, kafkaName, topicName, aclName))
			require.NoError(t, err)
			if tc.policy != "" {
				objs[aclName].SetAnnotations(map[string]string{
					"controllers.aiven.io/management-policy": tc.policy,
				})
			}

			s := NewSession(ctx, k8sClient)
			defer s.Destroy(t)
			require.NoError(t, s.ApplyObjects(objs[topicName], objs[aclName]))
			require.NoError(t, s.GetRunning(new(v1alpha1.KafkaTopic), topicName))

			acl := new(v1alpha1.KafkaACL)
			require.NoError(t, s.GetRunning(acl, aclName))
			originalID := acl.Status.ID
			require.NotEmpty(t, originalID)
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				_, err := getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, originalID)
				assert.NoError(collect, err)
			}, 2*time.Minute, 10*time.Second)

			// Delete only the external ACL. The CR keeps its spec, policy and cached ID.
			_, err = avnGen.ServiceKafkaAclDelete(ctx, cfg.Project, kafkaName, originalID)
			require.NoError(t, err)
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				_, err := getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, originalID)
				assert.True(collect, isNotFound(err), "deleted ACL %q is still visible: %v", originalID, err)
			}, 2*time.Minute, 10*time.Second)

			// Running alone is insufficient: it can remain set while the ACL is absent.
			// Wait for periodic reconciliation to publish a new ID backed by a real ACL.
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				got := new(v1alpha1.KafkaACL)
				key := types.NamespacedName{Name: aclName, Namespace: defaultNamespace}
				if !assert.NoError(collect, k8sClient.Get(ctx, key, got)) {
					return
				}
				assert.Equal(collect, acl.UID, got.UID)
				assert.Equal(collect, acl.Generation, got.Generation)
				assert.Equal(collect, acl.Spec, got.Spec)
				assert.Equal(collect, tc.policy, got.Annotations["controllers.aiven.io/management-policy"])
				if !assert.NotEmpty(collect, got.Status.ID) ||
					!assert.NotEqual(collect, originalID, got.Status.ID, "operator must replace the deleted ACL's cached ID") {
					return
				}

				restored, err := getKafkaACLbyID(ctx, avnGen, cfg.Project, kafkaName, got.Status.ID)
				if !assert.NoError(collect, err) {
					return
				}
				assert.Equal(collect, acl.Spec.Topic, restored.Topic)
				assert.Equal(collect, acl.Spec.Username, restored.Username)
				assert.Equal(collect, acl.Spec.Permission, restored.Permission)
				assert.True(collect, controllers.IsReadyToUse(got))
			}, 3*time.Minute, 5*time.Second)
		})
	}
}

func getKafkaACLbyID(ctx context.Context, avnGen avngen.Client, projectName, serviceName, aclID string) (*kafka.ServiceKafkaAclListOut, error) {
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
