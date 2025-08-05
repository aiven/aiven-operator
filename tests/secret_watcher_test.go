package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// TestServiceUserSecretWatch tests ServiceUser password changes via connInfoSecretSource
func TestServiceUserSecretWatch(t *testing.T) {
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("secret-watch-pg")
	userName := randName("secret-watch-user")
	secretName := randName("password-secret")
	yml := getServiceUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, serviceName))

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Username)

	initialPassword := userAvn.Password

	initialAnnotations := user.GetAnnotations()
	var initialAnnotationValue string
	if initialAnnotations != nil {
		initialAnnotationValue = initialAnnotations["controllers.aiven.io/secret-source-updated"]
	}

	secret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: user.Namespace,
	}, secret)
	require.NoError(t, err)

	// update password to new value
	secret.Data["password"] = []byte("updated-password-67890")
	err = k8sClient.Update(ctx, secret)
	require.NoError(t, err)

	// wait for the secret watcher to trigger reconciliation
	require.Eventually(t, func() bool {
		updatedUser := &v1alpha1.ServiceUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, updatedUser)
		if err != nil {
			return false
		}

		annotations := updatedUser.GetAnnotations()
		if annotations == nil {
			return false
		}

		newAnnotationValue := annotations["controllers.aiven.io/secret-source-updated"]
		return newAnnotationValue != "" && newAnnotationValue != initialAnnotationValue
	}, 1*time.Minute, 5*time.Second, "secret watcher should trigger reconciliation annotation update")

	// wait for the ServiceUser controller to process the password change
	require.Eventually(t, func() bool {
		updatedUser := &v1alpha1.ServiceUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, updatedUser)
		if err != nil {
			return false
		}

		annotations := updatedUser.GetAnnotations()
		return annotations != nil && annotations["controllers.aiven.io/secret-source-updated"] != ""
	}, 1*time.Minute, 10*time.Second, "ServiceUser should be reconciled with new password")

	// verify the password actually changed in Aiven
	finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, finalUserAvn.Username)

	assert.Equal(t, "updated-password-67890", finalUserAvn.Password, "Password should be updated to the new value from the secret")
	assert.NotEqual(t, initialPassword, finalUserAvn.Password, "Password should have changed from initial value")
}

func getServiceUserWithSecretSourceYaml(project, serviceName, userName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: aW5pdGlhbFBhc3N3b3JkMTIzNDU= # initialPassword12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[5]s
  plan: startup-16

  userConfig:
    pg_version: "15"
---
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-service-user-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName)
}

// TestClickhouseUserSecretWatch tests ClickhouseUser password changes via connInfoSecretSource
func TestClickhouseUserSecretWatch(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("secret-watch-ch")
	userName := randName("secret-watch-ch-user")
	secretName := randName("ch-password-secret")
	yml := getClickhouseUserWithSecretSourceYaml(cfg.Project, serviceName, userName, secretName, cfg.PrimaryCloudName)
	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	ch := new(v1alpha1.Clickhouse)
	require.NoError(t, s.GetRunning(ch, serviceName))

	user := new(v1alpha1.ClickhouseUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	userAvn, err := getClickHouseUserByID(ctx, avnGen, cfg.Project, serviceName, user.Status.UUID)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Name)

	secret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      user.Spec.ConnInfoSecretTarget.Name,
		Namespace: user.Namespace,
	}, secret)
	require.NoError(t, err)
	assert.Equal(t, "clickhouse-password-12345", string(secret.Data["PASSWORD"]), "Initial password should match the secret")

	require.NoError(t, pingClickhouse(
		ctx,
		secret.Data["HOST"],
		secret.Data["PORT"],
		secret.Data["USERNAME"],
		secret.Data["PASSWORD"],
	), "initial password should allow ClickHouse connection")

	passwordSecret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: user.Namespace,
	}, passwordSecret)
	require.NoError(t, err)

	passwordSecret.Data["PASSWORD"] = []byte("updated-clickhouse-password-67890")
	err = k8sClient.Update(ctx, passwordSecret)
	require.NoError(t, err)

	initialAnnotations := user.GetAnnotations()
	var initialAnnotationValue string
	if initialAnnotations != nil {
		initialAnnotationValue = initialAnnotations["controllers.aiven.io/secret-source-updated"]
	}

	// wait for the secret watcher to trigger reconciliation
	require.Eventually(t, func() bool {
		updatedUser := &v1alpha1.ClickhouseUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, updatedUser)
		if err != nil {
			return false
		}

		annotations := updatedUser.GetAnnotations()
		if annotations == nil {
			return false
		}

		newAnnotationValue := annotations["controllers.aiven.io/secret-source-updated"]
		return newAnnotationValue != "" && newAnnotationValue != initialAnnotationValue
	}, 1*time.Minute, 5*time.Second, "secret watcher should trigger reconciliation annotation update")

	// wait for the ClickhouseUser controller to process the password change
	require.Eventually(t, func() bool {
		updatedUser := &v1alpha1.ClickhouseUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, updatedUser)
		if err != nil {
			return false
		}

		annotations := updatedUser.GetAnnotations()
		return annotations != nil && annotations["controllers.aiven.io/secret-source-updated"] != ""
	}, 1*time.Minute, 10*time.Second, "ClickhouseUser should be reconciled with new password")

	// verify the password was updated in the target secret
	require.Eventually(t, func() bool {
		finalSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      user.Spec.ConnInfoSecretTarget.Name,
			Namespace: user.Namespace,
		}, finalSecret)
		if err != nil {
			return false
		}

		actualPassword := string(finalSecret.Data["PASSWORD"])
		return actualPassword == "updated-clickhouse-password-67890"
	}, 1*time.Minute, 5*time.Second, "password should be updated to the new value from the secret")

	// get the final secret
	finalSecret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      user.Spec.ConnInfoSecretTarget.Name,
		Namespace: user.Namespace,
	}, finalSecret)
	require.NoError(t, err)

	assert.NoError(t, pingClickhouse(
		ctx,
		finalSecret.Data["HOST"],
		finalSecret.Data["PORT"],
		finalSecret.Data["USERNAME"],
		finalSecret.Data["PASSWORD"],
	), "updated password should allow ClickHouse connection")

	// test that the OLD password no longer works
	assert.Error(t, pingClickhouse(
		ctx,
		finalSecret.Data["HOST"],
		finalSecret.Data["PORT"],
		finalSecret.Data["USERNAME"],
		[]byte("clickhouse-password-12345"),
	), "old password should no longer work after update")
}

func getClickhouseUserWithSecretSourceYaml(project, serviceName, userName, secretName, cloudName string) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  PASSWORD: Y2xpY2tob3VzZS1wYXNzd29yZC0xMjM0NQ== # clickhouse-password-12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[5]s
  plan: startup-16

  userConfig:
    public_access:
      clickhouse: true
---
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-clickhouse-user-secret

  connInfoSecretSource:
    name: %[4]s
    passwordKey: PASSWORD

  project: %[1]s
  serviceName: %[2]s
`, project, serviceName, userName, secretName, cloudName)
}

// TestCrossNamespaceSecretWatch tests secret watching across namespaces
func TestCrossNamespaceSecretWatch(t *testing.T) {
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("cross-ns-pg")
	userName := randName("cross-ns-user")
	secretName := randName("cross-ns-secret")
	secretNamespace := "test-secrets"

	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: secretNamespace},
	}
	err := k8sClient.Create(ctx, testNs)
	require.NoError(t, err)

	defer func() {
		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer deleteCancel()
		_ = k8sClient.Delete(deleteCtx, testNs)
	}()

	yml := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
  namespace: %[5]s
data:
  password: Y3Jvc3MtbnMtcGFzc3dvcmQtMTIzNDU= # cross-ns-password-12345 # gitleaks:allow
---
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: %[2]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: %[1]s
  cloudName: %[6]s
  plan: startup-16

  userConfig:
    pg_version: "15"
---
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: %[3]s
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: cross-ns-service-user-secret

  connInfoSecretSource:
    name: %[4]s
    namespace: %[5]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
`, cfg.Project, serviceName, userName, secretName, secretNamespace, cfg.PrimaryCloudName)

	s := NewSession(ctx, k8sClient, cfg.Project)

	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	// wait for PostgreSQL service to be running
	pg := new(v1alpha1.PostgreSQL)
	require.NoError(t, s.GetRunning(pg, serviceName))

	// wait for ServiceUser to be running
	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Username)

	initialPassword := userAvn.Password

	secret := &corev1.Secret{}
	err = k8sClient.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: secretNamespace,
	}, secret)
	require.NoError(t, err)

	secret.Data["password"] = []byte("updated-cross-ns-password-67890")
	err = k8sClient.Update(ctx, secret)
	require.NoError(t, err)

	// wait for the secret watcher to trigger reconciliation
	require.Eventually(t, func() bool {
		updatedUser := &v1alpha1.ServiceUser{}
		err = k8sClient.Get(ctx, types.NamespacedName{
			Name:      userName,
			Namespace: user.Namespace,
		}, updatedUser)
		if err != nil {
			return false
		}

		annotations := updatedUser.GetAnnotations()
		if annotations == nil {
			return false
		}

		return annotations["controllers.aiven.io/secret-source-updated"] != ""
	}, 1*time.Minute, 5*time.Second, "Cross-namespace secret watcher should trigger reconciliation")

	finalUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, finalUserAvn.Username)

	assert.Equal(t, "updated-cross-ns-password-67890", finalUserAvn.Password, "Cross-namespace password should be updated to the new value from the secret")
	assert.NotEqual(t, initialPassword, finalUserAvn.Password, "Cross-namespace password should have changed from initial value")
}
