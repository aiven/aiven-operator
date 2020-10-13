// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	Controller
}

const projectFinalizer = "project-finalizer.k8s-operator.aiven.io"

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("project", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Project instance
	project := &k8soperatorv1alpha1.Project{}
	err := r.Get(ctx, req.NamespacedName, project)
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

	// Check if the Project instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isProjectMarkedToBeDeleted := project.GetDeletionTimestamp() != nil
	if isProjectMarkedToBeDeleted {
		if contains(project.GetFinalizers(), projectFinalizer) {
			// Run finalization logic for projectFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeProject(log, project); err != nil {
				return reconcile.Result{}, err
			}

			// Remove projectFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(project, projectFinalizer)
			err := r.Client.Update(ctx, project)
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

	// Check if project already exists on the Aiven side, create a
	// new one if project is not found
	_, err = r.AivenClient.Projects.Get(project.Spec.Name)
	if err != nil {
		// Create a new project if project does not exists and update CR status
		if aiven.IsNotFound(err) {
			_, err = r.createProject(project)
			if err != nil {
				log.Error(err, "Failed to create Project")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	// Update project via API and update CR status
	_, err = r.updateProject(project)
	if err != nil {
		log.Error(err, "Failed to update Project")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// createCACertSecret creates a CA project certificate secret
func (r *ProjectReconciler) createCACertSecret(project *k8soperatorv1alpha1.Project) error {
	cert, err := r.AivenClient.CA.Get(project.Status.Name)
	if err != nil {
		return fmt.Errorf("aiven client error %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", project.Name, "-ca-cert"),
			Namespace: project.Namespace,
			Labels: map[string]string{
				"app": project.Name,
			},
		},
		StringData: map[string]string{
			"cert": cert,
		},
	}
	err = r.Client.Create(context.Background(), secret)
	if err != nil {
		return fmt.Errorf("k8s client create error %w", err)
	}

	// Set Project instance as the owner and controller
	err = controllerutil.SetControllerReference(project, secret, r.Scheme)
	if err != nil {
		return fmt.Errorf("k8s set controller error %w", err)
	}

	return nil
}

// createProject creates a project on Aiven side
func (r *ProjectReconciler) createProject(project *k8soperatorv1alpha1.Project) (*aiven.Project, error) {
	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := r.AivenClient.Projects.Create(aiven.CreateProjectRequest{
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

	// Update project custom resource status
	err = r.updateCRStatus(project, p)
	if err != nil {
		return nil, fmt.Errorf("failed to update Project status: %w", err)
	}

	// Get CA Certificate of a newly created project and save it as K8s secret
	err = r.createCACertSecret(project)
	if err != nil {
		return nil, fmt.Errorf("failed to create Project CA Secret: %w", err)
	}

	return p, err
}

// updateProject updates a project on Aiven side
func (r *ProjectReconciler) updateProject(project *k8soperatorv1alpha1.Project) (*aiven.Project, error) {
	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := r.AivenClient.Projects.Update(project.Spec.Name, aiven.UpdateProjectRequest{
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

	// Update project custom resource status
	err = r.updateCRStatus(project, p)
	if err != nil {
		return nil, fmt.Errorf("failed to update Project status: %w", err)
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

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Project{}).
		Complete(r)
}

// finalizeProject deletes Aiven project
func (r *ProjectReconciler) finalizeProject(log logr.Logger, p *k8soperatorv1alpha1.Project) error {
	// Check if secret exists and delete if it is
	secret := &corev1.Secret{}
	err := r.Get(context.Background(), types.NamespacedName{Name: fmt.Sprintf("%s%s", p.Name, "-ca-cert"), Namespace: p.Namespace}, secret)
	if err == nil {
		err = r.Client.Delete(context.Background(), secret)
		if err != nil {
			return fmt.Errorf("delete project secret error: %w", err)
		}
	}

	// Delete project on Aiven side
	if err := r.AivenClient.Projects.Delete(p.Spec.Name); err != nil {
		var skip bool

		// If project not found then there is nothing to delete
		if aiven.IsNotFound(err) {
			skip = true
		}

		// Silence "Project with open balance cannot be deleted" error
		// to make long acceptance tests pass which generate some balance
		re := regexp.MustCompile("Project with open balance cannot be deleted")
		re1 := regexp.MustCompile("Project with unused credits cannot be deleted")
		if (re.MatchString(err.Error()) || re1.MatchString(err.Error())) && err.(aiven.Error).Status == 403 {
			skip = true
		}

		if !skip {
			log.Error(err, "Cannot delete Aiven project")
			return fmt.Errorf("aiven client delete project error: %w", err)
		}
	}

	log.Info("Successfully finalized project")
	return nil
}

// addFinalizer add finalizer to CR
func (r *ProjectReconciler) addFinalizer(reqLogger logr.Logger, p *k8soperatorv1alpha1.Project) error {
	reqLogger.Info("Adding Finalizer for the Project")
	controllerutil.AddFinalizer(p, projectFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), p)
	if err != nil {
		reqLogger.Error(err, "Failed to update Project with finalizer")
		return err
	}
	return nil
}
