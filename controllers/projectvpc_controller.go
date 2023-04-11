// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var isDependencyError = v1alpha1.ErrorSubstrChecker(
	"VPC cannot be deleted while there are services in it",
	"VPC cannot be deleted while there are services migrating from it",
)

// ProjectVPCReconciler reconciles a ProjectVPC object
type ProjectVPCReconciler struct {
	Controller
}

type ProjectVPCHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &ProjectVPCHandler{}, &v1alpha1.ProjectVPC{})
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h *ProjectVPCHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
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

func (h *ProjectVPCHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return false, err
	}

	vpc, err := avn.VPCs.Get(projectVPC.Spec.Project, projectVPC.Status.ID)
	if aiven.IsNotFound(err) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	switch vpc.State {
	case "DELETING", "DELETED":
		return true, nil
	}

	err = avn.VPCs.Delete(projectVPC.Spec.Project, projectVPC.Status.ID)
	if isDependencyError(err) {
		return false, fmt.Errorf("%w: %s", v1alpha1.ErrDeleteDependencies, err)
	}

	return false, nil
}

func (h *ProjectVPCHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	projectVPC, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	vpc, err := avn.VPCs.Get(projectVPC.Spec.Project, projectVPC.Status.ID)
	if err != nil {
		return nil, err
	}

	projectVPC.Status.State = vpc.State
	if vpc.State == "ACTIVE" {
		meta.SetStatusCondition(&projectVPC.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return nil, nil
}

func (h *ProjectVPCHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}

func (h *ProjectVPCHandler) convert(i client.Object) (*v1alpha1.ProjectVPC, error) {
	vpc, ok := i.(*v1alpha1.ProjectVPC)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ProjectVPC")
	}

	return vpc, nil
}
