//go:build kafka

package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestKafkaSchema(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafka, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafka.GetName()
	schemaName := randName("kafka-schema")
	// Keep the subject over the old 63-character operator limit to verify the CRD accepts it.
	subjectName := randName("kafka-schema-subject-name-longer-than-sixty-three-characters")
	require.Greater(t, len(subjectName), 63)
	yml := getKafkaSchemaYaml(cfg.Project, kafkaName, schemaName, subjectName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// THEN
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafka.GetName())
	assert.Equal(t, serviceRunningState, kafka.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)
	require.NotNil(t, kafka.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), kafka.Spec.UserConfig.SchemaRegistry)

	// This test implements the following scenario and expects IDs/Versions:
	// Schema A -> ID:1, Version:1
	// Schema B -> ID:2, Version:2
	// Revert to A -> ID:1, Version:1

	// STEP 1: Schema A
	// KafkaSchema test
	schemaA := new(v1alpha1.KafkaSchema)
	require.NoError(t, s.GetRunning(schemaA, schemaName))
	assert.Equal(t, schemaName, schemaA.Name)
	assert.Equal(t, subjectName, schemaA.Spec.SubjectName)
	assert.Equal(t, kafkaName, schemaA.Spec.ServiceName)
	assert.Equal(t, kafkaschemaregistry.SchemaTypeAvro, schemaA.Spec.SchemaType)
	assert.Equal(t, kafkaschemaregistry.CompatibilityTypeBackward, schemaA.Spec.CompatibilityLevel)

	// Compares to the returned schema from Aiven API.
	avnSchemaA, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, subjectName, schemaA.Status.Version)
	require.NoError(t, err)
	assert.Equal(t, schemaA.Status.ID, avnSchemaA.Id)
	assert.Equal(t, schemaA.Status.Version, avnSchemaA.Version)
	assert.Empty(t, avnSchemaA.SchemaType) // Empty means "AVRO", which is the default schema type.

	// Can't compare the schema directly because of the different types.
	// Turns them into a struct with the same types.
	type schemaType struct {
		Default   any           `json:"default,omitempty"`
		Fields    []*schemaType `json:"fields,omitempty"`
		Doc       string        `json:"doc"`
		Name      string        `json:"name"`
		Namespace string        `json:"namespace"`
		Type      string        `json:"type"`
	}

	actualSchema := new(schemaType)
	err = json.Unmarshal([]byte(schemaA.Spec.Schema), &actualSchema)
	require.NoError(t, err)
	expectedSchema := &schemaType{
		Default: nil,
		Fields: []*schemaType{
			{
				Default:   float64(5),
				Doc:       "field_doc",
				Name:      "field_name",
				Namespace: "field_namespace",
				Type:      "int",
			},
		},
		Doc:       "example_doc",
		Name:      "example_name",
		Namespace: "example_namespace",
		Type:      "record",
	}
	assert.Empty(t, cmp.Diff(expectedSchema, actualSchema))

	// STEP 2: Schema B updates the schema
	schemaB := schemaA.DeepCopy()
	schemaB.Spec.Schema = strings.ReplaceAll(schemaA.Spec.Schema, "example_namespace", "example_namespace_updated")
	require.NoError(t, k8sClient.Update(ctx, schemaB))
	require.NoError(t, s.GetRunning(schemaB, schemaName))

	// The update schema has a new ID and version
	assert.NotEqual(t, schemaB.Status.ID, schemaA.Status.ID)
	assert.Greater(t, schemaB.Status.Version, schemaA.Status.Version)

	// Compares to the returned schema from Aiven API.
	avnSchemaB, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, subjectName, schemaB.Status.Version)
	require.NoError(t, err)
	assert.Equal(t, schemaB.Status.ID, avnSchemaB.Id)
	assert.Equal(t, schemaB.Status.Version, avnSchemaB.Version)

	// STEP 3: Revert to Schema A
	schemaC := schemaB.DeepCopy()
	schemaC.Spec.Schema = schemaA.Spec.Schema
	require.NoError(t, k8sClient.Update(ctx, schemaC))
	require.NoError(t, s.GetRunning(schemaC, schemaName))

	// The update schema has the old ID and the old version
	assert.Equal(t, schemaC.Status.ID, schemaA.Status.ID)
	assert.Equal(t, schemaC.Status.Version, schemaA.Status.Version)

	// Compares to the returned schema from Aiven API.
	avnSchemaC, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, subjectName, schemaC.Status.Version)
	require.NoError(t, err)
	assert.Equal(t, schemaC.Status.ID, avnSchemaC.Id)
	assert.Equal(t, schemaC.Status.Version, avnSchemaC.Version)

	// Validates deleting, because deleted kafka drops schemas, and we want to be sure deletion works
	subjectExists := func() error {
		list, err := avnGen.ServiceSchemaRegistrySubjects(ctx, cfg.Project, kafkaName)
		if err != nil {
			return fmt.Errorf("cannot list Kafka Subjects: %w", err)
		}
		for _, subject := range list {
			if subject == subjectName {
				return nil // Found the subject
			}
		}
		return controllers.NewNotFound(fmt.Sprintf("Kafka Subject %q not found", subjectName))
	}

	// Then deletes it until it is not found
	// First proves that it won't give false positive on GET
	require.NoError(t, subjectExists())
	assert.NoError(t, s.Delete(schemaA, subjectExists))
}

func TestKafkaSchemaReferences(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	kafka, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafka.GetName()
	s := NewSession(ctx, k8sClient)

	// subjectExists returns nil when the named subject still appears in the
	// registry's active listing, NotFound otherwise. The active listing excludes soft-deleted subjects.
	subjectExists := func(subjectName string) func() error {
		return func() error {
			list, err := avnGen.ServiceSchemaRegistrySubjects(ctx, cfg.Project, kafkaName)
			if err != nil {
				return fmt.Errorf("cannot list Kafka Subjects: %w", err)
			}
			for _, subject := range list {
				if subject == subjectName {
					return nil
				}
			}
			return controllers.NewNotFound(fmt.Sprintf("Kafka Subject %q not found", subjectName))
		}
	}

	t.Run("explicit subject and version", func(t *testing.T) {
		refSchemaName := randName("kafka-schema-ref")
		refSubjectName := randName("kafka-schema-ref")
		mainSchemaName := randName("kafka-schema-main")
		mainSubjectName := randName("kafka-schema-main")

		// Create the referenced (base) schema first
		refYml := getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, refSchemaName, refSubjectName)
		require.NoError(t, s.Apply(refYml))

		refSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(refSchema, refSchemaName))
		assert.Equal(t, refSubjectName, refSchema.Spec.SubjectName)
		assert.Equal(t, kafkaschemaregistry.SchemaTypeProtobuf, refSchema.Spec.SchemaType)
		assert.Equal(t, 1, refSchema.Status.Version)

		// Create a schema that references the base schema
		mainYml := getKafkaSchemaRefYaml(cfg.Project, kafkaName, mainSchemaName, mainSubjectName, refSubjectName, refSchema.Status.Version)
		require.NoError(t, s.Apply(mainYml))

		mainSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(mainSchema, mainSchemaName))
		assert.Equal(t, mainSubjectName, mainSchema.Spec.SubjectName)
		assert.Equal(t, kafkaschemaregistry.SchemaTypeProtobuf, mainSchema.Spec.SchemaType)

		// Verify the references are set in the spec
		require.Len(t, mainSchema.Spec.References, 1)
		assert.Equal(t, "customer.proto", mainSchema.Spec.References[0].Name)
		assert.Equal(t, refSubjectName, mainSchema.Spec.References[0].Subject)
		assert.Equal(t, refSchema.Status.Version, mainSchema.Spec.References[0].Version)

		// Verify the schema was created with references
		avnSchema, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, mainSchema.Status.Version)
		require.NoError(t, err)
		assert.Equal(t, mainSchema.Status.ID, avnSchema.Id)
		require.Len(t, avnSchema.References, 1)
		assert.Equal(t, "customer.proto", avnSchema.References[0].Name)
		assert.Equal(t, refSubjectName, avnSchema.References[0].Subject)
		assert.Equal(t, refSchema.Status.Version, avnSchema.References[0].Version)

		// Delete the dependent first.
		assert.NoError(t, s.Delete(mainSchema, subjectExists(mainSubjectName)))
		assert.NoError(t, s.Delete(refSchema, subjectExists(refSubjectName)))
	})

	// 1. referent moves -> dependent must propagate to the new version
	// 2. dependent moves (own spec) -> reference still pins the same referent version
	// 3. delete dependent then referent -> both succeed
	t.Run("kafkaSchemaRef tracks referent on both sides of edits and cleans up", func(t *testing.T) {
		refSchemaName := randName("kafka-schema-ref-track")
		refSubjectName := randName("kafka-schema-ref-track")
		mainSchemaName := randName("kafka-schema-main-track")
		mainSubjectName := randName("kafka-schema-main-track")

		// Apply referent, then dependent.
		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, refSchemaName, refSubjectName)))

		refSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(refSchema, refSchemaName))
		initialRefVersion := refSchema.Status.Version
		require.Greater(t, initialRefVersion, 0, "referent must have a registry version")

		require.NoError(t, s.Apply(getKafkaSchemaKafkaSchemaRefYaml(cfg.Project, kafkaName, mainSchemaName, mainSubjectName, refSchemaName)))

		mainSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(mainSchema, mainSchemaName))
		initialMainVersion := mainSchema.Status.Version

		avnSchema, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, initialMainVersion)
		require.NoError(t, err)
		require.Len(t, avnSchema.References, 1)
		assert.Equal(t, "customer.proto", avnSchema.References[0].Name)
		assert.Equal(t, refSubjectName, avnSchema.References[0].Subject)
		assert.Equal(t, initialRefVersion, avnSchema.References[0].Version, "dependent must reference the referent's initial version")

		// Mutate the referent.
		updatedRef := refSchema.DeepCopy()
		updatedRef.Spec.Schema = strings.ReplaceAll(refSchema.Spec.Schema, "string email", "string email_address")
		require.NoError(t, k8sClient.Update(ctx, updatedRef))
		require.NoError(t, s.GetRunning(updatedRef, refSchemaName))
		require.Greater(t, updatedRef.Status.Version, initialRefVersion, "referent did not advance to a new version")
		newRefVersion := updatedRef.Status.Version

		require.NoError(t, retryForever(ctx, "dependent advances to new referent version", func() (bool, error) {
			latestMain := new(v1alpha1.KafkaSchema)
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: mainSchemaName, Namespace: defaultNamespace}, latestMain); err != nil {
				return true, err
			}
			got, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, latestMain.Status.Version)
			if err != nil {
				return true, err
			}
			if len(got.References) != 1 {
				return true, fmt.Errorf("expected 1 reference, got %d", len(got.References))
			}
			if got.References[0].Version != newRefVersion {
				return true, nil
			}
			return false, nil
		}))

		// Edit the dependent's own spec.schema. The new dependent version must still reference the
		// referent at the same version.
		latestMain := new(v1alpha1.KafkaSchema)
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Name: mainSchemaName, Namespace: defaultNamespace}, latestMain))
		mainBeforeSelfEdit := latestMain.Status.Version

		updatedMain := latestMain.DeepCopy()
		updatedMain.Spec.Schema = strings.ReplaceAll(latestMain.Spec.Schema, "string order_id = 1;", "string order_id = 1;\n      string note = 3;")
		require.NoError(t, k8sClient.Update(ctx, updatedMain))
		require.NoError(t, s.GetRunning(updatedMain, mainSchemaName))
		require.Greater(t, updatedMain.Status.Version, mainBeforeSelfEdit, "dependent did not advance after self-edit")

		require.NoError(t, retryForever(ctx, "dependent self-edit preserves referent version", func() (bool, error) {
			latest := new(v1alpha1.KafkaSchema)
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: mainSchemaName, Namespace: defaultNamespace}, latest); err != nil {
				return true, err
			}
			got, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, latest.Status.Version)
			if err != nil {
				return true, err
			}
			if len(got.References) != 1 {
				return true, fmt.Errorf("expected 1 reference after dependent self-edit, got %d", len(got.References))
			}
			if got.References[0].Subject != refSubjectName {
				return false, fmt.Errorf("dependent's reference subject changed: %s", got.References[0].Subject)
			}
			if got.References[0].Version != newRefVersion {
				return true, nil
			}
			return false, nil
		}))

		// The referent itself never moved.
		stillRef := new(v1alpha1.KafkaSchema)
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Name: refSchemaName, Namespace: defaultNamespace}, stillRef))
		assert.Equal(t, newRefVersion, stillRef.Status.Version, "referent must not have advanced during dependent self-edit")

		require.NoError(t, s.Delete(updatedMain, subjectExists(mainSubjectName)))
		require.NoError(t, s.Delete(refSchema, subjectExists(refSubjectName)))
	})

	t.Run("kafkaSchemaRef removed from spec drops references in the registry", func(t *testing.T) {
		refSchemaName := randName("kafka-schema-ref-drop")
		refSubjectName := randName("kafka-schema-ref-drop")
		mainSchemaName := randName("kafka-schema-main-drop")
		mainSubjectName := randName("kafka-schema-main-drop")

		// 1) Apply referent first.
		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, refSchemaName, refSubjectName)))
		refSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(refSchema, refSchemaName))
		require.Greater(t, refSchema.Status.Version, 0, "referent must have a registry version")

		// 2) Apply dependent that imports the referent via kafkaSchemaRef.
		require.NoError(t, s.Apply(getKafkaSchemaKafkaSchemaRefYaml(cfg.Project, kafkaName, mainSchemaName, mainSubjectName, refSchemaName)))
		mainSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(mainSchema, mainSchemaName))

		avnSchema, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, mainSchema.Status.Version)
		require.NoError(t, err)
		require.Len(t, avnSchema.References, 1, "dependent must start with one reference attached")

		// 3) Edit the dependent.
		updatedMain := mainSchema.DeepCopy()
		updatedMain.Spec.References = nil
		updatedMain.Spec.Schema = `syntax = "proto3";
message Order {
  string order_id = 1;
}
`
		require.NoError(t, k8sClient.Update(ctx, updatedMain))
		require.NoError(t, s.GetRunning(updatedMain, mainSchemaName))

		// 4) The registry must report zero references.
		require.NoError(t, retryForever(ctx, "registry drops references when spec drops them", func() (bool, error) {
			latestMain := new(v1alpha1.KafkaSchema)
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: mainSchemaName, Namespace: defaultNamespace}, latestMain); err != nil {
				return true, err
			}
			got, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, latestMain.Status.Version)
			if err != nil {
				return true, err
			}
			if len(got.References) != 0 {
				// Still stale. Retry until the operator re-POSTs without references.
				return true, nil
			}
			return false, nil
		}))

		assert.NoError(t, s.Delete(updatedMain, subjectExists(mainSubjectName)))
	})

	t.Run("referent stays in Terminating while dependent exists, then cleans up", func(t *testing.T) {
		refSchemaName := randName("kafka-schema-guard-ref")
		refSubjectName := randName("kafka-schema-guard-ref")
		mainSchemaName := randName("kafka-schema-guard-main")
		mainSubjectName := randName("kafka-schema-guard-main")

		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, refSchemaName, refSubjectName)))
		refSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(refSchema, refSchemaName))

		require.NoError(t, s.Apply(getKafkaSchemaKafkaSchemaRefYaml(cfg.Project, kafkaName, mainSchemaName, mainSubjectName, refSchemaName)))
		mainSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(mainSchema, mainSchemaName))

		// The k8s API server accepts it (sets deletionTimestamp),
		// but the operator refuses while the dependent's index entry still exists.
		require.NoError(t, k8sClient.Delete(ctx, refSchema))

		require.NoError(t, retryForever(ctx, "referent stays in Terminating while dependent exists", func() (bool, error) {
			latest := new(v1alpha1.KafkaSchema)
			err := k8sClient.Get(ctx, client.ObjectKey{Name: refSchemaName, Namespace: defaultNamespace}, latest)
			if isNotFound(err) {
				return false, fmt.Errorf("referent vanished while dependent still references it — local guard regressed")
			}
			if err != nil {
				return true, err
			}
			if latest.DeletionTimestamp.IsZero() {
				return true, nil // delete hasn't landed yet, retry
			}
			return false, nil
		}))

		// Subject must still be in the registry.
		require.NoError(t, subjectExists(refSubjectName)(),
			"referent's subject must still be registered while the local guard holds the delete back")

		// Delete the dependent.
		require.NoError(t, s.Delete(mainSchema, subjectExists(mainSubjectName)))

		require.NoError(t, retryForever(ctx, "referent finishes terminating after dependent is removed", func() (bool, error) {
			err := k8sClient.Get(ctx, client.ObjectKey{Name: refSchemaName, Namespace: defaultNamespace}, new(v1alpha1.KafkaSchema))
			return !isNotFound(err), nil
		}))
		err := subjectExists(refSubjectName)()
		require.True(t, isNotFound(err), "referent's subject must be gone from the active listing after the CR finishes terminating: %v", err)
	})

	// With A -> B -> C all connected via kafkaSchemaRef, mutating A must cause B to re-POST
	// against A's new version AND C to re-POST against B's new version.
	t.Run("3-level chain converges when root advances", func(t *testing.T) {
		aSchemaName := randName("kafka-schema-chain-a")
		aSubjectName := randName("kafka-schema-chain-a")
		bSchemaName := randName("kafka-schema-chain-b")
		bSubjectName := randName("kafka-schema-chain-b")
		cSchemaName := randName("kafka-schema-chain-c")
		cSubjectName := randName("kafka-schema-chain-c")

		// A: leaf referent. customer.proto contributes the Customer type.
		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, aSchemaName, aSubjectName)))
		aSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(aSchema, aSchemaName))
		initialAVersion := aSchema.Status.Version
		require.Greater(t, initialAVersion, 0, "A must have a registry version")

		// B: depends on A via kafkaSchemaRef. order.proto imports customer.proto.
		require.NoError(t, s.Apply(getKafkaSchemaChainMidYaml(cfg.Project, kafkaName, bSchemaName, bSubjectName, aSchemaName)))
		bSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(bSchema, bSchemaName))

		avnB, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, bSubjectName, bSchema.Status.Version)
		require.NoError(t, err)
		require.Len(t, avnB.References, 1, "B must reference A")
		require.Equal(t, initialAVersion, avnB.References[0].Version, "B must reference A's initial version")

		// C: depends on B via kafkaSchemaRef. order_confirmation.proto imports order.proto.
		require.NoError(t, s.Apply(getKafkaSchemaChainTopYaml(cfg.Project, kafkaName, cSchemaName, cSubjectName, bSchemaName)))
		cSchema := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(cSchema, cSchemaName))

		avnC, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, cSubjectName, cSchema.Status.Version)
		require.NoError(t, err)
		require.Len(t, avnC.References, 1, "C must reference B")
		require.Equal(t, bSchema.Status.Version, avnC.References[0].Version, "C must reference B's initial version")

		// Mutate A. Aiven assigns A v2. B and C must propagate.
		updatedA := aSchema.DeepCopy()
		updatedA.Spec.Schema = strings.ReplaceAll(aSchema.Spec.Schema, "string email", "string email_address")
		require.NoError(t, k8sClient.Update(ctx, updatedA))
		require.NoError(t, s.GetRunning(updatedA, aSchemaName))
		require.Greater(t, updatedA.Status.Version, initialAVersion, "A did not advance")
		newAVersion := updatedA.Status.Version

		var newBVersion int
		require.NoError(t, retryForever(ctx, "B propagates A's new version", func() (bool, error) {
			latestB := new(v1alpha1.KafkaSchema)
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: bSchemaName, Namespace: defaultNamespace}, latestB); err != nil {
				return true, err
			}
			got, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, bSubjectName, latestB.Status.Version)
			if err != nil {
				return true, err
			}
			if len(got.References) != 1 {
				return true, fmt.Errorf("expected 1 reference on B, got %d", len(got.References))
			}
			if got.References[0].Version != newAVersion {
				return true, nil // still stale
			}
			newBVersion = latestB.Status.Version
			return false, nil
		}))
		require.Greater(t, newBVersion, bSchema.Status.Version, "B did not advance to a new registry version")

		require.NoError(t, retryForever(ctx, "C propagates B's new version", func() (bool, error) {
			latestC := new(v1alpha1.KafkaSchema)
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: cSchemaName, Namespace: defaultNamespace}, latestC); err != nil {
				return true, err
			}
			got, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, cSubjectName, latestC.Status.Version)
			if err != nil {
				return true, err
			}
			if len(got.References) != 1 {
				return true, fmt.Errorf("expected 1 reference on C, got %d", len(got.References))
			}
			if got.References[0].Version != newBVersion {
				return true, nil // still stale
			}
			return false, nil
		}))

		assert.NoError(t, s.Delete(cSchema, subjectExists(cSubjectName)))
		assert.NoError(t, s.Delete(bSchema, subjectExists(bSubjectName)))
		assert.NoError(t, s.Delete(aSchema, subjectExists(aSubjectName)))
	})

	// Hard-delete purges the subject's metadata, re-applied KafkaSchema with
	// the same subjectName starts at version 1.
	t.Run("subject can be recreated cleanly after delete", func(t *testing.T) {
		schemaName := randName("kafka-schema-recreate")
		subjectName := randName("kafka-schema-recreate")

		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, schemaName, subjectName)))
		first := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(first, schemaName))

		require.NoError(t, s.Delete(first, subjectExists(subjectName)))

		require.NoError(t, s.Apply(getKafkaSchemaRefBaseYaml(cfg.Project, kafkaName, schemaName, subjectName)))
		second := new(v1alpha1.KafkaSchema)
		require.NoError(t, s.GetRunning(second, schemaName))

		assert.Equal(t, 1, second.Status.Version, "recreated subject must start at version 1; soft-delete-only would have preserved the old version counter")

		require.NoError(t, s.Delete(second, subjectExists(subjectName)))
	})
}

func TestKafkaSchemaReferencesValidation(t *testing.T) {
	t.Parallel()

	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	testCases := []struct {
		name                   string
		yaml                   string
		expectErrorMsgContains string
	}{
		{
			name: "references with AVRO schema type",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: avro-with-refs
  schemaType: AVRO
  schema: '{}'
  references:
    - name: other.avsc
      subject: other-subject
      version: 1
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "references are only supported for PROTOBUF and JSON schema types",
		},
		{
			name: "reference with empty name",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: empty-ref-name
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: ""
      subject: valid-subject
      version: 1
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "spec.references[0].name",
		},
		{
			name: "reference with empty subject",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: empty-ref-subject
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
      subject: ""
      version: 1
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "spec.references[0].subject in body should be at least 1 chars long",
		},
		{
			name: "reference with zero version",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: zero-ref-version
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
      subject: valid-subject
      version: 0
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "spec.references[0].version in body should be greater than or equal to 1",
		},
		{
			name: "reference sets both explicit and kafkaSchemaRef",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: both-ref-kinds
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
      subject: valid-subject
      version: 1
      kafkaSchemaRef:
        name: some-other-schema
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "set both subject and version, or set kafkaSchemaRef, but not both",
		},
		{
			name: "reference sets neither explicit nor kafkaSchemaRef",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: empty-ref
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "set both subject and version, or set kafkaSchemaRef, but not both",
		},
		{
			// Reference names must be unique within a single KafkaSchema.
			name: "duplicate reference name",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  serviceName: fake-kafka
  subjectName: dup-ref-name
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
      subject: subject-a
      version: 1
    - name: customer.proto
      subject: subject-b
      version: 1
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "duplicate",
		},
		{
			name: "kafkaSchemaRef points at the owning KafkaSchema",
			yaml: fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[1]s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %[2]s
  serviceName: fake-kafka
  subjectName: self-ref
  schemaType: PROTOBUF
  schema: 'syntax = "proto3";'
  references:
    - name: customer.proto
      kafkaSchemaRef:
        name: %[1]s
`, randName("kafka-schema-val"), cfg.Project),
			expectErrorMsgContains: "kafkaSchemaRef cannot point to the KafkaSchema itself",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Apply(tc.yaml)
			assert.ErrorContains(t, err, tc.expectErrorMsgContains)
		})
	}
}

func getKafkaSchemaRefYaml(project, kafkaName, schemaName, subjectName, refSubject string, refVersion int) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: PROTOBUF
  schema: |
    syntax = "proto3";
    import "customer.proto";
    message Order {
      string order_id = 1;
      Customer customer = 2;
    }
  references:
    - name: customer.proto
      subject: %[5]s
      version: %[6]d
`, project, kafkaName, schemaName, subjectName, refSubject, refVersion)
}

// getKafkaSchemaKafkaSchemaRefYaml builds a dependent KafkaSchema that resolves
// its reference via kafkaSchemaRef instead of pinning subject+version.
func getKafkaSchemaKafkaSchemaRefYaml(project, kafkaName, schemaName, subjectName, refName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: PROTOBUF
  schema: |
    syntax = "proto3";
    import "customer.proto";
    message Order {
      string order_id = 1;
      Customer customer = 2;
    }
  references:
    - name: customer.proto
      kafkaSchemaRef:
        name: %[5]s
`, project, kafkaName, schemaName, subjectName, refName)
}

// getKafkaSchemaChainMidYaml builds B in the A -> B -> C chain. B imports
// customer.proto from A and exports an Order message that C will import.
func getKafkaSchemaChainMidYaml(project, kafkaName, schemaName, subjectName, refName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: PROTOBUF
  schema: |
    syntax = "proto3";
    import "customer.proto";
    message Order {
      string order_id = 1;
      Customer customer = 2;
    }
  references:
    - name: customer.proto
      kafkaSchemaRef:
        name: %[5]s
`, project, kafkaName, schemaName, subjectName, refName)
}

// getKafkaSchemaChainTopYaml builds C in the A -> B -> C chain. C imports
// order.proto from B.
func getKafkaSchemaChainTopYaml(project, kafkaName, schemaName, subjectName, refName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: PROTOBUF
  schema: |
    syntax = "proto3";
    import "order.proto";
    message OrderConfirmation {
      Order order = 1;
      string confirmation_id = 2;
    }
  references:
    - name: order.proto
      kafkaSchemaRef:
        name: %[5]s
`, project, kafkaName, schemaName, subjectName, refName)
}

func getKafkaSchemaRefBaseYaml(project, kafkaName, schemaName, subjectName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: PROTOBUF
  schema: |
    syntax = "proto3";
    message Customer {
      string name = 1;
      string email = 2;
    }
`, project, kafkaName, schemaName, subjectName)
}

func getKafkaSchemaYaml(project, kafkaName, schemaName, subjectName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  subjectName: %[4]s
  schemaType: AVRO
  compatibilityLevel: BACKWARD
  schema: |
    {
        "doc": "example_doc",
        "fields": [{
            "default": 5,
            "doc": "field_doc",
            "name": "field_name",
            "namespace": "field_namespace",
            "type": "int"
        }],
        "name": "example_name",
        "namespace": "example_namespace",
        "type": "record"
    }
`, project, kafkaName, schemaName, subjectName)
}
