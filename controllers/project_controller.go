// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const projectFinalizer = "finalizer.k8s-operator.aiven.io"

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("project", req.NamespacedName)

	// Check if aiven-token secret already exists
	token := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: "aiven-token", Namespace: req.Namespace}, token)
	if err != nil {
		log.Error(err, "aiven-token secret is missing, required by the Aiven client", "Project.Namespace", req.Namespace, "Project.Name", req.Name)
		return reconcile.Result{}, err
	}

	// Fetch the Project instance
	project := &k8soperatorv1alpha1.Project{}
	err = r.Get(ctx, req.NamespacedName, project)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Project resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Get Aiven client
	aivenC, err := r.GetClient(token)
	if err != nil {
		log.Error(err, "Failed to initiate Aiven client", "Project.Namespace", project.Namespace, "Project.Name", project.Name)
		return ctrl.Result{}, err
	}

	// Check if the Project instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isProjectMarkedToBeDeleted := project.GetDeletionTimestamp() != nil
	if isProjectMarkedToBeDeleted {
		if contains(project.GetFinalizers(), projectFinalizer) {
			// Run finalization logic for projectFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeProject(log, project, aivenC); err != nil {
				return reconcile.Result{}, err
			}

			// Remove projectFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(project, projectFinalizer)
			err := r.Client.Update(context.TODO(), project)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(project.GetFinalizers(), projectFinalizer) {
		if err := r.addFinalizer(log, project); err != nil {
			return reconcile.Result{}, err
		}
	}

	var aivenProject *aiven.Project

	// Check if project already exists on the Aiven side, create a
	// new one if project is not found or update existing
	_, err = aivenC.Projects.Get(project.Spec.Name)
	if err != nil {
		// Create a new project if project does not exists
		if aiven.IsNotFound(err) {
			aivenProject, err = r.createProject(project, aivenC)
			if err != nil {
				log.Error(err, "Failed to create Project", "Project.Namespace", project.Namespace, "Project.Name", project.Name)
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	} else {
		// Update project via API and update CR status
		aivenProject, err = r.updateProject(project, aivenC)
		if err != nil {
			log.Error(err, "Failed to update Project", "Project.Namespace", project.Namespace, "Project.Name", project.Name)
			return ctrl.Result{}, err
		}
	}

	// Update project custom resource status
	errS := r.updateCRStatus(project, aivenProject)
	if errS != nil {
		log.Error(errS, "Failed to update Project status", "Project.Namespace", project.Namespace, "Project.Name", project.Name)
		return ctrl.Result{}, errS
	}

	return ctrl.Result{}, nil
}

// createProject creates a project on Aiven side
func (r *ProjectReconciler) createProject(project *k8soperatorv1alpha1.Project, aivenC *aiven.Client) (*aiven.Project, error) {
	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := aivenC.Projects.Create(aiven.CreateProjectRequest{
		BillingAddress:   &project.Spec.BillingAddress,
		BillingEmails:    billingEmails,
		BillingExtraText: &project.Spec.BillingExtraText,
		CardID:           project.Spec.CardId,
		Cloud:            project.Spec.Cloud,
		CopyFromProject:  project.Spec.CopyFromProject,
		CountryCode:      &project.Spec.CountryCode,
		Project:          project.Spec.Name,
		AccountId:        project.Spec.AccountId,
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return nil, err
	}

	return p, err
}

// updateProject updates a project on Aiven side
func (r *ProjectReconciler) updateProject(project *k8soperatorv1alpha1.Project, aivenC *aiven.Client) (*aiven.Project, error) {
	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := aivenC.Projects.Update(project.Spec.Name, aiven.UpdateProjectRequest{
		BillingAddress:   &project.Spec.BillingAddress,
		BillingEmails:    billingEmails,
		BillingExtraText: &project.Spec.BillingExtraText,
		CardID:           project.Spec.CardId,
		Cloud:            project.Spec.Cloud,
		CountryCode:      &project.Spec.CountryCode,
		AccountId:        project.Spec.AccountId,
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return nil, err
	}

	return p, err
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ProjectReconciler) updateCRStatus(project *k8soperatorv1alpha1.Project, p *aiven.Project) error {
	project.Status.Name = p.Name
	project.Status.AccountId = p.AccountId
	project.Status.BillingAddress = p.BillingAddress
	project.Status.BillingEmails = p.GetBillingEmailsAsStringSlice()
	project.Status.TechnicalEmails = p.GetTechnicalEmailsAsStringSlice()
	project.Status.BillingExtraText = p.BillingExtraText
	project.Status.CardId = p.Card.CardID
	project.Status.Cloud = p.DefaultCloud
	project.Status.CountryCode = p.CountryCode
	project.Status.VatId = p.VatID
	project.Status.CopyFromProject = p.CopyFromProject
	project.Status.BillingCurrency = p.BillingCurrency

	err := r.Status().Update(context.Background(), project)
	if err != nil {
		return err
	}

	return nil
}

// deploymentForProject returns a project Deployment object
func (r *ProjectReconciler) deploymentForProject(p *k8soperatorv1alpha1.Project) *appsv1.Deployment {
	ls := labelsForProject(p.Name)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
		},
	}

	// Set Project instance as the owner and controller
	_ = ctrl.SetControllerReference(p, dep, r.Scheme)
	return dep
}

// labelsForProject returns the labels for selecting the resources
// belonging to the given project CR name.
func labelsForProject(name string) map[string]string {
	return map[string]string{"app": "project", "project_cr": name}
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Project{}).
		Complete(r)
}

// GetClient retrieves an Aiven client
func (r *ProjectReconciler) GetClient(s *corev1.Secret) (*aiven.Client, error) {
	var token string

	if v, ok := s.Data["token"]; ok {
		token = string(v)
	} else {
		return nil, fmt.Errorf("cannot initialize Aiven client, kubernetes secret has no `token` key")
	}

	c, err := aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return nil, err
	}

	return c, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// finalizeProject deletes Aiven project
func (r *ProjectReconciler) finalizeProject(log logr.Logger, p *k8soperatorv1alpha1.Project, c *aiven.Client) error {
	if err := c.Projects.Delete(p.Spec.Name); err != nil {
		log.Error(err, "Cannot delete Aiven project", "Project.Namespace", p.Namespace, "Project.Name", p.Name)
		return err
	}

	log.Info("Successfully finalized project")
	return nil
}

// addFinalizer add finalizer to CR
func (r *ProjectReconciler) addFinalizer(reqLogger logr.Logger, p *k8soperatorv1alpha1.Project) error {
	reqLogger.Info("Adding Finalizer for the Project")
	controllerutil.AddFinalizer(p, projectFinalizer)

	// Update CR
	err := r.Client.Update(context.TODO(), p)
	if err != nil {
		reqLogger.Error(err, "Failed to update Project with finalizer")
		return err
	}
	return nil
}
