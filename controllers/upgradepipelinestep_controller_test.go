package controllers

import (
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/upgradepipeline"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

const yamlUpgradePipelineStep = `
apiVersion: aiven.io/v1alpha1
kind: UpgradePipelineStep
metadata:
  name: test-step
  namespace: default
spec:
  organizationId: org123
  sourceProjectName: sandbox
  sourceServiceName: billing-pg-sandbox
  destinationProjectName: prod
  destinationServiceName: billing-pg-prod
  autoValidationDelayDays: 7
`

func TestUpgradePipelineStepReconciler(t *testing.T) {
	t.Parallel()

	runScenario := func(t *testing.T, step *v1alpha1.UpgradePipelineStep, avn avngen.Client, additionalObjects ...client.Object) (*Reconciler[*v1alpha1.UpgradePipelineStep], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		objects := append([]client.Object{step}, additionalObjects...)

		r := newUpgradePipelineStepReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.UpgradePipelineStep{}).
				WithObjects(objects...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.UpgradePipelineStep])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{
				Name:      step.Name,
				Namespace: step.Namespace,
			},
		})
		return r, res, err
	}

	t.Run("Creates upgrade pipeline step on Aiven", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepCreateIn) bool {
				return *in.AutoValidationDelayDays == step.Spec.AutoValidationDelayDays &&
					in.SourceProjectName == step.Spec.SourceProjectName &&
					in.SourceServiceName == step.Spec.SourceServiceName &&
					in.DestinationProjectName == step.Spec.DestinationProjectName &&
					in.DestinationServiceName == step.Spec.DestinationServiceName
			})).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-create",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "step-create", got.Status.ID)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Observes existing upgrade pipeline step by status ID", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Status.ID = "step-observe"
		validatedAt := time.Date(2026, 5, 19, 10, 30, 0, 0, time.UTC)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepGet(mock.Anything, step.Spec.OrganizationID, step.Status.ID).
			Return(&upgradepipeline.UpgradePipelineStepGetOut{
				AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				LastValidation: upgradepipeline.LastValidationOut{
					Comment:         "validated",
					ValidatedAt:     validatedAt,
					ValidatedByUser: "alice@example.com",
				},
				SourceProjectName: step.Spec.SourceProjectName,
				SourceServiceName: step.Spec.SourceServiceName,
				StepId:            "step-observe",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-observe", got.Status.ID)
		require.NotNil(t, got.Status.LastValidation)
		require.Equal(t, "validated", got.Status.LastValidation.Comment)
		require.Equal(t, "alice@example.com", got.Status.LastValidation.ValidatedByUser)
		require.NotNil(t, got.Status.LastValidation.ValidatedAt)
		require.True(t, got.Status.LastValidation.ValidatedAt.Time.Equal(validatedAt))
	})

	t.Run("Adopts existing upgrade pipeline step without creating", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-adopt",
					},
				},
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-adopt", got.Status.ID)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Adopts existing upgrade pipeline step and updates delay", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Spec.AutoValidationDelayDays = 3

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: 7,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-adopt-update",
					},
				},
			}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepUpdate(mock.Anything, step.Spec.OrganizationID, "step-adopt-update", mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepUpdateIn) bool {
				return *in.AutoValidationDelayDays == step.Spec.AutoValidationDelayDays
			})).
			Return(&upgradepipeline.UpgradePipelineStepUpdateOut{
				AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-adopt-update",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-adopt-update", got.Status.ID)
	})

	t.Run("Updates auto validation delay", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 2
		step.Status.ID = "step-update"
		step.Spec.AutoValidationDelayDays = 3

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepGet(mock.Anything, step.Spec.OrganizationID, step.Status.ID).
			Return(&upgradepipeline.UpgradePipelineStepGetOut{
				AutoValidationDelayDays: 7,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-update",
			}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepUpdate(mock.Anything, step.Spec.OrganizationID, step.Status.ID, mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepUpdateIn) bool {
				return *in.AutoValidationDelayDays == step.Spec.AutoValidationDelayDays
			})).
			Return(&upgradepipeline.UpgradePipelineStepUpdateOut{
				AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-update",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Recreates upgrade pipeline step when status ID is missing remotely", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 2
		step.Status.ID = "step-missing"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepGet(mock.Anything, step.Spec.OrganizationID, step.Status.ID).
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.Anything).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: step.Spec.AutoValidationDelayDays,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-recreated",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-recreated", got.Status.ID)
	})

	t.Run("Deletes upgrade pipeline step and removes finalizer", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Finalizers = []string{instanceDeletionFinalizer}
		step.Status.ID = "step-delete"
		now := metav1.Now()
		step.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepDelete(mock.Anything, step.Spec.OrganizationID, step.Status.ID).
			Return(nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		err = r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Removes finalizer when remote upgrade pipeline step is already missing", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Finalizers = []string{instanceDeletionFinalizer}
		step.Status.ID = "step-delete-missing"
		now := metav1.Now()
		step.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepDelete(mock.Anything, step.Spec.OrganizationID, step.Status.ID).
			Return(newAivenError(404, "not found")).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		err = r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
