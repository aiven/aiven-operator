// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	Controller
}

type KafkaSchemaHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas/status,verbs=get;update;patch

func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaschema", req.NamespacedName)
	log.Info("Reconciling Aiven Kafka Schema")

	const finalizer = "kafkaschema-finalizer.k8s-operator.aiven.io"
	schema := &k8soperatorv1alpha1.KafkaSchema{}
	return r.reconcileInstance(&KafkaSchemaHandler{}, ctx, log, req, schema, finalizer)
}

func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaSchema{}).
		Complete(r)
}

func (h KafkaSchemaHandler) create(_ logr.Logger, i client.Object) (client.Object, error) {
	schema, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	// create Kafka Schema Subject
	_, err = aivenClient.KafkaSubjectSchemas.Add(
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
		_, err := aivenClient.KafkaSubjectSchemas.UpdateConfiguration(
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

	h.setStatus(schema, version)

	return schema, nil
}

func (h KafkaSchemaHandler) delete(log logr.Logger, i client.Object) (client.Object, bool, error) {
	schema, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	err = aivenClient.KafkaSubjectSchemas.Delete(schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.Schema)
	if err != nil && !aiven.IsNotFound(err) {
		return nil, false, fmt.Errorf("aiven client delete Kafka Schema error: %w", err)
	}

	log.Info("Successfully finalized Kafka Schema")

	return nil, true, nil
}

func (h KafkaSchemaHandler) exists(_ logr.Logger, i client.Object) (bool, error) {
	schema, err := h.convert(i)
	if err != nil {
		return false, err
	}

	return schema.Status.Version != 0, nil
}

func (h KafkaSchemaHandler) update(log logr.Logger, i client.Object) (client.Object, error) {
	return h.create(log, i)
}

func (h KafkaSchemaHandler) getSecret(_ logr.Logger, _ client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h KafkaSchemaHandler) checkPreconditions(_ logr.Logger, i client.Object) bool {
	schema, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(schema.Spec.Project, schema.Spec.ServiceName)
}

func (h KafkaSchemaHandler) isActive(_ logr.Logger, _ client.Object) (bool, error) {
	return true, nil
}

func (h KafkaSchemaHandler) setStatus(schema *k8soperatorv1alpha1.KafkaSchema, v int) {
	schema.Status.Project = schema.Spec.Project
	schema.Status.ServiceName = schema.Spec.ServiceName
	schema.Status.SubjectName = schema.Spec.SubjectName
	schema.Status.Schema = schema.Spec.Schema
	schema.Status.CompatibilityLevel = schema.Spec.CompatibilityLevel
	schema.Status.Version = v
}

func (h KafkaSchemaHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaSchema, error) {
	schema, ok := i.(*k8soperatorv1alpha1.KafkaSchema)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaSchema")
	}

	return schema, nil
}

func (h KafkaSchemaHandler) getLastVersion(schema *k8soperatorv1alpha1.KafkaSchema) (int, error) {
	ver, err := aivenClient.KafkaSubjectSchemas.GetVersions(
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
