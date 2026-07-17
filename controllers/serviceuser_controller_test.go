package controllers

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
					Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate,
					},
					{
						Component:                 "kafka",
						Host:                      "kafka-sasl.example.com",
						Port:                      9093,
						KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl,
					},
					{Component: "schema_registry", Host: "schema.example.com", Port: 8081},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "valkey.example.com", Port: 6379}},
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
					Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "valkey", Host: "host", Port: 6379}},
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
				Components:  []service.ComponentOut{{Component: "kafka", Host: "host", Port: 9092}},
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
