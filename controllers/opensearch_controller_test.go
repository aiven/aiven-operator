// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/opensearch"
)

var _ = Describe("OpenSearch Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		os          *v1alpha1.OpenSearch
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-os-acc-" + generateRandomID()
		os = osSpec(serviceName, namespace)

		By("Creating a new oPENsEARCH CR instance")
		Expect(k8sClient.Create(ctx, os)).Should(Succeed())

		rLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdOs := &v1alpha1.OpenSearch{}
		// We'll need to retry getting this newly created OpenSearch,
		// given that creation may not immediately happen.
		By("by retrieving os instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdOs)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting OpenSearch service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdOs)
			if err == nil {
				return meta.IsStatusConditionTrue(createdOs.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdOs.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating OpenSearch reconciler behaviour", func() {
		It("should createOrUpdate a new OpenSearch service", func() {
			createdOs := &v1alpha1.OpenSearch{}
			lookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdOs)).Should(Succeed())

			By("by checking that after creation of a OpenSearch service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PASSWORD"]).NotTo(BeEmpty())

			Expect(createdOs.Status.State).Should(Equal("RUNNING"))

			// Userconfig test
			expectedIPFilter := []*opensearchuserconfig.IpFilter{
				{
					Network: "10.20.0.0/16",
				},
				{
					Network:     "0.0.0.0",
					Description: anyPointer("whatever"),
				},
			}
			Expect(createdOs.Spec.UserConfig.IpFilter).Should(Equal(expectedIPFilter))
		})
	})

	AfterEach(func() {
		By("Ensures that OpenSearch instance was deleted")
		ensureDelete(ctx, os)
	})
})

func osSpec(serviceName, namespace string) *v1alpha1.OpenSearch {
	return &v1alpha1.OpenSearch{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "OpenSearch",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.OpenSearchSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				DiskSpace: "240Gib",
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "business-4",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
				AuthSecretRef: v1alpha1.AuthSecretReference{
					Name: secretRefName,
					Key:  secretRefKey,
				},
			},
			UserConfig: &opensearchuserconfig.OpensearchUserConfig{
				IpFilter: []*opensearchuserconfig.IpFilter{
					{
						Network: "10.20.0.0/16",
					},
					{
						Network:     "0.0.0.0",
						Description: anyPointer("whatever"),
					},
				},
			},
		},
	}
}
