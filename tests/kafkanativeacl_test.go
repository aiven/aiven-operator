//go:build kafka

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestKafkaNativeACL(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafkaName := randName("kafka-native-acl")
	aclName := randName("kafka-acl")
	yml, err := loadExampleYaml("kafkanativeacl.yaml", map[string]string{
		"doc[0].metadata.name":                  kafkaName,
		"doc[0].spec.project":                   cfg.Project,
		"doc[0].spec.connInfoSecretTarget.name": kafkaName,
		"doc[1].metadata.name":                  aclName,
		"doc[1].spec.project":                   cfg.Project,
		"doc[1].spec.serviceName":               kafkaName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	acl := new(v1alpha1.KafkaNativeACL)
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

	// KafkaNativeACL
	aclAvn, err := avnGen.ServiceKafkaNativeAclGet(ctx, cfg.Project, kafkaName, acl.Status.ID)
	require.NoError(t, err)
	assert.True(t, meta.IsStatusConditionTrue(acl.Status.Conditions, "Running"))
	assert.Equal(t, aclAvn.Host, acl.Spec.Host)
	assert.Equal(t, aclAvn.Operation, acl.Spec.Operation)
	assert.Equal(t, aclAvn.PatternType, acl.Spec.PatternType)
	assert.Equal(t, aclAvn.PermissionType, acl.Spec.PermissionType)
	assert.Equal(t, aclAvn.Principal, acl.Spec.Principal)
	assert.Equal(t, aclAvn.ResourceName, acl.Spec.ResourceName)
	assert.Equal(t, aclAvn.ResourceType, acl.Spec.ResourceType)

	// Validate delete by new ID
	assert.NoError(t, s.Delete(acl, func() error {
		_, err = avnGen.ServiceKafkaNativeAclGet(ctx, cfg.Project, kafkaName, acl.Status.ID)
		return err
	}))
}

// TestKafkaNativeACLAdoption covers what happens when a matching ACL is already present on Aiven.
func TestKafkaNativeACLAdoption(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	kafkaService, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafkaService.GetName()

	t.Run("Aiven refuses a second identical entry", func(t *testing.T) {
		in := newForeignNativeACL(randName("dup"))
		spec := foreignNativeACLSpec(in)
		t.Cleanup(func() { deleteKafkaNativeACLMatches(t, cfg.Project, kafkaName, spec) })

		created, err := avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err, "Aiven refused an empty host, so normalizeACLHost guards a state that cannot occur")

		stored, err := findKafkaNativeACLByID(ctx, cfg.Project, kafkaName, created.Id)
		require.NoError(t, err)
		require.NotNil(t, stored)
		t.Logf("an empty host was stored as %q", stored.Host)

		_, err = avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, in)
		require.Error(t, err, "a duplicate add must not succeed")
		assert.True(t, avngen.IsAlreadyExists(err), "expected 409 already exists, got: %s", err)

		ids, err := findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "the refused add must not have created a second entry")
	})

	// An ACL that exists before any CR does, as created in the Console or by Terraform.
	t.Run("adopts an entry created outside the operator", func(t *testing.T) {
		aclName := randName("native-acl-foreign")
		in := newForeignNativeACL(randName("foreign"))
		spec := foreignNativeACLSpec(in)
		t.Cleanup(func() { deleteKafkaNativeACLMatches(t, cfg.Project, kafkaName, spec) })

		created, err := avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)
		foreignID := created.Id

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		// The CR carries host "*", the CRD default; the entry above was created with "".
		require.NoError(t, s.Apply(getKafkaNativeACLYaml(
			cfg.Project, kafkaName, aclName,
			spec.Host, string(spec.PatternType), spec.ResourceName, spec.Principal,
		)))

		adopted := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(adopted, aclName))

		assert.Equal(t, foreignID, adopted.Status.ID,
			"the CR must adopt the pre-existing ACL, not create a second one")

		ids, err := findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "adoption must not leave a duplicate ACL behind")

		// An adopted ACL is deleted with the CR, the trade-off the CHANGELOG warns about.
		assert.NoError(t, s.Delete(adopted, func() error {
			_, err := avnGen.ServiceKafkaNativeAclGet(ctx, cfg.Project, kafkaName, foreignID)
			return err
		}))
	})

	t.Run("re-adopts an entry after an Orphan deletion", func(t *testing.T) {
		principal := "User:" + randName("adopt")
		prefixedName := randName("native-acl-prefixed")
		wildcardName := randName("native-acl-wildcard")

		// The risky shape: PREFIXED pattern with an explicit host.
		prefixedYml := getKafkaNativeACLYaml(
			cfg.Project, kafkaName, prefixedName,
			"10.20.30.40", "PREFIXED", randName("adopt-prefix"), principal,
		)
		// The common shape: host "*", which is also the CRD default.
		wildcardYml := getKafkaNativeACLYaml(
			cfg.Project, kafkaName, wildcardName,
			"*", "LITERAL", randName("adopt-topic"), principal,
		)

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(prefixedYml+"\n---\n"+wildcardYml))

		prefixed := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(prefixed, prefixedName))

		wildcard := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(wildcard, wildcardName))

		// Capture before deleting: s.Delete leaves the local object in an undefined state.
		originalID := prefixed.Status.ID
		prefixedSpec := prefixed.Spec
		require.NotEmpty(t, originalID)

		t.Cleanup(func() {
			if err := avnGen.ServiceKafkaNativeAclDelete(context.Background(), cfg.Project, kafkaName, originalID); err != nil && !avngen.IsNotFound(err) {
				t.Logf("leftover orphaned ACL %s: %s", originalID, err)
			}
		})

		// The matcher's core assumption, checked for both shapes.
		prefixedAvn, err := findKafkaNativeACLByID(ctx, cfg.Project, kafkaName, originalID)
		require.NoError(t, err)
		require.NotNil(t, prefixedAvn)
		assertNativeACLEchoedVerbatim(t, prefixedAvn, prefixedSpec)

		wildcardAvn, err := findKafkaNativeACLByID(ctx, cfg.Project, kafkaName, wildcard.Status.ID)
		require.NoError(t, err)
		require.NotNil(t, wildcardAvn)
		assertNativeACLEchoedVerbatim(t, wildcardAvn, wildcard.Spec)

		// Orphan the ACL, then drop the CR: this is the state a user lands in after
		// `kubectl delete` with the Orphan policy.
		require.NoError(t, setOrphanDeletionPolicy(ctx, prefixed))
		require.NoError(t, s.Delete(prefixed, func() error {
			_, err := avnGen.ServiceKafkaNativeAclGet(ctx, cfg.Project, kafkaName, originalID)
			return err
		}))

		gone := new(v1alpha1.KafkaNativeACL)
		err = k8sClient.Get(ctx, types.NamespacedName{Name: prefixedName, Namespace: defaultNamespace}, gone)
		require.True(t, isNotFound(err), "CR should be gone")

		orphaned, err := findKafkaNativeACLByID(ctx, cfg.Project, kafkaName, originalID)
		require.NoError(t, err)
		require.NotNil(t, orphaned, "Orphan policy should have left the ACL on Aiven")

		// Re-apply the identical spec, without the Orphan annotation this time.
		require.NoError(t, s.Apply(prefixedYml))

		readopted := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(readopted, prefixedName))

		assert.Equal(t, originalID, readopted.Status.ID,
			"re-applied CR must adopt the existing ACL, not create a new one")

		ids, err := findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, prefixedSpec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "adoption must not leave a duplicate ACL behind")

		// The adopted ACL is deletable through the CR again.
		assert.NoError(t, s.Delete(readopted, func() error {
			_, err := avnGen.ServiceKafkaNativeAclGet(ctx, cfg.Project, kafkaName, originalID)
			return err
		}))
	})

	t.Run("Aiven assigns a new ID when an entry is recreated", func(t *testing.T) {
		in := newForeignNativeACL(randName("reid"))
		spec := foreignNativeACLSpec(in)
		t.Cleanup(func() { deleteKafkaNativeACLMatches(t, cfg.Project, kafkaName, spec) })

		first, err := avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)
		require.NoError(t, avnGen.ServiceKafkaNativeAclDelete(ctx, cfg.Project, kafkaName, first.Id))

		second, err := avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, in)
		require.NoError(t, err)

		assert.NotEqual(t, first.Id, second.Id,
			"Aiven reused the ACL ID, so a cached ID could never go stale and adoption by content is unnecessary")
	})

	t.Run("re-adopts the live entry when Status.ID no longer matches it", func(t *testing.T) {
		aclName := randName("native-acl-stale")
		spec := foreignNativeACLSpec(newForeignNativeACL(randName("stale")))
		t.Cleanup(func() { deleteKafkaNativeACLMatches(t, cfg.Project, kafkaName, spec) })

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(getKafkaNativeACLYaml(
			cfg.Project, kafkaName, aclName,
			spec.Host, string(spec.PatternType), spec.ResourceName, spec.Principal,
		)))

		acl := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(acl, aclName))

		liveID := acl.Status.ID
		require.NotEmpty(t, liveID)

		// Point the cache at nothing, as `kubectl patch --subresource=status` does. The entry
		// itself is untouched, so only the operator's own bookkeeping is wrong.
		require.NoError(t, k8sClient.Status().Patch(ctx, acl, client.RawPatch(
			types.MergePatchType, []byte(`{"status":{"id":"stale-id-that-does-not-exist"}}`))))

		// What is under test: Observe matches the spec against the live list and repairs
		// Status.ID from what it finds there. Trusting Status.ID instead, as the operator used
		// to, reports the ACL missing and retries Create against an entry that already exists,
		// so the ID would never come back and this wait would time out.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			got := new(v1alpha1.KafkaNativeACL)
			if !assert.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: aclName, Namespace: defaultNamespace}, got)) {
				return
			}
			assert.Equal(collect, liveID, got.Status.ID)
		}, 3*time.Minute, 2*time.Second)

		ids, err := findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "a stale cache must not produce a duplicate entry")
	})

	// The entry is destroyed and an identical one is created outside the operator.
	t.Run("converges on the live entry after an out-of-band recreate", func(t *testing.T) {
		aclName := randName("native-acl-recreated")
		spec := foreignNativeACLSpec(newForeignNativeACL(randName("recreate")))
		t.Cleanup(func() { deleteKafkaNativeACLMatches(t, cfg.Project, kafkaName, spec) })

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(getKafkaNativeACLYaml(
			cfg.Project, kafkaName, aclName,
			spec.Host, string(spec.PatternType), spec.ResourceName, spec.Principal,
		)))

		acl := new(v1alpha1.KafkaNativeACL)
		require.NoError(t, s.GetRunning(acl, aclName))

		cachedID := acl.Status.ID
		require.NotEmpty(t, cachedID)

		require.NoError(t, avnGen.ServiceKafkaNativeAclDelete(ctx, cfg.Project, kafkaName, cachedID))

		if _, err := avnGen.ServiceKafkaNativeAclAdd(ctx, cfg.Project, kafkaName, foreignNativeACLAddIn(spec)); err != nil {
			// The operator got there first, so it recreated the entry through Create rather
			// than adopting one. Convergence still has to hold.
			require.True(t, avngen.IsAlreadyExists(err), "unexpected add error: %s", err)
			t.Log("the operator recreated the entry before this add")
		}

		ids, err := findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		require.Len(t, ids, 1, "the recreate must leave exactly one entry")

		recreatedID := ids[0]
		require.NotEqual(t, cachedID, recreatedID, "Aiven reused the ID, so the cache never went stale")

		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			got := new(v1alpha1.KafkaNativeACL)
			if !assert.NoError(collect, k8sClient.Get(ctx, types.NamespacedName{Name: aclName, Namespace: defaultNamespace}, got)) {
				return
			}
			assert.Equal(collect, recreatedID, got.Status.ID)
		}, 3*time.Minute, 2*time.Second)

		ids, err = findKafkaNativeACLMatches(ctx, cfg.Project, kafkaName, spec)
		require.NoError(t, err)
		assert.Len(t, ids, 1, "converging must not produce a duplicate entry")
	})
}

// newForeignNativeACL builds an ACL to create straight through the API, as the Console or
// Terraform would. The host is left empty rather than "*" to exercise normalizeACLHost.
func newForeignNativeACL(suffix string) *kafka.ServiceKafkaNativeAclAddIn {
	return &kafka.ServiceKafkaNativeAclAddIn{
		Host:           anyPointer(""),
		Operation:      "Read",
		PatternType:    "LITERAL",
		PermissionType: "ALLOW",
		Principal:      "User:" + suffix,
		ResourceName:   suffix + "-topic",
		ResourceType:   "Topic",
	}
}

// foreignNativeACLSpec is the CR spec with the same identity, carrying the "*" host default.
func foreignNativeACLSpec(in *kafka.ServiceKafkaNativeAclAddIn) v1alpha1.KafkaNativeACLSpec {
	return v1alpha1.KafkaNativeACLSpec{
		Host:           "*",
		Operation:      in.Operation,
		PatternType:    in.PatternType,
		PermissionType: in.PermissionType,
		Principal:      in.Principal,
		ResourceName:   in.ResourceName,
		ResourceType:   in.ResourceType,
	}
}

// foreignNativeACLAddIn is the API payload for an entry with the spec's identity, matching
// what the controller sends on Create.
func foreignNativeACLAddIn(spec v1alpha1.KafkaNativeACLSpec) *kafka.ServiceKafkaNativeAclAddIn {
	return &kafka.ServiceKafkaNativeAclAddIn{
		Host:           &spec.Host,
		Operation:      spec.Operation,
		PatternType:    spec.PatternType,
		PermissionType: spec.PermissionType,
		Principal:      spec.Principal,
		ResourceName:   spec.ResourceName,
		ResourceType:   spec.ResourceType,
	}
}

// The adoption path: a CR deleted with deletion-policy Orphan leaves the ACL on Aiven, and
// re-applying the same spec must adopt that entry instead of creating a second one.
func getKafkaNativeACLYaml(project, service, name, host, patternType, resourceName, principal string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaNativeACL
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  host: %[4]q
  operation: Read
  patternType: %[5]s
  permissionType: ALLOW
  principal: %[7]q
  resourceName: %[6]q
  resourceType: Topic
`, project, service, name, host, patternType, resourceName, principal)
}

// setOrphanDeletionPolicy annotates the resource so deleting it keeps the Aiven object.
func setOrphanDeletionPolicy(ctx context.Context, obj client.Object) error {
	patch := []byte(`{"metadata":{"annotations":{"controllers.aiven.io/deletion-policy":"Orphan"}}}`)
	return k8sClient.Patch(ctx, obj, client.RawPatch(types.MergePatchType, patch))
}

func findKafkaNativeACLByID(ctx context.Context, project, service, id string) (*kafka.KafkaAclOut, error) {
	list, err := avnGen.ServiceKafkaNativeAclList(ctx, project, service)
	if err != nil {
		return nil, err
	}

	for i := range list.KafkaAcl {
		if list.KafkaAcl[i].Id == id {
			return &list.KafkaAcl[i], nil
		}
	}

	return nil, nil
}

// findKafkaNativeACLMatches returns the IDs of the remote ACLs matching the spec identity.
// Host is normalized as the controller does, because Aiven may report an empty host for what the spec calls "*".
func findKafkaNativeACLMatches(ctx context.Context, project, service string, spec v1alpha1.KafkaNativeACLSpec) ([]string, error) {
	list, err := avnGen.ServiceKafkaNativeAclList(ctx, project, service)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, v := range list.KafkaAcl {
		if wildcardHost(v.Host) == wildcardHost(spec.Host) &&
			v.Principal == spec.Principal &&
			v.ResourceName == spec.ResourceName &&
			v.Operation == spec.Operation &&
			v.PatternType == spec.PatternType &&
			string(v.PermissionType) == string(spec.PermissionType) &&
			v.ResourceType == spec.ResourceType {
			ids = append(ids, v.Id)
		}
	}

	return ids, nil
}

// wildcardHost mirrors the controller's normalizeACLHost.
func wildcardHost(host string) string {
	if host == "" {
		return "*"
	}
	return host
}

// deleteKafkaNativeACLMatches removes every remote ACL matching the spec identity. Entries
// created directly against the API are owned by no CR, so nothing else cleans them up.
func deleteKafkaNativeACLMatches(t *testing.T, project, service string, spec v1alpha1.KafkaNativeACLSpec) {
	t.Helper()
	ctx := context.Background()

	ids, err := findKafkaNativeACLMatches(ctx, project, service, spec)
	if err != nil {
		t.Logf("cannot list Kafka-native ACLs for cleanup: %s", err)
		return
	}

	for _, id := range ids {
		if err := avnGen.ServiceKafkaNativeAclDelete(ctx, project, service, id); err != nil && !avngen.IsNotFound(err) {
			t.Logf("leftover Kafka-native ACL %s: %s", id, err)
		}
	}
}

// assertNativeACLEchoedVerbatim checks that every field the adoption matcher compares
// round-trips through the Aiven API unchanged.
func assertNativeACLEchoedVerbatim(t *testing.T, avn *kafka.KafkaAclOut, spec v1alpha1.KafkaNativeACLSpec) {
	t.Helper()
	assert.Equal(t, spec.Host, avn.Host, "host was not echoed verbatim")
	assert.Equal(t, spec.Principal, avn.Principal, "principal was not echoed verbatim")
	assert.Equal(t, spec.ResourceName, avn.ResourceName, "resourceName was not echoed verbatim")
	assert.Equal(t, spec.Operation, avn.Operation)
	assert.Equal(t, spec.PatternType, avn.PatternType)
	assert.EqualValues(t, spec.PermissionType, avn.PermissionType)
	assert.Equal(t, spec.ResourceType, avn.ResourceType)
}
