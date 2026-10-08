package controllers

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const yamlServiceUser = `
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: test-user
  namespace: default
spec:
  project: test-project
  serviceName: test-service
`

// primaryComponent is the dynamic primary component HOST and PORT come from when no route is set.
func primaryComponent(name, host string, port int) service.ComponentOut {
	return service.ComponentOut{Component: name, Host: host, Port: port, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary}
}

func Test_newServiceUserReconciler(t *testing.T) {
	t.Parallel()

	r := newServiceUserReconciler(Controller{}).(*Reconciler[*v1alpha1.ServiceUser])
	require.NotNil(t, r.options)
	require.Equal(t, serviceUserMaxConcurrentReconciles, r.options.MaxConcurrentReconciles)
}

func TestAccessControlMatches(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		desired *v1alpha1.ServiceUserAccessControl
		actual  *service.AccessControlOut
	}

	t.Run("Matches", func(t *testing.T) {
		testCases := []testCase{
			{
				name:    "unmanaged ACL always matches",
				desired: nil,
				actual: &service.AccessControlOut{
					ValkeyAclKeys: []string{"cache:*"},
				},
			},
			{
				name:    "managed empty ACL matches missing remote ACL",
				desired: &v1alpha1.ServiceUserAccessControl{},
				actual:  nil,
			},
			{
				name: "exact match succeeds",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLKeys:       []string{"cache:*"},
					ValkeyACLCommands:   []string{"-acl", "+get"},
					ValkeyACLCategories: []string{"+@read", "-@dangerous"},
					ValkeyACLChannels:   []string{"events*"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclKeys:       []string{"cache:*"},
					ValkeyAclCommands:   []string{"-acl", "+get"},
					ValkeyAclCategories: []string{"+@read", "-@dangerous"},
					ValkeyAclChannels:   []string{"events*"},
				},
			},
			{
				name: "keys ignore order",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLKeys:       []string{"cache:*", "session:*"},
					ValkeyACLCommands:   []string{"-acl", "+get"},
					ValkeyACLCategories: []string{"+@read"},
					ValkeyACLChannels:   []string{"events*"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclKeys:       []string{"session:*", "cache:*"},
					ValkeyAclCommands:   []string{"-acl", "+get"},
					ValkeyAclCategories: []string{"+@read"},
					ValkeyAclChannels:   []string{"events*"},
				},
			},
			{
				name: "channels ignore order",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLChannels: []string{"events*", "updates*"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclChannels: []string{"updates*", "events*"},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				require.True(t, accessControlMatches(tc.desired, tc.actual))
			})
		}
	})

	t.Run("Doesn't match", func(t *testing.T) {
		testCases := []testCase{
			{
				name: "managed non-empty ACL does not match missing remote ACL",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLKeys: []string{"cache:*"},
				},
				actual: nil,
			},
			{
				name: "keys compare duplicate counts",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLKeys: []string{"cache:*", "cache:*"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclKeys: []string{"cache:*"},
				},
			},
			{
				name: "commands keep order significant",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLCommands: []string{"-acl", "+get"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclCommands: []string{"+get", "-acl"},
				},
			},
			{
				name: "command differences are detected",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLCommands: []string{"-acl"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclCommands: []string{"-slowlog"},
				},
			},
			{
				name: "categories keep order significant",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLCategories: []string{"+@read", "-@dangerous"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclCategories: []string{"-@dangerous", "+@read"},
				},
			},
			{
				name: "category differences are detected",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLCategories: []string{"+@read"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclCategories: []string{"+@write"},
				},
			},
			{
				name: "channels compare duplicate counts",
				desired: &v1alpha1.ServiceUserAccessControl{
					ValkeyACLChannels: []string{"events*", "events*"},
				},
				actual: &service.AccessControlOut{
					ValkeyAclChannels: []string{"events*"},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				require.False(t, accessControlMatches(tc.desired, tc.actual))
			})
		}
	})
}

func TestServiceUserReconciler(t *testing.T) {
	t.Parallel()

	runScenarioErr := func(t *testing.T, user *v1alpha1.ServiceUser, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.ServiceUser], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{user}, additionalObjects...)

		r := newServiceUserReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ServiceUser{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ServiceUser])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}
		r.jitter = nil // deterministic RequeueAfter

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, user *v1alpha1.ServiceUser, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.ServiceUser], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, user, avn, additionalObjects...)
		require.NoError(t, err)
		return r, res
	}

	equalManagedSlice := func(actual *[]string, expected []string) bool {
		if actual == nil || *actual == nil {
			return false
		}

		return slices.Equal(*actual, expected)
	}

	matchValkeyAccessControl := func(expected *v1alpha1.ServiceUserAccessControl) func(*service.AccessControlIn) bool {
		return func(in *service.AccessControlIn) bool {
			if expected == nil {
				return in == nil
			}

			return equalManagedSlice(in.ValkeyAclKeys, expected.ValkeyACLKeys) &&
				equalManagedSlice(in.ValkeyAclCommands, expected.ValkeyACLCommands) &&
				equalManagedSlice(in.ValkeyAclCategories, expected.ValkeyACLCategories) &&
				equalManagedSlice(in.ValkeyAclChannels, expected.ValkeyACLChannels)
		}
	}

	valkeyAccessControlOut := func(in *v1alpha1.ServiceUserAccessControl) *service.AccessControlOut {
		if in == nil {
			return nil
		}

		return &service.AccessControlOut{
			ValkeyAclKeys:       slices.Clone(in.ValkeyACLKeys),
			ValkeyAclCommands:   slices.Clone(in.ValkeyACLCommands),
			ValkeyAclCategories: slices.Clone(in.ValkeyACLCategories),
			ValkeyAclChannels:   slices.Clone(in.ValkeyACLChannels),
		}
	}

	t.Run("Requeues when service preconditions aren't met", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Creates ServiceUser on Aiven when it doesn't exist", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceUserCreateIn) bool {
				return in.Username == user.Name
			})).
			Return(&service.ServiceUserCreateOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Creates a MySQL user with the requested authentication and source password", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}
		password := "source-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(password)},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "mysql",
				Components:  []service.ComponentOut{primaryComponent("mysql", "host", 3306)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, &service.ServiceUserCreateIn{
				Username:       user.Name,
				Authentication: user.Spec.Authentication,
			}).
			Return(&service.ServiceUserCreateOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, &service.ServiceUserCredentialsModifyIn{
				NewPassword:    &password,
				Authentication: user.Spec.Authentication,
				Operation:      service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
			}).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: password, Authentication: user.Spec.Authentication}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
		require.Equal(t, []byte(password), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Retries transient not found after create before publishing secrets", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
			user.Generation = 1

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
				Return(&service.ServiceGetOut{
					State:       service.ServiceStateTypeRunning,
					ServiceType: "kafka",
					Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
				}, nil).Twice()
			avn.EXPECT().
				ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
				Return(nil, newAivenError(404, "not found")).Once()
			avn.EXPECT().
				ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceUserCreateIn) bool {
					return in.Username == user.Name
				})).
				Return(&service.ServiceUserCreateOut{}, nil).Once()
			avn.EXPECT().
				ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
				Return(nil, newAivenError(404, "not found")).Once()
			avn.EXPECT().
				ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
				Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

			r, res := runScenario(t, user, avn)
			require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

			secret := &corev1.Secret{}
			require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
			require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
		})
	})

	t.Run("Creates ServiceUser with managed Valkey ACL", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{
			ValkeyACLKeys:       []string{"prefix_*:*"},
			ValkeyACLCommands:   []string{"-acl"},
			ValkeyACLCategories: []string{"+@all"},
			ValkeyACLChannels:   []string{"some*chan"},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceUserCreateIn) bool {
				return in.Username == user.Name && matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Return(&service.ServiceUserCreateOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "pw",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Normalizes empty managed Valkey ACL block to empty arrays on create", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceUserCreateIn) bool {
				return in.Username == user.Name && matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Return(&service.ServiceUserCreateOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "pw",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		_, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
	})

	t.Run("Updates ServiceUser when generation isn't processed yet", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}

		srcPassword := "external-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(srcPassword)},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.NewPassword != nil && *in.NewPassword == srcPassword &&
					in.Operation == service.ServiceUserCredentialsModifyOperationTypeResetCredentials
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: srcPassword}, nil).Twice()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(srcPassword), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Updates ServiceUser access control when generation isn't processed yet", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{
			ValkeyACLKeys:       []string{"prefix_*:*"},
			ValkeyACLCommands:   []string{"-acl"},
			ValkeyACLCategories: []string{"+@all"},
			ValkeyACLChannels:   []string{"some*chan"},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "pw",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.Operation == service.ServiceUserCredentialsModifyOperationTypeSetAccessControl &&
					in.NewPassword == nil &&
					matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Updates ServiceUser access control before resetting password", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{
			ValkeyACLKeys:       []string{"prefix_*:*"},
			ValkeyACLCommands:   []string{"-acl"},
			ValkeyACLCategories: []string{"+@all"},
			ValkeyACLChannels:   []string{"some*chan"},
		}
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}

		srcPassword := "external-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(srcPassword)},
		}
		operations := make([]service.ServiceUserCredentialsModifyOperationType, 0, 2)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "current-password",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.Operation == service.ServiceUserCredentialsModifyOperationTypeSetAccessControl &&
					in.NewPassword == nil &&
					matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Run(func(_ context.Context, _, _, _ string, in *service.ServiceUserCredentialsModifyIn) {
				operations = append(operations, in.Operation)
			}).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.Operation == service.ServiceUserCredentialsModifyOperationTypeResetCredentials &&
					in.NewPassword != nil && *in.NewPassword == srcPassword &&
					in.AccessControl == nil
			})).
			Run(func(_ context.Context, _, _, _ string, in *service.ServiceUserCredentialsModifyIn) {
				operations = append(operations, in.Operation)
			}).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      srcPassword,
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
		require.Equal(t, []service.ServiceUserCredentialsModifyOperationType{
			service.ServiceUserCredentialsModifyOperationTypeSetAccessControl,
			service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		}, operations)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(srcPassword), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Publishes secrets and requeues when ServiceUser is up to date with managed ACL", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{
			ValkeyACLKeys:       []string{"prefix_*:*"},
			ValkeyACLCommands:   []string{"-acl"},
			ValkeyACLCategories: []string{"+@all"},
			ValkeyACLChannels:   []string{"some*chan"},
		}
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "pw",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Publishes Kafka endpoint keys for externally managed Kafka service", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		accessCert := "access-cert"
		accessKey := "access-key"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components: []service.ComponentOut{
					{
						Component:                 "kafka",
						Host:                      "kafka-cert.example.com",
						Port:                      9092,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate,
					},
					{
						Component:                 "kafka",
						Host:                      "kafka-sasl.example.com",
						Port:                      9093,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl,
					},
					{Component: "schema_registry", Host: "schema.example.com", Port: 8081, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:   user.Name,
				Password:   "pw",
				AccessCert: &accessCert,
				AccessKey:  &accessKey,
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("kafka-cert.example.com"), secret.Data["SERVICEUSER_HOST"])
		require.Equal(t, []byte("9092"), secret.Data["SERVICEUSER_PORT"])
		require.Equal(t, []byte(user.Name), secret.Data["SERVICEUSER_USERNAME"])
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
		require.Equal(t, []byte("access-cert"), secret.Data["SERVICEUSER_ACCESS_CERT"])
		require.Equal(t, []byte("access-key"), secret.Data["SERVICEUSER_ACCESS_KEY"])
		require.Equal(t, []byte("ca"), secret.Data["SERVICEUSER_CA_CERT"])
		require.Equal(t, []byte("kafka-sasl.example.com"), secret.Data["SERVICEUSER_SASL_HOST"])
		require.Equal(t, []byte("9093"), secret.Data["SERVICEUSER_SASL_PORT"])
		require.Equal(t, []byte("schema.example.com"), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"])
		require.Equal(t, []byte("8081"), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"])
	})

	t.Run("Publishes only available Kafka endpoint keys", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components: []service.ComponentOut{
					{
						Component:                 "kafka",
						Host:                      "kafka-sasl.example.com",
						Port:                      9093,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl,
					},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("kafka-sasl.example.com"), secret.Data["SERVICEUSER_SASL_HOST"])
		require.Equal(t, []byte("9093"), secret.Data["SERVICEUSER_SASL_PORT"])
		require.NotContains(t, secret.Data, "SERVICEUSER_SCHEMA_REGISTRY_HOST")
		require.NotContains(t, secret.Data, "SERVICEUSER_SCHEMA_REGISTRY_PORT")
	})

	t.Run("Preserves existing Kafka endpoint keys when components are temporarily absent", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: user.Name, Namespace: user.Namespace},
			Data: map[string][]byte{
				"SERVICEUSER_SASL_HOST":            []byte("old-sasl.example.com"),
				"SERVICEUSER_SASL_PORT":            []byte("9093"),
				"SERVICEUSER_SCHEMA_REGISTRY_HOST": []byte("old-schema.example.com"),
				"SERVICEUSER_SCHEMA_REGISTRY_PORT": []byte("8081"),
			},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components: []service.ComponentOut{
					{
						Component:                 "kafka",
						Host:                      "kafka-cert.example.com",
						Port:                      9092,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate,
					},
				},
				UserConfig: map[string]any{
					"kafka_authentication_methods": map[string]any{"sasl": true},
					"schema_registry":              true,
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn, existingSecret)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("old-sasl.example.com"), secret.Data["SERVICEUSER_SASL_HOST"])
		require.Equal(t, []byte("9093"), secret.Data["SERVICEUSER_SASL_PORT"])
		require.Equal(t, []byte("old-schema.example.com"), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"])
		require.Equal(t, []byte("8081"), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"])
	})

	t.Run("Clears stale Kafka endpoint keys when endpoints are explicitly disabled", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: user.Name, Namespace: user.Namespace},
			Data: map[string][]byte{
				"SERVICEUSER_SASL_HOST":            []byte("old-sasl.example.com"),
				"SERVICEUSER_SASL_PORT":            []byte("9093"),
				"SERVICEUSER_SCHEMA_REGISTRY_HOST": []byte("old-schema.example.com"),
				"SERVICEUSER_SCHEMA_REGISTRY_PORT": []byte("8081"),
			},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components: []service.ComponentOut{
					{
						Component:                 "kafka",
						Host:                      "kafka-cert.example.com",
						Port:                      9092,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate,
					},
				},
				UserConfig: map[string]any{
					"kafka_authentication_methods": map[string]any{"sasl": false},
					"schema_registry":              false,
					"schema_registry_config":       "unexpected-shape",
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn, existingSecret)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(""), secret.Data["SERVICEUSER_SASL_HOST"])
		require.Equal(t, []byte(""), secret.Data["SERVICEUSER_SASL_PORT"])
		require.Equal(t, []byte(""), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_HOST"])
		require.Equal(t, []byte(""), secret.Data["SERVICEUSER_SCHEMA_REGISTRY_PORT"])
	})

	t.Run("Logs and skips Kafka endpoint cleanup when user config cannot be decoded", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components: []service.ComponentOut{
					{
						Component:                 "kafka",
						Host:                      "kafka-cert.example.com",
						Port:                      9092,
						Route:                     service.RouteTypeDynamic,
						Usage:                     service.UsageTypePrimary,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate,
					},
				},
				UserConfig: map[string]any{
					"schema_registry": "not-a-bool",
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		sink := &logRecorderSink{}
		ctx := logr.NewContext(t.Context(), logr.New(sink))
		_, details, err := (&ServiceUserController{avnGen: avn}).fetchUser(ctx, user, false)
		require.NoError(t, err)
		require.NotContains(t, details, "SERVICEUSER_SASL_HOST")
		require.NotContains(t, details, "SERVICEUSER_SASL_PORT")
		require.NotContains(t, details, "SERVICEUSER_SCHEMA_REGISTRY_HOST")
		require.NotContains(t, details, "SERVICEUSER_SCHEMA_REGISTRY_PORT")
		require.Contains(t, sink.logs, "ERROR: unable to decode Kafka user config, keeping existing optional Kafka endpoint keys")
	})

	t.Run("Doesn't publish Kafka endpoint keys for non-Kafka service", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "valkey.example.com", 6379)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.NotContains(t, secret.Data, "SERVICEUSER_SASL_HOST")
		require.NotContains(t, secret.Data, "SERVICEUSER_SASL_PORT")
		require.NotContains(t, secret.Data, "SERVICEUSER_SCHEMA_REGISTRY_HOST")
		require.NotContains(t, secret.Data, "SERVICEUSER_SCHEMA_REGISTRY_PORT")
	})

	t.Run("Waits with a precondition when the requested route has no component", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{processedGenerationAnnotation: "1", instanceIsRunningAnnotation: "true"}
		user.Spec.ConnInfoSecretRoute = service.RouteTypePrivatelink

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "pg",
				ServiceName: user.Spec.ServiceName,
				Components:  []service.ComponentOut{primaryComponent("pg", "host", 5432)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		// No ServiceUserCreate expectation: the user exists, only the secret waits for the route.
		_, err := (&ServiceUserController{avnGen: avn}).Observe(t.Context(), user)
		require.ErrorIs(t, err, errPreconditionNotMet)
		require.ErrorContains(t, err, `route "privatelink"`)
		require.ErrorContains(t, err, user.Spec.ServiceName)
	})

	t.Run("Create surfaces the precondition after creating the user", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.ConnInfoSecretRoute = service.RouteTypePrivatelink

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceUserCreateOut{Username: user.Name}, nil).Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "pg",
				ServiceName: user.Spec.ServiceName,
				Components:  []service.ComponentOut{primaryComponent("pg", "host", 5432)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		c := &ServiceUserController{Client: fake.NewClientBuilder().WithScheme(scheme).Build(), avnGen: avn}
		_, err := c.Create(t.Context(), user)
		require.ErrorIs(t, err, errPreconditionNotMet)
		require.ErrorContains(t, err, `route "privatelink"`)
	})

	t.Run("Pushes nothing while the requested route is missing", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		// A rotation is stamped and access control drifted, but the route wait comes first.
		user.Annotations = map[string]string{instanceIsRunningAnnotation: "true", secretSourceUpdatedAnnotation: "123"}
		user.Spec.ConnInfoSecretRoute = service.RouteTypePrivatelink
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{ValkeyACLKeys: []string{"prefix_*:*"}}
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte("external-secret-password")},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				ServiceName: user.Spec.ServiceName,
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		// No ServiceUserCredentialsModify expectation: nothing is pushed while the route is missing.
		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
		require.Contains(t, recorderEvents(r.Recorder.(*record.FakeRecorder)),
			`Warning PreconditionsNotMet preconditions are not met: waiting for an external change: component "valkey" with route "privatelink" not found on service "`+user.Spec.ServiceName+`"`)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.NotContains(t, got.Annotations, processedGenerationAnnotation, "the wait must not mark the generation processed")
		cond := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError)
		require.NotNil(t, cond)
		require.Equal(t, string(errConditionPreconditions), cond.Reason)

		secret := &corev1.Secret{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err), "secret must not be written while the route is missing")
	})

	t.Run("Pushes pending changes once the requested route appears", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{instanceIsRunningAnnotation: "true", secretSourceUpdatedAnnotation: "123"}
		user.Spec.ConnInfoSecretRoute = service.RouteTypePrivatelink
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{ValkeyACLKeys: []string{"prefix_*:*"}}
		srcPassword := "external-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(srcPassword)},
		}

		withoutRoute := &service.ServiceGetOut{
			State:       service.ServiceStateTypeRunning,
			ServiceType: "valkey",
			ServiceName: user.Spec.ServiceName,
			Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
		}
		withRoute := &service.ServiceGetOut{
			State:       service.ServiceStateTypeRunning,
			ServiceType: "valkey",
			ServiceName: user.Spec.ServiceName,
			Components: []service.ComponentOut{
				primaryComponent("valkey", "host", 6379),
				{Component: "valkey", Host: "pl-host", Port: 26379, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary},
			},
		}

		avn := avngen.NewMockClient(t)
		// First reconcile waits; the second observes the route, updates and rebuilds the secret.
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(withoutRoute, nil).Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(withRoute, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Times(3)
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.Operation == service.ServiceUserCredentialsModifyOperationTypeSetAccessControl &&
					matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.NewPassword != nil && *in.NewPassword == srcPassword &&
					in.Operation == service.ServiceUserCredentialsModifyOperationTypeResetCredentials
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Times(3)

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{NamespacedName: types.NamespacedName{Name: user.Name, Namespace: user.Namespace}})
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pl-host"), secret.Data["SERVICEUSER_HOST"])
		require.Equal(t, []byte("26379"), secret.Data["SERVICEUSER_PORT"])
	})

	t.Run("Retries transient not found for ready ServiceUser before treating it as absent", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
			user.Generation = 1
			user.Annotations = map[string]string{
				processedGenerationAnnotation: "1",
				instanceIsRunningAnnotation:   "true",
			}

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
				Return(&service.ServiceGetOut{
					State:       service.ServiceStateTypeRunning,
					ServiceType: "kafka",
					Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
				}, nil).Once()
			avn.EXPECT().
				ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
				Return(nil, newAivenError(404, "not found")).Twice()
			avn.EXPECT().
				ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
				Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
			avn.EXPECT().
				ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

			r, res := runScenario(t, user, avn)
			require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

			secret := &corev1.Secret{}
			require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
			require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
		})
	})

	t.Run("Observe with no source secret publishes whatever the API returns, including empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Once()
		// Observe path does not retry empty-password — single fetch.
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: ""}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(""), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Empty password on Observe with source secret heals via Update", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}

		srcPassword := "external-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(srcPassword)},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Twice()
		// Observe sees empty.
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: ""}, nil).Once()
		// Update pushes the source-secret password.
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.NewPassword != nil && *in.NewPassword == srcPassword &&
					in.Operation == service.ServiceUserCredentialsModifyOperationTypeResetCredentials
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		// Post-Update fetch sees the populated password.
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: srcPassword}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(srcPassword), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Leaves authentication unchanged when the spec omits it", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "mysql",
				Components:  []service.ComponentOut{primaryComponent("mysql", "host", 3306)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw", Authentication: service.AuthenticationTypeMysqlNativePassword}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Doesn't detect authentication drift when the API omits the field", func(t *testing.T) {
		for _, serviceType := range []string{"mysql", "pg"} {
			t.Run(serviceType, func(t *testing.T) {
				user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
				user.Generation = 1
				user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
				user.Annotations = map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				}

				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
					Return(&service.ServiceGetOut{
						State:       service.ServiceStateTypeRunning,
						ServiceType: serviceType,
						Components:  []service.ComponentOut{primaryComponent(serviceType, "host", 1234)},
					}, nil).Once()
				avn.EXPECT().
					ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
					Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
				avn.EXPECT().
					ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

				r, res := runScenario(t, user, avn)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
				secret := &corev1.Secret{}
				require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
				require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
			})
		}
	})

	t.Run("Applies authentication to existing users while preserving their password", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			serviceType string
			generation  int64
			annotations map[string]string
			desired     service.AuthenticationType
			actual      service.AuthenticationType
		}{
			{
				name:        "repairs an external change during periodic reconciliation",
				serviceType: "mysql",
				generation:  1,
				annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
				desired: service.AuthenticationTypeMysqlNativePassword,
				actual:  service.AuthenticationTypeCachingSha2Password,
			},
			{
				name:        "applies a changed spec",
				serviceType: "mysql",
				generation:  2,
				annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
				desired: service.AuthenticationTypeCachingSha2Password,
				actual:  service.AuthenticationTypeMysqlNativePassword,
			},
			{
				name:        "adopts an existing Aiven user",
				serviceType: "mysql",
				generation:  1,
				desired:     service.AuthenticationTypeMysqlNativePassword,
				actual:      service.AuthenticationTypeCachingSha2Password,
			},
			{
				name:        "uses authentication when another service exposes it",
				serviceType: "future-service",
				generation:  1,
				annotations: map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				},
				desired: service.AuthenticationTypeMysqlNativePassword,
				actual:  service.AuthenticationTypeCachingSha2Password,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
				user.Generation = tc.generation
				user.Annotations = tc.annotations
				user.Spec.Username = "existing_database_user"
				user.Spec.Authentication = tc.desired
				password := "existing-password"

				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
					Return(&service.ServiceGetOut{
						State:       service.ServiceStateTypeRunning,
						ServiceType: tc.serviceType,
						Components:  []service.ComponentOut{primaryComponent(tc.serviceType, "host", 3306)},
					}, nil).Times(3)
				avn.EXPECT().
					ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username).
					Return(&service.ServiceUserGetOut{Username: user.Spec.Username, Password: password, Authentication: tc.actual}, nil).Twice()
				avn.EXPECT().
					ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username, &service.ServiceUserCredentialsModifyIn{
						NewPassword:    &password,
						Authentication: tc.desired,
						Operation:      service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
					}).
					Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
				avn.EXPECT().
					ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username).
					Return(&service.ServiceUserGetOut{Username: user.Spec.Username, Password: password, Authentication: tc.desired}, nil).Twice()
				avn.EXPECT().
					ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Times(3)

				r, res := runScenario(t, user, avn)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

				secret := &corev1.Secret{}
				require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
				require.Equal(t, []byte(user.Spec.Username), secret.Data["SERVICEUSER_USERNAME"])
				require.Equal(t, []byte(password), secret.Data["SERVICEUSER_PASSWORD"])

				rec := r.Recorder.(*record.FakeRecorder)
				require.True(t, slices.ContainsFunc(recorderEvents(rec), func(e string) bool {
					return strings.Contains(e, eventAuthenticationReset)
				}), "expected %s event", eventAuthenticationReset)

				// Once the method matches, the next poll must not reset credentials again.
				res, err := r.Reconcile(t.Context(), ctrlruntime.Request{NamespacedName: client.ObjectKeyFromObject(user)})
				require.NoError(t, err)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
				require.False(t, slices.ContainsFunc(recorderEvents(rec), func(e string) bool {
					return strings.Contains(e, eventAuthenticationReset)
				}), "unexpected %s event: authentication already matched", eventAuthenticationReset)
			})
		}
	})

	t.Run("Doesn't reset matching authentication when another spec change triggers Update", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 2
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "mysql",
				Components:  []service.ComponentOut{primaryComponent("mysql", "host", 3306)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw", Authentication: user.Spec.Authentication}, nil).Times(3)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
		require.False(t, slices.ContainsFunc(recorderEvents(r.Recorder.(*record.FakeRecorder)), func(e string) bool {
			return strings.Contains(e, eventAuthenticationReset)
		}), "unexpected %s event: authentication already matched", eventAuthenticationReset)
	})

	t.Run("Skips authentication during Update when the API omits the field", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			password string
		}{
			{name: "password is available", password: "existing-password"},
			{name: "password is unavailable"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
				user.Generation = 2
				user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
				user.Annotations = map[string]string{
					processedGenerationAnnotation: "1",
					instanceIsRunningAnnotation:   "true",
				}

				avn := avngen.NewMockClient(t)
				avn.EXPECT().
					ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
					Return(&service.ServiceGetOut{
						State:       service.ServiceStateTypeRunning,
						ServiceType: "pg",
						Components:  []service.ComponentOut{primaryComponent("pg", "host", 5432)},
					}, nil).Twice()
				avn.EXPECT().
					ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
					Return(&service.ServiceUserGetOut{Username: user.Name, Password: tc.password}, nil).Times(3)
				avn.EXPECT().
					ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

				r, res := runScenario(t, user, avn)
				require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
				got := &v1alpha1.ServiceUser{}
				require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
				require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
				require.Equal(t, user.Spec.Authentication, got.Spec.Authentication)
				secret := &corev1.Secret{}
				require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
				require.Equal(t, tc.password, string(secret.Data["SERVICEUSER_PASSWORD"]))
			})
		}
	})

	t.Run("Updates the source password without authentication when the API omits the field", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 2
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		password := "source-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(password)},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "pg",
				Components:  []service.ComponentOut{primaryComponent("pg", "host", 5432)},
			}, nil).Times(3)
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "previous-password"}, nil).Twice()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, &service.ServiceUserCredentialsModifyIn{
				NewPassword: &password,
				Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
			}).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: password}, nil).Twice()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Times(3)

		r, res := runScenario(t, user, avn, src)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
		require.Equal(t, []byte(password), secret.Data["SERVICEUSER_PASSWORD"])
		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
		require.Equal(t, user.Spec.Authentication, got.Spec.Authentication)

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{NamespacedName: client.ObjectKeyFromObject(user)})
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
	})

	t.Run("Reports authentication drift without resetting an unknown password", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: user.Name, Namespace: user.Namespace},
			Data:       map[string][]byte{"SERVICEUSER_PASSWORD": []byte("published-password")},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "mysql",
				Components:  []service.ComponentOut{primaryComponent("mysql", "host", 3306)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Authentication: service.AuthenticationTypeCachingSha2Password}, nil).Twice()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, _, err := runScenarioErr(t, user, avn, secret)
		require.ErrorContains(t, err, "cannot change service user authentication without a known password")
		require.False(t, slices.ContainsFunc(recorderEvents(r.Recorder.(*record.FakeRecorder)), func(e string) bool {
			return strings.Contains(e, eventAuthenticationReset)
		}), "unexpected %s event: nothing was reset", eventAuthenticationReset)
		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
		require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
		gotSecret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(secret), gotSecret))
		require.Equal(t, secret.Data, gotSecret.Data)
	})

	t.Run("Retries a failed authentication update using the source password", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Authentication = service.AuthenticationTypeMysqlNativePassword
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{Name: "src", PasswordKey: "PASSWORD"}
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		password := "source-secret-password"
		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "src", Namespace: user.Namespace},
			Data:       map[string][]byte{"PASSWORD": []byte(password)},
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "mysql",
				Components:  []service.ComponentOut{primaryComponent("mysql", "host", 3306)},
			}, nil).Times(3)
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "api-password", Authentication: service.AuthenticationTypeCachingSha2Password}, nil).Times(4)
		request := &service.ServiceUserCredentialsModifyIn{
			NewPassword:    &password,
			Authentication: user.Spec.Authentication,
			Operation:      service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		}
		apiErr := newAivenError(500, "authentication update failed")
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, request).
			Return(nil, apiErr).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Times(3)

		r, _, err := runScenarioErr(t, user, avn, src)
		require.ErrorIs(t, err, apiErr)
		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
		condition := meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError)
		require.NotNil(t, condition)
		require.Contains(t, condition.Message, "authentication update failed")

		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, request).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: password, Authentication: user.Spec.Authentication}, nil).Once()

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{NamespacedName: client.ObjectKeyFromObject(user)})
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), got))
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), client.ObjectKeyFromObject(user), secret))
		require.Equal(t, []byte(password), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Repairs managed Valkey ACL drift during periodic reconcile", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.AccessControl = &v1alpha1.ServiceUserAccessControl{
			ValkeyACLKeys:       []string{"prefix_*:*"},
			ValkeyACLCommands:   []string{"-acl"},
			ValkeyACLCategories: []string{"+@all"},
			ValkeyACLChannels:   []string{"some*chan"},
		}
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "valkey",
				Components:  []service.ComponentOut{primaryComponent("valkey", "host", 6379)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username: user.Name,
				Password: "pw",
				AccessControl: &service.AccessControlOut{
					ValkeyAclKeys:       []string{"different:*"},
					ValkeyAclCommands:   []string{"-acl"},
					ValkeyAclCategories: []string{"+@all"},
					ValkeyAclChannels:   []string{"some*chan"},
				},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserCredentialsModify(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name, mock.MatchedBy(func(in *service.ServiceUserCredentialsModifyIn) bool {
				return in.Operation == service.ServiceUserCredentialsModifyOperationTypeSetAccessControl &&
					in.NewPassword == nil &&
					matchValkeyAccessControl(user.Spec.AccessControl)(in.AccessControl)
			})).
			Return(&service.ServiceUserCredentialsModifyOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(&service.ServiceUserGetOut{
				Username:      user.Name,
				Password:      "pw",
				AccessControl: valkeyAccessControlOut(user.Spec.AccessControl),
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Twice()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Returns error when create races with existing ServiceUser", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(409, "already exists")).Once()

		r, _, err := runScenarioErr(t, user, avn)
		require.EqualError(t, err, `unable to create or update instance at aiven: creating service user: [409 ]: already exists`)

		got := &v1alpha1.ServiceUser{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Empty(t, got.Annotations)

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletes ServiceUser and removes finalizer on deletion", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		user.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Name).
			Return(nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceUser{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Creates ServiceUser on Aiven using spec.username override", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		// A legal Aiven username that cannot be a Kubernetes object name.
		user.Spec.Username = "test_team_test_app_1a2b3c4d_abc"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:       service.ServiceStateTypeRunning,
				ServiceType: "kafka",
				Components:  []service.ComponentOut{primaryComponent("kafka", "host", 9092)},
			}, nil).Twice()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			ServiceUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *service.ServiceUserCreateIn) bool {
				return in.Username == user.Spec.Username
			})).
			Return(&service.ServiceUserCreateOut{}, nil).Once()
		avn.EXPECT().
			ServiceUserGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username).
			Return(&service.ServiceUserGetOut{Username: user.Spec.Username, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		// The secret keeps the resource's name; the username inside it is the override.
		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte(user.Spec.Username), secret.Data["SERVICEUSER_USERNAME"])
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
	})

	t.Run("Skips deletion at Aiven when another resource manages the same username", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Username = "shared_user"
		user.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		user.DeletionTimestamp = &now

		other := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		other.Name = "other-user"
		other.Namespace = "other-namespace"
		other.Spec.Username = "shared_user"

		// No ServiceUserDelete expected: the mock fails the test on an unexpected call.
		avn := avngen.NewMockClient(t)

		r, res := runScenario(t, user, avn, other)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceUser{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))

		// The deleted resource gets a Warning event explaining the skip,
		// and no event claiming the user is gone at Aiven
		rec := r.Recorder.(*record.FakeRecorder)
		var skipped, deleted bool
		for len(rec.Events) > 0 {
			e := <-rec.Events
			skipped = skipped || strings.Contains(e, eventSkippedDeletionAtAiven)
			deleted = deleted || strings.Contains(e, eventSuccessfullyDeletedAtAiven)
		}
		require.True(t, skipped, "expected %s warning event", eventSkippedDeletionAtAiven)
		require.False(t, deleted, "unexpected %s event: nothing was deleted at Aiven", eventSuccessfullyDeletedAtAiven)
	})

	t.Run("Skips deletion at Aiven when another resource's name matches the username", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Username = "other-user"
		user.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		user.DeletionTimestamp = &now

		other := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		other.Name = "other-user"

		avn := avngen.NewMockClient(t)

		r, res := runScenario(t, user, avn, other)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceUser{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletes at Aiven when the resource sharing the username is also being deleted", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Username = "shared_user"
		user.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		user.DeletionTimestamp = &now

		other := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		other.Name = "other-user"
		other.Spec.Username = "shared_user"
		other.Finalizers = []string{instanceDeletionFinalizer}
		other.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, "shared_user").
			Return(nil).Once()

		r, res := runScenario(t, user, avn, other)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceUser{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletes ServiceUser using spec.username override", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ServiceUser](t, yamlServiceUser)
		user.Generation = 1
		user.Spec.Username = "test_team_test_app_1a2b3c4d_abc"
		user.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		user.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Spec.Username).
			Return(nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ServiceUser{}
		err := r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}

func TestServiceUserSecretDetails(t *testing.T) {
	accessCert, accessKey := "cert", "key"
	u := &service.ServiceUserGetOut{Username: "alice", Password: "pw", AccessCert: &accessCert, AccessKey: &accessKey}
	base := func(host, port string) SecretDetails {
		return SecretDetails{
			"P_HOST": host, "P_PORT": port, "P_USERNAME": "alice", "P_PASSWORD": "pw",
			"P_ACCESS_CERT": "cert", "P_ACCESS_KEY": "key", "P_CA_CERT": "ca",
		}
	}
	withSasl := func(d SecretDetails, host, port string) SecretDetails {
		d["P_SASL_HOST"], d["P_SASL_PORT"] = host, port
		return d
	}
	withSchemaRegistry := func(d SecretDetails, host, port string) SecretDetails {
		d["P_SCHEMA_REGISTRY_HOST"], d["P_SCHEMA_REGISTRY_PORT"] = host, port
		return d
	}
	withKafka := func(d SecretDetails, saslHost, saslPort, srHost, srPort string) SecretDetails {
		return withSchemaRegistry(withSasl(d, saslHost, saslPort), srHost, srPort)
	}
	droppingComponent := func(name string) func() []service.ComponentOut {
		return func() []service.ComponentOut {
			return without(testComponents(), func(c service.ComponentOut) bool { return c.Component == name })
		}
	}
	droppingKafkaAuth := func(auth service.KafkaAuthenticationMethodType) func() []service.ComponentOut {
		return func() []service.ComponentOut {
			return without(testComponents(), func(c service.ComponentOut) bool { return c.KafkaAuthenticationMethod == auth })
		}
	}

	cases := []struct {
		name         string
		serviceType  string
		route        service.RouteType             // spec value; empty keeps the legacy selection
		components   func() []service.ComponentOut // nil means testComponents
		userConfig   map[string]any
		want         SecretDetails
		wantReversed SecretDetails // legacy cases only: the selection follows API order
		wantErr      []string
		hardErr      bool // wantErr is not a precondition
	}{
		{
			name: "unset route keeps the legacy pg selection: first entry in API order", serviceType: "pg",
			want: base("pg-dynamic-primary", "1"), wantReversed: base("pg-public-primary", "4"),
		},
		{name: "dynamic pg", serviceType: "pg", route: service.RouteTypeDynamic, want: base("pg-dynamic-primary", "1")},
		{name: "privatelink pg", serviceType: "pg", route: service.RouteTypePrivatelink, want: base("pg-privatelink-primary", "3")},
		{name: "public pg", serviceType: "pg", route: service.RouteTypePublic, want: base("pg-public-primary", "4")},
		{
			name: "unset route keeps the legacy kafka selection: first kafka, last sasl and schema registry", serviceType: "kafka",
			want:         withKafka(base("kafka-dynamic-cert", "5"), "kafka-privatelink-sasl", "8", "sr-privatelink", "10"),
			wantReversed: withKafka(base("kafka-privatelink-sasl", "8"), "kafka-dynamic-sasl", "6", "sr-dynamic", "9"),
		},
		{
			name: "unset route with letsencrypt sasl takes whichever sasl entry is listed last", serviceType: "kafka",
			components:   func() []service.ComponentOut { return withLetsencryptSasl(testComponents()) },
			want:         withKafka(base("kafka-dynamic-cert", "5"), "kafka-privatelink-sasl-letsencrypt", "12", "sr-privatelink", "10"),
			wantReversed: withKafka(base("kafka-privatelink-sasl-letsencrypt", "12"), "kafka-dynamic-sasl", "6", "sr-dynamic", "9"),
		},
		{
			name: "dynamic kafka picks the dynamic primary entries", serviceType: "kafka", route: service.RouteTypeDynamic,
			want: withKafka(base("kafka-dynamic-cert", "5"), "kafka-dynamic-sasl", "6", "sr-dynamic", "9"),
		},
		{
			name: "privatelink kafka", serviceType: "kafka", route: service.RouteTypePrivatelink,
			want: withKafka(base("kafka-privatelink-cert", "7"), "kafka-privatelink-sasl", "8", "sr-privatelink", "10"),
		},
		{
			name: "kafka without certificate auth takes the sasl entry for HOST", serviceType: "kafka", route: service.RouteTypePrivatelink,
			components: droppingKafkaAuth(service.KafkaAuthenticationMethodTypeCertificate),
			want:       withKafka(base("kafka-privatelink-sasl", "8"), "kafka-privatelink-sasl", "8", "sr-privatelink", "10"),
		},
		{
			name: "kafka with letsencrypt sasl keeps the project CA sasl entry", serviceType: "kafka", route: service.RouteTypeDynamic,
			components: func() []service.ComponentOut { return withLetsencryptSasl(testComponents()) },
			want:       withKafka(base("kafka-dynamic-cert", "5"), "kafka-dynamic-sasl", "6", "sr-dynamic", "9"),
		},
		{
			name: "kafka privatelink with letsencrypt sasl keeps the project CA sasl entry", serviceType: "kafka", route: service.RouteTypePrivatelink,
			components: func() []service.ComponentOut { return withLetsencryptSasl(testComponents()) },
			want:       withKafka(base("kafka-privatelink-cert", "7"), "kafka-privatelink-sasl", "8", "sr-privatelink", "10"),
		},
		{
			name: "kafka without certificate auth and with letsencrypt sasl takes the project CA sasl entry for HOST", serviceType: "kafka", route: service.RouteTypePrivatelink,
			components: func() []service.ComponentOut {
				return withLetsencryptSasl(droppingKafkaAuth(service.KafkaAuthenticationMethodTypeCertificate)())
			},
			want: withKafka(base("kafka-privatelink-sasl", "8"), "kafka-privatelink-sasl", "8", "sr-privatelink", "10"),
		},
		{
			name: "kafka dynamic without schema registry blanks the schema registry keys", serviceType: "kafka", route: service.RouteTypeDynamic,
			components: droppingComponent("schema_registry"),
			want:       withKafka(base("kafka-dynamic-cert", "5"), "kafka-dynamic-sasl", "6", "", ""),
		},
		{
			name: "kafka privatelink without schema registry blanks the schema registry keys", serviceType: "kafka", route: service.RouteTypePrivatelink,
			components: droppingComponent("schema_registry"),
			want:       withKafka(base("kafka-privatelink-cert", "7"), "kafka-privatelink-sasl", "8", "", ""),
		},
		{
			name: "kafka blanks disabled sasl and schema registry", serviceType: "kafka", route: service.RouteTypePrivatelink,
			userConfig: map[string]any{"kafka_authentication_methods": map[string]any{"sasl": false}, "schema_registry": false},
			want:       withKafka(base("kafka-privatelink-cert", "7"), "", "", "", ""),
		},
		{
			name: "missing route is a precondition", serviceType: "pg", route: service.RouteTypePrivate,
			wantErr: []string{`component "pg"`, `route "private"`, `service "svc"`},
		},
		{
			name: "missing route is a precondition for kafka", serviceType: "kafka", route: service.RouteTypePrivate,
			wantErr: []string{`component "kafka"`, `route "private"`, `service "svc"`},
		},
		{
			name: "missing component on the unset route is an error", serviceType: "mysql",
			wantErr: []string{`service component "mysql" not found`}, hardErr: true,
		},
		{
			name: "missing component on the dynamic route is an error", serviceType: "mysql", route: service.RouteTypeDynamic,
			wantErr: []string{`service component "mysql" not found`}, hardErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			components := tc.components
			if components == nil {
				components = testComponents
			}
			run := func(components []service.ComponentOut) (SecretDetails, error) {
				svc := &service.ServiceGetOut{ServiceType: tc.serviceType, ServiceName: "svc", Components: components, UserConfig: tc.userConfig}
				return serviceUserSecretDetails(t.Context(), svc, u, "ca", "P_", tc.route)
			}
			listed, err := run(components())
			rev, errRev := run(reversed(components()))

			if tc.wantErr != nil {
				for _, e := range []error{err, errRev} {
					if tc.hardErr {
						require.NotErrorIs(t, e, errPreconditionNotMet)
					} else {
						require.ErrorIs(t, e, errPreconditionNotMet)
					}
					for _, want := range tc.wantErr {
						require.ErrorContains(t, e, want)
					}
				}
				require.Nil(t, listed)
				require.Nil(t, rev)
				return
			}

			require.NoError(t, err)
			require.NoError(t, errRev)
			require.Equal(t, tc.want, listed)
			if tc.wantReversed != nil {
				require.Equal(t, tc.wantReversed, rev, "the legacy selection follows API order")
				return
			}
			require.Equal(t, listed, rev)
		})
	}

	// Real API data: with two PrivateLink connections every listener is listed once per
	// connection, so the route alone does not identify a component. The operator takes the first
	// in API order (connection 1) and, unlike every case above, the choice depends on that order.
	t.Run("recorded kafka with two privatelink connections takes the first connection", func(t *testing.T) {
		const pl1 = "privatelink-1-rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com"
		const pl2 = "privatelink-2-rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com"
		run := func(components []service.ComponentOut, route service.RouteType) SecretDetails {
			svc := &service.ServiceGetOut{ServiceType: "kafka", ServiceName: "svc", Components: components}
			details, err := serviceUserSecretDetails(t.Context(), svc, u, "ca", "P_", route)
			require.NoError(t, err)
			return details
		}

		require.Equal(t,
			withKafka(base(pl1, "23220"), pl1, "23224", pl1, "14614"),
			run(recordedKafkaPrivatelinkComponents(), service.RouteTypePrivatelink))
		require.Equal(t,
			withKafka(base(pl2, "23228"), pl2, "23232", pl2, "14614"),
			run(reversed(recordedKafkaPrivatelinkComponents()), service.RouteTypePrivatelink),
			"ties between connections follow API order")
		require.Equal(t,
			withKafka(base("rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com", "14611"),
				"rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com", "14622",
				"rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com", "14614"),
			run(recordedKafkaPrivatelinkComponents(), service.RouteTypeDynamic),
			"the dynamic route is unaffected by the privatelink entries")
		require.Equal(t,
			withKafka(base("rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com", "14611"), pl2, "23232", pl2, "14614"),
			run(recordedKafkaPrivatelinkComponents(), routeLegacy),
			"the unset route keeps the pre-field secret: dynamic HOST, SASL and schema registry from the last listed connection")
	})
}
