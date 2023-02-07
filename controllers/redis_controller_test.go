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
	redisuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/redis"
)

var _ = Describe("Redis Controller using secret", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		redis       *v1alpha1.Redis
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-redis-acc-with-secret-" + generateRandomID()
		redis = redisSpec(serviceName, namespace, true)

		By("Creating a new Redis CR instance")
		Expect(k8sClient.Create(ctx, redis)).Should(Succeed())

		rLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdRedis := &v1alpha1.Redis{}
		// We'll need to retry getting this newly created Redis,
		// given that creation may not immediately happen.
		By("by retrieving redis instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdRedis)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Redis service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdRedis)
			if err == nil {
				return meta.IsStatusConditionTrue(createdRedis.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdRedis.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Redis reconciler behaviour", func() {
		It("should createOrUpdate a new Redis service", func() {
			createdRedis := &v1alpha1.Redis{}
			lookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdRedis)).Should(Succeed())

			By("by checking that after creation of a Redis service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PASSWORD"]).NotTo(BeEmpty())

			Expect(createdRedis.Status.State).Should(Equal("RUNNING"))
		})
	})

	AfterEach(func() {
		By("Ensures that Redis instance was deleted")
		ensureDelete(ctx, redis)
	})
})

var _ = Describe("Redis Controller using default token", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		redis       *v1alpha1.Redis
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-redis-acc-default-token-" + generateRandomID()
		redis = redisSpec(serviceName, namespace, false)

		By("Creating a new Redis CR instance")
		Expect(k8sClient.Create(ctx, redis)).Should(Succeed())

		rLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdRedis := &v1alpha1.Redis{}
		// We'll need to retry getting this newly created Redis,
		// given that creation may not immediately happen.
		By("by retrieving redis instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdRedis)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Redis service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, rLookupKey, createdRedis)
			if err == nil {
				return meta.IsStatusConditionTrue(createdRedis.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdRedis.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Redis reconciler behaviour", func() {
		It("should createOrUpdate a new Redis service", func() {
			createdRedis := &v1alpha1.Redis{}
			lookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdRedis)).Should(Succeed())

			By("by checking that after creation of a Redis service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdSecret.Data["HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PASSWORD"]).NotTo(BeEmpty())

			Expect(createdRedis.Status.State).Should(Equal("RUNNING"))

			// Userconfig test
			expectedIPFilter := []*redisuserconfig.IpFilter{
				{
					Network: "10.20.0.0/16",
				},
				{
					Network:     "0.0.0.0",
					Description: anyPointer("whatever"),
				},
			}
			Expect(createdRedis.Spec.UserConfig.IpFilter).Should(Equal(expectedIPFilter))
		})
	})

	AfterEach(func() {
		By("Ensures that Redis instance was deleted")
		ensureDelete(ctx, redis)
	})
})

func redisSpec(serviceName, namespace string, useSecret bool) *v1alpha1.Redis {
	var authSecretReference v1alpha1.AuthSecretReference
	if useSecret {
		authSecretReference = v1alpha1.AuthSecretReference{
			Name: secretRefName,
			Key:  secretRefKey,
		}
	}
	return &v1alpha1.Redis{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Redis",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.RedisSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:       os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:          "business-4",
				CloudName:     "google-europe-west1",
				AuthSecretRef: authSecretReference,
			},
			UserConfig: &redisuserconfig.RedisUserConfig{
				IpFilter: []*redisuserconfig.IpFilter{
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
