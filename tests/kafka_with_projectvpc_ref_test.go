//go:build kafka

package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getKafkaWithProjectVPCRefYaml(project, vpcName, kafkaName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: ProjectVPC
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  networkCidr: 10.0.0.0/24

---

apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  projectVPCRef:
    name: %[2]s
`, project, vpcName, kafkaName, cloudName)
}

// TestKafkaWithProjectVPCRef tests Kafka.Spec.ProjectVPCRef
func TestKafkaWithProjectVPCRef(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	vpcName := randName("kafka-vpc")
	kafkaName := randName("kafka-vpc")
	yml := getKafkaWithProjectVPCRefYaml(cfg.Project, vpcName, kafkaName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(kafka, kafkaName))

	projectVPC := new(v1alpha1.ProjectVPC)
	require.NoError(t, s.GetRunning(projectVPC, vpcName))

	// THEN
	kafkaAvn, err := avnGen.ServiceGet(ctx, cfg.Project, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.ServiceName, kafka.GetName())
	assert.Equal(t, serviceRunningState, kafka.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, kafkaAvn.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// Validates VPC
	require.NotNil(t, kafka.Spec.ProjectVPCRef)
	assert.Equal(t, vpcName, kafka.Spec.ProjectVPCRef.Name)
	assert.Equal(t, projectVPC.Status.ID, kafkaAvn.ProjectVpcId)

	vpcAvn, err := avnGen.VpcGet(ctx, cfg.Project, projectVPC.Status.ID)
	require.NoError(t, err)
	assert.EqualValues(t, "ACTIVE", vpcAvn.State)
	assert.Equal(t, vpcAvn.State, projectVPC.Status.State)
	assert.Equal(t, vpcAvn.CloudName, projectVPC.Spec.CloudName)
	assert.Equal(t, "10.0.0.0/24", projectVPC.Spec.NetworkCidr)
	assert.Equal(t, vpcAvn.NetworkCidr, projectVPC.Spec.NetworkCidr)
}
