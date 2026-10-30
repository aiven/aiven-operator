//go:build valkey

package tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceUserValkeyAccessControl(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	serviceName := randName("valkey-su")
	userName := randName("valkey-user")
	sourceSecretName := randName("valkey-su-source")
	targetSecretName := randName("valkey-su-target")

	const (
		initialPassword = "InitialValkeyPassword123!"
		updatedPassword = "UpdatedValkeyPassword67890!"
	)

	initialAccessControl := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"cache:*"},
		ValkeyACLCommands:   []string{"-acl"},
		ValkeyACLCategories: []string{"+@all"},
		ValkeyACLChannels:   []string{"initial*"},
	}

	updatedAccessControl := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"cache:updated:*"},
		ValkeyACLCommands:   []string{"-slowlog"},
		ValkeyACLCategories: []string{"+@read"},
		ValkeyACLChannels:   []string{"updated*"},
	}

	yml, err := getServiceUserValkeyCreateYAML(
		cfg.Project,
		cfg.PrimaryCloudName,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		initialPassword,
		initialAccessControl,
	)
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient)

	// Cleans test afterward
	defer s.Destroy(t)

	// WHEN
	require.NoError(t, s.Apply(yml))

	valkey := new(v1alpha1.Valkey)
	require.NoError(t, s.GetRunning(valkey, serviceName))

	t.Run("CreateAndGetUserType", func(t *testing.T) {
		apiUserName := randName("valkey-type")
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
			defer cancel()
			err := avnGen.ServiceUserDelete(cleanupCtx, cfg.Project, serviceName, apiUserName)
			if !avngen.IsNotFound(err) {
				assert.NoError(t, err, "cleaning up the user created directly through the API must succeed")
			}
		})

		created, err := avnGen.ServiceUserCreate(ctx, cfg.Project, serviceName, &service.ServiceUserCreateIn{
			Username: apiUserName,
			AccessControl: &service.AccessControlIn{
				ValkeyAclKeys:       &initialAccessControl.ValkeyACLKeys,
				ValkeyAclCommands:   &initialAccessControl.ValkeyACLCommands,
				ValkeyAclCategories: &initialAccessControl.ValkeyACLCategories,
				ValkeyAclChannels:   &initialAccessControl.ValkeyACLChannels,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, apiUserName, created.Username)
		// Characterize the current API inconsistency: POST reports normal, while GET reports regular.
		assert.Equal(t, "normal", created.Type, "Valkey POST currently reports the generic account type")

		var observed *service.ServiceUserGetOut
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			var err error
			observed, err = avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, apiUserName)
			require.NoError(c, err)
		}, 3*time.Minute, 10*time.Second, "the created Valkey user must become readable through GET")
		assert.Equal(t, apiUserName, observed.Username)
		assert.Equal(t, "regular", observed.Type, "Valkey GET must report the service-specific account type")
		t.Logf("Valkey user type: POST=%q, GET=%q", created.Type, observed.Type)
	})

	user := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(user, userName))

	// THEN
	require.Eventually(t, func() bool {
		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err != nil {
			return false
		}

		return userAvn.Password == initialPassword &&
			serviceUserValkeyAccessControlEquals(userAvn.AccessControl, initialAccessControl)
	}, 3*time.Minute, 10*time.Second, "ServiceUser should be created with managed Valkey ACL and custom password")

	userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, userName, userAvn.Username)
	assert.Equal(t, userAvn.Type, user.Status.Type, "status.type must match GET after creation")
	assert.Equal(t, initialPassword, userAvn.Password)
	assert.True(t, serviceUserValkeyAccessControlEquals(userAvn.AccessControl, initialAccessControl))

	secret, err := s.GetSecret(targetSecretName)
	require.NoError(t, err)
	assert.NotEmpty(t, secret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_USERNAME"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_PASSWORD"])
	assert.NotEmpty(t, secret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, secret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, userName, string(secret.Data["SERVICEUSER_USERNAME"]))
	assert.Equal(t, initialPassword, string(secret.Data["SERVICEUSER_PASSWORD"]))

	updatedYml := getServiceUserValkeyUpdateYAML(
		cfg.Project,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		updatedPassword,
		updatedAccessControl,
	)

	require.NoError(t, s.Apply(updatedYml))

	updatedUser := new(v1alpha1.ServiceUser)
	require.NoError(t, s.GetRunning(updatedUser, userName))

	require.Eventually(t, func() bool {
		userAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err != nil {
			return false
		}

		secret, err := s.GetSecret(targetSecretName)
		if err != nil {
			return false
		}

		return userAvn.Password == updatedPassword &&
			serviceUserValkeyAccessControlEquals(userAvn.AccessControl, updatedAccessControl) &&
			string(secret.Data["SERVICEUSER_PASSWORD"]) == updatedPassword
	}, 3*time.Minute, 10*time.Second, "ServiceUser should reconcile Valkey ACL and password changes together")

	updatedUserAvn, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
	require.NoError(t, err)
	assert.Equal(t, updatedUserAvn.Type, updatedUser.Status.Type, "status.type must match GET after updating ACL and password")
	assert.Equal(t, updatedPassword, updatedUserAvn.Password)
	assert.True(t, serviceUserValkeyAccessControlEquals(updatedUserAvn.AccessControl, updatedAccessControl))

	updatedSecret, err := s.GetSecret(targetSecretName)
	require.NoError(t, err)
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_HOST"])
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_PORT"])
	assert.NotEmpty(t, updatedSecret.Data["SERVICEUSER_CA_CERT"])
	assert.Contains(t, updatedSecret.Data, "SERVICEUSER_ACCESS_CERT")
	assert.Contains(t, updatedSecret.Data, "SERVICEUSER_ACCESS_KEY")
	assert.Equal(t, userName, string(updatedSecret.Data["SERVICEUSER_USERNAME"]))
	assert.Equal(t, updatedPassword, string(updatedSecret.Data["SERVICEUSER_PASSWORD"]))

	assert.NoError(t, s.Delete(updatedUser, func() error {
		_, err = avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		return err
	}))
}

func TestServiceUserValkeyAdoptionAppliesACLBeforeSecret(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)
	ctx, cancel := testCtx()
	defer cancel()
	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	serviceName := randName("valkey-adoption")
	userName := randName("valkey-adopt-user")
	targetSecretName := randName("valkey-adopt-secret")
	yml, err := loadExampleYaml("valkey.yaml", map[string]string{
		"metadata.name":                  serviceName,
		"spec.project":                   cfg.Project,
		"spec.cloudName":                 cfg.PrimaryCloudName,
		"spec.connInfoSecretTarget.name": serviceName,
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(yml))
	require.NoError(t, s.GetRunning(new(v1alpha1.Valkey), serviceName))

	broadAccess := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"*"},
		ValkeyACLCommands:   []string{"-acl"},
		ValkeyACLCategories: []string{"+@all"},
		ValkeyACLChannels:   []string{"*"},
	}
	wantedAccess := &v1alpha1.ServiceUserAccessControl{
		ValkeyACLKeys:       []string{"cache:*"},
		ValkeyACLCommands:   []string{"-acl"},
		ValkeyACLCategories: []string{"+@read"},
		ValkeyACLChannels:   []string{"read:*"},
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()
		err := avnGen.ServiceUserDelete(cleanupCtx, cfg.Project, serviceName, userName)
		if !avngen.IsNotFound(err) {
			assert.NoError(t, err, "cleaning up the adopted Aiven user must succeed")
		}
	})
	_, err = avnGen.ServiceUserCreate(ctx, cfg.Project, serviceName, &service.ServiceUserCreateIn{
		Username: userName,
		AccessControl: &service.AccessControlIn{
			ValkeyAclKeys:       &broadAccess.ValkeyACLKeys,
			ValkeyAclCommands:   &broadAccess.ValkeyACLCommands,
			ValkeyAclCategories: &broadAccess.ValkeyACLCategories,
			ValkeyAclChannels:   &broadAccess.ValkeyACLChannels,
		},
	})
	require.NoError(t, err)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		u, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		require.NoError(c, err)
		assert.NotEmpty(c, u.Password)
		assert.True(c, serviceUserValkeyAccessControlEquals(u.AccessControl, broadAccess))
	}, 3*time.Minute, 5*time.Second, "the existing user must have a readable password and broad ACL")

	usersPath := fmt.Sprintf("/v1/project/%s/service/%s/user", url.PathEscape(cfg.Project), url.PathEscape(serviceName))
	getKey := http.MethodGet + " " + usersPath + "/" + url.PathEscape(userName)
	createKey := http.MethodPost + " " + usersPath
	var injectedNotFound, observedConflict atomic.Bool
	aivenAPIHooks.Store(getKey, aivenAPIHook(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		if injectedNotFound.CompareAndSwap(false, true) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"User not found","errors":[]}`))
			return
		}
		next.ServeHTTP(w, r)
	}))
	aivenAPIHooks.Store(createKey, aivenAPIHook(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		// Forward POST unchanged and record the real API's response.
		response := httptest.NewRecorder()
		next.ServeHTTP(response, r)
		if response.Code == http.StatusConflict {
			observedConflict.Store(true)
		}
		for name, values := range response.Header() {
			w.Header()[name] = values
		}
		w.WriteHeader(response.Code)
		_, _ = w.Write(response.Body.Bytes())
	}))
	t.Cleanup(func() {
		aivenAPIHooks.Delete(getKey)
		aivenAPIHooks.Delete(createKey)
	})

	// An empty target lets the existing Secret update hook capture the first credential publication.
	target := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: targetSecretName, Namespace: defaultNamespace},
	}
	require.NoError(t, s.ApplyObjects(target))
	secretKey := client.ObjectKeyFromObject(target)
	type publicationSnapshot struct {
		accessControl *service.AccessControlOut
		err           error
	}
	publications := make(chan publicationSnapshot, 1)
	var captured atomic.Bool
	secretUpdateHooks.Store(secretKey, secretUpdateHook(func(ctx context.Context, secret *corev1.Secret, update func() error) error {
		if captured.Load() || string(secret.Data["SERVICEUSER_USERNAME"]) != userName {
			return update()
		}
		u, readErr := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		if err := update(); err != nil {
			return err
		}
		if captured.CompareAndSwap(false, true) {
			snapshot := publicationSnapshot{err: readErr}
			if u != nil {
				snapshot.accessControl = u.AccessControl
			}
			publications <- snapshot
		}
		return nil
	}))
	t.Cleanup(func() { secretUpdateHooks.Delete(secretKey) })

	user := &v1alpha1.ServiceUser{
		TypeMeta:   metav1.TypeMeta{APIVersion: "aiven.io/v1alpha1", Kind: "ServiceUser"},
		ObjectMeta: metav1.ObjectMeta{Name: userName, Namespace: defaultNamespace},
	}
	user.Spec.Project = cfg.Project
	user.Spec.ServiceName = serviceName
	user.Spec.AuthSecretRef = &v1alpha1.AuthSecretReference{Name: secretRefName, Key: secretRefKey}
	user.Spec.ConnInfoSecretTarget.Name = targetSecretName
	user.Spec.AccessControl = wantedAccess
	require.NoError(t, s.ApplyObjects(user))

	var first publicationSnapshot
	select {
	case first = <-publications:
	case <-time.After(3 * time.Minute):
		t.Fatal("the adopted user must publish connection credentials")
	}
	require.NoError(t, first.err, "reading the real ACL at first credential publication must succeed")
	require.NotNil(t, first.accessControl)
	require.NoError(t, s.GetRunning(user, userName))
	assert.True(t, injectedNotFound.Load(), "Observe must receive the injected 404")
	assert.True(t, observedConflict.Load(), "Create must receive a real 409 from Aiven")
	t.Logf("At first credential publication: injected GET 404=%t, real POST 409=%t, ACL keys=%v, categories=%v",
		injectedNotFound.Load(), observedConflict.Load(), first.accessControl.ValkeyAclKeys, first.accessControl.ValkeyAclCategories)
	assert.True(t, serviceUserValkeyAccessControlEquals(first.accessControl, wantedAccess),
		"the desired ACL must be applied before publishing credentials; requested: %+v; observed at first publication: %+v", wantedAccess, first.accessControl)

	published, err := s.GetSecret(targetSecretName)
	require.NoError(t, err)
	assert.Equal(t, userName, string(published.Data["SERVICEUSER_USERNAME"]))
	assert.NotEmpty(t, published.Data["SERVICEUSER_PASSWORD"])
	assert.NoError(t, s.Delete(user, func() error {
		_, err := avnGen.ServiceUserGet(ctx, cfg.Project, serviceName, userName)
		return err
	}))
}

func getServiceUserValkeyCreateYAML(
	project, cloudName, serviceName, userName, sourceSecretName, targetSecretName, password string,
	accessControl *v1alpha1.ServiceUserAccessControl,
) (string, error) {
	valkeyYml, err := loadExampleYaml("valkey.yaml", map[string]string{
		"metadata.name":                  serviceName,
		"spec.project":                   project,
		"spec.cloudName":                 cloudName,
		"spec.connInfoSecretTarget.name": serviceName,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n---\n%s", valkeyYml, getServiceUserValkeyUpdateYAML(
		project,
		serviceName,
		userName,
		sourceSecretName,
		targetSecretName,
		password,
		accessControl,
	)), nil
}

func getServiceUserValkeyUpdateYAML(
	project, serviceName, userName, sourceSecretName, targetSecretName, password string,
	accessControl *v1alpha1.ServiceUserAccessControl,
) string {
	return fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %[4]s
data:
  password: %[6]s
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
    name: %[5]s

  connInfoSecretSource:
    name: %[4]s
    passwordKey: password

  project: %[1]s
  serviceName: %[2]s
%[7]s
`, project, serviceName, userName, sourceSecretName, targetSecretName, base64.StdEncoding.EncodeToString([]byte(password)), serviceUserValkeyAccessControlBlock(accessControl))
}

func serviceUserValkeyAccessControlBlock(accessControl *v1alpha1.ServiceUserAccessControl) string {
	if accessControl == nil {
		return ""
	}

	return fmt.Sprintf(`  accessControl:
    valkeyAclKeys:%s
    valkeyAclCommands:%s
    valkeyAclCategories:%s
    valkeyAclChannels:%s
`, yamlStringList(accessControl.ValkeyACLKeys), yamlStringList(accessControl.ValkeyACLCommands), yamlStringList(accessControl.ValkeyACLCategories), yamlStringList(accessControl.ValkeyACLChannels))
}

func yamlStringList(items []string) string {
	if len(items) == 0 {
		return " []"
	}

	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n      - %q", item)
	}

	return b.String()
}

func serviceUserValkeyAccessControlEquals(
	actual *service.AccessControlOut,
	expected *v1alpha1.ServiceUserAccessControl,
) bool {
	if expected == nil {
		return actual == nil
	}

	if actual == nil {
		return false
	}

	return slices.Equal(actual.ValkeyAclKeys, expected.ValkeyACLKeys) &&
		slices.Equal(actual.ValkeyAclCommands, expected.ValkeyACLCommands) &&
		slices.Equal(actual.ValkeyAclCategories, expected.ValkeyACLCategories) &&
		slices.Equal(actual.ValkeyAclChannels, expected.ValkeyACLChannels)
}
