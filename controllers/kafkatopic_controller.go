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
	"time"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// KafkaTopicReconciler reconciles a KafkaTopic object
type KafkaTopicReconciler struct {
	Controller
}

const kafkaTopicFinalizer = "kafkatopic-finalizer.k8s-operator.aiven.io"

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=kafkatopics/status,verbs=get;update;patch

func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkatopic", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the KafkaTopic instance
	topic := &k8soperatorv1alpha1.KafkaTopic{}
	err := r.Get(ctx, req.NamespacedName, topic)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("KafkaTopic resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get KafkaTopic")
		return ctrl.Result{}, err
	}

	// Check if the Kafka Topic instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isKafkaTopicMarkedToBeDeleted := topic.GetDeletionTimestamp() != nil
	if isKafkaTopicMarkedToBeDeleted {
		if contains(topic.GetFinalizers(), kafkaTopicFinalizer) {
			// Run finalization logic for kafkaTopicFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalize(log, topic); err != nil {
				return reconcile.Result{}, err
			}

			// Remove kafkaTopicFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(topic, kafkaTopicFinalizer)
			err := r.Client.Update(ctx, topic)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(topic.GetFinalizers(), kafkaTopicFinalizer) {
		if err := r.addFinalizer(log, topic); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if Kafka Topic already exists on the Aiven side, createTopic a
	// new one if it is not found
	t, err := r.AivenClient.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil {
		if aivenError, ok := err.(aiven.Error); ok {
			// Getting topic info can sometimes temporarily fail with 501 and 502. Don't
			// treat that as fatal error but keep on retrying instead.
			if aivenError.Status == 501 || aivenError.Status == 502 {
				return ctrl.Result{
					Requeue:      true,
					RequeueAfter: 10 * time.Second,
				}, nil
			}
		}

		// Create a new Kafka Topic if it does not exists and update CR status
		if aiven.IsNotFound(err) {
			err = r.createTopic(topic)
			if err != nil && !aiven.IsAlreadyExists(err) {
				log.Error(err, "Failed to createTopic KafkaTopic")
				return ctrl.Result{}, err
			}

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: 10 * time.Second,
			}, nil
		}
		return ctrl.Result{}, err
	}

	// Check Kafka Topic status and wait until it is ACTIVE
	if t != nil && t.State != "ACTIVE" {
		log.Info("Kafka Topic state is " + t.State + ", waiting to become ACTIVE")
		err = r.updateCRStatus(topic, t)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: 10 * time.Second,
		}, nil
	}

	return ctrl.Result{}, r.updateTopic(topic)
}

func (r *KafkaTopicReconciler) createTopic(topic *k8soperatorv1alpha1.KafkaTopic) error {
	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err := r.AivenClient.KafkaTopics.Create(topic.Spec.Project, topic.Spec.ServiceName, aiven.CreateKafkaTopicRequest{
		Partitions:  &topic.Spec.Partitions,
		Replication: &topic.Spec.Replication,
		TopicName:   topic.Spec.TopicName,
		Tags:        tags,
		Config:      convertKafkaTopicConfig(topic),
	})
	if err != nil {
		return err
	}

	t, err := r.AivenClient.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil {
		return err
	}

	return r.updateCRStatus(topic, t)
}

func (r *KafkaTopicReconciler) updateTopic(topic *k8soperatorv1alpha1.KafkaTopic) error {
	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err := r.AivenClient.KafkaTopics.Update(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName,
		aiven.UpdateKafkaTopicRequest{
			Partitions:  &topic.Spec.Partitions,
			Replication: &topic.Spec.Replication,
			Tags:        tags,
			Config:      convertKafkaTopicConfig(topic),
		})
	if err != nil {
		return fmt.Errorf("cannot update Kafka Topic: %w", err)
	}

	t, err := r.AivenClient.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil {
		return fmt.Errorf("cannot get Kafka Topic after update: %w", err)
	}

	return r.updateCRStatus(topic, t)
}

func convertKafkaTopicConfig(topic *k8soperatorv1alpha1.KafkaTopic) aiven.KafkaTopicConfig {
	return aiven.KafkaTopicConfig{
		CleanupPolicy:                   topic.Spec.Config.CleanupPolicy,
		CompressionType:                 topic.Spec.Config.CompressionType,
		DeleteRetentionMs:               topic.Spec.Config.DeleteRetentionMs,
		FileDeleteDelayMs:               topic.Spec.Config.FileDeleteDelayMs,
		FlushMessages:                   topic.Spec.Config.FlushMessages,
		FlushMs:                         topic.Spec.Config.FlushMs,
		IndexIntervalBytes:              topic.Spec.Config.IndexIntervalBytes,
		MaxCompactionLagMs:              topic.Spec.Config.MaxCompactionLagMs,
		MaxMessageBytes:                 topic.Spec.Config.MaxMessageBytes,
		MessageDownconversionEnable:     topic.Spec.Config.MessageDownconversionEnable,
		MessageFormatVersion:            topic.Spec.Config.MessageFormatVersion,
		MessageTimestampDifferenceMaxMs: topic.Spec.Config.MessageTimestampDifferenceMaxMs,
		MessageTimestampType:            topic.Spec.Config.MessageTimestampType,
		MinCompactionLagMs:              topic.Spec.Config.MinCompactionLagMs,
		MinInsyncReplicas:               topic.Spec.Config.MinInsyncReplicas,
		Preallocate:                     topic.Spec.Config.Preallocate,
		RetentionBytes:                  topic.Spec.Config.RetentionBytes,
		RetentionMs:                     topic.Spec.Config.RetentionMs,
		SegmentBytes:                    topic.Spec.Config.SegmentBytes,
		SegmentIndexBytes:               topic.Spec.Config.SegmentIndexBytes,
		SegmentJitterMs:                 topic.Spec.Config.SegmentJitterMs,
		SegmentMs:                       topic.Spec.Config.SegmentMs,
		UncleanLeaderElectionEnable:     topic.Spec.Config.UncleanLeaderElectionEnable,
	}
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *KafkaTopicReconciler) updateCRStatus(topic *k8soperatorv1alpha1.KafkaTopic, t *aiven.KafkaTopic) error {
	var tags []k8soperatorv1alpha1.KafkaTopicTag
	for _, tt := range t.Tags {
		tags = append(tags, k8soperatorv1alpha1.KafkaTopicTag{
			Key:   tt.Key,
			Value: tt.Value,
		})
	}

	topic.Status.Project = topic.Spec.Project
	topic.Status.ServiceName = topic.Spec.ServiceName
	topic.Status.TopicName = t.TopicName
	topic.Status.Partitions = len(t.Partitions)
	topic.Status.Replication = t.Replication
	topic.Status.Tags = tags
	topic.Status.State = t.State

	topic.Status.Config.CleanupPolicy = t.Config.CleanupPolicy.Value
	topic.Status.Config.CompressionType = t.Config.CompressionType.Value
	topic.Status.Config.DeleteRetentionMs = &t.Config.DeleteRetentionMs.Value
	topic.Status.Config.FileDeleteDelayMs = &t.Config.FileDeleteDelayMs.Value
	topic.Status.Config.FlushMessages = &t.Config.FlushMessages.Value
	topic.Status.Config.FlushMs = &t.Config.FlushMs.Value
	topic.Status.Config.IndexIntervalBytes = &t.Config.IndexIntervalBytes.Value
	topic.Status.Config.MaxCompactionLagMs = &t.Config.MaxCompactionLagMs.Value
	topic.Status.Config.MaxMessageBytes = &t.Config.MaxMessageBytes.Value
	topic.Status.Config.MessageDownconversionEnable = &t.Config.MessageDownconversionEnable.Value
	topic.Status.Config.MessageFormatVersion = t.Config.MessageFormatVersion.Value
	topic.Status.Config.MessageTimestampDifferenceMaxMs = &t.Config.MessageTimestampDifferenceMaxMs.Value
	topic.Status.Config.MessageTimestampType = t.Config.MessageTimestampType.Value

	// Float is currently not supported
	//topic.Status.Config.MinCleanableDirtyRatio = t.Config.MinCleanableDirtyRatio.Value
	topic.Status.Config.MinCompactionLagMs = &t.Config.MinCompactionLagMs.Value
	topic.Status.Config.MinInsyncReplicas = &t.Config.MinInsyncReplicas.Value
	topic.Status.Config.Preallocate = &t.Config.Preallocate.Value
	topic.Status.Config.RetentionBytes = &t.Config.RetentionBytes.Value
	topic.Status.Config.RetentionMs = &t.Config.RetentionMs.Value
	topic.Status.Config.SegmentBytes = &t.Config.SegmentBytes.Value
	topic.Status.Config.SegmentIndexBytes = &t.Config.SegmentIndexBytes.Value
	topic.Status.Config.SegmentMs = &t.Config.SegmentMs.Value
	topic.Status.Config.SegmentJitterMs = &t.Config.SegmentJitterMs.Value
	topic.Status.Config.UncleanLeaderElectionEnable = &t.Config.UncleanLeaderElectionEnable.Value

	err := r.Status().Update(context.Background(), topic)
	if err != nil {
		return fmt.Errorf("cannot update CR status: %w", err)
	}

	return nil
}

// finalizeProject deletes Aiven Kafka Topic
func (r *KafkaTopicReconciler) finalize(log logr.Logger, t *k8soperatorv1alpha1.KafkaTopic) error {
	// Delete project on Aiven side
	err := r.AivenClient.KafkaTopics.Delete(t.Spec.Project, t.Spec.ServiceName, t.Spec.TopicName)
	if err != nil && !aiven.IsNotFound(err) {
		return err
	}

	log.Info("Successfully finalized Kafka Topic")
	return nil
}

// addFinalizer add finalizer to CR
func (r *KafkaTopicReconciler) addFinalizer(l logr.Logger, t *k8soperatorv1alpha1.KafkaTopic) error {
	l.Info("Adding Finalizer for the Kafka Topic")
	controllerutil.AddFinalizer(t, kafkaTopicFinalizer)

	// Update CR
	err := r.Client.Update(context.Background(), t)
	if err != nil {
		l.Error(err, "Failed to update KafkaTopic with finalizer")
		return err
	}
	return nil
}

func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaTopic{}).
		Complete(r)
}
