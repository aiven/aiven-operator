// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/upgradepipeline"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newUpgradePipelineStepReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.UpgradePipelineStep] {
			return &UpgradePipelineStepController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
}

//+kubebuilder:rbac:groups=aiven.io,resources=upgradepipelinesteps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=upgradepipelinesteps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=upgradepipelinesteps/finalizers,verbs=get;create;update

// UpgradePipelineStepController reconciles an UpgradePipelineStep object.
type UpgradePipelineStepController struct {
	client.Client
	avnGen avngen.Client
}

func (r *UpgradePipelineStepController) Observe(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (Observation, error) {
	if cr.Status.ID == "" {
		step, err := r.lookupStep(ctx, cr)
		if err != nil {
			return Observation{}, err
		}
		if step == nil {
			return Observation{ResourceExists: false}, nil
		}

		r.applyStatus(cr, *step)

		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: !r.hasDrift(cr, *step),
		}, nil
	}

	step, err := r.avnGen.UpgradePipelineStepGet(
		ctx,
		cr.Spec.OrganizationID,
		cr.Status.ID,
	)
	if err != nil {
		if isNotFound(err) {
			cr.Status.ID = ""
			cr.Status.LastValidation = nil
			return Observation{ResourceExists: false}, nil
		}

		return Observation{}, fmt.Errorf("getting upgrade pipeline step: %w", err)
	}

	r.applyStatus(cr, upgradepipeline.StepOut(*step))

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: !r.hasDrift(cr, upgradepipeline.StepOut(*step)),
	}, nil
}

func (r *UpgradePipelineStepController) Create(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (CreateResult, error) {
	step, err := r.avnGen.UpgradePipelineStepCreate(
		ctx,
		cr.Spec.OrganizationID,
		&upgradepipeline.UpgradePipelineStepCreateIn{
			AutoValidationDelayDays: &cr.Spec.AutoValidationDelayDays,
			DestinationProjectName:  cr.Spec.DestinationProjectName,
			DestinationServiceName:  cr.Spec.DestinationServiceName,
			SourceProjectName:       cr.Spec.SourceProjectName,
			SourceServiceName:       cr.Spec.SourceServiceName,
		},
	)
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating upgrade pipeline step: %w", err)
	}

	r.applyStatus(cr, upgradepipeline.StepOut(*step))

	return CreateResult{}, nil
}

func (r *UpgradePipelineStepController) Update(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (UpdateResult, error) {
	step, err := r.avnGen.UpgradePipelineStepUpdate(
		ctx,
		cr.Spec.OrganizationID,
		cr.Status.ID,
		&upgradepipeline.UpgradePipelineStepUpdateIn{
			AutoValidationDelayDays: &cr.Spec.AutoValidationDelayDays,
		},
	)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("updating upgrade pipeline step: %w", err)
	}

	r.applyStatus(cr, upgradepipeline.StepOut(*step))

	return UpdateResult{}, nil
}

func (r *UpgradePipelineStepController) Delete(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) error {
	if cr.Status.ID == "" {
		return nil
	}

	if err := r.avnGen.UpgradePipelineStepDelete(ctx, cr.Spec.OrganizationID, cr.Status.ID); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting upgrade pipeline step: %w", err)
	}

	return nil
}

func (r *UpgradePipelineStepController) lookupStep(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (*upgradepipeline.StepOut, error) {
	out, err := r.avnGen.UpgradePipelineStepList(ctx, cr.Spec.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("listing upgrade pipeline steps: %w", err)
	}

	var existing *upgradepipeline.StepOut
	for i := range out.Steps {
		step := &out.Steps[i]
		if step.SourceProjectName != cr.Spec.SourceProjectName ||
			step.SourceServiceName != cr.Spec.SourceServiceName ||
			step.DestinationProjectName != cr.Spec.DestinationProjectName ||
			step.DestinationServiceName != cr.Spec.DestinationServiceName {
			continue
		}
		if existing != nil {
			return nil, fmt.Errorf("found multiple upgrade pipeline steps matching %s/%s -> %s/%s",
				cr.Spec.SourceProjectName,
				cr.Spec.SourceServiceName,
				cr.Spec.DestinationProjectName,
				cr.Spec.DestinationServiceName,
			)
		}

		existing = step
	}

	return existing, nil
}

func (*UpgradePipelineStepController) applyStatus(cr *v1alpha1.UpgradePipelineStep, step upgradepipeline.StepOut) {
	cr.Status.ID = step.StepId
	cr.Status.LastValidation = &v1alpha1.UpgradePipelineStepLastValidationStatus{
		Comment:         step.LastValidation.Comment,
		ValidatedAt:     NilIfZero(metav1.NewTime(step.LastValidation.ValidatedAt)),
		ValidatedByUser: step.LastValidation.ValidatedByUser,
	}
	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&cr.ObjectMeta, instanceIsRunningAnnotation, "true")
}

func (*UpgradePipelineStepController) hasDrift(cr *v1alpha1.UpgradePipelineStep, step upgradepipeline.StepOut) bool {
	return cr.Spec.AutoValidationDelayDays != step.AutoValidationDelayDays
}
