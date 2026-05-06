// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas/finalizers,verbs=get;create;update

// KafkaSchemaController reconciles a KafkaSchema object.
type KafkaSchemaController struct {
	client.Client
	avnGen avngen.Client
}

func newKafkaSchemaReconciler(c Controller) reconcilerType {
	return newManagedReconciler(
		c,
		func(c Controller, avnGen avngen.Client) AivenController[*v1alpha1.KafkaSchema] {
			return &KafkaSchemaController{Client: c.Client, avnGen: avnGen}
		},
		nil,
	)
}

func (r *KafkaSchemaController) Observe(ctx context.Context, schema *v1alpha1.KafkaSchema) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, schema.Spec.Project, schema.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	versions, err := r.avnGen.ServiceSchemaRegistrySubjectVersionsGet(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	)
	switch {
	case isServerError(err):
		// The service is operational but the schema registry may not yet be ready.
		// Surface this as a precondition miss so the reconciler does a soft requeue.
		return Observation{}, fmt.Errorf("%w: schema registry not ready", errPreconditionNotMet)
	case isNotFound(err):
		// Subject is not registered yet
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("listing Kafka Schema versions: %w", err)
	}

	if schema.Status.ID == 0 {
		// No ID tracked yet, fall through to Create; it is idempotent.
		return Observation{ResourceExists: false}, nil
	}

	for _, v := range versions {
		got, err := r.avnGen.ServiceSchemaRegistrySubjectVersionGet(
			ctx,
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			v,
		)
		if err != nil {
			return Observation{}, fmt.Errorf("getting Kafka Schema version %d: %w", v, err)
		}

		if got.Id != schema.Status.ID {
			continue
		}

		schema.Status.Version = got.Version
		meta.SetStatusCondition(&schema.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
		metav1.SetMetaDataAnnotation(&schema.ObjectMeta, instanceIsRunningAnnotation, "true")

		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: hasLatestGeneration(schema),
		}, nil
	}

	// Tracked version is not visible, maybe eventual-consistency lag after POST
	return Observation{}, fmt.Errorf("%w: tracked schema ID %d not visible in registry", errPreconditionNotMet, schema.Status.ID)
}

func (r *KafkaSchemaController) Create(ctx context.Context, schema *v1alpha1.KafkaSchema) (CreateResult, error) {
	delete(schema.GetAnnotations(), instanceIsRunningAnnotation)
	if err := r.applySchema(ctx, schema); err != nil {
		return CreateResult{}, err
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&schema.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&schema.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *KafkaSchemaController) Update(ctx context.Context, schema *v1alpha1.KafkaSchema) (UpdateResult, error) {
	delete(schema.GetAnnotations(), instanceIsRunningAnnotation)
	if err := r.applySchema(ctx, schema); err != nil {
		return UpdateResult{}, err
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&schema.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&schema.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

// applySchema handles schema creation and updates idempotently:
// - New schemas get a new ID and version 1
// - Schema updates get a new ID and version
// - Submitting the same schema is idempotent (returns existing ID and version)
//
// Example:
// Schema A -> ID:1, Version:1
// Schema B -> ID:2, Version:2
// Revert to A -> ID:1, Version:1
func (r *KafkaSchemaController) applySchema(ctx context.Context, schema *v1alpha1.KafkaSchema) error {
	postIn := &kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn{
		Schema:     schema.Spec.Schema,
		SchemaType: schema.Spec.SchemaType,
	}

	if len(schema.Spec.References) > 0 {
		refs := make([]kafkaschemaregistry.ReferenceIn, len(schema.Spec.References))
		for i, ref := range schema.Spec.References {
			refs[i] = kafkaschemaregistry.ReferenceIn{
				Name:    ref.Name,
				Subject: ref.Subject,
				Version: ref.Version,
			}
		}
		postIn.References = &refs
	}

	schemaID, err := r.avnGen.ServiceSchemaRegistrySubjectVersionPost(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		postIn,
	)
	if err != nil {
		return fmt.Errorf("cannot add Kafka Schema Subject: %w", err)
	}

	// ID is used by Observe to look up the version, which may take some time to appear.
	schema.Status.ID = schemaID

	if schema.Spec.CompatibilityLevel != "" {
		if _, err := r.avnGen.ServiceSchemaRegistrySubjectConfigPut(
			ctx,
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			&kafkaschemaregistry.ServiceSchemaRegistrySubjectConfigPutIn{Compatibility: schema.Spec.CompatibilityLevel},
		); err != nil {
			return fmt.Errorf("cannot update Kafka Schema Configuration: %w", err)
		}
	}

	return nil
}

func (r *KafkaSchemaController) Delete(ctx context.Context, schema *v1alpha1.KafkaSchema) error {
	// Soft delete: the schema is still accessible via ?deleted=true.
	err := r.avnGen.ServiceSchemaRegistrySubjectDelete(ctx, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName)
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}
