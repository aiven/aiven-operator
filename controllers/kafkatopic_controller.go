// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// KafkaTopicReconciler reconciles a KafkaTopic object
type KafkaTopicReconciler struct {
	Controller
}

type KafkaTopicHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/status,verbs=get;update;patch

func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	topic := &k8soperatorv1alpha1.KafkaTopic{}
	err := r.Get(ctx, req.NamespacedName, topic)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, topic.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &KafkaTopicHandler{
		client: c,
	}, topic)
}

func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaTopic{}).
		Complete(r)
}

func (h KafkaTopicHandler) createOrUpdate(i client.Object) (client.Object, error) {
	topic, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	exists, err := h.exists(topic)
	if err != nil {
		return nil, err
	}

	var reason string
	if !exists {
		err = h.client.KafkaTopics.Create(topic.Spec.Project, topic.Spec.ServiceName, aiven.CreateKafkaTopicRequest{
			Partitions:  &topic.Spec.Partitions,
			Replication: &topic.Spec.Replication,
			TopicName:   topic.Spec.TopicName,
			Tags:        tags,
			Config:      convertKafkaTopicConfig(topic),
		})
		if err != nil && !aiven.IsAlreadyExists(err) {
			return nil, err
		}

		reason = "Created"
	} else {
		err = h.client.KafkaTopics.Update(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName,
			aiven.UpdateKafkaTopicRequest{
				Partitions:  &topic.Spec.Partitions,
				Replication: &topic.Spec.Replication,
				Tags:        tags,
				Config:      convertKafkaTopicConfig(topic),
			})
		if err != nil {
			return nil, fmt.Errorf("cannot update Kafka Topic: %w", err)
		}

		reason = "Updated"
	}

	meta.SetStatusCondition(&topic.Status.Conditions,
		getInitializedCondition(reason,
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&topic.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, reason,
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&topic.ObjectMeta,
		processedGeneration, strconv.FormatInt(topic.GetGeneration(), 10))

	return topic, nil
}

func (h KafkaTopicHandler) delete(i client.Object) (bool, error) {
	topic, err := h.convert(i)
	if err != nil {
		return false, err
	}

	// Delete project on Aiven side
	err = h.client.KafkaTopics.Delete(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
	if err != nil && !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h KafkaTopicHandler) exists(topic *k8soperatorv1alpha1.KafkaTopic) (bool, error) {
	t, err := h.client.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
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

func (h KafkaTopicHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	topic, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	isActive, err := h.isActive(topic)
	if err != nil {
		return nil, nil, err
	}

	if isActive {
		meta.SetStatusCondition(&topic.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "Get",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&topic.ObjectMeta, isRunning, "true")
	}

	return topic, nil, err
}

func (h KafkaTopicHandler) checkPreconditions(i client.Object) bool {
	topic, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(h.client, topic.Spec.Project, topic.Spec.ServiceName)
}

func (h KafkaTopicHandler) isActive(topic *k8soperatorv1alpha1.KafkaTopic) (bool, error) {
	t, err := h.client.KafkaTopics.Get(topic.Spec.Project, topic.Spec.ServiceName, topic.Spec.TopicName)
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
