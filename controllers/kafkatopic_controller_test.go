package controllers

import (
	"context"
	"github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"
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
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafka-topic-acc-" + generateRandomID()
		topicName = "k8s-test-topic-acc-" + generateRandomID()
		kafka = kafkaSpec(serviceName, namespace)
		topic = kafkaTopicSpec(serviceName, topicName, namespace)

		By("Creating a new Kafka instance")
		Expect(k8sClient.Create(ctx, kafka)).Should(Succeed())

		By("by waiting Kafka service status to become RUNNING")
		Eventually(func() string {
			kafkaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
			createdKafka := &v1alpha1.Kafka{}
			err := k8sClient.Get(ctx, kafkaLookupKey, createdKafka)
			if err == nil {
				return createdKafka.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

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
		Eventually(func() string {
			err := k8sClient.Get(ctx, lookupKey, createdTopic)
			if err == nil {
				return createdTopic.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("ACTIVE"))
	})

	Context("Validating Kafka reconciler behaviour", func() {
		It("should create a new Kafka Topic", func() {
			createdTopic := &v1alpha1.KafkaTopic{}
			lookupKey := types.NamespacedName{Name: topicName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdTopic)).Should(Succeed())

			// Let's make sure our Kafka Topic status was properly populated.
			By("by checking that after creation KafkaTopic status fields were properly populated")
			Expect(createdTopic.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdTopic.Status.State).Should(Equal("ACTIVE"))
			Expect(createdTopic.Status.Project).Should(Equal(os.Getenv("AIVEN_PROJECT_NAME")))
			Expect(createdTopic.Status.TopicName).Should(Equal(topicName))
			Expect(createdTopic.Status.Partitions).Should(Equal(3))
			Expect(createdTopic.Status.Replication).Should(Equal(2))
			Expect(len(createdTopic.Status.Tags)).Should(Equal(1))
			Expect(createdTopic.Status.Tags[0].Key).Should(Equal("key1"))
			Expect(createdTopic.Status.Tags[0].Value).Should(Equal("val1"))

			By("by checking finalizers")
			Expect(createdTopic.GetFinalizers()).ToNot(BeEmpty())
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
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "KafkaTopic",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      topic,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaTopicSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			TopicName:   topic,
			Partitions:  3,
			Replication: 2,
			Tags: []v1alpha1.KafkaTopicTag{
				{
					Key:   "key1",
					Value: "val1",
				},
			},
		},
	}
}
