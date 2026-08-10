// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/vpc"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var isDependencyError = v1alpha1.ErrorSubstrChecker(
	"VPC cannot be deleted while there are services in it",
	"VPC cannot be deleted while there are services migrating from it",
)

//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/finalizers,verbs=get;create;update

// ProjectVPCController reconciles a ProjectVPC object.
type ProjectVPCController struct {
	avnGen avngen.Client
}

func newProjectVPCReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(_ Controller, avnGen avngen.Client) AivenController[*v1alpha1.ProjectVPC] {
			return &ProjectVPCController{avnGen: avnGen}
		},
		nil,
	)
}

func (r *ProjectVPCController) Observe(ctx context.Context, projectVPC *v1alpha1.ProjectVPC) (Observation, error) {
	// An empty status.ID means we have not recorded a VPC yet. Before creating one, adopt any
	// existing VPC that already matches the spec.
	if projectVPC.Status.ID == "" {
		vpcs, err := r.avnGen.VpcList(ctx, projectVPC.Spec.Project)
		if err != nil {
			return Observation{}, fmt.Errorf("cannot list project VPCs: %w", err)
		}

		var pending *vpc.VpcOut
		for i, v := range vpcs {
			if !vpcMatchesSpec(v, projectVPC.Spec) {
				continue
			}

			// A VPC still being torn down can be neither adopted nor replaced.
			if isVPCGone(v.State) {
				pending = &vpcs[i]
				continue
			}

			projectVPC.Status.ID = v.ProjectVpcId
			return r.observeState(projectVPC, v.State), nil
		}

		if pending != nil {
			return Observation{}, fmt.Errorf("%w: vpc %s with the same cloudName and networkCidr is %s, waiting for it to be removed before creating a replacement",
				errPreconditionNotMet, pending.ProjectVpcId, pending.State)
		}

		// No matching VPC exists yet; let the reconciler create one.
		return Observation{ResourceExists: false}, nil
	}

	avnVpc, err := r.avnGen.VpcGet(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
	switch {
	case isNotFound(err):
		// The VPC vanished on Aiven side; trigger recreation like other migrated resources.
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("cannot get project VPC: %w", err)
	}

	return r.observeState(projectVPC, avnVpc.State), nil
}

// vpcMatchesSpec reports whether the remote VPC is the one this resource describes.
// cloudName and networkCidr uniquely identify a VPC within a project.
func vpcMatchesSpec(v vpc.VpcOut, spec v1alpha1.ProjectVPCSpec) bool {
	return v.CloudName == spec.CloudName && v.NetworkCidr == spec.NetworkCidr
}

// isVPCGone reports whether Aiven has already torn the VPC down.
func isVPCGone(state vpc.VpcStateType) bool {
	return state == vpc.VpcStateTypeDeleting || state == vpc.VpcStateTypeDeleted
}

// observeState records the observed VPC state on the status and reports whether the resource
// is up to date. It marks the instance running only once Aiven reports the VPC ACTIVE.
func (r *ProjectVPCController) observeState(projectVPC *v1alpha1.ProjectVPC, state vpc.VpcStateType) Observation {
	projectVPC.Status.State = state

	// The VPC transitions APPROVED -> ACTIVE asynchronously, so only mark it running
	// once Aiven reports it ACTIVE. An APPROVED VPC stays not-running and keeps requeueing.
	if state == vpc.VpcStateTypeActive {
		markInstanceRunning(projectVPC)
	}

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(projectVPC),
	}
}

func (r *ProjectVPCController) Create(ctx context.Context, projectVPC *v1alpha1.ProjectVPC) (CreateResult, error) {
	delete(projectVPC.GetAnnotations(), instanceIsRunningAnnotation)

	avnVpc, err := r.avnGen.VpcCreate(ctx, projectVPC.Spec.Project, &vpc.VpcCreateIn{
		CloudName:          projectVPC.Spec.CloudName,
		NetworkCidr:        projectVPC.Spec.NetworkCidr,
		PeeringConnections: make([]vpc.PeeringConnectionIn, 0),
	})
	if err != nil {
		return CreateResult{}, fmt.Errorf("cannot create project VPC on Aiven side: %w", err)
	}

	projectVPC.Status.ID = avnVpc.ProjectVpcId

	const reason = "Created"
	meta.SetStatusCondition(&projectVPC.Status.Conditions, getInitializedCondition(reason, "Successfully created the instance in Aiven"))
	meta.SetStatusCondition(&projectVPC.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *ProjectVPCController) Update(_ context.Context, _ *v1alpha1.ProjectVPC) (UpdateResult, error) {
	// ProjectVPC spec is fully immutable, so this is a no-op.
	return UpdateResult{}, nil
}

func (r *ProjectVPCController) Delete(ctx context.Context, projectVPC *v1alpha1.ProjectVPC) error {
	// Nothing was ever created on Aiven side.
	if projectVPC.Status.ID == "" {
		return nil
	}

	avnVpc, err := r.avnGen.VpcGet(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// Deletion already confirmed on Aiven side; remove the finalizer.
	if isVPCGone(avnVpc.State) {
		return nil
	}

	services, err := r.avnGen.ServiceList(ctx, projectVPC.Spec.Project)
	if err != nil {
		return err
	}

	for _, s := range services {
		if s.ProjectVpcId == projectVPC.Status.ID {
			return fmt.Errorf("%w: vpc has dependent service %q in state %q", v1alpha1.ErrDeleteDependencies, s.ServiceName, s.State)
		}
	}

	_, err = r.avnGen.VpcDelete(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
	if isDependencyError(err) {
		return fmt.Errorf("%w: %w", v1alpha1.ErrDeleteDependencies, err)
	}
	if err != nil {
		return err
	}

	// Delete was accepted; wait for the state to flip to DELETING/DELETED before the finalizer is removed.
	return fmt.Errorf("%w: vpc deletion accepted, waiting for DELETING state", errDeletionInProgress)
}
