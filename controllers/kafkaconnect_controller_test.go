// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("KafkaConnect Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		kafkaconnect *v1alpha1.KafkaConnect
		serviceName  string
		ctx          context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-kafkaconnect-acc-" + generateRandomID()
		kafkaconnect = kafkaConnectSpec(serviceName, namespace)

		By("Creating a new KafkaConnect CR instance")
		Expect(k8sClient.Create(ctx, kafkaconnect)).Should(Succeed())

		kcLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdKafkaConnect := &v1alpha1.KafkaConnect{}
		// We'll need to retry getting this newly created KafkaConnect,
		// given that creation may not immediately happen.
		By("by retrieving Kafka Connect instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Kafka Connect service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)
			if err == nil {
				return meta.IsStatusConditionTrue(createdKafkaConnect.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdKafkaConnect.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating KafkaConnect reconciler behaviour", func() {
		It("should createOrUpdate a new Kafka Connect service", func() {
			createdKafkaConnect := &v1alpha1.KafkaConnect{}
			kcLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, kcLookupKey, createdKafkaConnect)).Should(Succeed())

			By("by checking that after KafkaConnect service was created")
			Expect(meta.IsStatusConditionTrue(createdKafkaConnect.Status.Conditions, conditionTypeRunning)).Should(BeTrue())
			Expect(createdKafkaConnect.Status.State).Should(Equal("RUNNING"))
		})
	})

	AfterEach(func() {
		By("Ensures that KafkaConnect instance was deleted")
		ensureDelete(ctx, kafkaconnect)
	})
})

func kafkaConnectSpec(serviceName, namespace string) *v1alpha1.KafkaConnect {
	return &v1alpha1.KafkaConnect{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "KafkaConnect",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.KafkaConnectSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
