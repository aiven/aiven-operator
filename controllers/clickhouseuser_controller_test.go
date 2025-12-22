package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func Test_newClickhouseUserReconciler(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	controller := Controller{
		Client: k8sClient,
	}

	r := newClickhouseUserReconciler(controller)

	rec, ok := r.(*Reconciler[*v1alpha1.ClickhouseUser])
	require.True(t, ok)

	obj := rec.newObj()
	require.IsType(t, &v1alpha1.ClickhouseUser{}, obj)

	ctrl := rec.newController(nil)
	userCtrl, ok := ctrl.(*ClickhouseUserController)
	require.True(t, ok)
	require.Equal(t, k8sClient, userCtrl.Client)
}

func TestClickhouseUserController_Observe(t *testing.T) {
	t.Parallel()

	t.Run("Returns error when service is not operational", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		_, err := ctrl.Observe(t.Context(), user)

		require.EqualError(t, err, "preconditions are not met: [404 ]: service not found")
	})

	t.Run("Sets ResourceExists and UUID when user exists", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9000"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-1"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: nil,
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.False(t, obs.ResourceUpToDate)
		require.Equal(t, "uuid-1", user.Status.UUID)
	})

	t.Run("Marks resource up to date when IsReadyToUse", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		user.SetGeneration(1)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-1"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.True(t, obs.ResourceUpToDate)
	})

	t.Run("Returns error when listing users fails", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		_, err := ctrl.Observe(t.Context(), user)

		require.EqualError(t, err, "listing Clickhouse users: "+assert.AnError.Error())
	})

	t.Run("Returns empty observation when user does not exist", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: "other-user", Uuid: "uuid-other"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.Equal(t, Observation{}, obs)
	})

	t.Run("Returns error when getting service details fails in Observe", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		_, err := ctrl.Observe(t.Context(), user)

		require.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("Populates SecretDetails in external mode using ConnInfoSecretSource", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "src",
			PasswordKey: "PASSWORD",
		}

		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "src",
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"PASSWORD": []byte("external-observe-password"),
			},
		}

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, src).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "8443"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-ext"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.Equal(t, "uuid-ext", user.Status.UUID)
		require.Equal(t, "external-observe-password", obs.SecretDetails["PASSWORD"])
	})

	t.Run("Populates SecretDetails in operator mode using password from Aiven API", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		apiPassword := "aiven-api-password"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-api", Password: &apiPassword}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.Equal(t, "uuid-api", user.Status.UUID)
		require.Equal(t, apiPassword, obs.SecretDetails["PASSWORD"])
	})

	t.Run("Omits password in operator mode when Aiven API does not expose it", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-nopw"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)

		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.Equal(t, "uuid-nopw", user.Status.UUID)
		require.NotContains(t, obs.SecretDetails, "PASSWORD")
	})

	t.Run("Returns error when reading password from source secret fails in external mode", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "missing-src",
			PasswordKey: "PASSWORD",
		}

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{State: service.ServiceStateTypeRunning}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.GetUsername(), Uuid: "uuid-err"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Observe(t.Context(), user)

		require.EqualError(t, err, `failed to read connInfoSecretSource default/missing-src: secrets "missing-src" not found`)
	})
}

func TestClickhouseUserController_Create(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("External mode uses password from ConnInfoSecretSource", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "src",
			PasswordKey: "PASSWORD",
		}

		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "src",
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"PASSWORD": []byte("external-secret-password"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, src).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password != nil && *in.Password == "external-secret-password"
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-1"}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHousePasswordReset(mock.Anything, user.Spec.Project, user.Spec.ServiceName, "uuid-1", mock.MatchedBy(func(in *clickhouse.ServiceClickHousePasswordResetIn) bool {
				return in.Password != nil && *in.Password == "external-secret-password"
			})).
			Return("external-secret-password", nil).
			Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				ServiceUriParams: map[string]string{"host": "host", "port": "5432"},
			}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)

		require.NoError(t, err)
		require.Equal(t, "uuid-1", user.Status.UUID)
		require.True(t, meta.IsStatusConditionTrue(user.Status.Conditions, conditionTypeRunning))
		require.Equal(t, "true", user.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "external-secret-password", res.SecretDetails["PASSWORD"])
	})

	t.Run("Operator mode ignores password from existing connection Secret on Create", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		connSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"CLICKHOUSEUSER_PASSWORD": []byte("ignored-conn-secret-password"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, connSecret).
			Build()

		avn := avngen.NewMockClient(t)
		genPassword := "generated-from-create-with-existing-secret"
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-2", Password: &genPassword}, nil).
			Once()

		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9000"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)

		require.NoError(t, err)
		require.Equal(t, "uuid-2", user.Status.UUID)
		require.Equal(t, genPassword, res.SecretDetails["PASSWORD"])
	})

	t.Run("Operator mode generates password from Create response", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)

		genPassword := "generated-from-create"
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-3", Password: &genPassword}, nil).
			Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)

		require.NoError(t, err)
		require.Equal(t, "uuid-3", user.Status.UUID)
		require.Equal(t, genPassword, res.SecretDetails["PASSWORD"])
	})

	t.Run("Operator mode generates password via PasswordReset when Create response is empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-4"}, nil).
			Once()

		resetPassword := "generated-from-reset"
		avn.EXPECT().
			ServiceClickHousePasswordReset(mock.Anything, user.Spec.Project, user.Spec.ServiceName, "uuid-4", mock.AnythingOfType("*clickhouse.ServiceClickHousePasswordResetIn")).
			Return(resetPassword, nil).
			Once()

		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)

		require.NoError(t, err)
		require.Equal(t, "uuid-4", user.Status.UUID)
		require.Equal(t, resetPassword, res.SecretDetails["PASSWORD"])
	})

	t.Run("Returns error when reading password from source secret fails in Create", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "missing-src",
			PasswordKey: "PASSWORD",
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
		}

		_, err := ctrl.Create(t.Context(), user)

		require.EqualError(t, err, `failed to read connInfoSecretSource default/missing-src: secrets "missing-src" not found`)
	})

	t.Run("Wraps error from ServiceClickHouseUserCreate", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		// No external password and no connection secret -> desiredPassword == "" with no error.
		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.AnythingOfType("*clickhouse.ServiceClickHouseUserCreateIn")).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Create(t.Context(), user)

		require.EqualError(t, err, "creating Clickhouse user: "+assert.AnError.Error())
	})

	t.Run("Wraps error from PasswordReset when Create response is empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-err-reset"}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHousePasswordReset(mock.Anything, user.Spec.Project, user.Spec.ServiceName, "uuid-err-reset", mock.AnythingOfType("*clickhouse.ServiceClickHousePasswordResetIn")).
			Return("", assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Create(t.Context(), user)

		require.EqualError(t, err, "resetting Clickhouse user password: "+assert.AnError.Error())
	})

	t.Run("Wraps error from buildConnectionDetails", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)

		genPassword := "generated-from-create"
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.GetUsername() && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-build-err", Password: &genPassword}, nil).
			Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Create(t.Context(), user)

		require.EqualError(t, err, "building connection details: getting service details: "+assert.AnError.Error())
	})
}

func TestClickhouseUserController_Update(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	t.Run("External mode enforces password from ConnInfoSecretSource", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-1"
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "src",
			PasswordKey: "PASSWORD",
		}

		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "src",
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"PASSWORD": []byte("external-update-password"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, src).
			Build()

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceClickHousePasswordReset(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, mock.MatchedBy(func(in *clickhouse.ServiceClickHousePasswordResetIn) bool {
				return in.Password != nil && *in.Password == "external-update-password"
			})).
			Return("external-update-password", nil).
			Once()
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Update(t.Context(), user)

		require.NoError(t, err)
		require.True(t, meta.IsStatusConditionTrue(user.Status.Conditions, conditionTypeRunning))
		require.Equal(t, "true", user.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "external-update-password", res.SecretDetails["PASSWORD"])
	})

	t.Run("Operator mode does not change password and omits it from SecretDetails on Update", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-2"

		connSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"CLICKHOUSEUSER_PASSWORD": []byte("conn-update-password"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, connSecret).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		res, err := ctrl.Update(t.Context(), user)

		require.NoError(t, err)
		require.NotContains(t, res.SecretDetails, "PASSWORD")
	})

	t.Run("Returns error when reading password from source secret fails in Update", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-desired-fail"
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "missing-src",
			PasswordKey: "PASSWORD",
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
		}

		_, err := ctrl.Update(t.Context(), user)

		require.EqualError(t, err, `failed to read connInfoSecretSource default/missing-src: secrets "missing-src" not found`)
	})

	t.Run("Wraps error from ServiceClickHousePasswordReset", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-reset-fail"
		user.Spec.ConnInfoSecretSource = &v1alpha1.ConnInfoSecretSource{
			Name:        "src",
			PasswordKey: "PASSWORD",
		}

		src := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "src",
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"PASSWORD": []byte("external-update-password"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, src).
			Build()

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceClickHousePasswordReset(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID, mock.MatchedBy(func(in *clickhouse.ServiceClickHousePasswordResetIn) bool {
				return in.Password != nil && *in.Password == "external-update-password"
			})).
			Return("", assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Update(t.Context(), user)

		require.EqualError(t, err, "resetting Clickhouse user password: "+assert.AnError.Error())
	})

	t.Run("Wraps error from buildConnectionDetails", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-build-err"

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user).
			Build()

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			Client: k8sClient,
			avnGen: avn,
		}

		_, err := ctrl.Update(t.Context(), user)

		require.EqualError(t, err, "building connection details: getting service details: "+assert.AnError.Error())
	})
}

func TestClickhouseUserController_Delete(t *testing.T) {
	t.Parallel()

	t.Run("No-op when UUID is empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		ctrl := &ClickhouseUserController{}

		err := ctrl.Delete(t.Context(), user)

		require.NoError(t, err)
	})

	t.Run("No-op for built-in user", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-1"
		user.Name = defaultBuiltInUser

		ctrl := &ClickhouseUserController{}

		err := ctrl.Delete(t.Context(), user)

		require.NoError(t, err)
	})

	t.Run("No-op when spec.username is built-in", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-spec-builtin"
		user.Name = "custom-name"
		user.Spec.Username = defaultBuiltInUser

		avn := avngen.NewMockClient(t)

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		err := ctrl.Delete(t.Context(), user)

		require.NoError(t, err)
	})

	t.Run("Deletes when metadata.name is built-in but spec.username is not", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-spec-custom"
		user.Name = defaultBuiltInUser
		user.Spec.Username = "custom-user"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID).
			Return(nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		err := ctrl.Delete(t.Context(), user)

		require.NoError(t, err)
	})

	t.Run("Treats not found as success", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-2"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID).
			Return(newAivenError(404, "not found")).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		err := ctrl.Delete(t.Context(), user)

		require.NoError(t, err)
	})

	t.Run("Wraps delete errors", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-3"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID).
			Return(assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		err := ctrl.Delete(t.Context(), user)

		require.EqualError(t, err, "deleting Clickhouse user: "+assert.AnError.Error())
	})

	t.Run("Succeeds when delete returns nil", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Status.UUID = "uuid-4"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceClickHouseUserDelete(mock.Anything, user.Spec.Project, user.Spec.ServiceName, user.Status.UUID).
			Return(nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		err := ctrl.Delete(t.Context(), user)
		require.NoError(t, err)
	})
}

func TestClickhouseUserController_buildConnectionDetails(t *testing.T) {
	t.Parallel()

	user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

	t.Run("Success path populates prefixed and legacy keys", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "8443"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		details, err := ctrl.buildConnectionDetails(t.Context(), user, "pw")

		require.NoError(t, err)
		prefix := getSecretPrefix(user)
		require.Equal(t, "host", details[prefix+"HOST"])
		require.Equal(t, "8443", details[prefix+"PORT"])
		require.Equal(t, "pw", details[prefix+"PASSWORD"])
		require.Equal(t, "host", details["HOST"])
		require.Equal(t, "8443", details["PORT"])
		require.Equal(t, "pw", details["PASSWORD"])
	})

	t.Run("Skips password keys when password is empty", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "8443"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		details, err := ctrl.buildConnectionDetails(t.Context(), user, "")

		require.NoError(t, err)
		prefix := getSecretPrefix(user)
		require.Equal(t, "host", details[prefix+"HOST"])
		require.Equal(t, "8443", details[prefix+"PORT"])
		require.NotContains(t, details, prefix+"PASSWORD")
		require.Equal(t, "host", details["HOST"])
		require.Equal(t, "8443", details["PORT"])
		require.NotContains(t, details, "PASSWORD")
	})

	t.Run("Returns error when ServiceGet fails", func(t *testing.T) {
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(nil, assert.AnError).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		_, err := ctrl.buildConnectionDetails(t.Context(), user, "pw")

		require.EqualError(t, err, "getting service details: "+assert.AnError.Error())
	})
}

func TestClickhouseUser_BackwardCompatibility(t *testing.T) {
	t.Parallel()

	t.Run("Create uses metadata.name when spec.username is empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Name = "metadata-name"
		user.Spec.Username = ""

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.Name && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-compat-create-metadata", Password: ptr("mypassword")}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)
		require.NoError(t, err)

		prefix := getSecretPrefix(user)
		require.Equal(t, user.Name, res.SecretDetails[prefix+"USERNAME"])
		require.Equal(t, user.Name, res.SecretDetails["USERNAME"])
	})

	t.Run("Observe matches Aiven users by metadata.name when spec.username is empty", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Name = "metadata-name"
		user.Spec.Username = ""

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.Name, Uuid: "uuid-compat-observe-metadata"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.Equal(t, "uuid-compat-observe-metadata", user.Status.UUID)

		prefix := getSecretPrefix(user)
		require.Equal(t, user.Name, obs.SecretDetails[prefix+"USERNAME"])
		require.Equal(t, user.Name, obs.SecretDetails["USERNAME"])
	})

	t.Run("Create uses spec.username and publishes it to SecretDetails", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Name = "metadata-name"
		user.Spec.Username = "spec-username"

		avn := avngen.NewMockClient(t)

		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserCreate(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.MatchedBy(func(in *clickhouse.ServiceClickHouseUserCreateIn) bool {
				return in.Name == user.Spec.Username && in.Password == nil
			})).
			Return(&clickhouse.ServiceClickHouseUserCreateOut{Uuid: "uuid-compat-create", Password: ptr("mypassword")}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		res, err := ctrl.Create(t.Context(), user)
		require.NoError(t, err)

		prefix := getSecretPrefix(user)
		require.Equal(t, user.Spec.Username, res.SecretDetails[prefix+"USERNAME"])
		require.Equal(t, user.Spec.Username, res.SecretDetails["USERNAME"])
	})

	t.Run("Observe matches Aiven users by spec.username and publishes it to SecretDetails", func(t *testing.T) {
		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
		user.Name = "metadata-name"
		user.Spec.Username = "spec-username"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGet(mock.Anything, user.Spec.Project, user.Spec.ServiceName, mock.Anything).
			Return(&service.ServiceGetOut{
				State:            service.ServiceStateTypeRunning,
				ServiceUriParams: map[string]string{"host": "host", "port": "9440"},
			}, nil).
			Once()
		avn.EXPECT().
			ServiceClickHouseUserList(mock.Anything, user.Spec.Project, user.Spec.ServiceName).
			Return([]clickhouse.UserOut{{Name: user.Spec.Username, Uuid: "uuid-compat-observe"}}, nil).
			Once()

		ctrl := &ClickhouseUserController{
			avnGen: avn,
		}

		obs, err := ctrl.Observe(t.Context(), user)
		require.NoError(t, err)
		require.True(t, obs.ResourceExists)
		require.Equal(t, "uuid-compat-observe", user.Status.UUID)

		prefix := getSecretPrefix(user)
		require.Equal(t, user.Spec.Username, obs.SecretDetails[prefix+"USERNAME"])
		require.Equal(t, user.Spec.Username, obs.SecretDetails["USERNAME"])
	})

	t.Run("publishSecretDetails preserves existing password keys when omitted from details", func(t *testing.T) {
		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		user := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)

		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Name,
				Namespace: user.Namespace,
			},
			Data: map[string][]byte{
				"PASSWORD":                []byte("old-password"),
				"CLICKHOUSEUSER_PASSWORD": []byte("old-prefixed-password"),
				"EXTRA":                   []byte("keep-me"),
			},
		}

		k8sClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(user, existingSecret).
			Build()

		r := &Reconciler[*v1alpha1.ClickhouseUser]{
			Controller: Controller{
				Client:   k8sClient,
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(10),
			},
			newSecret: newSecret,
		}

		details := buildConnectionDetailsFromService(
			&service.ServiceGetOut{ServiceUriParams: map[string]string{"host": "host", "port": "9440"}},
			user,
			"",
		)

		require.NoError(t, r.publishSecretDetails(t.Context(), user, details))

		updated := &corev1.Secret{}
		require.NoError(t, k8sClient.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, updated))
		require.Equal(t, []byte("old-password"), updated.Data["PASSWORD"])
		require.Equal(t, []byte("old-prefixed-password"), updated.Data["CLICKHOUSEUSER_PASSWORD"])
		require.Equal(t, []byte("keep-me"), updated.Data["EXTRA"])
	})
}
