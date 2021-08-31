// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	Controller
}

// ProjectHandler handles an Aiven project
type ProjectHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ProjectHandler{}, &v1alpha1.Project{})
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Project{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ProjectHandler) getLongCardID(client *aiven.Client, cardID string) (*string, error) {
	if cardID == "" {
		return nil, nil
	}

	card, err := client.CardsHandler.Get(cardID)
	if err != nil {
		return nil, err
	}

	if card == nil {
		return nil, nil
	}

	return &card.CardID, nil
}

// create creates a project on Aiven side
func (h ProjectHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	project, err := h.convert(i)
	if err != nil {
		return err
	}

	var billingEmails *[]*aiven.ContactEmail
	if len(project.Spec.BillingEmails) > 0 {
		billingEmails = aiven.ContactEmailFromStringSlice(project.Spec.BillingEmails)
	}

	var technicalEmails *[]*aiven.ContactEmail
	if len(project.Spec.TechnicalEmails) > 0 {
		technicalEmails = aiven.ContactEmailFromStringSlice(project.Spec.TechnicalEmails)
	}

	exists, err := h.exists(avn, project)
	if err != nil {
		return fmt.Errorf("project does not exists: %w", err)
	}

	cardID, err := h.getLongCardID(avn, project.Spec.CardID)
	if err != nil {
		return fmt.Errorf("cannot get long card id: %w", err)
	}

	var reason string
	var p *aiven.Project
	if !exists {
		p, err = avn.Projects.Create(aiven.CreateProjectRequest{
			BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
			BillingEmails:    billingEmails,
			BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
			CardID:           cardID,
			Cloud:            toOptionalStringPointer(project.Spec.Cloud),
			CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
			AccountId:        toOptionalStringPointer(project.Spec.AccountID),
			TechnicalEmails:  technicalEmails,
			BillingCurrency:  project.Spec.BillingCurrency,
			Project:          project.Name,

			// only set during creation
			BillingGroupId:  project.Spec.BillingGroupID,
			CopyFromProject: project.Spec.CopyFromProject,
		})
		if err != nil {
			return fmt.Errorf("failed to create project on aiven side: %w", err)
		}

		reason = "Created"
	} else {
		p, err = avn.Projects.Update(project.Name, aiven.UpdateProjectRequest{
			BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
			BillingEmails:    billingEmails,
			BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
			CardID:           cardID,
			Cloud:            toOptionalStringPointer(project.Spec.Cloud),
			CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
			AccountId:        toOptionalStringPointer(project.Spec.AccountID),
			TechnicalEmails:  technicalEmails,
			BillingCurrency:  project.Spec.BillingCurrency,
		})
		if err != nil {
			return fmt.Errorf("failed to update project on aiven side: %w", err)
		}

		reason = "Updated"
	}

	project.Status.VatID = p.VatID
	project.Status.EstimatedBalance = p.EstimatedBalance
	project.Status.AvailableCredits = p.AvailableCredits
	project.Status.Country = p.Country
	project.Status.PaymentMethod = p.PaymentMethod

	meta.SetStatusCondition(&project.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&project.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&project.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(project.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ProjectHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	project, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	cert, err := avn.CA.Get(project.Name)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	meta.SetStatusCondition(&project.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&project.ObjectMeta, instanceIsRunningAnnotation, "true")

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(project),
			Namespace: project.Namespace,
		},
		StringData: map[string]string{
			"CA_CERT": cert,
		},
	}, nil
}

// exists checks if project already exists on Aiven side
func (h ProjectHandler) exists(avn *aiven.Client, project *v1alpha1.Project) (bool, error) {
	pr, err := avn.Projects.Get(project.Name)
	if aiven.IsNotFound(err) {
		return false, nil
	}

	return pr != nil, err
}

// delete deletes Aiven project
func (h ProjectHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	project, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	if err := avn.Projects.Delete(project.Name); err != nil {
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
			return false, fmt.Errorf("aiven client delete project error: %w", err)
		}
	}

	return true, nil
}

func (h ProjectHandler) fetchOwners(ctx context.Context, i client.Object) ([]client.Object, error) {
	return nil, nil
}

func (h ProjectHandler) getSecretName(project *v1alpha1.Project) string {
	if project.Spec.ConnInfoSecretTarget.Name != "" {
		return project.Spec.ConnInfoSecretTarget.Name
	}
	return project.Name
}

func (h ProjectHandler) convert(i client.Object) (*v1alpha1.Project, error) {
	p, ok := i.(*v1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to project")
	}

	return p, nil
}

func (h ProjectHandler) checkPreconditions(_ *aiven.Client, _ client.Object) (bool, error) {
	return true, nil
}
