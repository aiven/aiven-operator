// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
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

func (h KafkaSchemaHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	schema, err := h.convert(obj)
	if err != nil {
		return err
	}

	// createOrUpdate Kafka Schema Subject
	_, err = avn.KafkaSubjectSchemas.Add(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		aiven.KafkaSchemaSubject{
			Schema: schema.Spec.Schema,
		},
	)
	if err != nil {
		return fmt.Errorf("cannot add Kafka Schema Subject: %w", err)
	}

	// set compatibility level if defined for a newly created Kafka Schema Subject
	if schema.Spec.CompatibilityLevel != "" {
		_, err := avn.KafkaSubjectSchemas.UpdateConfiguration(
			ctx,
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			schema.Spec.CompatibilityLevel,
		)
		if err != nil {
			return fmt.Errorf("cannot update Kafka Schema Configuration: %w", err)
		}
	}

	// get last version
	version, err := h.getLastVersion(ctx, avn, schema)
	if err != nil {
		return fmt.Errorf("cannot get Kafka Schema Subject version: %w", err)
	}

	schema.Status.Version = version

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

func (h KafkaSchemaHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	schema, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = avn.KafkaSubjectSchemas.Delete(ctx, schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName)
	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka Schema error: %w", err)
	}

	return true, nil
}

func (h KafkaSchemaHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
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

func (h KafkaSchemaHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
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

func (h KafkaSchemaHandler) getLastVersion(ctx context.Context, avn *aiven.Client, schema *v1alpha1.KafkaSchema) (int, error) {
	ver, err := avn.KafkaSubjectSchemas.GetVersions(
		ctx,
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName)
	if err != nil {
		return 0, err
	}

	var latestVersion int
	for _, v := range ver.Versions {
		if v > latestVersion {
			latestVersion = v
		}
	}

	return latestVersion, nil
}
