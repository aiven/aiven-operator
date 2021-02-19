// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const projectVPCFinalizer = "projectvpc-finalizer.k8s-operator.aiven.io"

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

	// Check if the Project VPC instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isProjectVPCMarkedToBeDeleted := projectVPC.GetDeletionTimestamp() != nil
	if isProjectVPCMarkedToBeDeleted {
		if contains(projectVPC.GetFinalizers(), projectVPCFinalizer) {
			vpc, err := r.getProjectVPC(projectVPC)
			if err != nil {
				return reconcile.Result{}, err
			}

			if vpc == nil {
				// Remove projectVPCFinalizer. Once all finalizers have been
				// removed, the object will be deleted.
				controllerutil.RemoveFinalizer(projectVPC, projectVPCFinalizer)
				err := r.Client.Update(ctx, projectVPC)
				if err != nil {
					return reconcile.Result{}, err
				}

				return reconcile.Result{}, nil
			}

			if vpc.State != "DELETING" && vpc.State != "DELETED" {
				// Run finalization logic for projectVPCFinalizer. If the
				// finalization logic fails, don't remove the finalizer so
				// that we can retry during the next reconciliation.
				if err := r.finalizeProjectVPC(log, projectVPC); err != nil {
					log.Error(err, "unable to finalize Project VPC")
					return reconcile.Result{
						Requeue:      true,
						RequeueAfter: 5 * time.Second,
					}, nil
				}
			}

			if vpc.State == "DELETED" {
				// Remove projectVPCFinalizer. Once all finalizers have been
				// removed, the object will be deleted.
				controllerutil.RemoveFinalizer(projectVPC, projectVPCFinalizer)
				err := r.Client.Update(ctx, projectVPC)
				if err != nil {
					return reconcile.Result{}, err
				}
			}

			log.Info("Got " + vpc.State + " state while waiting for VPC connection to be DELETED")
			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
			}, nil

		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(projectVPC.GetFinalizers(), projectVPCFinalizer) {
		if err := r.addFinalizer(log, projectVPC); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if project VPC  already exists on the Aiven side, create a
	// new one if it is not found
	vpc, err := r.getProjectVPC(projectVPC)
	if err != nil {
		return ctrl.Result{}, err
	}

	if vpc == nil {
		_, err = r.createProjectVPC(projectVPC)
		if err != nil {
			log.Error(err, "Failed to create Project VPC")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
			}, err
		}
	} else {
		// Check project VPC status and wait until it is ACTIVE
		if vpc.State != "ACTIVE" {
			log.Info("Project VPC state is " + vpc.State + ", waiting to become ACTIVE")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 5 * time.Second,
			}, nil
		}
	}

	return ctrl.Result{}, nil
}

// getProjectVPC retrieves a project VPC from the list
func (r *ProjectVPCReconciler) getProjectVPC(projectVPC *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpcs, err := r.AivenClient.VPCs.List(projectVPC.Spec.Project)
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

// createProjectVPC creates a project VPC on Aiven side
func (r *ProjectVPCReconciler) createProjectVPC(projectVPC *k8soperatorv1alpha1.ProjectVPC) (*aiven.VPC, error) {
	vpc, err := r.AivenClient.VPCs.Create(projectVPC.Spec.Project, aiven.CreateVPCRequest{
		CloudName:   projectVPC.Spec.CloudName,
		NetworkCIDR: projectVPC.Spec.NetworkCidr,
	})
	if err != nil {
		return nil, err
	}

	err = r.updateCRStatus(projectVPC, vpc)
	if err != nil {
		return nil, err
	}

	return vpc, err
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ProjectVPCReconciler) updateCRStatus(projectVPC *k8soperatorv1alpha1.ProjectVPC, vpc *aiven.VPC) error {
	projectVPC.Status.Project = projectVPC.Spec.Project
	projectVPC.Status.CloudName = vpc.CloudName
	projectVPC.Status.Id = vpc.ProjectVPCID
	projectVPC.Status.State = vpc.State
	projectVPC.Status.NetworkCidr = vpc.NetworkCIDR

	return r.Status().Update(context.Background(), projectVPC)
}

func (r *ProjectVPCReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ProjectVPC{}).
		Complete(r)
}

// finalizeProjectVPC deletes Aiven project VPC
func (r *ProjectVPCReconciler) finalizeProjectVPC(log logr.Logger, p *k8soperatorv1alpha1.ProjectVPC) error {
	// Delete project VPC on Aiven side
	if err := r.AivenClient.VPCs.Delete(p.Status.Project, p.Status.Id); err != nil {
		// If project not found then there is nothing to delete
		if aiven.IsNotFound(err) {
			return nil
		}
		return err
	}

	log.Info("Successfully finalized project VPC")
	return nil
}

// addFinalizer add finalizer to CR
func (r *ProjectVPCReconciler) addFinalizer(reqLogger logr.Logger, projectVPC *k8soperatorv1alpha1.ProjectVPC) error {
	reqLogger.Info("Adding Finalizer for the Project VPC")
	controllerutil.AddFinalizer(projectVPC, projectVPCFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), projectVPC)
	if err != nil {
		reqLogger.Error(err, "Failed to update Project VPC with finalizer")
		return err
	}
	return nil
}
