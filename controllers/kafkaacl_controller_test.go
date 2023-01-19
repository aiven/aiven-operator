// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	"github.com/aiven/aiven-go-client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
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
		projectName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		projectName = os.Getenv("AIVEN_PROJECT_NAME")
		serviceName = "k8s-test-kafka-acl-acc-" + generateRandomID()
		topicName = "k8s-test-kafka-acl-acc-" + generateRandomID()
		userName = "k9s-acl1"
		kafka = kafkaSpec(serviceName, namespace)
		kafkaTopic = kafkaTopicSpec(serviceName, topicName, namespace)
		acl = kafkaACLSpec(projectName, serviceName, topicName, userName, namespace)

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

			By("checking that after KafkaACL was created")
			Expect(meta.IsStatusConditionTrue(createdACL.Status.Conditions, conditionTypeRunning)).Should(BeTrue())

			By("checking initial permission before update")
			Expect(createdACL.Spec.Permission).Should(Equal("admin"))

			By(`updating permission from "admin" to "write" and making sure it has a new ID`)
			createdACL.Spec.Permission = "write"
			Expect(k8sClient.Update(ctx, createdACL)).Should(Succeed())

			// It takes time
			var updatedACL *v1alpha1.KafkaACL
			Eventually(func() bool {
				updatedACL = &v1alpha1.KafkaACL{}
				err := k8sClient.Get(ctx, lookupKey, updatedACL)
				if err == nil {
					return updatedACL.Status.ID != createdACL.Status.ID
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(updatedACL.Status.ID).ShouldNot(Equal(""))
			Expect(updatedACL.Status.ID).ShouldNot(Equal(createdACL.Status.ID))
			Expect(updatedACL.Spec.Permission).Should(Equal("write"))

			By("checking that updated ACL exists at Aiven")
			a, err := aivenClient.KafkaACLs.Get(projectName, updatedACL.Spec.ServiceName, updatedACL.Status.ID)
			Expect(err).To(BeNil())
			Expect(a.ID).Should(Equal(updatedACL.Status.ID))
			Expect(a.Permission).Should(Equal("write"))

			By("checking that old ACL removed at Aiven")
			o, err := aivenClient.KafkaACLs.Get(projectName, updatedACL.Spec.ServiceName, createdACL.Status.ID)
			Expect(o).To(BeNil())
			Expect(aiven.IsNotFound(err)).To(BeTrue())
		})
	})

	AfterEach(func() {
		By("Ensures that Kafka ACL exists in Aiven")
		createdACL := &v1alpha1.KafkaACL{}
		lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}
		err := k8sClient.Get(ctx, lookupKey, createdACL)
		Expect(err).To(BeNil())
		a, err := aivenClient.KafkaACLs.Get(projectName, createdACL.Spec.ServiceName, createdACL.Status.ID)
		Expect(err).To(BeNil())
		Expect(a).NotTo(BeNil())

		By("Ensures that Kafka ACL instance was deleted")
		ensureDelete(ctx, acl)

		By("Ensures that Kafka ACL is deleted on Aiven side")
		a, err = aivenClient.KafkaACLs.Get(projectName, createdACL.Spec.ServiceName, createdACL.Status.ID)
		Expect(a).To(BeNil())
		Expect(aiven.IsNotFound(err)).To(BeTrue())

		By("Ensures that Kafka Topic instance was deleted")
		ensureDelete(ctx, kafkaTopic)

		By("Ensures that Kafka instance was deleted")
		ensureDelete(ctx, kafka)
	})
})

func kafkaACLSpec(project, service, topic, userName, namespace string) *v1alpha1.KafkaACL {
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
			Project:     project,
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
