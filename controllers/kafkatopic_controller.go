// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	"github.com/aiven/go-client-codegen/handler/service"
	"golang.org/x/sync/singleflight"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const kafkaTopicMaxConcurrentReconciles = 20

func newKafkaTopicReconciler(c Controller) reconcilerType {
	return &Reconciler[*v1alpha1.KafkaTopic]{
		Controller:              c,
		newAivenGeneratedClient: NewAivenGeneratedClient,
		newObj: func() *v1alpha1.KafkaTopic {
			return &v1alpha1.KafkaTopic{}
		},
		newController: func(avnGen avngen.Client) AivenController[*v1alpha1.KafkaTopic] {
			return &KafkaTopicController{
				Client: c.Client,
				avnGen: avnGen,
			}
		},
		customizeBuilder: func(b *ctrlbuilder.Builder) *ctrlbuilder.Builder {
			return b.WithOptions(controller.Options{MaxConcurrentReconciles: kafkaTopicMaxConcurrentReconciles})
		},
	}
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkatopics/finalizers,verbs=get;create;update

// KafkaTopicController reconciles a KafkaTopic object
type KafkaTopicController struct {
	client.Client
	avnGen avngen.Client
}

// singleflight group for ServiceKafkaTopicList calls
var topicListCallGroup singleflight.Group

func (r *KafkaTopicController) Observe(ctx context.Context, topic *v1alpha1.KafkaTopic) (Observation, error) {
	if err := r.checkPreconditions(ctx, topic); err != nil {
		return Observation{}, err
	}

	callKey := fmt.Sprintf("%s/%s", topic.Spec.Project, topic.Spec.ServiceName)
	targetTopicName := topic.GetTopicName()

	// let requeuing handle retries
	result, err, _ := topicListCallGroup.Do(callKey, func() (any, error) {
		return r.avnGen.ServiceKafkaTopicList(ctx, topic.Spec.Project, topic.Spec.ServiceName)
	})

	switch {
	case isServerError(err):
		// Getting topic info can sometimes temporarily fail with 5xx.
		// Don't treat that as a fatal error but keep on retrying instead.
		// When this happens during a spec update, assume the topic exists if it was applied before.
		return Observation{
			ResourceExists:   wasEverApplied(topic) || hasIsRunningAnnotation(topic),
			ResourceUpToDate: hasLatestGeneration(topic),
		}, nil
	case err != nil:
		return Observation{}, err
	}

	topicList, ok := result.([]kafkatopic.TopicOut)
	if !ok {
		return Observation{}, fmt.Errorf("unexpected result type from ServiceKafkaTopicList") // this should not happen
	}

	for _, topicInfo := range topicList {
		if topicInfo.TopicName != targetTopicName {
			continue
		}

		topic.Status.State = topicInfo.State
		if topic.Status.State == kafkatopic.TopicStateTypeActive {
			meta.SetStatusCondition(&topic.Status.Conditions,
				getRunningCondition(metav1.ConditionTrue, "CheckRunning",
					"Instance is running on Aiven side"))

			metav1.SetMetaDataAnnotation(&topic.ObjectMeta, instanceIsRunningAnnotation, "true")
		}

		return Observation{
			ResourceExists:   true,
			ResourceUpToDate: hasLatestGeneration(topic),
		}, nil
	}

	// Topic not found in list.
	// Treat this as existence drift: reconcile will Create.
	return Observation{ResourceExists: false}, nil
}

func (r *KafkaTopicController) Create(ctx context.Context, topic *v1alpha1.KafkaTopic) (CreateResult, error) {
	delete(topic.GetAnnotations(), instanceIsRunningAnnotation)

	tags := make([]kafkatopic.TagIn, 0, len(topic.Spec.Tags))
	for _, t := range topic.Spec.Tags {
		tags = append(tags, kafkatopic.TagIn{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err := r.avnGen.ServiceKafkaTopicCreate(ctx, topic.Spec.Project, topic.Spec.ServiceName, &kafkatopic.ServiceKafkaTopicCreateIn{
		Partitions:  &topic.Spec.Partitions,
		Replication: &topic.Spec.Replication,
		TopicName:   topic.GetTopicName(),
		Tags:        &tags,
		Config:      convertKafkaTopicConfig(topic),
	})
	if err != nil {
		return CreateResult{}, fmt.Errorf("creating Kafka topic: %w", err)
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&topic.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&topic.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return CreateResult{}, nil
}

func (r *KafkaTopicController) Update(ctx context.Context, topic *v1alpha1.KafkaTopic) (UpdateResult, error) {
	delete(topic.GetAnnotations(), instanceIsRunningAnnotation)

	tags := make([]kafkatopic.TagIn, 0, len(topic.Spec.Tags))
	for _, t := range topic.Spec.Tags {
		tags = append(tags, kafkatopic.TagIn{
			Key:   t.Key,
			Value: t.Value,
		})
	}

	err := r.avnGen.ServiceKafkaTopicUpdate(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName(),
		&kafkatopic.ServiceKafkaTopicUpdateIn{
			Partitions:  &topic.Spec.Partitions,
			Replication: &topic.Spec.Replication,
			Tags:        &tags,
			Config:      convertKafkaTopicConfig(topic),
		})
	if err != nil {
		return UpdateResult{}, fmt.Errorf("cannot update Kafka Topic: %w", err)
	}

	const reason = "CreatedOrUpdated"
	meta.SetStatusCondition(&topic.Status.Conditions, getInitializedCondition(reason, "Successfully created or updated the instance in Aiven"))
	meta.SetStatusCondition(&topic.Status.Conditions, getRunningCondition(metav1.ConditionUnknown, reason, "Successfully created or updated the instance in Aiven, status remains unknown"))

	return UpdateResult{}, nil
}

func (r *KafkaTopicController) Delete(ctx context.Context, topic *v1alpha1.KafkaTopic) error {
	if fromAnyPointer(topic.Spec.TerminationProtection) {
		return errTerminationProtectionOn
	}

	err := r.avnGen.ServiceKafkaTopicDelete(ctx, topic.Spec.Project, topic.Spec.ServiceName, topic.GetTopicName())
	if err != nil && !isNotFound(err) {
		return err
	}

	return nil
}

func (r *KafkaTopicController) checkPreconditions(ctx context.Context, topic *v1alpha1.KafkaTopic) error {
	s, err := r.avnGen.ServiceGet(ctx, topic.Spec.Project, topic.Spec.ServiceName)
	if isNotFound(err) {
		return errPreconditionNotMet
	}
	if err != nil {
		return err
	}

	running := 0
	for _, node := range s.NodeStates {
		if node.State == service.NodeStateTypeRunning {
			running++
		}
	}

	// Replication factor requires enough nodes running.
	// But we want to get the backend validation error if the value is too high.
	if running < min(len(s.NodeStates), topic.Spec.Replication) {
		return errPreconditionNotMet
	}

	return nil
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
		DisklessEnable:                  topic.Spec.Config.DisklessEnable,
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
