// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	Controller
}

// ProjectHandler handles an Aiven project
type ProjectHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("project", req.NamespacedName)
	log.Info("Reconciling Aiven Project")

	const projectFinalizer = "project-finalize.k8s-operator.aiven.io"
	project := &k8soperatorv1alpha1.Project{}
	return r.reconcileInstance(&ProjectHandler{}, ctx, log, req, project, projectFinalizer)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Project{}).
		Complete(r)
}

// create creates a project on Aiven side
func (h *ProjectHandler) create(log logr.Logger, i client.Object) (client.Object, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating a new project")

	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := aivenClient.Projects.Create(aiven.CreateProjectRequest{
		BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
		BillingEmails:    billingEmails,
		BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
		CardID:           toOptionalStringPointer(project.Spec.CardId),
		Cloud:            toOptionalStringPointer(project.Spec.Cloud),
		CopyFromProject:  project.Spec.CopyFromProject,
		CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
		Project:          project.Spec.Name,
		AccountId:        toOptionalStringPointer(project.Spec.AccountId),
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Project on Aiven side: %w", err)
	}

	h.setStatus(project, p)

	return project, nil
}

func (*ProjectHandler) setStatus(project *k8soperatorv1alpha1.Project, p *aiven.Project) {
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
}

func (h *ProjectHandler) getSecret(log logr.Logger, i client.Object) (*corev1.Secret, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating a Project secret with CA certificate")

	cert, err := aivenClient.CA.Get(project.Status.Name)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	return &corev1.Secret{
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
	}, nil
}

// update updates a project on Aiven side
func (h *ProjectHandler) update(log logr.Logger, i client.Object) (client.Object, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Updating Project")

	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := aivenClient.Projects.Update(project.Spec.Name, aiven.UpdateProjectRequest{
		BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
		BillingEmails:    billingEmails,
		BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
		CardID:           toOptionalStringPointer(project.Spec.CardId),
		Cloud:            toOptionalStringPointer(project.Spec.Cloud),
		CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
		AccountId:        toOptionalStringPointer(project.Spec.AccountId),
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update Project on Aiven side: %w", err)
	}

	h.setStatus(project, p)

	return project, nil
}

// exists checks if project already exists on Aiven side
func (h *ProjectHandler) exists(log logr.Logger, i client.Object) (bool, error) {
	project, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("Checking if project exists")

	pr, err := aivenClient.Projects.Get(project.Spec.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return pr != nil, err
}

// delete deletes Aiven project
func (h *ProjectHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	log.Info("Finalizing project")

	// Delete project on Aiven side
	if err := aivenClient.Projects.Delete(project.Spec.Name); err != nil {
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
			return nil, false, fmt.Errorf("aiven client delete project error: %w", err)
		}
	}

	log.Info("Successfully finalized project on Aiven side")
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", project.Name, "-ca-cert"),
			Namespace: project.Namespace,
		},
	}, true, nil
}

func (h *ProjectHandler) convert(i client.Object) (*k8soperatorv1alpha1.Project, error) {
	p, ok := i.(*k8soperatorv1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to project")
	}

	return p, nil
}

func (h *ProjectHandler) isActive(logr.Logger, client.Object) (bool, error) {
	return true, nil
}

func (h *ProjectHandler) checkPreconditions(logr.Logger, client.Object) bool {
	return true
}
