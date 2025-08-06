// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	proj "github.com/aiven/go-client-codegen/handler/project"
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

func (h ProjectHandler) getLongCardID(ctx context.Context, avnGen avngen.Client, cardID string) (*string, error) {
	if cardID == "" {
		return nil, nil
	}

	// Uses the deprecated UserCreditCardsList method to retrieve credit cards.
	cards, err := avnGen.UserCreditCardsList(ctx) // nolint:staticcheck
	if err != nil {
		return nil, err
	}

	for _, card := range cards {
		if card.Last4 == cardID || card.CardId == cardID {
			return &card.CardId, nil
		}
	}

	return nil, nil
}

// create creates a project on Aiven side
func (h ProjectHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	project, err := h.convert(obj)
	if err != nil {
		return err
	}

	billingEmails := make([]proj.BillingEmailIn, 0, len(project.Spec.BillingEmails))
	for _, v := range project.Spec.BillingEmails {
		billingEmails = append(billingEmails, proj.BillingEmailIn{Email: v})
	}

	technicalEmails := make([]proj.TechEmailIn, 0, len(project.Spec.TechnicalEmails))
	for _, v := range project.Spec.TechnicalEmails {
		technicalEmails = append(technicalEmails, proj.TechEmailIn{Email: v})
	}

	exists, err := h.exists(ctx, avnGen, project)
	if err != nil {
		return fmt.Errorf("project does not exists: %w", err)
	}

	cardID, err := h.getLongCardID(ctx, avnGen, project.Spec.CardID)
	if err != nil {
		return fmt.Errorf("cannot get long card id: %w", err)
	}

	if !exists {
		req := &proj.ProjectCreateIn{
			CardId:           cardID,
			Project:          project.Name,
			BillingCurrency:  project.Spec.BillingCurrency,
			BillingEmails:    &billingEmails,
			Tags:             &project.Spec.Tags,
			TechEmails:       &technicalEmails,
			BillingAddress:   NilIfZero(project.Spec.BillingAddress),
			BillingExtraText: NilIfZero(project.Spec.BillingExtraText),
			Cloud:            NilIfZero(project.Spec.Cloud),
			CountryCode:      NilIfZero(project.Spec.CountryCode),
			AccountId:        NilIfZero(project.Spec.AccountID),
			BillingGroupId:   NilIfZero(project.Spec.BillingGroupID),
			CopyFromProject:  NilIfZero(project.Spec.CopyFromProject),
		}

		p, err := avnGen.ProjectCreate(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create project on aiven side: %w", err)
		}

		project.Status.VatID = p.VatId
		project.Status.EstimatedBalance = p.EstimatedBalance
		project.Status.AvailableCredits = fromAnyPointer(p.AvailableCredits)
		project.Status.Country = p.Country
		project.Status.PaymentMethod = p.PaymentMethod
	} else {
		req := &proj.ProjectUpdateIn{
			CardId:           cardID,
			BillingEmails:    &billingEmails,
			BillingAddress:   NilIfZero(project.Spec.BillingAddress),
			BillingExtraText: NilIfZero(project.Spec.BillingExtraText),
			Cloud:            NilIfZero(project.Spec.Cloud),
			CountryCode:      NilIfZero(project.Spec.CountryCode),
			AccountId:        NilIfZero(project.Spec.AccountID),
			TechEmails:       &technicalEmails,
			BillingCurrency:  project.Spec.BillingCurrency,
			Tags:             &project.Spec.Tags,
		}

		p, err := avnGen.ProjectUpdate(ctx, project.Name, req)
		if err != nil {
			return fmt.Errorf("failed to update project on aiven side: %w", err)
		}

		project.Status.VatID = p.VatId
		project.Status.EstimatedBalance = p.EstimatedBalance
		project.Status.AvailableCredits = fromAnyPointer(p.AvailableCredits)
		project.Status.Country = p.Country
		project.Status.PaymentMethod = p.PaymentMethod
	}

	return nil
}

func (h ProjectHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
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
func (h ProjectHandler) exists(ctx context.Context, client avngen.Client, project *v1alpha1.Project) (bool, error) {
	pr, err := client.ProjectGet(ctx, project.Name)
	if isNotFound(err) {
		return false, nil
	}

	return pr != nil, err
}

// delete deletes Aiven project
func (h ProjectHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	project, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	err = avnGen.ProjectDelete(ctx, project.Name)
	return isDeleted(err)
}

func (h ProjectHandler) convert(i client.Object) (*v1alpha1.Project, error) {
	p, ok := i.(*v1alpha1.Project)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to project")
	}

	return p, nil
}

func (h ProjectHandler) checkPreconditions(_ context.Context, _ avngen.Client, _ client.Object) (bool, error) {
	return true, nil
}
