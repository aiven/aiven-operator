// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// kafkaSchemaAppliedFingerprintAnnotation stores a hash of the last
// schema body + resolved references + compatibility level.
const kafkaSchemaAppliedFingerprintAnnotation = "controllers.aiven.io/kafka-schema-applied"

// kafkaSchemaRefIndex is the cache index key for finding KafkaSchemas that
// reference another KafkaSchema by name.
const kafkaSchemaRefIndex = "spec.references.kafkaSchemaRef.name"

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
	).WithIndexes(registerKafkaSchemaRefIndex).
		WithWatches(func(b *builder.Builder) *builder.Builder {
			return b.Watches(
				&v1alpha1.KafkaSchema{},
				handler.EnqueueRequestsFromMapFunc(findKafkaSchemasReferencing(c.Client)),
				builder.WithPredicates(kafkaSchemaVersionChangedPredicate()),
			)
		})
}

// registerKafkaSchemaRefIndex indexes KafkaSchemas by the referent names in Spec.References[*].KafkaSchemaRef.Name.
func registerKafkaSchemaRefIndex(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &v1alpha1.KafkaSchema{}, kafkaSchemaRefIndex, kafkaSchemaRefIndexValues)
}

// kafkaSchemaRefIndexValues extracts referent names from a KafkaSchema for the index.
func kafkaSchemaRefIndexValues(obj client.Object) []string {
	s, ok := obj.(*v1alpha1.KafkaSchema)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(s.Spec.References))
	for _, ref := range s.Spec.References {
		if ref.KafkaSchemaRef != nil {
			names = append(names, ref.KafkaSchemaRef.Name)
		}
	}

	return names
}

// findKafkaSchemasReferencing enqueues every KafkaSchema in the same namespace that has a kafkaSchemaRef pointing at.
func findKafkaSchemasReferencing(k client.Client) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []reconcile.Request {
		target, ok := obj.(*v1alpha1.KafkaSchema)
		if !ok {
			return nil
		}
		var list v1alpha1.KafkaSchemaList
		if err := k.List(ctx, &list,
			client.InNamespace(target.GetNamespace()),
			client.MatchingFields{kafkaSchemaRefIndex: target.GetName()},
		); err != nil {
			return nil
		}

		out := make([]reconcile.Request, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}

		return out
	}
}

// kafkaSchemaVersionChangedPredicate enqueues dependents only when a referent changes.
func kafkaSchemaVersionChangedPredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			o, okO := e.ObjectOld.(*v1alpha1.KafkaSchema)
			n, okN := e.ObjectNew.(*v1alpha1.KafkaSchema)
			if !okO || !okN {
				return true
			}
			return o.Status.Version != n.Status.Version || o.Generation != n.Generation
		},
		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// Observe decides whether the registry already serves what the spec describes.
// Drift detection is driven by an annotation fingerprint of the last applied schema.
func (r *KafkaSchemaController) Observe(ctx context.Context, schema *v1alpha1.KafkaSchema) (Observation, error) {
	if _, err := getServiceIfOperational(ctx, r.avnGen, schema.Spec.Project, schema.Spec.ServiceName); err != nil {
		return Observation{}, err
	}

	_, err := r.avnGen.ServiceSchemaRegistrySubjectVersionsGet(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	)
	switch {
	case isServerError(err):
		// The service is operational but the schema registry may not yet be ready.
		return Observation{}, fmt.Errorf("%w: schema registry not ready", errPreconditionNotMet)
	case isNotFound(err):
		// Subject is not registered yet.
		return Observation{ResourceExists: false}, nil
	case err != nil:
		return Observation{}, fmt.Errorf("listing Kafka Schema versions: %w", err)
	}

	var resolvedRefs []kafkaschemaregistry.ReferenceIn
	if len(schema.Spec.References) > 0 {
		refs, err := r.resolveReferences(ctx, schema)
		switch {
		case errors.Is(err, errPreconditionNotMet):
			// Referent exists but its Status.Version is still 0 — soft-requeue.
			return Observation{}, err
		case err != nil:
			return Observation{}, fmt.Errorf("resolving references: %w", err)
		}
		resolvedRefs = refs
	}
	desiredFP := fingerprintSchema(schema, resolvedRefs)

	appliedFP, ok := schema.GetAnnotations()[kafkaSchemaAppliedFingerprintAnnotation]
	if !ok || appliedFP != desiredFP {
		return Observation{ResourceExists: true, ResourceUpToDate: false}, nil
	}

	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning", "Instance is running on Aiven side"))
	metav1.SetMetaDataAnnotation(&schema.ObjectMeta, instanceIsRunningAnnotation, "true")

	return Observation{
		ResourceExists:   true,
		ResourceUpToDate: hasLatestGeneration(schema),
	}, nil
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

	var resolvedRefs []kafkaschemaregistry.ReferenceIn
	if len(schema.Spec.References) > 0 {
		refs, err := r.resolveReferences(ctx, schema)
		if err != nil {
			return err
		}
		resolvedRefs = refs
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

	version, err := r.lookupVersionForID(ctx, schema, schemaID)
	if err != nil {
		return fmt.Errorf("locating version for schema ID %d: %w", schemaID, err)
	}
	schema.Status.ID = schemaID
	schema.Status.Version = version

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

	metav1.SetMetaDataAnnotation(
		&schema.ObjectMeta,
		kafkaSchemaAppliedFingerprintAnnotation,
		fingerprintSchema(schema, resolvedRefs),
	)

	return nil
}

// lookupVersionForID returns the registry version holding the given schema id.
func (r *KafkaSchemaController) lookupVersionForID(
	ctx context.Context, schema *v1alpha1.KafkaSchema, id int,
) (int, error) {
	versions, err := r.avnGen.ServiceSchemaRegistrySubjectVersionsGet(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	)
	switch {
	case isNotFound(err):
		// soft-requeue and retry
		return 0, fmt.Errorf("%w: subject not visible in registry yet", errPreconditionNotMet)
	case err != nil:
		return 0, fmt.Errorf("listing Kafka Schema versions: %w", err)
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })
	for _, v := range versions {
		got, err := r.avnGen.ServiceSchemaRegistrySubjectVersionGet(
			ctx,
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			v,
		)
		switch {
		case isNotFound(err):
			return 0, fmt.Errorf("%w: schema version %d not visible in registry yet", errPreconditionNotMet, v)
		case err != nil:
			return 0, fmt.Errorf("getting Kafka Schema version %d: %w", v, err)
		}
		if got.Id == id {
			return got.Version, nil
		}
	}

	return 0, fmt.Errorf("%w: schema ID %d not visible in registry yet", errPreconditionNotMet, id)
}

// fingerprintSchema returns a stable hash of the provided schema.
func fingerprintSchema(schema *v1alpha1.KafkaSchema, refs []kafkaschemaregistry.ReferenceIn) string {
	sorted := append([]kafkaschemaregistry.ReferenceIn(nil), refs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	payload := struct {
		Schema             string                                `json:"schema"`
		SchemaType         kafkaschemaregistry.SchemaType        `json:"schemaType"`
		CompatibilityLevel kafkaschemaregistry.CompatibilityType `json:"compatibilityLevel,omitempty"`
		References         []kafkaschemaregistry.ReferenceIn     `json:"references"`
	}{
		Schema:             schema.Spec.Schema,
		SchemaType:         schema.Spec.SchemaType,
		CompatibilityLevel: schema.Spec.CompatibilityLevel,
		References:         sorted,
	}

	buf, _ := json.Marshal(payload)
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:])
}

// resolveReferences turns Spec.References into the ReferenceIn slice.
func (r *KafkaSchemaController) resolveReferences(
	ctx context.Context,
	schema *v1alpha1.KafkaSchema,
) ([]kafkaschemaregistry.ReferenceIn, error) {
	refs := make([]kafkaschemaregistry.ReferenceIn, 0, len(schema.Spec.References))
	for _, ref := range schema.Spec.References {
		subject, version := ref.Subject, ref.Version
		if ref.KafkaSchemaRef != nil {
			target := &v1alpha1.KafkaSchema{}
			key := client.ObjectKey{
				Namespace: schema.GetNamespace(),
				Name:      ref.KafkaSchemaRef.Name,
			}
			if err := r.Get(ctx, key, target); err != nil {
				if apierrors.IsNotFound(err) {
					return nil, fmt.Errorf("%w: referenced KafkaSchema %s not found", errPreconditionNotMet, key)
				}
				return nil, fmt.Errorf("resolving kafkaSchemaRef %s: %w", key, err)
			}
			if target.Status.Version == 0 {
				return nil, fmt.Errorf("%w: referenced KafkaSchema %s has no version yet", errPreconditionNotMet, key)
			}
			subject = target.Spec.SubjectName
			version = target.Status.Version
		}

		refs = append(refs, kafkaschemaregistry.ReferenceIn{
			Name:    ref.Name,
			Subject: subject,
			Version: version,
		})
	}

	return refs, nil
}

func (r *KafkaSchemaController) Delete(ctx context.Context, schema *v1alpha1.KafkaSchema) error {
	// Block delete if any KafkaSchema in this namespace still imports us via kafkaSchemaRef.
	// Only catches kafkaSchemaRef dependents in the same namespace.
	dependents, err := r.findKafkaSchemaRefDependents(ctx, schema)
	if err != nil {
		return fmt.Errorf("checking for kafkaSchemaRef dependents: %w", err)
	}

	if len(dependents) > 0 {
		return fmt.Errorf("%w: still referenced by %s",
			v1alpha1.ErrDeleteDependencies, strings.Join(dependents, ", "))
	}

	// Two-step delete: soft-delete first, then hard-delete.
	// The schema registry requires this ordering — a hard-delete is only
	// allowed after a soft-delete on the same subject.
	//
	// Soft-delete leaves the subject's references attached in the registry's metadata,
	// which keeps any *referent* of this subject pinned.
	if err = r.avnGen.ServiceSchemaRegistrySubjectDelete(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	); err != nil && !isNotFound(err) {
		return fmt.Errorf("soft-deleting Kafka Schema Subject: %w", err)
	}

	if err = r.avnGen.ServiceSchemaRegistrySubjectDelete(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		kafkaschemaregistry.ServiceSchemaRegistrySubjectDeletePermanent(true),
	); err != nil && !isNotFound(err) {
		return fmt.Errorf("hard-deleting Kafka Schema Subject: %w", err)
	}
	return nil
}

// findKafkaSchemaRefDependents returns the sorted list with names of KafkaSchemas in the same
// namespace that reference schema via kafkaSchemaRef.
func (r *KafkaSchemaController) findKafkaSchemaRefDependents(
	ctx context.Context,
	schema *v1alpha1.KafkaSchema,
) ([]string, error) {
	var list v1alpha1.KafkaSchemaList
	if err := r.List(ctx, &list,
		client.InNamespace(schema.GetNamespace()),
		client.MatchingFields{kafkaSchemaRefIndex: schema.GetName()},
	); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for i := range list.Items {
		// CRD-level CEL blocks self-references at admission.
		// But we avoid a self-enqueue loop if a pre-existing object slipped through.
		if list.Items[i].GetName() == schema.GetName() {
			continue
		}
		names = append(names, list.Items[i].GetName())
	}
	sort.Strings(names)

	return names, nil
}
