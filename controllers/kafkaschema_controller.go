// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const schemaFinalizer = "kafkaschema-finalizer.k8s-operator.aiven.io"

// KafkaSchemaReconciler reconciles a KafkaSchema object
type KafkaSchemaReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaschemas/status,verbs=get;update;patch

func (r *KafkaSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaschema", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the KafkaSchema instance
	schema := &k8soperatorv1alpha1.KafkaSchema{}
	err := r.Get(ctx, req.NamespacedName, schema)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("KafkaSchema resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get KafkaSchema")
		return ctrl.Result{}, err
	}

	// Check if the Kafka Schema instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isSchemaMarkedToBeDeleted := schema.GetDeletionTimestamp() != nil
	if isSchemaMarkedToBeDeleted {
		if contains(schema.GetFinalizers(), schemaFinalizer) {
			// Run finalization logic for schemaFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalize(log, schema); err != nil {
				return reconcile.Result{}, err
			}

			// Remove schemaFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(schema, schemaFinalizer)
			err := r.Client.Update(ctx, schema)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(schema.GetFinalizers(), schemaFinalizer) {
		if err := r.addFinalizer(log, schema); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Kafka Schema via API and update CR status
	err = r.addNewSchema(schema, ctx)
	if err != nil {
		if aiven.IsNotFound(err) ||
			strings.Contains(err.Error(), "Internal server error") ||
			strings.Contains(err.Error(), "Please try again later") {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Second * 5,
			}, nil
		}
		log.Error(err, "Failed to add Kafka Schema Subject")
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
func (r *KafkaSchemaReconciler) updateCRStatus(ctx context.Context, schema *k8soperatorv1alpha1.KafkaSchema, v int) error {
	schema.Status.Project = schema.Spec.Project
	schema.Status.ServiceName = schema.Spec.ServiceName
	schema.Status.SubjectName = schema.Spec.SubjectName
	schema.Status.Schema = schema.Spec.Schema
	schema.Status.CompatibilityLevel = schema.Spec.CompatibilityLevel
	schema.Status.Version = v

	return r.Status().Update(ctx, schema)
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

func (r *KafkaSchemaReconciler) addNewSchema(schema *k8soperatorv1alpha1.KafkaSchema, ctx context.Context) error {
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
		return fmt.Errorf("cannot add Kafka Schema Subject: %w", err)
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
			return fmt.Errorf("cannot update Kafka Schema Configuration: %w", err)
		}
	}

	// get last version
	version, err := r.getLastVersion(schema)
	if err != nil {
		return fmt.Errorf("cannot get Kafka Schema Subject version: %w", err)
	}

	err = r.updateCRStatus(ctx, schema, version)
	if err != nil {
		return fmt.Errorf("cannot udapte Kafka Schema Subject CR status: %w", err)
	}

	return nil
}

// finalizeProject deletes Aiven Kafka Schema
func (r *KafkaSchemaReconciler) finalize(log logr.Logger, s *k8soperatorv1alpha1.KafkaSchema) error {
	err := r.AivenClient.KafkaSubjectSchemas.Delete(s.Spec.Project, s.Spec.ServiceName, s.Spec.Schema)
	if err != nil {
		// If schema not found then there is nothing to delete
		if aiven.IsNotFound(err) {
			return nil
		}

		log.Error(err, "Cannot delete Kafka Schema")
		return fmt.Errorf("aiven client delete Kafka Schema error: %w", err)
	}

	log.Info("Successfully finalized Kafka Schema")
	return nil
}

// addFinalizer add finalizer to CR
func (r *KafkaSchemaReconciler) addFinalizer(reqLogger logr.Logger, s *k8soperatorv1alpha1.KafkaSchema) error {
	reqLogger.Info("Adding Finalizer for the Kafka Schema")
	controllerutil.AddFinalizer(s, schemaFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), s)
	if err != nil {
		reqLogger.Error(err, "Failed to update Kafka Schema with finalizer")
		return err
	}
	return nil
}
