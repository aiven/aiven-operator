// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	"github.com/avast/retry-go"
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

func (h KafkaSchemaHandler) createOrUpdate(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	schema, err := h.convert(obj)
	if err != nil {
		return err
	}

	// Must poll kafka until the registry ready.
	// The client retries errors under the hood.
	_, err = avnGen.ServiceSchemaRegistrySubjectVersionsGet(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
	)
	if err != nil && !isNotFound(err) {
		return err
	}

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

	// set compatibility level if defined for a newly created Kafka Schema Subject
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

	// Gets the last version
	// Because of eventual consistency, we must poll the subject
	const (
		pollDelay    = 5 * time.Second
		pollAttempts = 10
	)

	err = retry.Do(
		func() error {
			version, err := getKafkaSchemaVersion(
				ctx,
				avnGen,
				schemaID,
				schema.Spec.Project,
				schema.Spec.ServiceName,
				schema.Spec.SubjectName,
			)
			schema.Status.Version = version
			return err
		},
		retry.Context(ctx),
		retry.RetryIf(isNotFound),
		retry.Delay(pollDelay),
		retry.Attempts(pollAttempts),
	)
	if err != nil {
		return fmt.Errorf("cannot get Kafka Schema Subject version: %w", err)
	}

	meta.SetStatusCondition(&schema.Status.Conditions,
		getInitializedCondition("Added",
			"Successfully created or updated the instance in Aiven"))

	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Added",
			"Successfully created or updated the instance in Aiven, status remains unknown"))

	metav1.SetMetaDataAnnotation(&schema.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(schema.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaSchemaHandler) delete(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = avnGen.ServiceSchemaRegistrySubjectDelete(ctx, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName)
	return isDeleted(err)
}

func (h KafkaSchemaHandler) get(_ context.Context, _ *aiven.Client, _ avngen.Client, obj client.Object) (*corev1.Secret, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&schema.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaSchemaHandler) checkPreconditions(ctx context.Context, _ *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	return checkServiceIsOperational(ctx, avnGen, schema.Spec.Project, schema.Spec.ServiceName)
}

func (h KafkaSchemaHandler) convert(i client.Object) (*v1alpha1.KafkaSchema, error) {
	schema, ok := i.(*v1alpha1.KafkaSchema)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaSchema")
	}

	return schema, nil
}

func getKafkaSchemaVersion(ctx context.Context, client avngen.Client, schemaID int, projectName, serviceName, subjectName string) (int, error) {
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
