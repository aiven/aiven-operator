// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// +kubebuilder:rbac:groups=aiven.io,resources=projects,verbs=get;list;watch;createOrUpdate;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("project", req.NamespacedName)
	log.Info("reconciling aiven project")

	const projectFinalizer = "project-finalize.aiven.io"
	project := &k8soperatorv1alpha1.Project{}
	return r.reconcileInstance(ctx, req, &ProjectHandler{}, project)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.Project{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// create creates a project on Aiven side
func (h ProjectHandler) createOrUpdate(i client.Object) error {
	project, err := h.convert(i)
	if err != nil {
		return err
	}

	log.Info("creating a new project")

	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := c.Projects.Create(aiven.CreateProjectRequest{
		BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
		BillingEmails:    billingEmails,
		BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
		CardID:           toOptionalStringPointer(project.Spec.CardID),
		Cloud:            toOptionalStringPointer(project.Spec.Cloud),
		CopyFromProject:  project.Spec.CopyFromProject,
		CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
		Project:          project.Name,
		AccountId:        toOptionalStringPointer(project.Spec.AccountID),
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return fmt.Errorf("failed to createOrUpdate Project on Aiven side: %w", err)
	}

	h.setStatus(project, p)

	return nil
}

func (*ProjectHandler) setStatus(project *k8soperatorv1alpha1.Project, p *aiven.Project) {
	project.Status.AccountID = p.AccountId
	project.Status.BillingAddress = p.BillingAddress
	project.Status.BillingEmails = p.GetBillingEmailsAsStringSlice()
	project.Status.TechnicalEmails = p.GetTechnicalEmailsAsStringSlice()
	project.Status.BillingExtraText = p.BillingExtraText
	project.Status.CardID = p.Card.CardID
	project.Status.Cloud = p.DefaultCloud
	project.Status.CountryCode = p.CountryCode
	project.Status.VatID = p.VatID
	project.Status.CopyFromProject = p.CopyFromProject
	project.Status.BillingCurrency = p.BillingCurrency
	project.Status.EstimatedBalance = p.EstimatedBalance
}

func (h ProjectHandler) get(i client.Object) (*corev1.Secret, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("creating a project secret with ca certificate")

	cert, err := c.CA.Get(project.Name)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(project),
			Namespace: project.Namespace,
			Labels: map[string]string{
				"app": project.Name,
			},
		},
		StringData: map[string]string{
			"CA_CERT": cert,
		},
	}, nil
}

// update updates a project on Aiven side
func (h ProjectHandler) update(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("updating project")

	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	p, err := c.Projects.Update(project.Name, aiven.UpdateProjectRequest{
		BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
		BillingEmails:    billingEmails,
		BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
		CardID:           toOptionalStringPointer(project.Spec.CardID),
		Cloud:            toOptionalStringPointer(project.Spec.Cloud),
		CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
		AccountId:        toOptionalStringPointer(project.Spec.AccountID),
		TechnicalEmails:  technicalEmails,
		BillingCurrency:  project.Spec.BillingCurrency,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update project on aiven side: %w", err)
	}

	h.setStatus(project, p)

	return project, nil
}

// exists checks if project already exists on Aiven side
func (h ProjectHandler) exists(c *aiven.Client, log logr.Logger, i client.Object) (bool, error) {
	project, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("checking if project exists")

	pr, err := c.Projects.Get(project.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return pr != nil, err
}

// delete deletes Aiven project
func (h ProjectHandler) delete(i client.Object) (bool, error) {
	project, err := h.convert(i)
	if err != nil {
		return false, err
	}

	log.Info("finalizing project")

	// Delete project on Aiven side
	if err := c.Projects.Delete(project.Name); err != nil {
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
			log.Error(err, "cannot delete aiven project")
			return false, fmt.Errorf("aiven client delete project error: %w", err)
		}
	}

	log.Info("successfully finalized project on aiven side")
	return true, nil
}

func (h ProjectHandler) getSecretName(project *k8soperatorv1alpha1.Project) string {
	if project.Spec.ConnInfoSecretTarget.Name != "" {
		return project.Spec.ConnInfoSecretTarget.Name
	}
	return project.Name
}

func (h ProjectHandler) convert(i client.Object) (*k8soperatorv1alpha1.Project, error) {
	p, ok := i.(*k8soperatorv1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to project")
	}

	return p, nil
}

func (h ProjectHandler) isActive(*aiven.Client, logr.Logger, client.Object) (bool, error) {
	return true, nil
}

func (h ProjectHandler) checkPreconditions(client.Object) bool {
	return true
}

func (h ProjectHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	project, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &project.Spec.AuthSecretRef
}
