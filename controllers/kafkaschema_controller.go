// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	Controller
}

type KafkaSchemaHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaschemas/status,verbs=get;update;patch

func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	schema := &k8soperatorv1alpha1.KafkaSchema{}
	err := r.Get(ctx, req.NamespacedName, schema)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, schema.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &KafkaSchemaHandler{
		client: c,
	}, schema)
}

func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaSchema{}).
		Complete(r)
}

func (h KafkaSchemaHandler) createOrUpdate(i client.Object) (client.Object, error) {
	schema, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	// createOrUpdate Kafka Schema Subject
	_, err = h.client.KafkaSubjectSchemas.Add(
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		aiven.KafkaSchemaSubject{
			Schema: schema.Spec.Schema,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("cannot add Kafka Schema Subject: %w", err)
	}

	// set compatibility level if defined for a newly created Kafka Schema Subject
	if schema.Spec.CompatibilityLevel != "" {
		_, err := h.client.KafkaSubjectSchemas.UpdateConfiguration(
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			schema.Spec.CompatibilityLevel,
		)
		if err != nil {
			return nil, fmt.Errorf("cannot update Kafka Schema Configuration: %w", err)
		}
	}

	// get last version
	version, err := h.getLastVersion(schema)
	if err != nil {
		return nil, fmt.Errorf("cannot get Kafka Schema Subject version: %w", err)
	}

	schema.Status.Version = version

	meta.SetStatusCondition(&schema.Status.Conditions,
		getInitializedCondition("Added",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Added",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&schema.ObjectMeta,
		processedGeneration, strconv.FormatInt(schema.GetGeneration(), formatIntBaseDecimal))

	return schema, nil
}

func (h KafkaSchemaHandler) delete(i client.Object) (bool, error) {
	schema, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = h.client.KafkaSubjectSchemas.Delete(schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.Schema)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka Schema error: %w", err)
	}

	return true, nil
}

func (h KafkaSchemaHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	schema, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	meta.SetStatusCondition(&schema.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&schema.ObjectMeta, isRunning, "true")

	return schema, nil, nil
}

func (h KafkaSchemaHandler) checkPreconditions(i client.Object) (bool, error) {
	schema, err := h.convert(i)
	if err != nil {
		return false, err
	}

	return checkServiceIsRunning(h.client, schema.Spec.Project, schema.Spec.ServiceName)
}

func (h KafkaSchemaHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaSchema, error) {
	schema, ok := i.(*k8soperatorv1alpha1.KafkaSchema)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaSchema")
	}

	return schema, nil
}

func (h KafkaSchemaHandler) getLastVersion(schema *k8soperatorv1alpha1.KafkaSchema) (int, error) {
	ver, err := h.client.KafkaSubjectSchemas.GetVersions(
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
