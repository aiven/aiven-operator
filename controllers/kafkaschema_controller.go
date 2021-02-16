// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"k8s.io/apimachinery/pkg/api/errors"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas/status,verbs=get;update;patch

func (r *KafkaSchemaReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("kafkaschema", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the KafkaSchema instance
	schema := &k8soperatorv1alpha1.KafkaSchema{}
	err := r.Get(ctx, req.NamespacedName, schema)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("KafkaSchema resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get KafkaSchema")
		return ctrl.Result{}, err
	}

	// Check if Kafka Schema already exists on the Aiven side, create a
	// new one if it is not found
	schemaExists, err := r.exists(schema.Spec.Project, schema.Spec.ServiceName, schema.Spec.SubjectName)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create a new project if project does not exists and update CR status
	if !schemaExists {
		err = r.create(schema)
		if err != nil {
			log.Error(err, "Failed to create Kafka Schema")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Update Kafka Schema via API and update CR status
	err = r.update(schema)
	if err != nil {
		log.Error(err, "Failed to update Kafka Schema")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KafkaSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaSchema{}).
		Complete(r)
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *KafkaSchemaReconciler) updateCRStatus(schema *k8soperatorv1alpha1.KafkaSchema, v int) error {
	schema.Status.Project = schema.Spec.Project
	schema.Status.ServiceName = schema.Spec.ServiceName
	schema.Status.SubjectName = schema.Spec.SubjectName
	schema.Status.CompatibilityLevel = schema.Spec.CompatibilityLevel
	schema.Status.Version = v

	return r.Status().Update(context.Background(), schema)
}

func (r *KafkaSchemaReconciler) getLastVersion(schema *k8soperatorv1alpha1.KafkaSchema) (int, error) {
	ver, err := r.AivenClient.KafkaSubjectSchemas.GetVersions(
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

func (r *KafkaSchemaReconciler) exists(project, serviceName, subjectName string) (bool, error) {
	l, err := r.AivenClient.KafkaSubjectSchemas.List(project, serviceName)
	if err != nil {
		return false, err
	}

	for _, s := range l.Subjects {
		if s == subjectName {
			return true, nil
		}
	}

	return false, nil
}

func (r *KafkaSchemaReconciler) update(schema *k8soperatorv1alpha1.KafkaSchema) error {
	// create Kafka Schema Subject
	_, err := r.AivenClient.KafkaSubjectSchemas.Add(
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		aiven.KafkaSchemaSubject{
			Schema: schema.Spec.Schema,
		},
	)
	if err != nil {
		return err
	}

	// set compatibility level if defined for a newly created Kafka Schema Subject
	if schema.Spec.CompatibilityLevel != "" {
		_, err := r.AivenClient.KafkaSubjectSchemas.UpdateConfiguration(
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			schema.Spec.CompatibilityLevel,
		)
		if err != nil {
			return err
		}
	}

	// get last version
	version, err := r.getLastVersion(schema)
	if err != nil {
		return err
	}

	err = r.updateCRStatus(schema, version)
	if err != nil {
		return err
	}

	return nil
}

func (r *KafkaSchemaReconciler) create(schema *k8soperatorv1alpha1.KafkaSchema) error {
	// create Kafka Schema Subject
	_, err := r.AivenClient.KafkaSubjectSchemas.Add(
		schema.Spec.Project,
		schema.Spec.ServiceName,
		schema.Spec.SubjectName,
		aiven.KafkaSchemaSubject{
			Schema: schema.Spec.Schema,
		},
	)
	if err != nil {
		return err
	}

	// set compatibility level if defined for a newly created Kafka Schema Subject
	if schema.Spec.CompatibilityLevel != "" {
		_, err := r.AivenClient.KafkaSubjectSchemas.UpdateConfiguration(
			schema.Spec.Project,
			schema.Spec.ServiceName,
			schema.Spec.SubjectName,
			schema.Spec.CompatibilityLevel,
		)
		if err != nil {
			return err
		}
	}

	// get last version
	version, err := r.getLastVersion(schema)
	if err != nil {
		return err
	}

	// newly created versions start from 1
	if version == 0 {
		return fmt.Errorf("kafka schema subject after creation has an empty list of versions")
	}

	err = r.updateCRStatus(schema, version)
	if err != nil {
		return err
	}

	return nil
}
