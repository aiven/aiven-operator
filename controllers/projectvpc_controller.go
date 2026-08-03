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

		for _, v := range vpcs {
			// cloudName + networkCidr uniquely identify a VPC within a project, and both are
			// immutable on the spec, so a match is unambiguously our resource.
			if v.CloudName == projectVPC.Spec.CloudName && v.NetworkCidr == projectVPC.Spec.NetworkCidr {
				projectVPC.Status.ID = v.ProjectVpcId
				return r.observeState(projectVPC, v.State), nil
			}
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

	switch avnVpc.State {
	case vpc.VpcStateTypeDeleting, vpc.VpcStateTypeDeleted:
		// Deletion already confirmed on Aiven side; remove the finalizer.
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
