package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func getProjectVPCYaml(project, vpcName string) string {
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
  cloudName: google-europe-west2
  networkCidr: 10.0.0.0/24
`, project, vpcName)
}

func getKafkaForProjectVPCYaml(project, vpcID, kafkaName string) string {
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
  cloudName: google-europe-west2
  plan: startup-2
  projectVpcId: %[2]s
`, project, vpcID, kafkaName)
}

// TestProjectVPCID Kafka.Spec.ProjectVPCID
// creates vpc and then creates kafka with given vpcID
func TestProjectVPCID(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	vpcName := randName("project-vpc-id")
	vpcYaml := getProjectVPCYaml(testProject, vpcName)
	vpcSession := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer vpcSession.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, vpcSession.Apply(vpcYaml))

	// Waits kube object
	vpc := new(v1alpha1.ProjectVPC)
	require.NoError(t, vpcSession.GetRunning(vpc, vpcName))

	// THEN
	// Validates VPC
	vpcAvn, err := avnClient.VPCs.Get(testProject, vpc.Status.ID)
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", vpcAvn.State)
	assert.Equal(t, vpcAvn.State, vpc.Status.State)
	assert.Equal(t, vpcAvn.CloudName, vpc.Spec.CloudName)
	assert.Equal(t, "10.0.0.0/24", vpc.Spec.NetworkCidr)
	assert.Equal(t, vpcAvn.NetworkCIDR, vpc.Spec.NetworkCidr)

	// Creates Kafka with given vpcID
	kafkaName := randName("project-vpc-id")
	kafkaYaml := getKafkaForProjectVPCYaml(testProject, vpc.Status.ID, kafkaName)
	kafkaSession := NewSession(k8sClient, avnClient, testProject)

	// Cleans test afterwards
	defer kafkaSession.Destroy()

	// WHEN
	// Applies given manifest
	require.NoError(t, kafkaSession.Apply(kafkaYaml))

	// Waits kube objects
	kafka := new(v1alpha1.Kafka)
	require.NoError(t, kafkaSession.GetRunning(kafka, kafkaName))

	// THEN
	kafkaAvn, err := avnClient.Services.Get(testProject, kafkaName)
	require.NoError(t, err)
	assert.Equal(t, kafkaAvn.Name, kafka.GetName())
	assert.Equal(t, "RUNNING", kafka.Status.State)
	assert.Equal(t, kafkaAvn.State, kafka.Status.State)
	assert.Equal(t, kafkaAvn.Plan, kafka.Spec.Plan)
	assert.Equal(t, kafkaAvn.CloudName, kafka.Spec.CloudName)

	// Validates VPC
	assert.Equal(t, vpc.Status.ID, kafka.Spec.ProjectVPCID)
	assert.Equal(t, anyPointer(vpc.Status.ID), kafkaAvn.ProjectVPCID)
}
