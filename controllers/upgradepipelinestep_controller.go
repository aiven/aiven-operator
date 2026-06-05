// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/upgradepipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const upgradePipelineStepLookupLimit = 30

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
	step, err := r.lookupStep(ctx, cr)
	if err != nil {
		return Observation{}, err
	}
	if step == nil {
		cr.Status.ID = ""
		cr.Status.LastValidation = nil
		markInstanceNotReconciled(cr)
		return Observation{ResourceExists: false}, nil
	}

	r.applyStatus(cr, *step)
	if r.hasDrift(cr, *step) {
		markInstanceNotReconciled(cr)
		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}
	markInstanceRunning(cr)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (r *UpgradePipelineStepController) Create(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (CreateResult, error) {
	step, err := r.avnGen.UpgradePipelineStepCreate(
		ctx,
		cr.Spec.OrganizationID,
		&upgradepipeline.UpgradePipelineStepCreateIn{
			AutoValidationDelayDays: cr.Spec.AutoValidationDelayDays,
			DestinationProjectName:  cr.Spec.DestinationProjectName,
			DestinationServiceName:  cr.Spec.DestinationServiceName,
			SourceProjectName:       cr.Spec.SourceProjectName,
			SourceServiceName:       cr.Spec.SourceServiceName,
		},
	)
	if err != nil {
		if isNotFound(err) {
			return CreateResult{}, fmt.Errorf("%w: creating upgrade pipeline step: %w", errPreconditionNotMet, err)
		}
		return CreateResult{}, fmt.Errorf("creating upgrade pipeline step: %w", err)
	}

	r.applyStatus(cr, upgradepipeline.StepOut(*step))
	markInstanceRunning(cr)

	return CreateResult{}, nil
}

func (r *UpgradePipelineStepController) Update(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (UpdateResult, error) {
	step, err := r.avnGen.UpgradePipelineStepUpdate(
		ctx,
		cr.Spec.OrganizationID,
		cr.Status.ID,
		&upgradepipeline.UpgradePipelineStepUpdateIn{
			AutoValidationDelayDays: cr.Spec.AutoValidationDelayDays,
		},
	)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("updating upgrade pipeline step: %w", err)
	}

	r.applyStatus(cr, upgradepipeline.StepOut(*step))
	markInstanceRunning(cr)

	return UpdateResult{}, nil
}

func (r *UpgradePipelineStepController) Delete(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) error {
	step, err := r.lookupStep(ctx, cr)
	if err != nil {
		return err
	}
	if step == nil {
		return nil
	}

	if err := r.avnGen.UpgradePipelineStepDelete(ctx, cr.Spec.OrganizationID, step.StepId); err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting upgrade pipeline step: %w", err)
	}

	return nil
}

func (r *UpgradePipelineStepController) lookupStep(ctx context.Context, cr *v1alpha1.UpgradePipelineStep) (*upgradepipeline.StepOut, error) {
	out, err := r.avnGen.UpgradePipelineStepList(ctx, cr.Spec.OrganizationID,
		upgradepipeline.UpgradePipelineStepListLimit(upgradePipelineStepLookupLimit),
		upgradepipeline.UpgradePipelineStepListSourceProjectName(cr.Spec.SourceProjectName),
		upgradepipeline.UpgradePipelineStepListSourceServiceName(cr.Spec.SourceServiceName),
		upgradepipeline.UpgradePipelineStepListDestinationProjectName(cr.Spec.DestinationProjectName),
		upgradepipeline.UpgradePipelineStepListDestinationServiceName(cr.Spec.DestinationServiceName),
	)
	if err != nil {
		return nil, fmt.Errorf("listing upgrade pipeline steps: %w", err)
	}

	// Check for the ambiguous results. We expect not more than one step to match the filters, but if there are more, something is wrong and we shouldn't proceed.
	if (out.TotalCount != nil && *out.TotalCount > 1) || (out.Next != nil && *out.Next != "") || len(out.Steps) > 1 {
		return nil, fmt.Errorf("found multiple upgrade pipeline steps matching %s/%s -> %s/%s",
			cr.Spec.SourceProjectName,
			cr.Spec.SourceServiceName,
			cr.Spec.DestinationProjectName,
			cr.Spec.DestinationServiceName,
		)
	}
	if len(out.Steps) == 0 {
		return nil, nil
	}

	step := out.Steps[0]
	if step.SourceProjectName != cr.Spec.SourceProjectName ||
		step.SourceServiceName != cr.Spec.SourceServiceName ||
		step.DestinationProjectName != cr.Spec.DestinationProjectName ||
		step.DestinationServiceName != cr.Spec.DestinationServiceName {
		return nil, fmt.Errorf("upgrade pipeline step list returned a step that does not match requested filters: got %s/%s -> %s/%s, want %s/%s -> %s/%s",
			step.SourceProjectName,
			step.SourceServiceName,
			step.DestinationProjectName,
			step.DestinationServiceName,
			cr.Spec.SourceProjectName,
			cr.Spec.SourceServiceName,
			cr.Spec.DestinationProjectName,
			cr.Spec.DestinationServiceName,
		)
	}

	return &step, nil
}

func (*UpgradePipelineStepController) applyStatus(cr *v1alpha1.UpgradePipelineStep, step upgradepipeline.StepOut) {
	cr.Status.ID = step.StepId
	lastValidation := &v1alpha1.UpgradePipelineStepLastValidationStatus{
		Comment:         step.LastValidation.Comment,
		ValidatedAt:     NilIfZero(metav1.NewTime(step.LastValidation.ValidatedAt)),
		ValidatedByUser: step.LastValidation.ValidatedByUser,
	}
	if lastValidation.Comment == "" && lastValidation.ValidatedAt == nil && lastValidation.ValidatedByUser == "" {
		cr.Status.LastValidation = nil
	} else {
		cr.Status.LastValidation = lastValidation
	}
}

func (*UpgradePipelineStepController) hasDrift(cr *v1alpha1.UpgradePipelineStep, step upgradepipeline.StepOut) bool {
	return cr.Spec.AutoValidationDelayDays != nil && *cr.Spec.AutoValidationDelayDays != step.AutoValidationDelayDays
}
