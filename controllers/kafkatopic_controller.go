// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/aiven/go-client-codegen/handler/service"
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

func newKafkaTopicReconciler(c Controller) reconcilerType {
	return &KafkaTopicReconciler{Controller: c}
}

type KafkaTopicHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/finalizers,verbs=get;create;update

func (r *KafkaTopicReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaTopicHandler{}, &v1alpha1.KafkaTopic{})
}

func (r *KafkaTopicReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaTopic{}).
		Complete(r)
}

func (h KafkaTopicHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) (bool, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	tags := make([]kafkatopic.TagIn, 0, len(topic.Spec.Tags))
	for _, t := range topic.Spec.Tags {
		tags = append(tags, kafkatopic.TagIn{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	// ServiceKafkaTopicGet quite often fails with 5xx errors.
	// So instead of trying to get the topic info, we'll just create it.
	// If the topic already exists, we'll update it.
	err = avnGen.ServiceKafkaTopicCreate(ctx, topic.Spec.Project, topic.Spec.ServiceName, &kafkatopic.ServiceKafkaTopicCreateIn{
		Partitions:  &topic.Spec.Partitions,
		Replication: &topic.Spec.Replication,
		TopicName:   topic.GetTopicName(),
		Tags:        &tags,
		Config:      convertKafkaTopicConfig(topic),
	})

	exists := isAlreadyExists(err)
	if exists {
		err = avnGen.ServiceKafkaTopicUpdate(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName(),
			&kafkatopic.ServiceKafkaTopicUpdateIn{
				Partitions:  &topic.Spec.Partitions,
				Replication: &topic.Spec.Replication,
				Tags:        &tags,
				Config:      convertKafkaTopicConfig(topic),
			})
		if err != nil {
			return false, fmt.Errorf("cannot update Kafka Topic: %w", err)
		}
	}

	switch {
	case isServerError(err):
		// Service is not ready yet, retry later.
		return false, nil
	case err != nil:
		return false, err
	}

	return !exists, nil
}

func (h KafkaTopicHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	if fromAnyPointer(topic.Spec.TerminationProtection) {
		return false, errTerminationProtectionOn
	}

	// Delete project on Aiven side
	err = avnGen.ServiceKafkaTopicDelete(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h KafkaTopicHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	avnTopic, err := avnGen.ServiceKafkaTopicGet(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())

	switch {
	case isServerError(err):
		// Getting topic info can sometimes temporarily fail with 5xx.
		// Don't treat that as a fatal error but keep on retrying instead.
		return nil, nil
	case err != nil:
		return nil, err
	}

	topic.Status.State = avnTopic.State
	if topic.Status.State == kafkatopic.TopicStateTypeActive {
		meta.SetStatusCondition(&topic.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		metav1.SetMetaDataAnnotation(&topic.ObjectMeta, instanceIsRunningAnnotation, "true")
	}

	return nil, err
}

func (h KafkaTopicHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	topic, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&topic.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	s, err := avnGen.ServiceGet(ctx, topic.Spec.Project, topic.Spec.ServiceName)
	if isNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	running := 0
	for _, node := range s.NodeStates {
		if node.State == service.NodeStateTypeRunning {
			running++
		}
	}
	// Replication factor requires enough nodes running.
	// But we want to get the backend validation error if the value is too high
	return running >= min(len(s.NodeStates), topic.Spec.Replication), nil
}

func (h KafkaTopicHandler) convert(i client.Object) (*v1alpha1.KafkaTopic, error) {
	topic, ok := i.(*v1alpha1.KafkaTopic)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaTopic")
	}

	return topic, nil
}

func convertKafkaTopicConfig(topic *v1alpha1.KafkaTopic) *kafkatopic.ConfigIn {
	if topic.Spec.Config == nil {
		return nil
	}

	return &kafkatopic.ConfigIn{
		CleanupPolicy:                   topic.Spec.Config.CleanupPolicy,
		CompressionType:                 topic.Spec.Config.CompressionType,
		DeleteRetentionMs:               topic.Spec.Config.DeleteRetentionMs,
		FileDeleteDelayMs:               topic.Spec.Config.FileDeleteDelayMs,
		FlushMessages:                   topic.Spec.Config.FlushMessages,
		FlushMs:                         topic.Spec.Config.FlushMs,
		IndexIntervalBytes:              topic.Spec.Config.IndexIntervalBytes,
		InklessEnable:                   topic.Spec.Config.InklessEnable,
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
		UncleanLeaderElectionEnable:     topic.Spec.Config.UncleanLeaderElectionEnable,
	}
}
