package controllers

import (
	"slices"
	"testing"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/upgradepipeline"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
	listQuery := func(step *v1alpha1.UpgradePipelineStep) any {
		return mock.MatchedBy(func(query [][2]string) bool {
			return slices.Equal(query, [][2]string{
				upgradepipeline.UpgradePipelineStepListLimit(upgradePipelineStepLookupLimit),
				upgradepipeline.UpgradePipelineStepListSourceProjectName(step.Spec.SourceProjectName),
				upgradepipeline.UpgradePipelineStepListSourceServiceName(step.Spec.SourceServiceName),
				upgradepipeline.UpgradePipelineStepListDestinationProjectName(step.Spec.DestinationProjectName),
				upgradepipeline.UpgradePipelineStepListDestinationServiceName(step.Spec.DestinationServiceName),
			})
		})
	}

	t.Run("Creates upgrade pipeline step on Aiven", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepCreateIn) bool {
				return in.AutoValidationDelayDays != nil &&
					*in.AutoValidationDelayDays == *step.Spec.AutoValidationDelayDays &&
					in.SourceProjectName == step.Spec.SourceProjectName &&
					in.SourceServiceName == step.Spec.SourceServiceName &&
					in.DestinationProjectName == step.Spec.DestinationProjectName &&
					in.DestinationServiceName == step.Spec.DestinationServiceName
			})).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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
		require.Nil(t, got.Status.LastValidation)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Creates upgrade pipeline step without auto validation delay", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Spec.AutoValidationDelayDays = nil

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepCreateIn) bool {
				return in.AutoValidationDelayDays == nil
			})).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: 7,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-create-default-delay",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-create-default-delay", got.Status.ID)
		require.Nil(t, got.Spec.AutoValidationDelayDays)
	})

	t.Run("Creates upgrade pipeline step with zero auto validation delay", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		autoValidationDelayDays := 0
		step.Spec.AutoValidationDelayDays = &autoValidationDelayDays

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepCreateIn) bool {
				return in.AutoValidationDelayDays != nil && *in.AutoValidationDelayDays == 0
			})).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: 0,
				DestinationProjectName:  step.Spec.DestinationProjectName,
				DestinationServiceName:  step.Spec.DestinationServiceName,
				SourceProjectName:       step.Spec.SourceProjectName,
				SourceServiceName:       step.Spec.SourceServiceName,
				StepId:                  "step-create-zero-delay",
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-create-zero-delay", got.Status.ID)
		require.NotNil(t, got.Spec.AutoValidationDelayDays)
		require.Equal(t, 0, *got.Spec.AutoValidationDelayDays)
	})

	t.Run("Requeues when creating upgrade pipeline step returns not found", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Empty(t, got.Status.ID)
		require.NotContains(t, got.Annotations, processedGenerationAnnotation)
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, string(ConditionTypeError)))
	})

	t.Run("Clears stale readiness when remote step is missing and create requeues", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Status.ID = "stale-step"
		metav1.SetMetaDataAnnotation(&step.ObjectMeta, processedGenerationAnnotation, "1")
		metav1.SetMetaDataAnnotation(&step.ObjectMeta, instanceIsRunningAnnotation, "true")

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.Anything).
			Return(nil, newAivenError(404, "service not found")).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.False(t, IsReadyToUse(got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("Observes existing upgrade pipeline step by spec and refreshes stale status ID", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Status.ID = "stale-step"
		validatedAt := time.Date(2026, 5, 19, 10, 30, 0, 0, time.UTC)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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
					},
				},
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
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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

	t.Run("Doesn't update auto validation delay when omitted", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		step.Spec.AutoValidationDelayDays = nil

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: 3,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-observe-default-delay",
					},
				},
			}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.Equal(t, "step-observe-default-delay", got.Status.ID)
		require.Nil(t, got.Spec.AutoValidationDelayDays)
	})

	t.Run("Fails observe when filtered list response has next page", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		nextPage := "next-page"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Next: &nextPage,
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-page-1",
					},
				},
			}, nil).Once()

		_, res, err := runScenario(t, step, avn)
		require.ErrorContains(t, err, "found multiple upgrade pipeline steps matching")
		require.Equal(t, ctrlruntime.Result{}, res)
	})

	t.Run("Fails observe when filtered list response has multiple total count", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		totalCount := 2

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-total-1",
					},
				},
				TotalCount: &totalCount,
			}, nil).Once()

		_, res, err := runScenario(t, step, avn)
		require.ErrorContains(t, err, "found multiple upgrade pipeline steps matching")
		require.Equal(t, ctrlruntime.Result{}, res)
	})

	t.Run("Fails observe when filtered list response has multiple matches", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-duplicate-1",
					},
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-duplicate-2",
					},
				},
			}, nil).Once()

		_, res, err := runScenario(t, step, avn)
		require.ErrorContains(t, err, "found multiple upgrade pipeline steps matching")
		require.Equal(t, ctrlruntime.Result{}, res)
	})

	t.Run("Fails observe when filtered list response doesn't match requested filters", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  "unexpected-service",
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-mismatch",
					},
				},
			}, nil).Once()

		_, res, err := runScenario(t, step, avn)
		require.ErrorContains(t, err, "does not match requested filters")
		require.Equal(t, ctrlruntime.Result{}, res)
	})

	t.Run("Adopts existing upgrade pipeline step and updates delay", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		autoValidationDelayDays := 3
		step.Spec.AutoValidationDelayDays = &autoValidationDelayDays

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
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
				return in.AutoValidationDelayDays != nil && *in.AutoValidationDelayDays == *step.Spec.AutoValidationDelayDays
			})).
			Return(&upgradepipeline.UpgradePipelineStepUpdateOut{
				AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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
		step.Status.ID = "stale-step"
		autoValidationDelayDays := 3
		step.Spec.AutoValidationDelayDays = &autoValidationDelayDays

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: 7,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-update",
					},
				},
			}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepUpdate(mock.Anything, step.Spec.OrganizationID, "step-update", mock.MatchedBy(func(in *upgradepipeline.UpgradePipelineStepUpdateIn) bool {
				return in.AutoValidationDelayDays != nil && *in.AutoValidationDelayDays == *step.Spec.AutoValidationDelayDays
			})).
			Return(&upgradepipeline.UpgradePipelineStepUpdateOut{
				AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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

	t.Run("Clears stale readiness when drifted step update requeues", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 1
		autoValidationDelayDays := 3
		step.Spec.AutoValidationDelayDays = &autoValidationDelayDays
		step.Status.ID = "stale-step"
		metav1.SetMetaDataAnnotation(&step.ObjectMeta, processedGenerationAnnotation, "1")
		metav1.SetMetaDataAnnotation(&step.ObjectMeta, instanceIsRunningAnnotation, "true")

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: 7,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-update",
					},
				},
			}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepUpdate(mock.Anything, step.Spec.OrganizationID, "step-update", mock.Anything).
			Return(nil, newAivenError(404, "step not found")).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got))
		require.False(t, IsReadyToUse(got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionFalse, condition.Status)
	})

	t.Run("Creates upgrade pipeline step when stale status ID isn't found by spec", func(t *testing.T) {
		step := newObjectFromYAML[v1alpha1.UpgradePipelineStep](t, yamlUpgradePipelineStep)
		step.Generation = 2
		step.Status.ID = "step-missing"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepCreate(mock.Anything, step.Spec.OrganizationID, mock.Anything).
			Return(&upgradepipeline.UpgradePipelineStepCreateOut{
				AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
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
		step.Status.ID = "stale-step"
		now := metav1.Now()
		step.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{
				Steps: []upgradepipeline.StepOut{
					{
						AutoValidationDelayDays: *step.Spec.AutoValidationDelayDays,
						DestinationProjectName:  step.Spec.DestinationProjectName,
						DestinationServiceName:  step.Spec.DestinationServiceName,
						SourceProjectName:       step.Spec.SourceProjectName,
						SourceServiceName:       step.Spec.SourceServiceName,
						StepId:                  "step-delete",
					},
				},
			}, nil).Once()
		avn.EXPECT().
			UpgradePipelineStepDelete(mock.Anything, step.Spec.OrganizationID, "step-delete").
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
			UpgradePipelineStepList(mock.Anything, step.Spec.OrganizationID, listQuery(step)).
			Return(&upgradepipeline.UpgradePipelineStepListOut{}, nil).Once()

		r, res, err := runScenario(t, step, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.UpgradePipelineStep{}
		err = r.Get(t.Context(), types.NamespacedName{Name: step.Name, Namespace: step.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})
}
