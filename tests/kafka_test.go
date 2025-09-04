//go:build kafka

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func getKafkaYaml(project, name, cloudName string) string {
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
  cloudName: %[3]s
  plan: business-4
  disk_space: 600GiB

  tags:
    env: test
    instance: foo

  userConfig:
    kafka_rest: true
    kafka_connect: true
    schema_registry: true
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16
    kafka_authentication_methods:
      sasl: true
`, project, name, cloudName)
}

func TestKafka(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	name := randName("kafka")
	yml := getKafkaYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	// Applies given manifest
	require.NoError(t, s.Apply(yml))

	// Waits kube objects
	ks := new(v1alpha1.Kafka)
	require.NoError(t, s.GetRunning(ks, name))

	// THEN
	ksAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, ksAvn.ServiceName, ks.GetName())
	assert.Equal(t, serviceRunningState, ks.Status.State)
	assert.Contains(t, serviceRunningStatesAiven, ksAvn.State)
	assert.Equal(t, ksAvn.Plan, ks.Spec.Plan)
	assert.Equal(t, ksAvn.CloudName, ks.Spec.CloudName)
	assert.Equal(t, "600GiB", ks.Spec.DiskSpace)
	assert.Equal(t, int(614400), *ksAvn.DiskSpaceMb)
	assert.Equal(t, map[string]string{"env": "test", "instance": "foo"}, ks.Spec.Tags)
	ksTags, err := avnGen.ProjectServiceTagsList(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, ksTags, ks.Spec.Tags)

	// UserConfig test
	require.NotNil(t, ks.Spec.UserConfig)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaRest)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaConnect)
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.SchemaRegistry)

	// Validates ip filters
	require.Len(t, ks.Spec.UserConfig.IpFilter, 2)

	// First entry
	assert.Equal(t, "0.0.0.0/32", ks.Spec.UserConfig.IpFilter[0].Network)
	assert.Equal(t, "bar", *ks.Spec.UserConfig.IpFilter[0].Description)

	// Second entry
	assert.Equal(t, "10.20.0.0/16", ks.Spec.UserConfig.IpFilter[1].Network)
	assert.Nil(t, ks.Spec.UserConfig.IpFilter[1].Description)

	// Compares with Aiven ip_filter
	var ipFilterAvn []*kafkauserconfig.IpFilter
	require.NoError(t, castInterface(ksAvn.UserConfig["ip_filter"], &ipFilterAvn))
	assert.Equal(t, ipFilterAvn, ks.Spec.UserConfig.IpFilter)

	// Secrets test
	secret, err := s.GetSecret(ks.GetName())
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["HOST"])
	assert.NotEmpty(t, secret.Data["PORT"])
	assert.NotEmpty(t, secret.Data["USERNAME"])
	assert.NotEmpty(t, secret.Data["PASSWORD"])
	assert.NotEmpty(t, secret.Data["ACCESS_CERT"])
	assert.NotEmpty(t, secret.Data["ACCESS_KEY"])
	assert.NotEmpty(t, secret.Data["CA_CERT"])

	// New secrets
	assert.NotEmpty(t, secret.Data["KAFKA_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_PORT"])
	assert.NotEmpty(t, secret.Data["KAFKA_USERNAME"])
	assert.NotEmpty(t, secret.Data["KAFKA_PASSWORD"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_CERT"])
	assert.NotEmpty(t, secret.Data["KAFKA_ACCESS_KEY"])
	assert.NotEmpty(t, secret.Data["KAFKA_CA_CERT"])

	// SASL test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaAuthenticationMethods.Sasl)
	assert.NotEmpty(t, secret.Data["KAFKA_SASL_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_SASL_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SASL_PORT"], secret.Data["KAFKA_PORT"])

	// Schema registry test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.SchemaRegistry)
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_URI"])
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_SCHEMA_REGISTRY_PORT"], secret.Data["KAFKA_SASL_PORT"])

	// Kafka Connect test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaConnect)
	assert.NotEmpty(t, secret.Data["KAFKA_CONNECT_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_CONNECT_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_CONNECT_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_CONNECT_PORT"], secret.Data["KAFKA_SASL_PORT"])

	// Kafka REST test
	assert.Equal(t, anyPointer(true), ks.Spec.UserConfig.KafkaRest)
	assert.NotEmpty(t, secret.Data["KAFKA_REST_URI"])
	assert.NotEmpty(t, secret.Data["KAFKA_REST_HOST"])
	assert.NotEmpty(t, secret.Data["KAFKA_REST_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_REST_PORT"], secret.Data["KAFKA_PORT"])
	assert.NotEqual(t, secret.Data["KAFKA_REST_PORT"], secret.Data["KAFKA_SASL_PORT"])

	// Tests service power off functionality
	// Note: Power on testing is handled generically in generic_service_handler_test.go
	// since it's consistent across services. Power off testing is done here since
	// the flow can vary by service type and may require service-specific steps.
	poweredOff := ks.DeepCopy()
	poweredOff.Spec.Powered = anyPointer(false)
	require.NoError(t, k8sClient.Update(ctx, poweredOff))
	require.NoError(t, s.GetRunning(poweredOff, name))

	poweredOffAvn, err := avnGen.ServiceGet(ctx, cfg.Project, name)
	require.NoError(t, err)
	assert.Equal(t, service.ServiceStateTypePoweroff, poweredOffAvn.State)
}

// TestKafkaControllerProvisioningAndUpdates tests the complete flow
func TestKafkaControllerProvisioningAndUpdates(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	name := randName("kafka-provisioning-flow")
	yml := getKafkaYaml(cfg.Project, name, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	require.NoError(t, s.Apply(yml))

	ks := new(v1alpha1.Kafka)

	// Step 1: wait for Kafka object to exist and be processed
	require.Eventually(t, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: defaultNamespace}, ks); err != nil {
			return false
		}

		annotations := ks.GetAnnotations()
		if annotations == nil {
			return false
		}

		_, exists := annotations["controllers.aiven.io/generation-was-processed"]
		return exists
	}, 30*time.Second, 2*time.Second, "Kafka should be processed by controller")

	initialAnnotations := ks.GetAnnotations()
	require.NotNil(t, initialAnnotations, "annotations should exist in initialized state")

	initialProcessedGen, exists := initialAnnotations["controllers.aiven.io/generation-was-processed"]
	require.True(t, exists, "processedGeneration should be set when initialized")

	initialCurrentGen := fmt.Sprintf("%d", ks.GetGeneration())
	assert.Equal(t, initialCurrentGen, initialProcessedGen, "processedGeneration should match generation when initialized")

	// Step 2: wait for Initialized condition
	require.Eventually(t, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks); err != nil {
			return false
		}

		for _, condition := range ks.Status.Conditions {
			if condition.Type == "Initialized" && condition.Status == metav1.ConditionTrue {
				return true
			}
		}
		return false
	}, 30*time.Second, 5*time.Second, "Kafka should become Initialized after createOrUpdate")

	// Step 3: wait for service to be actually running
	require.NoError(t, s.GetRunning(ks, name), "Kafka should eventually become running")

	// Step 4: validate all metadata
	runningAnnotations := ks.GetAnnotations()
	runningProcessedGen := runningAnnotations["controllers.aiven.io/generation-was-processed"]
	runningCurrentGen := fmt.Sprintf("%d", ks.GetGeneration())
	assert.Equal(t, runningCurrentGen, runningProcessedGen, "processedGeneration should remain current in running state")

	runningAnnotationValue := runningAnnotations["controllers.aiven.io/instance-is-running"]
	assert.Equal(t, "true", runningAnnotationValue, "instanceIsRunning should be 'true' for running service")

	var runningCondition *metav1.Condition
	for i := range ks.Status.Conditions {
		if ks.Status.Conditions[i].Type == "Running" {
			runningCondition = &ks.Status.Conditions[i]
			break
		}
	}
	require.NotNil(t, runningCondition, "Running condition should exist")
	assert.Equal(t, metav1.ConditionTrue, runningCondition.Status, "Running condition should be True")

	// update tags
	updated1 := ks.DeepCopy()
	if updated1.Spec.Tags == nil {
		updated1.Spec.Tags = make(map[string]string)
	}
	updated1.Spec.Tags["continuous-test"] = "change-1"
	require.NoError(t, k8sClient.Update(ctx, updated1))

	require.Eventually(t, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks); err != nil {
			return false
		}
		return ks.Spec.Tags != nil && ks.Spec.Tags["continuous-test"] == "change-1"
	}, 30*time.Second, 2*time.Second, "first tag change should be processed")

	require.Eventually(t, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks); err != nil {
			return false
		}

		annotations := ks.GetAnnotations()
		if annotations == nil {
			return false
		}

		processedGen, exists := annotations["controllers.aiven.io/generation-was-processed"]
		if !exists {
			return false
		}

		currentGen := fmt.Sprintf("%d", ks.GetGeneration())
		return processedGen == currentGen
	}, 30*time.Second, 2*time.Second, "first change should be processed and generation updated")

	change1Annotations := ks.GetAnnotations()
	change1ProcessedGen := change1Annotations["controllers.aiven.io/generation-was-processed"]
	change1CurrentGen := fmt.Sprintf("%d", ks.GetGeneration())
	assert.Equal(t, change1CurrentGen, change1ProcessedGen, "processedGeneration should match after first change")

	assert.NotNil(t, ks.Spec.Tags, "tags should be set")
	assert.Equal(t, "change-1", ks.Spec.Tags["continuous-test"], "Tag should be applied")

	// update disk space and add second tag
	updated2 := ks.DeepCopy()
	updated2.Spec.DiskSpace = "630GiB"
	if updated2.Spec.Tags == nil {
		updated2.Spec.Tags = make(map[string]string)
	}
	updated2.Spec.Tags["advanced-test"] = "change-2"
	require.NoError(t, k8sClient.Update(ctx, updated2))

	require.Eventually(t, func() bool {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks); err != nil {
			return false
		}

		if ks.Spec.DiskSpace != "630GiB" {
			return false
		}
		if ks.Spec.Tags == nil || ks.Spec.Tags["advanced-test"] != "change-2" {
			return false
		}

		annotations := ks.GetAnnotations()
		if annotations == nil {
			return false
		}

		processedGen, exists := annotations["controllers.aiven.io/generation-was-processed"]
		if !exists {
			return false
		}

		currentGen := fmt.Sprintf("%d", ks.GetGeneration())
		return processedGen == currentGen
	}, 2*time.Minute, 2*time.Second, "second change should be processed and generation updated")

	change2Annotations := ks.GetAnnotations()
	change2ProcessedGen := change2Annotations["controllers.aiven.io/generation-was-processed"]
	change2CurrentGen := fmt.Sprintf("%d", ks.GetGeneration())
	assert.Equal(t, change2CurrentGen, change2ProcessedGen, "processedGeneration should match after second change")

	assert.Equal(t, "630GiB", ks.Spec.DiskSpace, "disk space should be updated")

	finalRunningAnnotation := change2Annotations["controllers.aiven.io/instance-is-running"]
	assert.Equal(t, "true", finalRunningAnnotation, "service should remain running through continuous changes")

	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks))

	finalCurrentGen := fmt.Sprintf("%d", ks.GetGeneration())
	finalAnnotations := ks.GetAnnotations()
	finalProcessedGen, exists := finalAnnotations["controllers.aiven.io/generation-was-processed"]
	require.True(t, exists, "processedGeneration annotation should exist at end")
	assert.Equal(t, finalCurrentGen, finalProcessedGen, "processedGeneration should be current at end")

	assert.Equal(t, "change-1", ks.Spec.Tags["continuous-test"], "first tag change should persist")
	assert.Equal(t, "change-2", ks.Spec.Tags["advanced-test"], "second tag change should persist")

	finalRunningValue := finalAnnotations["controllers.aiven.io/instance-is-running"]
	assert.Equal(t, "true", finalRunningValue, "service should be running at test end")

	finalConditions := ks.Status.Conditions
	hasInitialized := false
	hasRunning := false
	hasError := false

	for _, condition := range finalConditions {
		switch condition.Type {
		case "Initialized":
			hasInitialized = true
			assert.Equal(t, metav1.ConditionTrue, condition.Status, "initialized condition should remain True")
		case "Running":
			hasRunning = true
			assert.Equal(t, metav1.ConditionTrue, condition.Status, "running condition should be True (powered-off is valid)")
		case "Error":
			hasError = true
			t.Logf("unexpected Error condition: %s - %s", condition.Reason, condition.Message)
		}
	}

	assert.True(t, hasInitialized, "should have Initialized condition")
	assert.True(t, hasRunning, "should have Running condition")
	assert.False(t, hasError, "should not have Error conditions in final state")
}

// TestKafkaController_SecretManagement tests secret creation, updates, and management
func TestKafkaController_SecretManagement(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	t.Run("SecretRecreation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testCtx()
		defer cancel()

		name := randName("kafka-secret-recreation")
		yml := getKafkaYaml(cfg.Project, name, cfg.PrimaryCloudName)
		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(yml))

		ks := new(v1alpha1.Kafka)
		require.NoError(t, s.GetRunning(ks, name))

		secret, err := s.GetSecret(ks.GetName())
		require.NoError(t, err, "connection secret should exist")

		requiredFields := []string{
			"KAFKA_HOST", "KAFKA_PORT", "KAFKA_USERNAME", "KAFKA_PASSWORD",
			"KAFKA_ACCESS_CERT", "KAFKA_ACCESS_KEY", "KAFKA_CA_CERT",
		}

		for _, field := range requiredFields {
			assert.NotEmpty(t, secret.Data[field], "secret should have %s field", field)
		}

		originalHost := string(secret.Data["KAFKA_HOST"])
		require.NotEmpty(t, originalHost, "original host should not be empty")

		require.NoError(t, k8sClient.Delete(ctx, secret))

		require.Eventually(t, func() bool {
			_, err := s.GetSecret(ks.GetName())
			return err != nil // should return error when secret doesn't exist
		}, 10*time.Second, 1*time.Second, "secret should be deleted")

		// trigger reconciliation with a spec change
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(ks), ks))
		updated := ks.DeepCopy()
		if updated.Spec.Tags == nil {
			updated.Spec.Tags = make(map[string]string)
		}
		updated.Spec.Tags["secret-test"] = fmt.Sprintf("trigger-%d", time.Now().Unix())
		require.NoError(t, k8sClient.Update(ctx, updated))

		// wait for secret to be recreated by controller
		require.Eventually(t, func() bool {
			recreatedSecret, err := s.GetSecret(ks.GetName())
			if err != nil {
				return false
			}

			for _, field := range requiredFields {
				if len(recreatedSecret.Data[field]) == 0 {
					return false
				}
			}

			newHost := string(recreatedSecret.Data["KAFKA_HOST"])
			return newHost != "" && newHost == originalHost
		}, 2*time.Minute, 5*time.Second, "secret should be recreated with all required fields")

		finalSecret, err := s.GetSecret(ks.GetName())
		require.NoError(t, err, "final secret should exist")
		finalHost := string(finalSecret.Data["KAFKA_HOST"])
		assert.Equal(t, originalHost, finalHost, "recreated secret should have same host")
	})

	t.Run("SecretDisabled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := testCtx()
		defer cancel()

		noSecretName := randName("kafka-no-secret")

		noSecretYml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: %s
spec:
  authSecretRef:
    name: aiven-token
    key: token
  project: %s
  cloudName: %s
  plan: startup-2
  connInfoSecretTargetDisabled: true
  userConfig:
    kafka_rest: true
`, noSecretName, cfg.Project, cfg.PrimaryCloudName)

		s := NewSession(ctx, k8sClient)
		defer s.Destroy(t)

		require.NoError(t, s.Apply(noSecretYml))

		noSecretKs := new(v1alpha1.Kafka)
		require.NoError(t, s.GetRunning(noSecretKs, noSecretName))

		_, err := s.GetSecret(noSecretName)
		assert.Error(t, err, "no secret should be created when connInfoSecretTargetDisabled is true")

		assert.Equal(t, serviceRunningState, noSecretKs.Status.State)
	})
}
