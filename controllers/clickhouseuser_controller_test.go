// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ClickhouseUser Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		ch          *v1alpha1.Clickhouse
		u           *v1alpha1.ClickhouseUser
		serviceName string
		userName    string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-ch-user-acc-" + generateRandomID()
		userName = "k8s-ch-user-acc-" + generateRandomID()
		ch = chSpec(serviceName, namespace)
		u = clickhouseUserSpec(serviceName, userName, namespace)

		By("Creating a new Clickhouse CR instance")
		Expect(k8sClient.Create(ctx, ch)).Should(Succeed())

		By("Creating a new ClickhouseUser CR instance")
		Expect(k8sClient.Create(ctx, u)).Should(Succeed())

		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving ClickhouseUser instance from k8s")
		Eventually(func() bool {
			suLookupKey := types.NamespacedName{Name: userName, Namespace: namespace}
			createdUser := &v1alpha1.ClickhouseUser{}
			err := k8sClient.Get(ctx, suLookupKey, createdUser)
			if err == nil {
				return meta.IsStatusConditionTrue(createdUser.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("Validating ClickhouseUser reconciler behaviour", func() {
		It("should createOrUpdate a new ClickhouseUser instance", func() {
			createdUser := &v1alpha1.ClickhouseUser{}
			lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdUser)).Should(Succeed())

			By("by checking that after creation ClickhouseUser UUID fields were properly populated")
			Expect(createdUser.Status.UUID).ToNot(BeEmpty())

			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: userName, Namespace: namespace}, createdSecret)).Should(Succeed())
			Expect(createdSecret.Data["USERNAME"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["PORT"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that Clickhouse User instance was deleted")
		ensureDelete(ctx, u)

		By("Ensures that Clickhouse instance was deleted")
		ensureDelete(ctx, ch)
	})
})

func clickhouseUserSpec(service, user, namespace string) *v1alpha1.ClickhouseUser {
	return &v1alpha1.ClickhouseUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "ClickhouseUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      user,
			Namespace: namespace,
		},
		Spec: v1alpha1.ClickhouseUserSpec{
			Project:     os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName: service,
			AuthSecretRef: &v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
