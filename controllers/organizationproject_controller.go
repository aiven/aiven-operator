// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/organizationprojects"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newOrganizationProjectReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.OrganizationProject] {
			return &OrganizationProjectController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		nil,
	)
}

//+kubebuilder:rbac:groups=aiven.io,resources=organizationprojects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=organizationprojects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=organizationprojects/finalizers,verbs=get;create;update

// OrganizationProjectController reconciles an OrganizationProject object.
type OrganizationProjectController struct {
	client.Client
	avnGen avngen.Client
}

func (r *OrganizationProjectController) Observe(ctx context.Context, cr *v1alpha1.OrganizationProject) (Observation, error) {
	got, err := r.avnGen.OrganizationProjectsGet(ctx, cr.Spec.OrganizationID, cr.Spec.ProjectID)
	if err != nil {
		if isNotFound(err) {
			return Observation{ResourceExists: false}, nil
		}
		return Observation{}, fmt.Errorf("getting organization project: %w", err)
	}

	inSync := hasLatestGeneration(cr)
	if inSync { // The spec was applied, compare the remote state
		inSync, err = r.orgProjectMatchesSpec(ctx, got, cr)
		if err != nil {
			return Observation{}, err
		}
	}

	if !inSync {
		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: false,
		}, nil
	}

	// The CA cert is only fetched to fill the connection secret, so skip the call
	// entirely when the secret is disabled: the details would be dropped anyway.
	var details SecretDetails
	if !cr.NoSecret() {
		cert, err := r.avnGen.ProjectKmsGetCA(ctx, cr.Spec.ProjectID)
		if err != nil {
			return Observation{}, fmt.Errorf("getting project KMS CA: %w", err)
		}

		details = SecretDetails{
			getSecretPrefix(cr) + "CA_CERT": cert,
		}
	}

	markInstanceRunning(cr)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		SecretDetails:    details,
	}, nil
}

func (r *OrganizationProjectController) Create(ctx context.Context, cr *v1alpha1.OrganizationProject) (CreateResult, error) {
	delete(cr.GetAnnotations(), instanceIsRunningAnnotation)

	parentID, err := r.resolveParentID(ctx, cr.Spec.ParentID)
	if err != nil {
		return CreateResult{}, err
	}

	techEmails := organizationProjectTechEmails(cr.Spec.TechnicalEmails)
	in := &organizationprojects.OrganizationProjectsCreateIn{
		ProjectId:      cr.Spec.ProjectID,
		BillingGroupId: cr.Spec.BillingGroupID,
		ParentId:       NilIfZero(parentID),
		BasePort:       cr.Spec.BasePort,
		Tags:           emptyIfNil(cr.Spec.Tags),
		TechEmails:     &techEmails,
	}

	if _, err := r.avnGen.OrganizationProjectsCreate(ctx, cr.Spec.OrganizationID, in); err != nil {
		if isServerError(err) {
			return CreateResult{}, fmt.Errorf("%w: creating organization project: %w", errPreconditionNotMet, err)
		}
		return CreateResult{}, fmt.Errorf("creating organization project: %w", err)
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&cr.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *OrganizationProjectController) Update(ctx context.Context, cr *v1alpha1.OrganizationProject) (UpdateResult, error) {
	delete(cr.GetAnnotations(), instanceIsRunningAnnotation)

	parentID, err := r.resolveParentID(ctx, cr.Spec.ParentID)
	if err != nil {
		return UpdateResult{}, err
	}

	techEmails := organizationProjectTechEmails(cr.Spec.TechnicalEmails)
	tags := emptyIfNil(cr.Spec.Tags)
	in := &organizationprojects.OrganizationProjectsUpdateIn{
		BillingGroupId: NilIfZero(cr.Spec.BillingGroupID),
		ParentId:       NilIfZero(parentID),
		BasePort:       cr.Spec.BasePort,
		Tags:           &tags,
		TechEmails:     &techEmails,
	}

	if _, err := r.avnGen.OrganizationProjectsUpdate(ctx, cr.Spec.OrganizationID, cr.Spec.ProjectID, in); err != nil {
		if isServerError(err) {
			return UpdateResult{}, fmt.Errorf("%w: updating organization project: %w", errPreconditionNotMet, err)
		}
		return UpdateResult{}, fmt.Errorf("updating organization project: %w", err)
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&cr.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

func (r *OrganizationProjectController) Delete(ctx context.Context, cr *v1alpha1.OrganizationProject) error {
	err := r.avnGen.OrganizationProjectsDelete(ctx, cr.Spec.OrganizationID, cr.Spec.ProjectID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("deleting organization project: %w", err)
	}

	return nil
}

// orgProjectMatchesSpec reports whether the remote project matches the spec.
func (r *OrganizationProjectController) orgProjectMatchesSpec(ctx context.Context, got *organizationprojects.OrganizationProjectsGetOut, cr *v1alpha1.OrganizationProject) (bool, error) {
	if cr.Spec.BillingGroupID != fromAnyPointer(got.BillingGroupId) {
		return false, nil
	}

	// A nil basePort in the spec means the field is unmanaged.
	if cr.Spec.BasePort != nil && *cr.Spec.BasePort != fromAnyPointer(got.BasePort) {
		return false, nil
	}

	if !maps.Equal(cr.Spec.Tags, got.Tags) {
		return false, nil
	}

	gotEmails := make([]string, len(got.TechEmails))
	for i, e := range got.TechEmails {
		gotEmails[i] = e.Email
	}
	if !cmp.Equal(normalizeTechEmails(cr.Spec.TechnicalEmails), normalizeTechEmails(gotEmails), cmpopts.EquateEmpty()) {
		return false, nil
	}

	parentID, err := r.resolveParentID(ctx, cr.Spec.ParentID)
	if err != nil {
		return false, err
	}

	return parentID == got.ParentId, nil
}

// resolveParentID converts an organization ID to its account ID form.
// Account IDs are passed through unchanged.
func (r *OrganizationProjectController) resolveParentID(ctx context.Context, parentID string) (string, error) {
	if !strings.HasPrefix(parentID, "org") {
		return parentID, nil
	}

	org, err := r.avnGen.OrganizationGet(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("converting organization ID %q to account ID: %w", parentID, err)
	}
	return org.AccountId, nil
}

// normalizeTechEmails mirrors how Aiven stores tech emails: domain lowercased duplicates dropped, sorted.
func normalizeTechEmails(emails []string) []string {
	out := make([]string, 0, len(emails))
	seen := make(map[string]struct{}, len(emails))
	for _, e := range emails {
		at := strings.LastIndex(e, "@")
		if at >= 0 {
			e = e[:at+1] + strings.ToLower(e[at+1:])
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	slices.Sort(out)
	return out
}

// organizationProjectTechEmails always returns a non-nil slice of technical emails.
func organizationProjectTechEmails(emails []string) []organizationprojects.TechEmailIn {
	techEmails := make([]organizationprojects.TechEmailIn, len(emails))
	for i, v := range emails {
		techEmails[i] = organizationprojects.TechEmailIn{Email: v}
	}
	return techEmails
}
