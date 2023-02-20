// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var _ = Describe("Kafka Topic Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafka       *v1alpha1.Kafka
		topic       *v1alpha1.KafkaTopic
		serviceName string
		topicName   string
		ctx         = context.Background()
	)

	BeforeEach(func() {
		serviceName = "k8s-test-kafka-topic-acc-" + generateRandomID()
		topicName = "k8s-test-topic-acc-" + generateRandomID()
		kafka = kafkaSpec(serviceName, namespace)
		topic = kafkaTopicSpec(serviceName, topicName, namespace)

		By("Creating a new Kafka instance")
		Expect(k8sClient.Create(ctx, kafka)).Should(Succeed())

		By("Creating a new KafkaTopic instance")
		Expect(k8sClient.Create(ctx, topic)).Should(Succeed())

		lookupKey := types.NamespacedName{Name: topicName, Namespace: namespace}
		createdTopic := &v1alpha1.KafkaTopic{}
		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving Kafka Topic instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, createdTopic)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Kafka Topic to become ACTIVE")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, lookupKey, createdTopic)
			if err == nil {
				return meta.IsStatusConditionTrue(createdTopic.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdTopic.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Kafka Topic reconciler behaviour", func() {
		It("should createOrUpdate a new Kafka Topic", func() {
			createdTopic := &v1alpha1.KafkaTopic{}
			lookupKey := types.NamespacedName{Name: topicName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdTopic)).Should(Succeed())

			By("by checking that after KafkaTopic was created")
			Expect(meta.IsStatusConditionTrue(createdTopic.Status.Conditions, conditionTypeRunning)).Should(BeTrue())
			Expect(createdTopic.Status.State).Should(Equal("ACTIVE"))

			By("by checking MinCleanableDirtyRatio")
			Expect(*createdTopic.Spec.Config.MinCleanableDirtyRatio).Should(Equal(0.5))
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka Topic instance was deleted")
		ensureDelete(ctx, topic)
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
	})
})

func kafkaTopicSpec(service, topic, namespace string) *v1alpha1.KafkaTopic {
	return &v1alpha1.KafkaTopic{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "KafkaTopic",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      topic,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaTopicSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			Partitions:  3,
			Replication: 2,
			Tags: []v1alpha1.KafkaTopicTag{
				{
					Key:   "key1",
					Value: "val1",
				},
			},
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
			Config: v1alpha1.KafkaTopicConfig{
				MinCleanableDirtyRatio: anyPointer(0.5),
			},
		},
	}
}
