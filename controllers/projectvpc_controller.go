// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/go-logr/logr"
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

func newProjectVPCReconciler(c Controller) reconcilerType {
	return &ProjectVPCReconciler{Controller: c}
}

type ProjectVPCHandler struct {
	log logr.Logger
}

//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=projectvpcs/finalizers,verbs=get;create;update

func (r *ProjectVPCReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, &ProjectVPCHandler{log: r.Log}, &v1alpha1.ProjectVPC{})
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ProjectVPC{}).
		Complete(r)
}

func (h *ProjectVPCHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, _ avngen.Client, obj client.Object, _ []client.Object) error {
	projectVPC, err := h.convert(obj)
	if err != nil {
		return err
	}

	vpc, err := avn.VPCs.Create(ctx, projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return err
	}

	projectVPC.Status.ID = vpc.ProjectVPCID

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	meta.SetStatusCondition(&projectVPC.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Successfully created or updated the instance in Aiven, status remains unknown"))

	metav1.SetMetaDataAnnotation(&projectVPC.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(projectVPC.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *ProjectVPCHandler) delete(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	projectVPC, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	vpc, err := avnGen.VpcGet(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
	if isNotFound(err) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	switch vpc.State {
	case "DELETING", "DELETED":
		return true, nil
	}

	services, err := avnGen.ServiceList(ctx, projectVPC.Spec.Project)
	if err != nil {
		return false, err
	}

	for _, s := range services {
		if s.ProjectVpcId == projectVPC.Status.ID {
			h.log.Info(fmt.Sprintf("vpc has dependent service %q in status %q", s.ServiceName, s.State))
			return false, nil
		}
	}

	_, err = avnGen.VpcDelete(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
	if isDependencyError(err) {
		return false, fmt.Errorf("%w: %w", v1alpha1.ErrDeleteDependencies, err)
	}

	return false, nil
}

func (h *ProjectVPCHandler) get(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	projectVPC, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	vpc, err := avnGen.VpcGet(ctx, projectVPC.Spec.Project, projectVPC.Status.ID)
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

func (h *ProjectVPCHandler) checkPreconditions(_ context.Context, _ *aiven.Client, _ avngen.Client, _ client.Object) (bool, error) {
	return true, nil
}

func (h *ProjectVPCHandler) convert(i client.Object) (*v1alpha1.ProjectVPC, error) {
	vpc, ok := i.(*v1alpha1.ProjectVPC)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ProjectVPC")
	}

	return vpc, nil
}
