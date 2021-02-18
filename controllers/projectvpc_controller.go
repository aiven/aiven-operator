// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"github.com/aiven/aiven-go-client"
	"k8s.io/apimachinery/pkg/api/errors"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ProjectVPCReconciler reconciles a ProjectVPC object
type ProjectVPCReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projectvpcs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projectvpcs/status,verbs=get;update;patch

func (r *ProjectVPCReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("projectvpc", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Project VPC instance
	projectVPC := &k8soperatorv1alpha1.ProjectVPC{}
	err := r.Get(ctx, req.NamespacedName, projectVPC)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Project VPC resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Project VPC")
		return ctrl.Result{}, err
	}

	_, err = r.createProjectVPC(projectVPC)
	if err != nil {
		log.Error(err, "Failed to create Project VPC")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createProjectVPC creates a project VPC on Aiven side
func (r *ProjectVPCReconciler) createProjectVPC(vpc *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	p, err := r.AivenClient.VPCs.Create(vpc.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   vpc.Spec.CloudName,
		NetworkCIDR: vpc.Spec.NetworkCidr,
	})
	if err != nil {
		return nil, err
	}

	return p, err
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ProjectVPC{}).
		Complete(r)
}
