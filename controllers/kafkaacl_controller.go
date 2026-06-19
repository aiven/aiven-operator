// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/finalizers,verbs=get;create;update

// KafkaACLController reconciles a KafkaACL object
type KafkaACLController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaACLReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaACL] {
			return &KafkaACLController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaACLController) Observe(ctx context.Context, acl *v1alpha1.KafkaACL) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, acl.Spec.Project, acl.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	// ACLs are immutable and identified by (topic, username, permission).
	id, err := r.findIDByContent(ctx, acl)
	switch {
	case isNotFound(err):
		// Nothing matches the spec. Status.ID set => spec changed, recreate via Update.
		// Status.ID empty => never created, go to Create.
		return Observation{ResourceExists: acl.Status.ID != ""}, nil
	case err != nil:
		return Observation{}, err
	}

	acl.Status.ID = id
	markInstanceRunning(acl)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (r *KafkaACLController) Create(ctx context.Context, acl *v1alpha1.KafkaACL) (CreateResult, error) {
	if err := r.applyACL(ctx, acl); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{}, nil
}

func (r *KafkaACLController) Update(ctx context.Context, acl *v1alpha1.KafkaACL) (UpdateResult, error) {
	if err := r.applyACL(ctx, acl); err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{}, nil
}

func (r *KafkaACLController) Delete(ctx context.Context, acl *v1alpha1.KafkaACL) error {
	return r.deleteACL(ctx, acl)
}

// applyACL recreates the ACL from scratch, it can't be modified in place.
func (r *KafkaACLController) applyACL(ctx context.Context, acl *v1alpha1.KafkaACL) error {
	delete(acl.GetAnnotations(), instanceIsRunningAnnotation)

	if err := r.deleteACL(ctx, acl); err != nil {
		return err
	}

	_, err := r.avnGen.ServiceKafkaAclAdd(
		ctx,
		acl.Spec.Project,
		acl.Spec.ServiceName,
		&kafka.ServiceKafkaAclAddIn{
			Permission: acl.Spec.Permission,
			Topic:      acl.Spec.Topic,
			Username:   acl.Spec.Username,
		},
	)
	if err != nil {
		return err
	}

	// Reset the old ID and resolve the newly created one.
	// The server doesn't return the ACL we created, but the list of all ACLs currently defined.
	acl.Status.ID = ""
	acl.Status.ID, err = r.getID(ctx, acl)
	return err
}

func (r *KafkaACLController) deleteACL(ctx context.Context, acl *v1alpha1.KafkaACL) error {
	id, err := r.getID(ctx, acl)
	if err == nil {
		_, err = r.avnGen.ServiceKafkaAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, id)
	}

	if err != nil && !isNotFound(err) {
		return fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return nil
}

// todo: remove in v1
// getID returns ACL's ID in < v0.5.1 compatible mode
func (r *KafkaACLController) getID(ctx context.Context, acl *v1alpha1.KafkaACL) (string, error) {
	// ACLs made prior to v0.5.1 doesn't have an ID.
	// This block is for fresh made ACLs only
	// The rest of this function tries to guess it filtering the list.
	if acl.Status.ID != "" {
		return acl.Status.ID, nil
	}

	// For old ACLs only
	return r.findIDByContent(ctx, acl)
}

// findIDByContent resolves the ID of the ACL matching the current spec.
func (r *KafkaACLController) findIDByContent(ctx context.Context, acl *v1alpha1.KafkaACL) (string, error) {
	list, err := r.avnGen.ServiceKafkaAclList(ctx, acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return "", err
	}

	// There could be multiple ACLs with same attributes.
	// Assume the one that was created is the last one matching.
	var latestID string
	for _, a := range list {
		if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
			latestID = fromAnyPointer(a.Id)
		}
	}

	if latestID != "" {
		return latestID, nil
	}

	return "", NewNotFound(fmt.Sprintf("Kafka ACL %q not found", acl.Name))
}
