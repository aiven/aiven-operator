// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"slices"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls/finalizers,verbs=get;create;update

// KafkaSchemaRegistryACLController reconciles a KafkaSchemaRegistryACL object
type KafkaSchemaRegistryACLController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaSchemaRegistryACLReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaSchemaRegistryACL] {
			return &KafkaSchemaRegistryACLController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaSchemaRegistryACLController) Observe(ctx context.Context, acl *v1alpha1.KafkaSchemaRegistryACL) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, acl.Spec.Project, acl.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	// Never created yet.
	if acl.Status.ACLId == "" {
		return Observation{ResourceExists: false}, nil
	}

	exists, err := r.exists(ctx, acl)
	if err != nil {
		return Observation{}, err
	}

	// Spec fields are immutable
	if !exists {
		return Observation{ResourceExists: false}, nil
	}

	markInstanceRunning(acl)

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: true,
	}, nil
}

func (r *KafkaSchemaRegistryACLController) Create(ctx context.Context, acl *v1alpha1.KafkaSchemaRegistryACL) (CreateResult, error) {
	delete(acl.GetAnnotations(), instanceIsRunningAnnotation)

	if err := r.addACL(ctx, acl); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{}, nil
}

// Update is no-op.
func (r *KafkaSchemaRegistryACLController) Update(_ context.Context, _ *v1alpha1.KafkaSchemaRegistryACL) (UpdateResult, error) {
	return UpdateResult{}, nil
}

func (r *KafkaSchemaRegistryACLController) Delete(ctx context.Context, acl *v1alpha1.KafkaSchemaRegistryACL) error {
	if acl.Status.ACLId == "" {
		return nil
	}

	_, err := r.avnGen.ServiceSchemaRegistryAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ACLId)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("cannot delete KafkaSchemaRegistryACL on Aiven side: %w", err)
	}
	return nil
}

// addACL creates the ACL on Aiven and resolves its ID from the returned list.
func (r *KafkaSchemaRegistryACLController) addACL(ctx context.Context, acl *v1alpha1.KafkaSchemaRegistryACL) error {
	in := kafkaschemaregistry.ServiceSchemaRegistryAclAddIn{
		Permission: kafkaschemaregistry.PermissionType(acl.Spec.Permission),
		Resource:   acl.Spec.Resource,
		Username:   acl.Spec.Username,
	}

	list, err := r.avnGen.ServiceSchemaRegistryAclAdd(ctx, acl.Spec.Project, acl.Spec.ServiceName, &in)
	if err != nil {
		return fmt.Errorf("cannot create KafkaSchemaRegistryACL on Aiven side: %w", err)
	}

	for _, v := range list {
		if in.Permission == v.Permission && in.Resource == v.Resource && in.Username == v.Username {
			// fixme: The ID is optional in the response.
			//  It must be a mistake.
			//  https://api.aiven.io/doc/#tag/Service:_Kafka/operation/ServiceSchemaRegistryAclAdd
			if v.Id == nil {
				return fmt.Errorf("received empty ID in response")
			}
			acl.Status.ACLId = *v.Id
			return nil
		}
	}

	return fmt.Errorf("created KafkaSchemaRegistryACL not found in Aiven response")
}

// exists reports whether the ACL identified by Status.ACLId is present on Aiven.
func (r *KafkaSchemaRegistryACLController) exists(
	ctx context.Context,
	acl *v1alpha1.KafkaSchemaRegistryACL,
) (bool, error) {
	list, err := r.avnGen.ServiceSchemaRegistryAclList(ctx, acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	return slices.ContainsFunc(
		list,
		func(v kafkaschemaregistry.AclOut) bool { return v.Id != nil && *v.Id == acl.Status.ACLId },
	), nil
}
