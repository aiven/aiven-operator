//go:build kafka

package tests

import (
	"context"
	"fmt"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	"github.com/aiven/aiven-operator/controllers"
)

func TestKafkaQuota(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	kafkaService, releaseKafka, err := sharedResources.AcquireKafka(ctx)
	require.NoError(t, err)
	defer releaseKafka()

	kafkaName := kafkaService.GetName()
	quotaName := randName("kafka-quota")
	user := randName("quota-user")
	clientID := randName("quota-client")

	yml := getKafkaQuotaYaml(cfg.Project, kafkaName, quotaName, user, clientID, 1000, 2000, 50)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	// Waits for the kube object to reach Running state
	quota := new(v1alpha1.KafkaQuota)
	require.NoError(t, s.GetRunning(quota, quotaName))

	// THEN
	assert.True(t, meta.IsStatusConditionTrue(quota.Status.Conditions, "Running"))
	assert.Equal(t, user, quota.Spec.User)
	assert.Equal(t, clientID, quota.Spec.ClientID)
	require.NotNil(t, quota.Spec.ConsumerByteRate)
	assert.EqualValues(t, 1000, *quota.Spec.ConsumerByteRate)
	require.NotNil(t, quota.Spec.ProducerByteRate)
	assert.EqualValues(t, 2000, *quota.Spec.ProducerByteRate)
	require.NotNil(t, quota.Spec.RequestPercentage)
	assert.InDelta(t, 50.0, *quota.Spec.RequestPercentage, 0.0001)

	// Verify the quota is present at Aiven side with matching values
	quotaAvn, err := getKafkaQuotaFromAiven(ctx, avnGen, cfg.Project, kafkaName, user, clientID)
	require.NoError(t, err)
	require.NotNil(t, quotaAvn.ConsumerByteRate)
	assert.InDelta(t, 1000.0, *quotaAvn.ConsumerByteRate, 0.0001)
	require.NotNil(t, quotaAvn.ProducerByteRate)
	assert.InDelta(t, 2000.0, *quotaAvn.ProducerByteRate, 0.0001)
	require.NotNil(t, quotaAvn.RequestPercentage)
	assert.InDelta(t, 50.0, *quotaAvn.RequestPercentage, 0.0001)

	// WHEN the quota is updated in-place
	quotaCopy := quota.DeepCopy()
	newConsumerRate := int64(5000)
	quotaCopy.Spec.ConsumerByteRate = &newConsumerRate
	require.NoError(t, k8sClient.Update(ctx, quotaCopy))

	// THEN the update is reflected on Aiven side
	updated := new(v1alpha1.KafkaQuota)
	require.NoError(t, s.GetRunning(updated, quotaName))
	require.NotNil(t, updated.Spec.ConsumerByteRate)
	assert.EqualValues(t, newConsumerRate, *updated.Spec.ConsumerByteRate)

	updatedAvn, err := getKafkaQuotaFromAiven(ctx, avnGen, cfg.Project, kafkaName, user, clientID)
	require.NoError(t, err)
	require.NotNil(t, updatedAvn.ConsumerByteRate)
	assert.InDelta(t, float64(newConsumerRate), *updatedAvn.ConsumerByteRate, 0.0001)

	// Validate delete: after deletion, describing the quota should return a payload with no
	// user/client-id (the API's way of saying "no quota is configured for this selector").
	assert.NoError(t, s.Delete(updated, func() error {
		got, err := avnGen.ServiceKafkaQuotaDescribe(
			ctx, cfg.Project, kafkaName,
			kafka.ServiceKafkaQuotaDescribeUser(user),
			kafka.ServiceKafkaQuotaDescribeClientId(clientID),
		)
		if err != nil {
			return err
		}
		if got == nil || (got.User == nil && got.ClientId == nil) {
			return controllers.NewNotFound(fmt.Sprintf("Kafka quota for user=%q client-id=%q not found", user, clientID))
		}
		return nil
	}))
}

func getKafkaQuotaYaml(project, kafkaName, quotaName, user, clientID string, consumer, producer int64, requestPercentage float64) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: KafkaQuota
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  serviceName: %[2]s
  user: %[4]s
  clientId: %[5]s
  consumerByteRate: %[6]d
  producerByteRate: %[7]d
  requestPercentage: %[8]v
`, project, kafkaName, quotaName, user, clientID, consumer, producer, requestPercentage)
}

// getKafkaQuotaFromAiven fetches the quota matching the given selector from the Aiven API.
// Returns a NotFound error if no matching quota exists so the caller can use it in delete-wait helpers.
func getKafkaQuotaFromAiven(ctx context.Context, avn avngen.Client, project, serviceName, user, clientID string) (*kafka.ServiceKafkaQuotaDescribeOut, error) {
	got, err := avn.ServiceKafkaQuotaDescribe(
		ctx, project, serviceName,
		kafka.ServiceKafkaQuotaDescribeUser(user),
		kafka.ServiceKafkaQuotaDescribeClientId(clientID),
	)
	if err != nil {
		return nil, err
	}
	if got == nil || (got.User == nil && got.ClientId == nil) {
		return nil, controllers.NewNotFound(fmt.Sprintf("Kafka quota for user=%q client-id=%q not found", user, clientID))
	}
	return got, nil
}
