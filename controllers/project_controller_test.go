package controllers

import (
	"errors"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	avnproject "github.com/aiven/go-client-codegen/handler/project"
	"github.com/aiven/go-client-codegen/handler/user"
	"github.com/stretchr/testify/assert"
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

func TestProjectReconciler(t *testing.T) {
	t.Parallel()

	const yamlProject = `
apiVersion: aiven.io/v1alpha1
kind: Project
metadata:
  name: test-project
  namespace: default
spec:
  accountId: test-account
  billingAddress: Helsinki
  billingEmails:
    - billing@example.com
  billingCurrency: EUR
  billingExtraText: PO-123
  billingGroupId: billing-group
  cardId: ""
  cloud: aws-eu-central-1
  copyFromProject: source-project
  countryCode: FI
  technicalEmails:
    - tech@example.com
  tags:
    env: test
`

	runScenario := func(t *testing.T, project *v1alpha1.Project, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.Project], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{project}, additionalObjects...)

		r := newProjectReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.Project{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(20),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.Project])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      project.Name,
				Namespace: project.Namespace,
			},
		})
		return r, res, err
	}

	t.Run("Creates Project on Aiven", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(nil, newAivenError(404, "project not found")).Once()
		avn.EXPECT().
			ProjectCreate(mock.Anything, mock.MatchedBy(func(in *avnproject.ProjectCreateIn) bool {
				return in.Project == project.Name &&
					in.AccountId != nil && *in.AccountId == project.Spec.AccountID &&
					in.BillingAddress != nil && *in.BillingAddress == project.Spec.BillingAddress &&
					in.BillingExtraText != nil && *in.BillingExtraText == project.Spec.BillingExtraText &&
					in.BillingGroupId != nil && *in.BillingGroupId == project.Spec.BillingGroupID &&
					in.CardId == nil &&
					in.Cloud != nil && *in.Cloud == project.Spec.Cloud &&
					in.CountryCode != nil && *in.CountryCode == project.Spec.CountryCode &&
					in.CopyFromProject != nil && *in.CopyFromProject == project.Spec.CopyFromProject &&
					in.BillingCurrency == project.Spec.BillingCurrency &&
					assert.Equal(t, projectBillingEmails(project.Spec.BillingEmails), *in.BillingEmails) &&
					assert.Equal(t, projectTechnicalEmails(project.Spec.TechnicalEmails), *in.TechEmails) &&
					in.Tags != nil &&
					assert.Equal(t, project.Spec.Tags, *in.Tags)
			})).
			Return(&avnproject.ProjectCreateOut{
				VatId:            "vat-1",
				EstimatedBalance: "10.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Equal(t, "100.00", got.Status.AvailableCredits)
		require.Equal(t, "Finland", got.Status.Country)
		require.NotEmpty(t, got.Status.EstimatedBalance)
		require.Equal(t, "card", got.Status.PaymentMethod)
		require.NotEmpty(t, got.Status.VatID)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionUnknown, condition.Status)

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Soft-requeues when Project create returns server error", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(nil, newAivenError(404, "project not found")).Once()
		avn.EXPECT().
			ProjectCreate(mock.Anything, mock.Anything).
			Return(nil, newAivenError(500, "temporary project create failure")).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Updates existing Project on Aiven", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		project.Spec.CardID = "4242"
		project.Annotations = map[string]string{
			processedGenerationAnnotation: "0",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(&avnproject.ProjectGetOut{ProjectName: project.Name}, nil).Once()
		avn.EXPECT().
			UserCreditCardsList(mock.Anything).
			Return([]user.CardOut{{CardId: "card-long-id", Last4: "4242"}}, nil).Once()
		avn.EXPECT().
			ProjectUpdate(mock.Anything, project.Name, mock.MatchedBy(func(in *avnproject.ProjectUpdateIn) bool {
				return in.AccountId != nil && *in.AccountId == project.Spec.AccountID &&
					in.BillingAddress != nil && *in.BillingAddress == project.Spec.BillingAddress &&
					in.BillingExtraText != nil && *in.BillingExtraText == project.Spec.BillingExtraText &&
					in.CardId != nil && *in.CardId == "card-long-id" &&
					in.Cloud != nil && *in.Cloud == project.Spec.Cloud &&
					in.CountryCode != nil && *in.CountryCode == project.Spec.CountryCode &&
					in.BillingCurrency == project.Spec.BillingCurrency &&
					assert.Equal(t, projectBillingEmails(project.Spec.BillingEmails), *in.BillingEmails) &&
					assert.Equal(t, projectTechnicalEmails(project.Spec.TechnicalEmails), *in.TechEmails) &&
					in.Tags != nil &&
					assert.Equal(t, project.Spec.Tags, *in.Tags)
			})).
			Return(&avnproject.ProjectUpdateOut{
				VatId:            "vat-2",
				EstimatedBalance: "20.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Equal(t, "100.00", got.Status.AvailableCredits)
		require.Equal(t, "Finland", got.Status.Country)
		require.NotEmpty(t, got.Status.EstimatedBalance)
		require.Equal(t, "card", got.Status.PaymentMethod)
		require.NotEmpty(t, got.Status.VatID)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionUnknown, condition.Status)

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Soft-requeues when Project update returns server error", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(&avnproject.ProjectGetOut{ProjectName: project.Name}, nil).Once()
		avn.EXPECT().
			ProjectUpdate(mock.Anything, project.Name, mock.Anything).
			Return(nil, newAivenError(500, "temporary project update failure")).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Fetches CA after Project mutation is processed", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		project.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(&avnproject.ProjectGetOut{
				VatId:            "vat-ready",
				EstimatedBalance: "40.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, project.Name).
			Return("ca-cert", nil).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "100.00", got.Status.AvailableCredits)
		require.Equal(t, "Finland", got.Status.Country)
		require.NotEmpty(t, got.Status.EstimatedBalance)
		require.Equal(t, "card", got.Status.PaymentMethod)
		require.NotEmpty(t, got.Status.VatID)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionTrue, condition.Status)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret))
		require.Equal(t, "ca-cert", string(secret.Data["CA_CERT"]))
		require.Equal(t, "ca-cert", string(secret.Data["PROJECT_CA_CERT"]))
	})

	t.Run("Observes Aiven when Project is ready", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		project.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(&avnproject.ProjectGetOut{
				VatId:            "vat-ready",
				EstimatedBalance: "40.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, project.Name).
			Return("ca-cert", nil).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "100.00", got.Status.AvailableCredits)
		require.Equal(t, "Finland", got.Status.Country)
		require.NotEmpty(t, got.Status.EstimatedBalance)
		require.Equal(t, "card", got.Status.PaymentMethod)
		require.NotEmpty(t, got.Status.VatID)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret))
		require.Equal(t, "ca-cert", string(secret.Data["CA_CERT"]))
		require.Equal(t, "ca-cert", string(secret.Data["PROJECT_CA_CERT"]))
	})

	t.Run("Does not mark Project running when CA fetch fails", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		project.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(&avnproject.ProjectGetOut{
				VatId:            "vat-error",
				EstimatedBalance: "50.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, project.Name).
			Return("", errors.New("ca unavailable")).Once()

		r, res, err := runScenario(t, project, avn)
		require.ErrorContains(t, err, "cannot observe the resource: getting project KMS CA: ca unavailable")
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		running := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.True(t, running == nil || running.Status != metav1.ConditionTrue)
		require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletes Project and removes finalizer when Aiven returns not found", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		project.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		project.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectDelete(mock.Anything, project.Name).
			Return(newAivenError(404, "project not found")).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.Project{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Skips connection secret when disabled", func(t *testing.T) {
		project := newObjectFromYAML[v1alpha1.Project](t, yamlProject)
		project.Generation = 1
		disabled := true
		project.Spec.ConnInfoSecretTargetDisabled = &disabled

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ProjectGet(mock.Anything, project.Name).
			Return(nil, newAivenError(404, "project not found")).Once()
		avn.EXPECT().
			ProjectCreate(mock.Anything, mock.MatchedBy(func(in *avnproject.ProjectCreateIn) bool {
				return in.Project == project.Name &&
					in.AccountId != nil && *in.AccountId == project.Spec.AccountID &&
					in.BillingAddress != nil && *in.BillingAddress == project.Spec.BillingAddress &&
					in.BillingExtraText != nil && *in.BillingExtraText == project.Spec.BillingExtraText &&
					in.BillingGroupId != nil && *in.BillingGroupId == project.Spec.BillingGroupID &&
					in.CardId == nil &&
					in.Cloud != nil && *in.Cloud == project.Spec.Cloud &&
					in.CountryCode != nil && *in.CountryCode == project.Spec.CountryCode &&
					in.CopyFromProject != nil && *in.CopyFromProject == project.Spec.CopyFromProject &&
					in.BillingCurrency == project.Spec.BillingCurrency &&
					assert.Equal(t, projectBillingEmails(project.Spec.BillingEmails), *in.BillingEmails) &&
					assert.Equal(t, projectTechnicalEmails(project.Spec.TechnicalEmails), *in.TechEmails) &&
					in.Tags != nil &&
					assert.Equal(t, project.Spec.Tags, *in.Tags)
			})).
			Return(&avnproject.ProjectCreateOut{
				VatId:            "vat-3",
				EstimatedBalance: "30.00",
				AvailableCredits: new("100.00"),
				Country:          "Finland",
				PaymentMethod:    "card",
			}, nil).Once()

		r, res, err := runScenario(t, project, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.Project{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, got))
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionUnknown, condition.Status)

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: project.Name, Namespace: project.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})
}
