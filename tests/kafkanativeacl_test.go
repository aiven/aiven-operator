package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

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
		"doc[0].metadata.name":    kafkaName,
		"doc[0].spec.project":     cfg.Project,
		"doc[1].metadata.name":    aclName,
		"doc[1].spec.project":     cfg.Project,
		"doc[1].spec.serviceName": kafkaName,
	})
	require.NoError(t, err)
	s := NewSession(ctx, k8sClient, cfg.Project)

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
