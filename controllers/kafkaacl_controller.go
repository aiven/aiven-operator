// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const kafkaACLFinalizer = "kafkaacl-finalizer.k8s-operator.aiven.io"

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaacls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkaacls/status,verbs=get;update;patch

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkaacl", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Kafka ACL instance
	acl := &k8soperatorv1alpha1.KafkaACL{}
	err := r.Get(ctx, req.NamespacedName, acl)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Kafka ACL resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Kafka ACL")
		return ctrl.Result{}, err
	}

	// Check if the Kafka ACL instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isKafkaACLMarkedToBeDeleted := acl.GetDeletionTimestamp() != nil
	if isKafkaACLMarkedToBeDeleted {
		if contains(acl.GetFinalizers(), kafkaACLFinalizer) {
			// Run finalization logic for kafkaACLFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalize(log, acl); err != nil {
				return reconcile.Result{}, err
			}

			// Remove kafkaACLFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(acl, kafkaACLFinalizer)
			err := r.Update(ctx, acl)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(acl.GetFinalizers(), kafkaACLFinalizer) {
		if err := r.addFinalizer(log, acl); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Kafka ACL already exists on the Aiven side, create a
	// new one if it is not found
	isACLExists, err := r.exists(acl)
	if err != nil {
		log.Error(err, "Failed to check if Kafka ACL exists")
		return ctrl.Result{}, err
	}

	// r.exists() updates status, so r.unchanged() has up-to-date info
	if isACLExists && r.unchanged(acl) {
		return ctrl.Result{}, nil
	}

	if isACLExists {
		// This is an update, so delete the old ACL first
		err := r.deleteACL(log, acl)
		if err != nil {
			log.Error(err, "Failed to delete old Kafka ACL")
			return ctrl.Result{}, err
		}
	}

	// Create a new Kafka ACL
	if err := r.createACL(acl); err != nil {
		log.Error(err, "Failed to create Kafka ACL")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KafkaACLReconciler) deleteACL(log logr.Logger, a *k8soperatorv1alpha1.KafkaACL) error {
	err := r.AivenClient.KafkaACLs.Delete(a.Status.Project, a.Status.ServiceName, a.Status.Id)
	if err != nil && !aiven.IsNotFound(err) {
		log.Error(err, "Cannot delete Kafka ACL")
		return fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}
	return nil
}

// finalize deletes Aiven Kafka ACL
func (r *KafkaACLReconciler) finalize(log logr.Logger, a *k8soperatorv1alpha1.KafkaACL) error {
	r.deleteACL(log, a)
	log.Info("Successfully finalized Kafka ACL")
	return nil
}

// addFinalizer adds finalizer to CR
func (r *KafkaACLReconciler) addFinalizer(reqLogger logr.Logger, a *k8soperatorv1alpha1.KafkaACL) error {
	reqLogger.Info("Adding Finalizer to Kafka ACL")
	controllerutil.AddFinalizer(a, kafkaACLFinalizer)

	// Update CR
	return r.Update(context.Background(), a)
}

func (r *KafkaACLReconciler) exists(acl *k8soperatorv1alpha1.KafkaACL) (bool, error) {
	var aivenACL *aiven.KafkaACL
	if acl.Status.Id != "" {
		var err error
		aivenACL, err = r.AivenClient.KafkaACLs.Get(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.Id)
		if err != nil {
			return false, err
		}
	} else {
		list, err := r.AivenClient.KafkaACLs.List(acl.Spec.Project, acl.Spec.ServiceName)
		if err != nil {
			return false, err
		}

		for _, a := range list {
			if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
				aivenACL = a
			}
		}
	}

	if aivenACL != nil {
		return true, r.updateCRStatus(acl, aivenACL)
	}

	return false, nil
}

func (r *KafkaACLReconciler) createACL(acl *k8soperatorv1alpha1.KafkaACL) error {
	a, err := r.AivenClient.KafkaACLs.Create(
		acl.Spec.Project,
		acl.Spec.ServiceName,
		aiven.CreateKafkaACLRequest{
			Permission: acl.Spec.Permission,
			Topic:      acl.Spec.Topic,
			Username:   acl.Spec.Username,
		},
	)
	if err != nil {
		return err
	}

	// Update custom resource status
	return r.updateCRStatus(acl, a)
}

func (r *KafkaACLReconciler) unchanged(acl *k8soperatorv1alpha1.KafkaACL) bool {
	return acl.Status.Username == acl.Spec.Username &&
		acl.Status.Permission == acl.Spec.Permission &&
		acl.Status.Topic == acl.Spec.Topic
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *KafkaACLReconciler) updateCRStatus(acl *k8soperatorv1alpha1.KafkaACL, a *aiven.KafkaACL) error {
	acl.Status.Project = acl.Spec.Project
	acl.Status.ServiceName = acl.Spec.ServiceName
	acl.Status.Username = a.Username
	acl.Status.Permission = a.Permission
	acl.Status.Topic = a.Topic
	acl.Status.Id = a.ID

	err := r.Status().Update(context.Background(), acl)
	if err != nil {
		return fmt.Errorf("failed to update Kafka ACL status: %w", err)
	}

	return nil
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaACL{}).
		Complete(r)
}
