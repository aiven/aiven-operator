//go:build kafka

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
  plan: startup-4

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
	topicName := randName("schema-registry-topic")
	aclName := randName("kafka-schema-registry-acl")
	yml := getKafkaSchemaRegistryACLYaml(cfg.Project, cfg.PrimaryCloudName, kafkaName, topicName, aclName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

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
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// KafkaTopic
	// todo: replace with code-generated client, when the API schema is fixed:
	//  json: cannot unmarshal string into Go struct field SynonymOut.topic.config.cleanup_policy.synonyms.value of type bool
	var topicAvn *kafkatopic.ServiceKafkaTopicGetOut
	// Kafka topics are eventually consistent in Aiven API, so we poll until they become readable
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		var getErr error
		topicAvn, getErr = avnGen.ServiceKafkaTopicGet(ctx, cfg.Project, kafkaName, topic.GetTopicName())
		assert.NoError(collect, getErr)
	}, 2*time.Minute, 10*time.Second)

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

// The adoption path: a CR deleted with deletion-policy Orphan leaves the ACL on Aiven, and
// re-applying the same spec must adopt that entry instead of creating a second one.
func getKafkaSchemaRegistryACLAdoptionYaml(project, service, name, resource, username string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchemaRegistryACL
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  resource: %[4]q
  username: %[5]q
  permission: schema_registry_read
`, project, service, name, resource, username)
}

// findKafkaSchemaRegistryACLMatches returns the IDs of the remote entries matching the spec
// identity, which is what Aiven uniquely indexes.
func findKafkaSchemaRegistryACLMatches(ctx context.Context, project, service string, spec v1alpha1.KafkaSchemaRegistryACLSpec) ([]string, error) {
	list, err := avnGen.ServiceSchemaRegistryAclList(ctx, project, service)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, v := range list {
		if v.Id != nil && string(v.Permission) == spec.Permission && v.Resource == spec.Resource && v.Username == spec.Username {
			ids = append(ids, *v.Id)
		}
	}

	return ids, nil
}

// deleteKafkaSchemaRegistryACLMatches removes every remote entry matching the spec identity.
// Entries created directly against the API are owned by no CR, so nothing else cleans them up.
func deleteKafkaSchemaRegistryACLMatches(t *testing.T, project, service string, spec v1alpha1.KafkaSchemaRegistryACLSpec) {
	t.Helper()
	ctx := context.Background()

	ids, err := findKafkaSchemaRegistryACLMatches(ctx, project, service, spec)
	if err != nil {
		t.Logf("cannot list Schema Registry ACLs for cleanup: %s", err)
		return
	}

	for _, id := range ids {
		if _, err := avnGen.ServiceSchemaRegistryAclDelete(ctx, project, service, id); err != nil && !avngen.IsNotFound(err) {
			t.Logf("leftover Schema Registry ACL %s: %s", id, err)
		}
	}
}

// TestKafkaSchemaRegistryACLAdoption covers what happens when a matching entry is already present on Aiven.
func TestKafkaSchemaRegistryACLAdoption(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	kafkaService, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafkaService.GetName()

	t.Run("Aiven refuses a second identical entry", func(t *testing.T) {
		spec := foreignSchemaRegistryACLSpec(randName("dup"))
		in := foreignSchemaRegistryACLAddIn(spec)
		t.Cleanup(func() { deleteKafkaSchemaRegistryACLMatches(t, cfg.Project, kafkaName, spec) })

		_, err := avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)

		_, err = avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, in)
		require.Error(t, err, "a duplicate add must not succeed")
		assert.True(t, avngen.IsAlreadyExists(err), "expected 409 already exists, got: %s", err)

		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "the refused add must not have created a second entry")
	})

	// An entry that exists before any CR does, as created in the Console or by Terraform.
	t.Run("adopts an entry created outside the operator", func(t *testing.T) {
		aclName := randName("sr-acl-foreign")
		spec := foreignSchemaRegistryACLSpec(randName("foreign"))
		in := foreignSchemaRegistryACLAddIn(spec)
		t.Cleanup(func() { deleteKafkaSchemaRegistryACLMatches(t, cfg.Project, kafkaName, spec) })

		_, err := avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)

		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		require.Len(t, ids, 1)
		foreignID := ids[0]

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(getKafkaSchemaRegistryACLAdoptionYaml(
			cfg.Project, kafkaName, aclName, spec.Resource, spec.Username,
		)))

		adopted := new(v1alpha1.KafkaSchemaRegistryACL)
		require.NoError(t, s.GetRunning(adopted, aclName))

		assert.Equal(t, foreignID, adopted.Status.ACLId,
			"the CR must adopt the pre-existing entry, not create a second one")

		ids, err = findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "adoption must not leave a duplicate entry behind")

		// An adopted entry is deleted together with the CR.
		assert.NoError(t, s.Delete(adopted, func() error {
			return schemaRegistryACLExists(ctx, kafkaName, spec, foreignID)
		}))
	})

	t.Run("re-adopts an entry after an Orphan deletion", func(t *testing.T) {
		aclName := randName("sr-acl-adopt")
		resource := "Subject:" + randName("adopt-subject")
		username := randName("adopt-user")

		yml := getKafkaSchemaRegistryACLAdoptionYaml(cfg.Project, kafkaName, aclName, resource, username)

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(yml))

		acl := new(v1alpha1.KafkaSchemaRegistryACL)
		require.NoError(t, s.GetRunning(acl, aclName))

		originalID := acl.Status.ACLId
		aclSpec := acl.Spec
		require.NotEmpty(t, originalID)

		t.Cleanup(func() {
			if _, err := avnGen.ServiceSchemaRegistryAclDelete(context.Background(), cfg.Project, kafkaName, originalID); err != nil && !avngen.IsNotFound(err) {
				t.Logf("leftover orphaned schema registry ACL %s: %s", originalID, err)
			}
		})

		// Sanity: exactly one entry, and it round-trips verbatim.
		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, aclSpec)
		require.NoError(t, err)
		require.Len(t, ids, 1)

		require.NoError(t, setOrphanDeletionPolicy(ctx, acl))
		require.NoError(t, s.Delete(acl, func() error { return nil }))

		gone := new(v1alpha1.KafkaSchemaRegistryACL)
		err = k8sClient.Get(ctx, types.NamespacedName{Name: aclName, Namespace: defaultNamespace}, gone)
		require.True(t, isNotFound(err), "CR should be gone")

		ids, err = findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, aclSpec)
		require.NoError(t, err)
		require.Len(t, ids, 1, "Orphan policy should have left the ACL on Aiven")

		require.NoError(t, s.Apply(yml))

		readopted := new(v1alpha1.KafkaSchemaRegistryACL)
		require.NoError(t, s.GetRunning(readopted, aclName))

		assert.Equal(t, originalID, readopted.Status.ACLId,
			"re-applied CR must adopt the existing ACL, not create a second one")

		ids, err = findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, aclSpec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "adoption must not leave a duplicate entry behind")

		assert.NoError(t, s.Delete(readopted, func() error {
			return schemaRegistryACLExists(ctx, kafkaName, aclSpec, originalID)
		}))
	})

	t.Run("Aiven assigns a new ID when an entry is recreated", func(t *testing.T) {
		spec := foreignSchemaRegistryACLSpec(randName("reid"))
		in := foreignSchemaRegistryACLAddIn(spec)
		t.Cleanup(func() { deleteKafkaSchemaRegistryACLMatches(t, cfg.Project, kafkaName, spec) })

		_, err := avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)

		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		require.Len(t, ids, 1)
		firstID := ids[0]

		_, err = avnGen.ServiceSchemaRegistryAclDelete(ctx, cfg.Project, kafkaName, firstID)
		require.NoError(t, err)

		_, err = avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)

		ids, err = findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		require.Len(t, ids, 1)

		assert.NotEqual(t, firstID, ids[0],
			"Aiven reused the ACL ID, so a cached ID could never go stale and adoption by content is unnecessary")
	})

	t.Run("re-adopts the live entry when Status.ACLId no longer matches it", func(t *testing.T) {
		aclName := randName("sr-acl-stale")
		spec := foreignSchemaRegistryACLSpec(randName("stale"))
		t.Cleanup(func() { deleteKafkaSchemaRegistryACLMatches(t, cfg.Project, kafkaName, spec) })

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(getKafkaSchemaRegistryACLAdoptionYaml(
			cfg.Project, kafkaName, aclName, spec.Resource, spec.Username,
		)))

		acl := new(v1alpha1.KafkaSchemaRegistryACL)
		require.NoError(t, s.GetRunning(acl, aclName))

		liveID := acl.Status.ACLId
		require.NotEmpty(t, liveID)

		// Point the cache at nothing, as `kubectl patch --subresource=status` does. The entry
		// itself is untouched, so only the operator's own bookkeeping is wrong.
		require.NoError(t, k8sClient.Status().Patch(ctx, acl, client.RawPatch(
			types.MergePatchType, []byte(`{"status":{"acl_id":"stale-id-that-does-not-exist"}}`))))

		// Observe matches the spec against the live list and repairs Status.ACLId from what it finds there.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			got := new(v1alpha1.KafkaSchemaRegistryACL)
			if !assert.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: aclName, Namespace: defaultNamespace}, got)) {
				return
			}
			assert.Equal(collect, liveID, got.Status.ACLId)
		}, 3*time.Minute, 2*time.Second)

		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "a stale cache must not produce a duplicate entry")
	})

	// The entry is destroyed and an identical one is created outside the operator.
	t.Run("converges on the live entry after an out-of-band recreate", func(t *testing.T) {
		aclName := randName("sr-acl-recreated")
		spec := foreignSchemaRegistryACLSpec(randName("recreate"))
		t.Cleanup(func() { deleteKafkaSchemaRegistryACLMatches(t, cfg.Project, kafkaName, spec) })

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(getKafkaSchemaRegistryACLAdoptionYaml(
			cfg.Project, kafkaName, aclName, spec.Resource, spec.Username,
		)))

		acl := new(v1alpha1.KafkaSchemaRegistryACL)
		require.NoError(t, s.GetRunning(acl, aclName))

		cachedID := acl.Status.ACLId
		require.NotEmpty(t, cachedID)

		_, err := avnGen.ServiceSchemaRegistryAclDelete(ctx, cfg.Project, kafkaName, cachedID)
		require.NoError(t, err)

		if _, err := avnGen.ServiceSchemaRegistryAclAdd(ctx, cfg.Project, kafkaName, foreignSchemaRegistryACLAddIn(spec)); err != nil {
			// The operator got there first, so it recreated the entry through Create rather
			// than adopting one. Convergence still has to hold.
			require.True(t, avngen.IsAlreadyExists(err), "unexpected add error: %s", err)
			t.Log("the operator recreated the entry before this add")
		}

		ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		require.Len(t, ids, 1, "the recreate must leave exactly one entry")

		recreatedID := ids[0]
		require.NotEqual(t, cachedID, recreatedID, "Aiven reused the ID, so the cache never went stale")

		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			got := new(v1alpha1.KafkaSchemaRegistryACL)
			if !assert.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: aclName, Namespace: defaultNamespace}, got)) {
				return
			}
			assert.Equal(collect, recreatedID, got.Status.ACLId)
		}, 3*time.Minute, 2*time.Second)

		ids, err = findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "converging must not produce a duplicate entry")
	})
}

// foreignSchemaRegistryACLSpec is a CR spec for an entry created straight through the API, as
// the Console or Terraform would.
func foreignSchemaRegistryACLSpec(suffix string) v1alpha1.KafkaSchemaRegistryACLSpec {
	return v1alpha1.KafkaSchemaRegistryACLSpec{
		Permission: "schema_registry_read",
		Resource:   "Subject:" + suffix,
		Username:   suffix,
	}
}

func foreignSchemaRegistryACLAddIn(spec v1alpha1.KafkaSchemaRegistryACLSpec) *kafkaschemaregistry.ServiceSchemaRegistryAclAddIn {
	return &kafkaschemaregistry.ServiceSchemaRegistryAclAddIn{
		Permission: kafkaschemaregistry.PermissionType(spec.Permission),
		Resource:   spec.Resource,
		Username:   spec.Username,
	}
}

// schemaRegistryACLExists reports nil while the entry is still on Aiven.
func schemaRegistryACLExists(ctx context.Context, service string, spec v1alpha1.KafkaSchemaRegistryACLSpec, id string) error {
	ids, err := findKafkaSchemaRegistryACLMatches(ctx, cfg.Project, service, spec)
	if err != nil {
		return err
	}

	if len(ids) > 0 {
		return nil
	}

	return controllers.NewNotFound("KafkaSchemaRegistryAcl not found with id " + id)
}
