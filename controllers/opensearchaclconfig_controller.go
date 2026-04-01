// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	avnopensearch "github.com/aiven/go-client-codegen/handler/opensearch"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func newOpenSearchACLConfigReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(_ Controller, avnGen avngen.Client) AivenController[*v1alpha1.OpenSearchACLConfig] {
			return &OpenSearchACLConfigController{avnGen: avnGen}
		},
		nil,
	)
}

// +kubebuilder:rbac:groups=aiven.io,resources=opensearchaclconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=opensearchaclconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aiven.io,resources=opensearchaclconfigs/finalizers,verbs=get;create;update

// OpenSearchACLConfigController reconciles an OpenSearchACLConfig object.
type OpenSearchACLConfigController struct {
	avnGen avngen.Client
}

func (r *OpenSearchACLConfigController) Observe(ctx context.Context, cr *v1alpha1.OpenSearchACLConfig) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, cr.Spec.Project, cr.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	actual, err := r.avnGen.ServiceOpenSearchAclGet(ctx, cr.Spec.Project, cr.Spec.ServiceName)
	if err != nil {
		return Observation{}, fmt.Errorf("getting OpenSearch ACL config: %w", err)
	}

	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&cr.ObjectMeta, instanceIsRunningAnnotation, "true")

	return Observation{
		ResourceExists: true,
		ResourceUpToDate: cr.Spec.Enabled == actual.OpensearchAclConfig.Enabled &&
			openSearchACLsMatch(cr.Spec.Acls, actual.OpensearchAclConfig.Acls),
	}, nil
}

func (r *OpenSearchACLConfigController) Create(ctx context.Context, cr *v1alpha1.OpenSearchACLConfig) (CreateResult, error) {
	// In practice OpenSearch services appear to expose a default ACL config immediately, so the normal reconcile path is Observe -> Update.
	// Keep Create in case the API behavior changes or we hit an unexpected race.
	_, err := r.avnGen.ServiceOpenSearchAclSet(
		ctx,
		cr.Spec.Project,
		cr.Spec.ServiceName,
		&avnopensearch.ServiceOpenSearchAclSetIn{
			OpensearchAclConfig: avnopensearch.OpensearchAclConfigIn{
				Acls:    buildOpenSearchACLsIn(cr.Spec.Acls),
				Enabled: cr.Spec.Enabled,
			},
		},
	)
	if err != nil {
		return CreateResult{}, fmt.Errorf("setting OpenSearch ACL config: %w", err)
	}

	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&cr.ObjectMeta, instanceIsRunningAnnotation, "true")
	return CreateResult{}, nil
}

func (r *OpenSearchACLConfigController) Update(ctx context.Context, cr *v1alpha1.OpenSearchACLConfig) (UpdateResult, error) {
	_, err := r.avnGen.ServiceOpenSearchAclSet(
		ctx,
		cr.Spec.Project,
		cr.Spec.ServiceName,
		&avnopensearch.ServiceOpenSearchAclSetIn{
			OpensearchAclConfig: avnopensearch.OpensearchAclConfigIn{
				Acls:    buildOpenSearchACLsIn(cr.Spec.Acls),
				Enabled: cr.Spec.Enabled,
			},
		},
	)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("setting OpenSearch ACL config: %w", err)
	}

	meta.SetStatusCondition(&cr.Status.Conditions, getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&cr.ObjectMeta, instanceIsRunningAnnotation, "true")
	return UpdateResult{}, nil
}

func (r *OpenSearchACLConfigController) Delete(ctx context.Context, cfg *v1alpha1.OpenSearchACLConfig) error {
	_, err := r.avnGen.ServiceOpenSearchAclSet(
		ctx,
		cfg.Spec.Project,
		cfg.Spec.ServiceName,
		&avnopensearch.ServiceOpenSearchAclSetIn{
			OpensearchAclConfig: avnopensearch.OpensearchAclConfigIn{
				Acls:    []avnopensearch.AclIn{},
				Enabled: false,
			},
		},
	)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("resetting OpenSearch ACL config: %w", err)
	}

	return nil
}

func buildOpenSearchRulesIn(rules []v1alpha1.OpenSearchACLConfigRule) []avnopensearch.RuleIn {
	out := make([]avnopensearch.RuleIn, 0, len(rules))
	for _, rule := range rules {
		out = append(out, avnopensearch.RuleIn(rule))
	}

	return out
}

func buildOpenSearchACLsIn(acls []v1alpha1.OpenSearchACLConfigACL) []avnopensearch.AclIn {
	out := make([]avnopensearch.AclIn, 0, len(acls))
	for _, acl := range acls {
		out = append(out, avnopensearch.AclIn{
			Username: acl.Username,
			Rules:    buildOpenSearchRulesIn(acl.Rules),
		})
	}

	return out
}

func openSearchACLsMatch(desired []v1alpha1.OpenSearchACLConfigACL, actual []avnopensearch.AclOut) bool {
	if len(desired) != len(actual) {
		return false
	}

	desiredRulesByUsername := make(map[string][]avnopensearch.RuleIn, len(desired))
	for _, desiredACL := range desired {
		desiredRulesByUsername[desiredACL.Username] = buildOpenSearchRulesIn(desiredACL.Rules)
	}

	seen := make(map[string]struct{}, len(actual))
	for _, actualACL := range actual {
		if _, ok := seen[actualACL.Username]; ok {
			return false
		}

		desiredRules, ok := desiredRulesByUsername[actualACL.Username]
		if !ok {
			return false
		}

		actualRules := make([]avnopensearch.RuleIn, len(actualACL.Rules))
		for i, rule := range actualACL.Rules {
			actualRules[i] = avnopensearch.RuleIn(rule)
		}

		if !lo.ElementsMatch(desiredRules, actualRules) {
			return false
		}

		seen[actualACL.Username] = struct{}{}
	}

	return true
}
