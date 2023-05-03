package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getKafkaSchemaYaml(project, kafkaName, schemaName, subjectName string) string {
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
  cloudName: google-europe-west1
  plan: startup-2
  userConfig:
    schema_registry: true

---

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

func TestKafkaSchema(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	kafkaName := randName("kafka-schema")
	schemaName := randName("kafka-schema")
	subjectName := randName("kafka-schema")
	yml := getKafkaSchemaYaml(testProject, kafkaName, schemaName, subjectName)
	s := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer s.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	// THEN
	// Kafka test
	kafkaAvn, err := avnClient.Services.Get(testProject, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.Name, kafka.GetName())
	assert.Equal(t, "RUNNING", kafka.Status.State)
	assert.Equal(t, kafkaAvn.State, kafka.Status.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)
	require.NotNil(t, kafka.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), kafka.Spec.UserConfig.SchemaRegistry)

	// KafkaSchema test
	schema := new(v1alpha1.KafkaSchema)
	require.NoError(t, s.GetRunning(schema, schemaName))
	assert.Equal(t, schemaName, schema.Name)
	assert.Equal(t, subjectName, schema.Spec.SubjectName)
	assert.Equal(t, kafkaName, schema.Spec.ServiceName)
	assert.Equal(t, "BACKWARD", schema.Spec.CompatibilityLevel)

	type schemaType struct {
		Default   any           `json:"default,omitempty"`
		Fields    []*schemaType `json:"fields,omitempty"`
		Doc       string        `json:"doc"`
		Name      string        `json:"name"`
		Namespace string        `json:"namespace"`
		Type      string        `json:"type"`
	}

	actualSchema := new(schemaType)
	err = json.Unmarshal([]byte(schema.Spec.Schema), &actualSchema)
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
	assert.Equal(t, "", cmp.Diff(expectedSchema, actualSchema))

	// Validates deleting, because deleted kafka drops schemas, and we want be sure deletion works
	assert.NoError(t, s.Delete(schema, func() error {
		_, err := avnClient.KafkaSubjectSchemas.Get(testProject, kafkaName, subjectName, 1)
		return err
	}))
}
