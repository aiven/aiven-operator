// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// ProjectVPCReconciler reconciles a ProjectVPC object
type ProjectVPCReconciler struct {
	Controller
}

type ProjectVPCHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	vpc := &k8soperatorv1alpha1.ProjectVPC{}
	err := r.Get(ctx, req.NamespacedName, vpc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, vpc.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &ProjectVPCHandler{
		client: c,
	}, vpc)
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h ProjectVPCHandler) createOrUpdate(i client.Object) (client.Object, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	vpc, err := h.client.VPCs.Create(projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return nil, err
	}

	projectVPC.Status.ID = vpc.ProjectVPCID

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getInitializedCondition("CreatedOrUpdate",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "CreatedOrUpdate",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta,
		processedGeneration, strconv.FormatInt(projectVPC.GetGeneration(), 10))

	return projectVPC, nil
}

func (h ProjectVPCHandler) delete(i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := h.getVPC(projectVPC)
	if err != nil {
		return false, err
	}

	if vpc == nil {
		return true, nil
	}

	if vpc.State != "DELETING" && vpc.State != "DELETED" {
		// Delete project VPC on Aiven side
		if err := h.client.VPCs.Delete(projectVPC.Spec.Project, projectVPC.Status.ID); err != nil && !aiven.IsNotFound(err) {
			return false, err
		}
	}

	if vpc.State == "DELETED" {
		return true, nil
	}

	return false, nil
}

func (h ProjectVPCHandler) getVPC(projectVPC *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpcs, err := h.client.VPCs.List(projectVPC.Spec.Project)
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

func (h ProjectVPCHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	vpc, err := h.getVPC(projectVPC)
	if err != nil {
		return nil, nil, err
	}

	if vpc.State == "ACTIVE" {
		meta.SetStatusCondition(&projectVPC.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "Get",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta, isRunning, "1")
	}

	return projectVPC, nil, nil
}

func (h ProjectVPCHandler) checkPreconditions(client.Object) bool {
	return true
}

func (h *ProjectVPCHandler) convert(i client.Object) (*k8soperatorv1alpha1.ProjectVPC, error) {
	vpc, ok := i.(*k8soperatorv1alpha1.ProjectVPC)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ProjectVPC")
	}

	return vpc, nil
}
