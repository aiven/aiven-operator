// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	proj "github.com/aiven/go-client-codegen/handler/project"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newProjectReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.Project] {
			return &ProjectController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
}

//+kubebuilder:rbac:groups=aiven.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=projects/finalizers,verbs=get;create;update

// ProjectController reconciles a Project object.
type ProjectController struct {
	client.Client
	avnGen avngen.Client
}

func (r *ProjectController) Observe(ctx context.Context, cr *v1alpha1.Project) (Observation, error) {
	p, err := r.avnGen.ProjectGet(ctx, cr.Name) // nolint:staticcheck
	if err != nil {
		if isNotFound(err) {
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, err
	}

	setProjectStatus(cr, p)

	if !hasLatestGeneration(cr) {
		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	cert, err := r.avnGen.ProjectKmsGetCA(ctx, cr.Name)
	if err != nil {
		return Observation{}, fmt.Errorf("getting project KMS CA: %w", err)
	}
	markInstanceRunning(cr)

	prefix := getSecretPrefix(cr)
	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		SecretDetails: SecretDetails{
			prefix + "CA_CERT": cert,
			// todo: remove in future releases
			"CA_CERT": cert,
		},
	}, nil
}

func (r *ProjectController) Create(ctx context.Context, cr *v1alpha1.Project) (CreateResult, error) {
	delete(cr.GetAnnotations(), instanceIsRunningAnnotation)

	cardID, err := r.getLongCardID(ctx, cr.Spec.CardID)
	if err != nil {
		return CreateResult{}, fmt.Errorf("getting long card id: %w", err)
	}

	billingEmails := projectBillingEmails(cr.Spec.BillingEmails)
	technicalEmails := projectTechnicalEmails(cr.Spec.TechnicalEmails)

	p, err := r.avnGen.ProjectCreate(ctx, &proj.ProjectCreateIn{
		CardId:           cardID,
		Project:          cr.Name,
		BillingCurrency:  cr.Spec.BillingCurrency,
		BillingEmails:    &billingEmails,
		Tags:             &cr.Spec.Tags,
		TechEmails:       &technicalEmails,
		BillingAddress:   NilIfZero(cr.Spec.BillingAddress),
		BillingExtraText: NilIfZero(cr.Spec.BillingExtraText),
		Cloud:            NilIfZero(cr.Spec.Cloud),
		CountryCode:      NilIfZero(cr.Spec.CountryCode),
		AccountId:        NilIfZero(cr.Spec.AccountID),
		BillingGroupId:   NilIfZero(cr.Spec.BillingGroupID),
		CopyFromProject:  NilIfZero(cr.Spec.CopyFromProject),
	})
	if err != nil {
		if isServerError(err) {
			return CreateResult{}, fmt.Errorf("%w: creating project: %w", errPreconditionNotMet, err)
		}

		return CreateResult{}, fmt.Errorf("creating project: %w", err)
	}

	setProjectStatus(cr, (*proj.ProjectGetOut)(p))

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&cr.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *ProjectController) Update(ctx context.Context, cr *v1alpha1.Project) (UpdateResult, error) {
	delete(cr.GetAnnotations(), instanceIsRunningAnnotation)

	cardID, err := r.getLongCardID(ctx, cr.Spec.CardID)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("getting long card id: %w", err)
	}

	billingEmails := projectBillingEmails(cr.Spec.BillingEmails)
	technicalEmails := projectTechnicalEmails(cr.Spec.TechnicalEmails)

	p, err := r.avnGen.ProjectUpdate(ctx, cr.Name, &proj.ProjectUpdateIn{ // nolint:staticcheck
		CardId:           cardID,
		BillingEmails:    &billingEmails,
		BillingAddress:   NilIfZero(cr.Spec.BillingAddress),
		BillingExtraText: NilIfZero(cr.Spec.BillingExtraText),
		Cloud:            NilIfZero(cr.Spec.Cloud),
		CountryCode:      NilIfZero(cr.Spec.CountryCode),
		AccountId:        NilIfZero(cr.Spec.AccountID),
		TechEmails:       &technicalEmails,
		BillingCurrency:  cr.Spec.BillingCurrency,
		Tags:             &cr.Spec.Tags,
	})
	if err != nil {
		if isServerError(err) {
			return UpdateResult{}, fmt.Errorf("%w: updating project: %w", errPreconditionNotMet, err)
		}

		return UpdateResult{}, fmt.Errorf("updating project: %w", err)
	}

	setProjectStatus(cr, (*proj.ProjectGetOut)(p))

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&cr.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

func (r *ProjectController) Delete(ctx context.Context, project *v1alpha1.Project) error {
	err := r.avnGen.ProjectDelete(ctx, project.Name) // nolint:staticcheck
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting project: %w", err)
	}

	return nil
}

func (r *ProjectController) getLongCardID(ctx context.Context, cardID string) (*string, error) {
	if cardID == "" {
		return nil, nil
	}

	// Uses the deprecated UserCreditCardsList method to retrieve credit cards.
	cards, err := r.avnGen.UserCreditCardsList(ctx) // nolint:staticcheck
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

func setProjectStatus(cr *v1alpha1.Project, p *proj.ProjectGetOut) {
	cr.Status.VatID = p.VatId
	cr.Status.EstimatedBalance = p.EstimatedBalance
	cr.Status.AvailableCredits = fromAnyPointer(p.AvailableCredits)
	cr.Status.Country = p.Country
	cr.Status.PaymentMethod = p.PaymentMethod
}

func projectBillingEmails(emails []string) []proj.BillingEmailIn {
	billingEmails := make([]proj.BillingEmailIn, len(emails))
	for i, v := range emails {
		billingEmails[i] = proj.BillingEmailIn{Email: v}
	}
	return billingEmails
}

func projectTechnicalEmails(emails []string) []proj.TechEmailIn {
	technicalEmails := make([]proj.TechEmailIn, len(emails))
	for i, v := range emails {
		technicalEmails[i] = proj.TechEmailIn{Email: v}
	}
	return technicalEmails
}
