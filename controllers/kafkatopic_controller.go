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

// KafkaTopicReconciler reconciles a KafkaTopic object
type KafkaTopicReconciler struct {
	Controller
}

type KafkaTopicHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/status,verbs=get;update;patch

func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaTopicHandler{}, &v1alpha1.KafkaTopic{})
}

func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaTopic{}).
		Complete(r)
}

func (h KafkaTopicHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	topic, err := h.convert(obj)
	if err != nil {
		return err
	}

	var tags []aiven.KafkaTopicTag
	for _, t := range topic.Spec.Tags {
		tags = append(tags, aiven.KafkaTopicTag{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	exists, err := h.exists(ctx, avn, topic)
	if err != nil {
		return err
	}

	var reason string
	if !exists {
		err = avn.KafkaTopics.Create(ctx, topic.Spec.Project, topic.Spec.ServiceName, aiven.CreateKafkaTopicRequest{
			Partitions:  &topic.Spec.Partitions,
			Replication: &topic.Spec.Replication,
			TopicName:   topic.GetTopicName(),
			Tags:        tags,
			Config:      convertKafkaTopicConfig(topic),
		})
		if err != nil && !isAlreadyExists(err) {
			return err
		}

		reason = "Created"
	} else {
		err = avn.KafkaTopics.Update(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName(),
			aiven.UpdateKafkaTopicRequest{
				Partitions:  &topic.Spec.Partitions,
				Replication: &topic.Spec.Replication,
				Tags:        tags,
				Config:      convertKafkaTopicConfig(topic),
			})
		if err != nil {
			return fmt.Errorf("cannot update Kafka Topic: %w", err)
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
		processedGenerationAnnotation, strconv.FormatInt(topic.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaTopicHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if fromAnyPointer(topic.Spec.TerminationProtection) {
		return false, errTerminationProtectionOn
	}

	// Delete project on Aiven side
	err = avn.KafkaTopics.Delete(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h KafkaTopicHandler) exists(ctx context.Context, avn *aiven.Client, topic *v1alpha1.KafkaTopic) (bool, error) {
	t, err := avn.KafkaTopics.Get(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())
	if err != nil && !isNotFound(err) {
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

func (h KafkaTopicHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	state, err := h.getState(ctx, avn, topic)
	if err != nil {
		return nil, err
	}

	topic.Status.State = state

	if state == "ACTIVE" {
		meta.SetStatusCondition(&topic.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&topic.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return nil, err
}

func (h KafkaTopicHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&topic.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(ctx, avn, avnGen, topic.Spec.Project, topic.Spec.ServiceName)
}

func (h KafkaTopicHandler) getState(ctx context.Context, avn *aiven.Client, topic *v1alpha1.KafkaTopic) (string, error) {
	t, err := avn.KafkaTopics.Get(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())
	if err != nil {
		if aivenError, ok := err.(aiven.Error); ok {
			// Getting topic info can sometimes temporarily fail with 501 and 502. Don't
			// treat that as fatal error but keep on retrying instead.
			if aivenError.Status == 501 || aivenError.Status == 502 {
				return "", nil
			}
		}
		return "", err
	}
	return t.State, nil
}

func (h KafkaTopicHandler) convert(i client.Object) (*v1alpha1.KafkaTopic, error) {
	topic, ok := i.(*v1alpha1.KafkaTopic)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaTopic")
	}

	return topic, nil
}

func convertKafkaTopicConfig(topic *v1alpha1.KafkaTopic) aiven.KafkaTopicConfig {
	return aiven.KafkaTopicConfig{
		CleanupPolicy:                   topic.Spec.Config.CleanupPolicy,
		CompressionType:                 topic.Spec.Config.CompressionType,
		DeleteRetentionMs:               topic.Spec.Config.DeleteRetentionMs,
		FileDeleteDelayMs:               topic.Spec.Config.FileDeleteDelayMs,
		FlushMessages:                   topic.Spec.Config.FlushMessages,
		FlushMs:                         topic.Spec.Config.FlushMs,
		IndexIntervalBytes:              topic.Spec.Config.IndexIntervalBytes,
		LocalRetentionBytes:             topic.Spec.Config.LocalRetentionBytes,
		LocalRetentionMs:                topic.Spec.Config.LocalRetentionMs,
		MaxCompactionLagMs:              topic.Spec.Config.MaxCompactionLagMs,
		MaxMessageBytes:                 topic.Spec.Config.MaxMessageBytes,
		MessageDownconversionEnable:     topic.Spec.Config.MessageDownconversionEnable,
		MessageFormatVersion:            topic.Spec.Config.MessageFormatVersion,
		MessageTimestampDifferenceMaxMs: topic.Spec.Config.MessageTimestampDifferenceMaxMs,
		MessageTimestampType:            topic.Spec.Config.MessageTimestampType,
		MinCleanableDirtyRatio:          topic.Spec.Config.MinCleanableDirtyRatio,
		MinCompactionLagMs:              topic.Spec.Config.MinCompactionLagMs,
		MinInsyncReplicas:               topic.Spec.Config.MinInsyncReplicas,
		Preallocate:                     topic.Spec.Config.Preallocate,
		RemoteStorageEnable:             topic.Spec.Config.RemoteStorageEnable,
		RetentionBytes:                  topic.Spec.Config.RetentionBytes,
		RetentionMs:                     topic.Spec.Config.RetentionMs,
		SegmentBytes:                    topic.Spec.Config.SegmentBytes,
		SegmentIndexBytes:               topic.Spec.Config.SegmentIndexBytes,
		SegmentJitterMs:                 topic.Spec.Config.SegmentJitterMs,
		SegmentMs:                       topic.Spec.Config.SegmentMs,
	}
}
