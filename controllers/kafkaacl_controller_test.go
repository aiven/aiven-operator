package controllers

import (
	"context"
	"os"
	"time"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Kafka ACL Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafka       *v1alpha1.Kafka
		kafkaTopic  *v1alpha1.KafkaTopic
		acl         *v1alpha1.KafkaACL
		serviceName string
		topicName   string
		userName    string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafka-acl-acc-" + generateRandomID()
		topicName = "k8s-test-kafka-acl-acc-" + generateRandomID()
		userName = "k9s-acl1"
		kafka = kafkaSpec(serviceName, namespace)
		kafkaTopic = kafkaTopicSpec(serviceName, topicName, namespace)
		acl = kafkaACLSpec(serviceName, topicName, userName, namespace)

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
		Expect(k8sClient.Create(ctx, kafkaTopic)).Should(Succeed())

		By("by waiting Kafka Topic to become ACTIVE")
		Eventually(func() string {
			lookupKey := types.NamespacedName{Name: topicName, Namespace: namespace}
			createdTopic := &v1alpha1.KafkaTopic{}
			err := k8sClient.Get(ctx, lookupKey, createdTopic)
			if err == nil {
				return createdTopic.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("ACTIVE"))

		time.Sleep(5 * time.Second)

		By("Creating a new KafkaACL instance")
		Expect(k8sClient.Create(ctx, acl)).Should(Succeed())

		time.Sleep(5 * time.Second)
	})

	Context("Validating Kafka ACL reconciler behaviour", func() {
		It("should create a new Kafka ACL", func() {
			createdACL := &v1alpha1.KafkaACL{}
			lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdACL)).Should(Succeed())

			By("by checking that after creation KafkaACL status fields were properly populated")
			Expect(createdACL.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdACL.Status.Project).Should(Equal(os.Getenv("AIVEN_PROJECT_NAME")))
			Expect(createdACL.Status.Permission).Should(Equal("admin"))
			Expect(createdACL.Status.Topic).Should(Equal(topicName))
			Expect(createdACL.Status.Username).Should(Equal(userName))

		})
	})

	AfterEach(func() {
		By("Ensures that Kafka Topic instance was deleted")
		ensureDelete(ctx, kafkaTopic)
		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
	})
})

func kafkaACLSpec(service, topic, userName, namespace string) *v1alpha1.KafkaACL {
	return &v1alpha1.KafkaACL{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "KafkaACL",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      userName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaACLSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			Permission:  "admin",
			Topic:       topic,
			Username:    userName,
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
