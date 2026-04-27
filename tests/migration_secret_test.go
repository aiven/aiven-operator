//go:build postgresql

package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestPgMigrationFromSecretSource(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	// GIVEN
	// 1. Create source PG service
	sourceName := randName("pg-mig-src")
	sourceYml := getSourcePgYaml(cfg.Project, sourceName, cfg.PrimaryCloudName)
	sourceSession := NewSession(ctx, k8sClient)
	defer sourceSession.Destroy(t)

	require.NoError(t, sourceSession.Apply(sourceYml))

	sourcePg := new(v1alpha1.PostgreSQL)
	require.NoError(t, sourceSession.GetRunning(sourcePg, sourceName))

	// 2. Get source service connection details from Aiven
	sourceAvn, err := avnGen.ServiceGet(ctx, cfg.Project, sourceName, service.ServiceGetIncludeSecrets(true))
	require.NoError(t, err)

	// 3. Create a Kubernetes Secret with the source connection details
	secretName := sourceName + "-migration-creds"
	migrationSecret := createMigrationSecretFromService(t, secretName, sourceAvn)
	require.NoError(t, k8sClient.Create(ctx, migrationSecret))
	defer func() {
		_ = k8sClient.Delete(ctx, migrationSecret)
	}()

	// WHEN
	// 4. Create target PG service with migrationSecretSource
	targetName := randName("pg-mig-tgt")
	targetYml := getTargetPgWithMigrationSecretYaml(cfg.Project, targetName, secretName, cfg.PrimaryCloudName)
	targetSession := NewSession(ctx, k8sClient)
	defer targetSession.Destroy(t)

	require.NoError(t, targetSession.Apply(targetYml))

	targetPg := new(v1alpha1.PostgreSQL)
	require.NoError(t, targetSession.GetRunning(targetPg, targetName))

	// THEN
	// 5. Verify target service is running
	assert.Equal(t, serviceRunningState, targetPg.Status.State)

	// 6. Verify migration config was sent to Aiven from the Secret
	targetAvn, err := avnGen.ServiceGet(ctx, cfg.Project, targetName)
	require.NoError(t, err)

	migrationConfig, ok := targetAvn.UserConfig["migration"].(map[string]any)
	require.True(t, ok, "migration config should be present in Aiven service user config")
	assert.Equal(t, sourceAvn.ServiceUriParams["host"], migrationConfig["host"])
	assert.Equal(t, sourceAvn.ServiceUriParams["dbname"], migrationConfig["dbname"])
	assert.Equal(t, sourceAvn.ServiceUriParams["user"], migrationConfig["username"])
	assert.Equal(t, "dump", migrationConfig["method"])

	// 7. Verify migrationSecretSource is set on the spec
	require.NotNil(t, targetPg.Spec.MigrationSecretSource)
	assert.Equal(t, secretName, targetPg.Spec.MigrationSecretSource.Name)

	// 8. Wait for migration to complete and verify the MigrationComplete condition
	require.NoError(t, retryForever(ctx, "wait for migration to complete", func() (bool, error) {
		pg := new(v1alpha1.PostgreSQL)
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: defaultNamespace}, pg); err != nil {
			return false, err
		}

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		if cond == nil {
			return true, nil // retry, condition not set yet
		}

		if cond.Status == metav1.ConditionTrue {
			return false, nil // done
		}

		if cond.Reason == v1alpha1.MigrationReasonFailed {
			return false, fmt.Errorf("migration failed: %s", cond.Message)
		}

		return true, nil // still in progress, retry
	}))

	// 9. Final check: MigrationComplete condition is True
	finalPg := new(v1alpha1.PostgreSQL)
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: defaultNamespace}, finalPg))
	cond := meta.FindStatusCondition(finalPg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
	require.NotNil(t, cond, "MigrationComplete condition should be set")
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, v1alpha1.MigrationReasonDone, cond.Reason)
}

func TestPgMigrationSecretTakesPrecedenceOverInline(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	// GIVEN
	// 1. Create source PG service
	sourceName := randName("pg-prec-src")
	sourceYml := getSourcePgYaml(cfg.Project, sourceName, cfg.PrimaryCloudName)
	sourceSession := NewSession(ctx, k8sClient)
	defer sourceSession.Destroy(t)

	require.NoError(t, sourceSession.Apply(sourceYml))

	sourcePg := new(v1alpha1.PostgreSQL)
	require.NoError(t, sourceSession.GetRunning(sourcePg, sourceName))

	sourceAvn, err := avnGen.ServiceGet(ctx, cfg.Project, sourceName, service.ServiceGetIncludeSecrets(true))
	require.NoError(t, err)

	// 2. Create Secret with real source connection details
	secretName := sourceName + "-migration-creds"
	migrationSecret := createMigrationSecretFromService(t, secretName, sourceAvn)
	require.NoError(t, k8sClient.Create(ctx, migrationSecret))
	defer func() {
		_ = k8sClient.Delete(ctx, migrationSecret)
	}()

	// WHEN
	// 3. Create target with BOTH migrationSecretSource AND inline migration config
	//    The inline config has bogus values that would fail if used
	targetName := randName("pg-prec-tgt")
	yml := fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  migrationSecretSource:
    name: %[3]s

  userConfig:
    migration:
      host: bogus-host-that-does-not-exist.example.com
      port: 9999
      password: wrong-password
`, cfg.Project, targetName, secretName, cfg.PrimaryCloudName)

	targetSession := NewSession(ctx, k8sClient)
	defer targetSession.Destroy(t)

	require.NoError(t, targetSession.Apply(yml))

	targetPg := new(v1alpha1.PostgreSQL)
	require.NoError(t, targetSession.GetRunning(targetPg, targetName))

	// THEN
	// Secret should take precedence: the real host from the Secret, not the bogus inline one
	targetAvn, err := avnGen.ServiceGet(ctx, cfg.Project, targetName)
	require.NoError(t, err)

	migrationConfig, ok := targetAvn.UserConfig["migration"].(map[string]any)
	require.True(t, ok, "migration config should be present")

	actualHost := fmt.Sprintf("%v", migrationConfig["host"])
	assert.NotEqual(t, "bogus-host-that-does-not-exist.example.com", actualHost,
		"inline migration host should be overridden by Secret")
	assert.True(t, strings.Contains(actualHost, sourceAvn.ServiceUriParams["host"]),
		"migration host should come from Secret, got: %s", actualHost)
}

func getSourcePgYaml(project, name, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[3]s
  plan: startup-4
`, project, name, cloudName)
}

func getTargetPgWithMigrationSecretYaml(project, name, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  migrationSecretSource:
    name: %[3]s
`, project, name, secretName, cloudName)
}

func getTargetPgWithMigrationSecretDeleteAfterYaml(project, name, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[4]s
  plan: startup-4

  migrationSecretSource:
    name: %[3]s
    deleteAfterMigration: true
`, project, name, secretName, cloudName)
}

func TestPgMigrationSecretDeleteAfterMigration(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	ctx, cancel := testCtx()
	defer cancel()

	// GIVEN
	// 1. Create source PG service
	sourceName := randName("pg-del-src")
	sourceYml := getSourcePgYaml(cfg.Project, sourceName, cfg.PrimaryCloudName)
	sourceSession := NewSession(ctx, k8sClient)
	defer sourceSession.Destroy(t)

	require.NoError(t, sourceSession.Apply(sourceYml))

	sourcePg := new(v1alpha1.PostgreSQL)
	require.NoError(t, sourceSession.GetRunning(sourcePg, sourceName))

	sourceAvn, err := avnGen.ServiceGet(ctx, cfg.Project, sourceName, service.ServiceGetIncludeSecrets(true))
	require.NoError(t, err)

	// 2. Create migration credentials Secret
	secretName := sourceName + "-migration-creds"
	migrationSecret := createMigrationSecretFromService(t, secretName, sourceAvn)
	require.NoError(t, k8sClient.Create(ctx, migrationSecret))
	// Best-effort cleanup; the operator is expected to delete it during the test.
	defer func() {
		_ = k8sClient.Delete(ctx, migrationSecret)
	}()

	// WHEN
	// 3. Create target PG with deleteAfterMigration: true
	targetName := randName("pg-del-tgt")
	targetYml := getTargetPgWithMigrationSecretDeleteAfterYaml(cfg.Project, targetName, secretName, cfg.PrimaryCloudName)
	targetSession := NewSession(ctx, k8sClient)
	defer targetSession.Destroy(t)

	require.NoError(t, targetSession.Apply(targetYml))

	targetPg := new(v1alpha1.PostgreSQL)
	require.NoError(t, targetSession.GetRunning(targetPg, targetName))

	// THEN
	// 4. Wait for migration to complete
	require.NoError(t, retryForever(ctx, "wait for migration to complete", func() (bool, error) {
		pg := new(v1alpha1.PostgreSQL)
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: defaultNamespace}, pg); err != nil {
			return false, err
		}

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		if cond == nil {
			return true, nil
		}
		if cond.Status == metav1.ConditionTrue {
			return false, nil
		}
		if cond.Reason == v1alpha1.MigrationReasonFailed {
			return false, fmt.Errorf("migration failed: %s", cond.Message)
		}
		return true, nil
	}))

	// 5. Wait for the operator to delete the Secret after completion
	require.NoError(t, retryForever(ctx, "wait for migration secret deletion", func() (bool, error) {
		err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: defaultNamespace}, &corev1.Secret{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}))

	// 6. Confirm Secret is gone
	err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: defaultNamespace}, &corev1.Secret{})
	assert.True(t, apierrors.IsNotFound(err), "migration secret should be deleted, got err: %v", err)

	// 7. Trigger a reconcile after deletion and verify the target stays healthy.
	//    The skip-read guard in createOrUpdate relies on MigrationComplete=True to
	//    avoid re-resolving the (now missing) Secret.
	require.NoError(t, k8sClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: defaultNamespace}, targetPg))
	if targetPg.Annotations == nil {
		targetPg.Annotations = map[string]string{}
	}
	targetPg.Annotations["aiven.io/trigger-reconcile"] = randName("touch")
	require.NoError(t, k8sClient.Update(ctx, targetPg))

	require.NoError(t, retryForever(ctx, "target stays running after secret deletion", func() (bool, error) {
		pg := new(v1alpha1.PostgreSQL)
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: targetName, Namespace: defaultNamespace}, pg); err != nil {
			return false, err
		}
		if pg.Status.State == serviceRunningState {
			return false, nil
		}
		return true, nil
	}))
}

// createMigrationSecretFromService creates a Kubernetes Secret with migration credentials
// extracted from a running Aiven service.
func createMigrationSecretFromService(t *testing.T, secretName string, sourceAvn *service.ServiceGetOut) *corev1.Secret {
	t.Helper()
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: defaultNamespace,
		},
		StringData: map[string]string{
			"host":     sourceAvn.ServiceUriParams["host"],
			"port":     sourceAvn.ServiceUriParams["port"],
			"password": sourceAvn.ServiceUriParams["password"],
			"dbname":   sourceAvn.ServiceUriParams["dbname"],
			"username": sourceAvn.ServiceUriParams["user"],
			"ssl":      "true",
			"method":   "dump",
		},
	}
}
