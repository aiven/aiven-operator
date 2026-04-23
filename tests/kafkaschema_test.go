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
	subjectName := randName("kafka-schema")
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

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafka, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafka.GetName()
	refSchemaName := randName("kafka-schema-ref")
	refSubjectName := randName("kafka-schema-ref")
	mainSchemaName := randName("kafka-schema-main")
	mainSubjectName := randName("kafka-schema-main")

	s := NewSession(ctx, k8sClient)

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

	// Cleanup: delete the referencing schema explicitly.
	// The referenced (base) schema is cleaned up by the shared Kafka service teardown,
	// because schema registry soft-delete does not clear the reference —
	// a referenced schema cannot be soft-deleted even after the referencing one is removed.
	assert.NoError(t, s.Delete(mainSchema, func() error {
		_, err := avnGen.ServiceSchemaRegistrySubjectVersionGet(ctx, cfg.Project, kafkaName, mainSubjectName, mainSchema.Status.Version)
		return err
	}))
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
			expectErrorMsgContains: "spec.references[0].subject",
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
			expectErrorMsgContains: "spec.references[0].version",
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
