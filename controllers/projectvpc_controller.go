// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// ProjectVPCReconciler reconciles a ProjectVPC object
type ProjectVPCReconciler struct {
	Controller
}

type ProjectVPCHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ProjectVPCHandler{}, &v1alpha1.ProjectVPC{})
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h ProjectVPCHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	projectVPC, err := h.convert(i)
	if err != nil {
		return err
	}

	vpc, err := avn.VPCs.Create(projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return err
	}

	projectVPC.Status.ID = vpc.ProjectVPCID

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getInitializedCondition("Created",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(projectVPC.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ProjectVPCHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(avn, projectVPC)
	if err != nil {
		return false, err
	}

	if vpc == nil {
		return true, nil
	}

	if vpc.State != "DELETING" && vpc.State != "DELETED" {
		// Delete project VPC on Aiven side
		if err := avn.VPCs.Delete(projectVPC.Spec.Project, projectVPC.Status.ID); err != nil && !aiven.IsNotFound(err) {
			return false, err
		}
	}

	if vpc.State == "DELETED" {
		return true, nil
	}

	return false, nil
}

func (h ProjectVPCHandler) getVPC(avn *aiven.Client, projectVPC *v1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpcs, err := avn.VPCs.List(projectVPC.Spec.Project)
	if err != nil {
		return nil, err
	}

	for _, vpc := range vpcs {
		if vpc.CloudName == projectVPC.Spec.CloudName {
			return vpc, nil
		}
	}

	return nil, nil
}

func (h ProjectVPCHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	vpc, err := h.getVPC(avn, projectVPC)
	if err != nil {
		return nil, err
	}

	if vpc.State == "ACTIVE" {
		meta.SetStatusCondition(&projectVPC.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return nil, nil
}

func (h ProjectVPCHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}

func (h *ProjectVPCHandler) convert(i client.Object) (*v1alpha1.ProjectVPC, error) {
	vpc, ok := i.(*v1alpha1.ProjectVPC)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ProjectVPC")
	}

	return vpc, nil
}
