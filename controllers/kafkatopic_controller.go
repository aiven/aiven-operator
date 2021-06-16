// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KafkaTopicReconciler reconciles a KafkaTopic object
type KafkaTopicReconciler struct {
	Controller
}

type KafkaTopicHandler struct {
	Handlers
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics,verbs=get;list;watch;createOrUpdate;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/status,verbs=get;update;patch

func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("kafkatopic", req.NamespacedName)

	log.Info("reconciling aiven kafka topic")

	const finalizer = "kafkatopic-finalizer.aiven.io"
	topic := &k8soperatorv1alpha1.KafkaTopic{}
	return r.reconcileInstance(ctx, req, &KafkaTopicHandler{}, topic)
}

func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaTopic{}).
		Complete(r)
}

func (h KafkaTopicHandler) createOrUpdate(i client.Object) error {
	topic, err := h.convert(i)
	if err != nil {
		return err
	}

	log.Info("creating a new kafka topic")

	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err = c.KafkaTopics.Create(topic.Spec.Project, topic.Spec.ServiceName, aiven.CreateKafkaTopicRequest{
		Partitions:  &topic.Spec.Partitions,
		Replication: &topic.Spec.Replication,
		TopicName:   topic.Spec.TopicName,
		Tags:        tags,
		Config:      convertKafkaTopicConfig(topic),
	})
	if err != nil && !aiven.IsAlreadyExists(err) {
		return err
	}

	t, err := c.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil {
		return err
	}

	h.setStatus(topic, t)

	return nil
}

func (h KafkaTopicHandler) delete(i client.Object) (bool, error) {
	topic, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	err = c.KafkaTopics.Delete(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	log.Info("successfully finalized kafka topic")

	return true, nil
}

func (h KafkaTopicHandler) exists(c *aiven.Client, _ logr.Logger, i client.Object) (bool, error) {
	topic, err := h.convert(i)
	if err != nil {
		return false, err
	}

	t, err := c.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil && !aiven.IsNotFound(err) {
		if aivenError, ok := err.(aiven.Error); ok {
			// Getting topic info can sometimes temporarily fail with 501 and 502. Don't
			// treat that as fatal error but keep on retrying instead.
			if aivenError.Status == 501 || aivenError.Status == 502 {
				return true, nil
			}
		}

		return false, err
	}

	return t != nil, nil
}

func (h KafkaTopicHandler) update(c *aiven.Client, log logr.Logger, i client.Object) (client.Object, error) {
	topic, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("updating a kafka topic")

	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err = c.KafkaTopics.Update(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName,
		aiven.UpdateKafkaTopicRequest{
			Partitions:  &topic.Spec.Partitions,
			Replication: &topic.Spec.Replication,
			Tags:        tags,
			Config:      convertKafkaTopicConfig(topic),
		})
	if err != nil {
		return nil, fmt.Errorf("cannot update Kafka Topic: %w", err)
	}

	t, err := c.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil {
		return nil, fmt.Errorf("cannot get Kafka Topic after update: %w", err)
	}

	h.setStatus(topic, t)

	return topic, nil
}

func (h KafkaTopicHandler) get(_ client.Object) (*corev1.Secret, error) {
	return nil, nil
}

func (h KafkaTopicHandler) checkPreconditions(i client.Object) bool {
	topic, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(c, topic.Spec.Project, topic.Spec.ServiceName)
}

func (h KafkaTopicHandler) isActive(c *aiven.Client, _ logr.Logger, i client.Object) (bool, error) {
	topic, err := h.convert(i)
	if err != nil {
		return false, err
	}

	t, err := c.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil && !aiven.IsNotFound(err) {
		if aivenError, ok := err.(aiven.Error); ok {
			// Getting topic info can sometimes temporarily fail with 501 and 502. Don't
			// treat that as fatal error but keep on retrying instead.
			if aivenError.Status == 501 || aivenError.Status == 502 {
				return false, nil
			}
		}

		return false, err
	}

	return t.State == "ACTIVE", nil
}

func (h KafkaTopicHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaTopic, error) {
	topic, ok := i.(*k8soperatorv1alpha1.KafkaTopic)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaTopic")
	}

	return topic, nil
}

func (h KafkaTopicHandler) setStatus(topic *k8soperatorv1alpha1.KafkaTopic, t *aiven.KafkaTopic) {
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

func (h KafkaTopicHandler) getSecretReference(i client.Object) *k8soperatorv1alpha1.AuthSecretReference {
	topic, err := h.convert(i)
	if err != nil {
		return nil
	}

	return &topic.Spec.AuthSecretRef
}
