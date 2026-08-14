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

//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls/finalizers,verbs=get;create;update

// KafkaNativeACLController reconciles a KafkaNativeACL object.
type KafkaNativeACLController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaNativeACLReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaNativeACL] {
			return &KafkaNativeACLController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaNativeACLController) Observe(ctx context.Context, acl *v1alpha1.KafkaNativeACL) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, acl.Spec.Project, acl.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	if acl.Status.ID == "" {
		// No stored ID — the ACL may already exist on Aiven, e.g. the CR was recreated after a
		// deletionPolicy:Orphan deletion, or the ACL was created outside the operator. Check the
		// live list before creating.
		list, err := r.avnGen.ServiceKafkaNativeAclList(ctx, acl.Spec.Project, acl.Spec.ServiceName)
		if err != nil {
			return Observation{}, fmt.Errorf("list Kafka-native ACLs error: %w", err)
		}
		for _, existing := range list.KafkaAcl {
			if nativeSpecMatches(acl.Spec, existing) {
				acl.Status.ID = existing.Id
				markInstanceRunning(acl)
				// The spec is immutable, so an adopted ACL is always up to date.
				return Observation{ResourceExists: true, ResourceUpToDate: true}, nil
			}
		}
		return Observation{ResourceExists: false}, nil
	}

	_, err := r.avnGen.ServiceKafkaNativeAclGet(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	switch {
	case isNotFound(err):
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("get Kafka-native ACL error: %w", err)
	}

	markInstanceRunning(acl)

	// The spec is immutable, so an existing ACL is always up to date.
	return Observation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (r *KafkaNativeACLController) Create(ctx context.Context, acl *v1alpha1.KafkaNativeACL) (CreateResult, error) {
	delete(acl.GetAnnotations(), instanceIsRunningAnnotation)

	in := &kafka.ServiceKafkaNativeAclAddIn{
		Host:           &acl.Spec.Host,
		Operation:      acl.Spec.Operation,
		PatternType:    acl.Spec.PatternType,
		PermissionType: acl.Spec.PermissionType,
		Principal:      acl.Spec.Principal,
		ResourceName:   acl.Spec.ResourceName,
		ResourceType:   acl.Spec.ResourceType,
	}

	rsp, err := r.avnGen.ServiceKafkaNativeAclAdd(ctx, acl.Spec.Project, acl.Spec.ServiceName, in)
	if err != nil {
		return CreateResult{}, fmt.Errorf("create Kafka-native ACL error: %w", err)
	}

	acl.Status.ID = rsp.Id
	markInstanceRunning(acl)

	return CreateResult{ResourceExists: true, ResourceUpToDate: true}, nil
}

// Update is a no-op: the spec is immutable, so an existing ACL never needs updating.
func (r *KafkaNativeACLController) Update(_ context.Context, acl *v1alpha1.KafkaNativeACL) (UpdateResult, error) {
	markInstanceRunning(acl)
	return UpdateResult{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (r *KafkaNativeACLController) Delete(ctx context.Context, acl *v1alpha1.KafkaNativeACL) error {
	err := r.avnGen.ServiceKafkaNativeAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("delete Kafka-native ACL error: %w", err)
	}
	return nil
}

// nativeSpecMatches reports whether an existing Kafka-native ACL from Aiven matches the CR
// spec. All immutable identifying fields are compared.
func nativeSpecMatches(spec v1alpha1.KafkaNativeACLSpec, existing kafka.KafkaAclOut) bool {
	return spec.Host == existing.Host &&
		spec.Principal == existing.Principal &&
		spec.ResourceName == existing.ResourceName &&
		spec.Operation == existing.Operation &&
		spec.PatternType == existing.PatternType &&
		string(spec.PermissionType) == string(existing.PermissionType) &&
		spec.ResourceType == existing.ResourceType
}
