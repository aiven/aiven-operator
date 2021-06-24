package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
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

		By("Creating a new KafkaTopic instance")
		Expect(k8sClient.Create(ctx, kafkaTopic)).Should(Succeed())

		By("Creating a new KafkaACL instance")
		Expect(k8sClient.Create(ctx, acl)).Should(Succeed())

		Eventually(func() bool {
			createdACL := &v1alpha1.KafkaACL{}
			lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}
			err := k8sClient.Get(ctx, lookupKey, createdACL)
			if err == nil {
				return meta.IsStatusConditionTrue(createdACL.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating Kafka ACL reconciler behaviour", func() {
		It("should createOrUpdate a new Kafka ACL", func() {
			createdACL := &v1alpha1.KafkaACL{}
			lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdACL)).Should(Succeed())

			By("by checking that after KafkaACL was created")
			Expect(meta.IsStatusConditionTrue(createdACL.Status.Conditions, conditionTypeRunning)).Should(BeTrue())
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka ACL instance was deleted")
		ensureDelete(ctx, acl)

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
