// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	Controller
}

func newProjectReconciler(c Controller) reconcilerType {
	return &ProjectReconciler{Controller: c}
}

// ProjectHandler handles an Aiven project
type ProjectHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=projects/finalizers,verbs=get;create;update

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ProjectHandler{}, &v1alpha1.Project{})
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Project{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (h ProjectHandler) getLongCardID(ctx context.Context, client *aiven.Client, cardID string) (*string, error) {
	if cardID == "" {
		return nil, nil
	}

	card, err := client.CardsHandler.Get(ctx, cardID)
	if err != nil {
		return nil, err
	}

	if card == nil {
		return nil, nil
	}

	return &card.CardID, nil
}

// create creates a project on Aiven side
func (h ProjectHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	project, err := h.convert(obj)
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

	exists, err := h.exists(ctx, avn, project)
	if err != nil {
		return fmt.Errorf("project does not exists: %w", err)
	}

	cardID, err := h.getLongCardID(ctx, avn, project.Spec.CardID)
	if err != nil {
		return fmt.Errorf("cannot get long card id: %w", err)
	}

	var reason string
	var p *aiven.Project
	if !exists {
		p, err = avn.Projects.Create(ctx, aiven.CreateProjectRequest{
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
			Tags:             project.Spec.Tags,

			// only set during creation
			BillingGroupId:  project.Spec.BillingGroupID,
			CopyFromProject: project.Spec.CopyFromProject,
		})
		if err != nil {
			return fmt.Errorf("failed to create project on aiven side: %w", err)
		}

		reason = "Created"
	} else {
		p, err = avn.Projects.Update(ctx, project.Name, aiven.UpdateProjectRequest{
			BillingAddress:   toOptionalStringPointer(project.Spec.BillingAddress),
			BillingEmails:    billingEmails,
			BillingExtraText: toOptionalStringPointer(project.Spec.BillingExtraText),
			CardID:           cardID,
			Cloud:            toOptionalStringPointer(project.Spec.Cloud),
			CountryCode:      toOptionalStringPointer(project.Spec.CountryCode),
			AccountId:        project.Spec.AccountID,
			TechnicalEmails:  technicalEmails,
			BillingCurrency:  project.Spec.BillingCurrency,
			Tags:             project.Spec.Tags,
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
			"Successfully created or updated the instance in Aiven"))

	meta.SetStatusCondition(&project.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Successfully created or updated the instance in Aiven, status remains unknown"))

	metav1.SetMetaDataAnnotation(&project.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(project.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ProjectHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	project, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	cert, err := avnGen.ProjectKmsGetCA(ctx, project.Name)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	meta.SetStatusCondition(&project.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&project.ObjectMeta, instanceIsRunningAnnotation, "true")

	prefix := getSecretPrefix(project)
	stringData := map[string]string{
		prefix + "CA_CERT": cert,
		// todo: remove in future releases
		"CA_CERT": cert,
	}
	return newSecret(project, stringData, false), nil
}

// exists checks if project already exists on Aiven side
func (h ProjectHandler) exists(ctx context.Context, avn *aiven.Client, project *v1alpha1.Project) (bool, error) {
	pr, err := avn.Projects.Get(ctx, project.Name)
	if isNotFound(err) {
		return false, nil
	}

	return pr != nil, err
}

// delete deletes Aiven project
func (h ProjectHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	project, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	if err := avn.Projects.Delete(ctx, project.Name); err != nil {
		var skip bool

		// If project not found then there is nothing to delete
		if isNotFound(err) {
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

func (h ProjectHandler) convert(i client.Object) (*v1alpha1.Project, error) {
	p, ok := i.(*v1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to project")
	}

	return p, nil
}

func (h ProjectHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	return true, nil
}
