package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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

func TestServiceUserReconciler(t *testing.T) {
	t.Parallel()

	runScenario := func(t *testing.T, user *v1alpha1.ServiceUser, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.ServiceUser], ctrlruntime.Result) {
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
		require.NoError(t, err)
		return r, res
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

	t.Run("Publishes secrets and requeues when ServiceUser is up to date", func(t *testing.T) {
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
			Return(&service.ServiceUserGetOut{Username: user.Name, Password: "pw"}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, user.Spec.Project).Return("ca", nil).Once()

		r, res := runScenario(t, user, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: user.Name, Namespace: user.Namespace}, secret))
		require.Equal(t, []byte("pw"), secret.Data["SERVICEUSER_PASSWORD"])
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
}
