// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	Controller
}

func newKafkaSchemaReconciler(c Controller) reconcilerType {
	return &KafkaSchemaReconciler{Controller: c}
}

type KafkaSchemaHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas/finalizers,verbs=get;create;update

func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaSchemaHandler{}, &v1alpha1.KafkaSchema{})
}

func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaSchema{}).
		Complete(r)
}

func (h KafkaSchemaHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	schema, err := h.convert(obj)
	if err != nil {
		return err
	}

	// This operation handles schema creation and updates idempotently:
	// - New schemas get a new ID and version 1
	// - Schema updates get a new ID and version
	// - Submitting the same schema is idempotent (returns existing ID and version)
	//
	// Example:
	// Schema A -> ID:1, Version:1
	// Schema B -> ID:2, Version:2
	// Revert to A -> ID:1, Version:1
	schemaID, err := avnGen.ServiceSchemaRegistrySubjectVersionPost(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		&kafkaschemaregistry.ServiceSchemaRegistrySubjectVersionPostIn{
			Schema:     schema.Spec.Schema,
			SchemaType: schema.Spec.SchemaType,
		},
	)
	if err != nil {
		return fmt.Errorf("cannot add Kafka Schema Subject: %w", err)
	}

	// The ID is used by the get() to poll the schema version which usually takes some time to be available.
	schema.Status.ID = schemaID

	// Sets compatibility level if defined
	if schema.Spec.CompatibilityLevel != "" {
		_, err := avnGen.ServiceSchemaRegistrySubjectConfigPut(
			ctx,
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			&kafkaschemaregistry.ServiceSchemaRegistrySubjectConfigPutIn{
				Compatibility: schema.Spec.CompatibilityLevel,
			},
		)
		if err != nil {
			return fmt.Errorf("cannot update Kafka Schema Configuration: %w", err)
		}
	}

	return nil
}

func (h KafkaSchemaHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// This is a soft delete, the schema still can be fetched with ?deleted=true flag.
	// The hard delete operation is recommended to be only used in development environments or when the topic needs to be recycled.
	err = avnGen.ServiceSchemaRegistrySubjectDelete(ctx, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName)
	return isDeleted(err)
}

func (h KafkaSchemaHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	version, err := getKafkaSchemaVersion(
		ctx,
		avnGen,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		schema.Status.ID,
	)

	switch {
	case isNotFound(err), isServerError(err):
		// Eventual consistency issue.
		// The resource is not replicated yet.
		return nil, nil
	case err != nil:
		return nil, err
	}

	schema.Status.Version = version
	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&schema.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaSchemaHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	ok, err := checkServiceIsOperational(ctx, avnGen, schema.Spec.Project, schema.Spec.ServiceName)
	if !ok || err != nil {
		return ok, err
	}

	// Makes a GET call to retry 5xx errors.
	// Even when the service is operational, the schema registry may not be ready yet.
	// The client retries errors under the hood, but sometimes it's not enough.
	// Let Kubernetes controller-runtime handle retries by returning false to requeue
	// the reconciliation request.
	_, err = avnGen.ServiceSchemaRegistrySubjectVersionsGet(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	)

	switch {
	case isServerError(err):
		// It is not a fatal error, just means that the schema registry is not ready yet.
		return false, nil
	case isNotFound(err):
		// The schema does not exist yet, which is fine.
		// If it exists, it will be updated in createOrUpdate.
		return true, nil
	}

	return err == nil, err
}

func (h KafkaSchemaHandler) convert(i client.Object) (*v1alpha1.KafkaSchema, error) {
	schema, ok := i.(*v1alpha1.KafkaSchema)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaSchema")
	}

	return schema, nil
}

func getKafkaSchemaVersion(ctx context.Context, client avngen.Client, projectName, serviceName, subjectName string, schemaID int) (int, error) {
	versions, err := client.ServiceSchemaRegistrySubjectVersionsGet(ctx, projectName, serviceName, subjectName)
	if err != nil {
		return 0, err
	}

	for _, v := range versions {
		version, err := client.ServiceSchemaRegistrySubjectVersionGet(ctx, projectName, serviceName, subjectName, v)
		if err != nil {
			return 0, err
		}

		if version.Id == schemaID {
			return version.Version, nil
		}
	}

	return 0, NewNotFound(fmt.Sprintf("the schema with id %d not found", schemaID))
}
