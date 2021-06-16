// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProjectVPCReconciler reconciles a ProjectVPC object
type ProjectVPCReconciler struct {
	Controller
}

type ProjectVPCHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;createOrUpdate;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("projectvpc", req.NamespacedName)
	log.Info("reconciling aiven project vpc")

	const finalizer = "projectvpc-finalizer.aiven.io"
	vpc := &k8soperatorv1alpha1.ProjectVPC{}
	return r.reconcileInstance(ctx, req, &ProjectVPCHandler{}, vpc)
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h ProjectVPCHandler) createOrUpdate(i client.Object) (error error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return err
	}

	vpc, err := c.VPCs.Create(projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return err
	}

	h.setStatus(projectVPC, vpc)

	return nil
}

func (h ProjectVPCHandler) delete(i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(c, projectVPC)
	if err != nil {
		return false, err
	}

	if vpc == nil {
		log.Info("successfully finalized project vpc")
		return true, nil
	}

	if vpc.State != "DELETING" && vpc.State != "DELETED" {
		// Delete project VPC on Aiven side
		if err := c.VPCs.Delete(projectVPC.Status.Project, projectVPC.Status.ID); err != nil && !aiven.IsNotFound(err) {
			return false, err
		}
	}

	if vpc.State == "DELETED" {
		log.Info("successfully finalized project vpc")
		return true, nil
	}

	return false, nil
}

func (h ProjectVPCHandler) exists(c *aiven.Client, _ logr.Logger, i client.Object) (exists bool, error error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(c, projectVPC)
	if err != nil {
		return false, err
	}

	return vpc != nil, nil
}

func (h ProjectVPCHandler) getVPC(c *aiven.Client, projectVPC *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpcs, err := c.VPCs.List(projectVPC.Spec.Project)
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

func (h ProjectVPCHandler) update(*aiven.Client, logr.Logger, client.Object) (client.Object, error) {
	return nil, nil
}

func (h ProjectVPCHandler) get(client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h ProjectVPCHandler) checkPreconditions(client.Object) bool {
	return true
}

func (h ProjectVPCHandler) isActive(c *aiven.Client, _ logr.Logger, i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(c, projectVPC)
	if err != nil {
		return false, err
	}

	return vpc.State == "ACTIVE", nil
}

func (h *ProjectVPCHandler) convert(i client.Object) (*k8soperatorv1alpha1.ProjectVPC, error) {
	vpc, ok := i.(*k8soperatorv1alpha1.ProjectVPC)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ProjectVPC")
	}

	return vpc, nil
}

func (h ProjectVPCHandler) setStatus(projectVPC *k8soperatorv1alpha1.ProjectVPC, vpc *aiven.VPC) {
	projectVPC.Status.Project = projectVPC.Spec.Project
	projectVPC.Status.CloudName = vpc.CloudName
	projectVPC.Status.ID = vpc.ProjectVPCID
	projectVPC.Status.State = vpc.State
	projectVPC.Status.NetworkCidr = vpc.NetworkCIDR
}

func (h ProjectVPCHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	vpc, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &vpc.Spec.AuthSecretRef
}
