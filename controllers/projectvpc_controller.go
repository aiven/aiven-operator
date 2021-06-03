// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
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

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("projectvpc", req.NamespacedName)
	log.Info("Reconciling Aiven ProjectVPC")

	const finalizer = "projectvpc-finalizer.k8s-operator.aiven.io"
	vpc := &k8soperatorv1alpha1.ProjectVPC{}
	return r.reconcileInstance(&ProjectVPCHandler{}, ctx, log, req, vpc, finalizer)
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h ProjectVPCHandler) create(_ logr.Logger, i client.Object) (createdObj client.Object, error error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	vpc, err := aivenClient.VPCs.Create(projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return nil, err
	}

	h.setStatus(projectVPC, vpc)

	return projectVPC, nil
}

func (h ProjectVPCHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	vpc, err := h.getVPC(projectVPC)
	if err != nil {
		return nil, false, err
	}

	if vpc == nil {
		log.Info("Successfully finalized project VPC")
		return nil, true, nil
	}

	if vpc.State != "DELETING" && vpc.State != "DELETED" {
		// Delete project VPC on Aiven side
		if err := aivenClient.VPCs.Delete(projectVPC.Status.Project, projectVPC.Status.Id); err != nil && !aiven.IsNotFound(err) {
			return nil, false, err
		}
	}

	if vpc.State == "DELETED" {
		log.Info("Successfully finalized project VPC")
		return nil, true, nil
	}

	return nil, false, nil
}

func (h ProjectVPCHandler) exists(_ logr.Logger, i client.Object) (exists bool, error error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(projectVPC)
	if err != nil {
		return false, err
	}

	return vpc != nil, nil
}

func (h ProjectVPCHandler) getVPC(projectVPC *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpcs, err := aivenClient.VPCs.List(projectVPC.Spec.Project)
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

func (h ProjectVPCHandler) update(logr.Logger, client.Object) (client.Object, error) {
	return nil, nil
}

func (h ProjectVPCHandler) getSecret(logr.Logger, client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h ProjectVPCHandler) checkPreconditions(logr.Logger, client.Object) bool {
	return true
}

func (h ProjectVPCHandler) isActive(_ logr.Logger, i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(projectVPC)
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
	projectVPC.Status.Id = vpc.ProjectVPCID
	projectVPC.Status.State = vpc.State
	projectVPC.Status.NetworkCidr = vpc.NetworkCIDR
}
