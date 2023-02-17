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
	grafanauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/grafana"
)

var _ = Describe("Grafana Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		grafana     *v1alpha1.Grafana
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-grafana-acc-" + generateRandomID()
		grafana = grafanaSpec(serviceName, namespace)

		By("Creating a new Grafana CR instance")
		Expect(k8sClient.Create(ctx, grafana)).Should(Succeed())

		grafanaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
		createdGrafana := &v1alpha1.Grafana{}
		// We'll need to retry getting this newly created Grafana,
		// given that creation may not immediately happen.
		By("by retrieving Grafana instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, grafanaLookupKey, createdGrafana)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting Grafana service status to become RUNNING")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, grafanaLookupKey, createdGrafana)
			if err == nil {
				return meta.IsStatusConditionTrue(createdGrafana.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdGrafana.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Grafana reconciler behaviour", func() {
		It("should createOrUpdate a new Grafana service", func() {
			createdGrafana := &v1alpha1.Grafana{}
			grafanaLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, grafanaLookupKey, createdGrafana)).Should(Succeed())

			By("by checking that after creation of a Grafana service secret is created")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, createdSecret)).Should(Succeed())

			// It is running
			Expect(createdGrafana.Status.State).Should(Equal("RUNNING"))

			// Secretes test
			Expect(createdSecret.Data["GRAFANA_HOST"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["GRAFANA_PORT"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["GRAFANA_USER"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["GRAFANA_PASSWORD"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["GRAFANA_URI"]).NotTo(BeEmpty())
			Expect(createdSecret.Data["GRAFANA_HOSTS"]).NotTo(BeEmpty())

			// User config test
			Expect(*createdGrafana.Spec.UserConfig.AlertingEnabled).Should(Equal(true))
			Expect(*createdGrafana.Spec.UserConfig.PublicAccess.Grafana).Should(Equal(true))

			// Ip filters test
			expectedIPFilter := []*grafanauserconfig.IpFilter{
				{
					Network: "10.20.0.0/16",
				},
				{
					Network:     "0.0.0.0",
					Description: anyPointer("whatever"),
				},
			}
			Expect(createdGrafana.Spec.UserConfig.IpFilter).Should(Equal(expectedIPFilter))
		})
	})

	AfterEach(func() {
		By("Ensures that Grafana instance was deleted")
		ensureDelete(ctx, grafana)
	})
})

func grafanaSpec(serviceName, namespace string) *v1alpha1.Grafana {
	return &v1alpha1.Grafana{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "aiven.io/v1alpha1",
			Kind:       "Grafana",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: v1alpha1.GrafanaSpec{
			ServiceCommonSpec: v1alpha1.ServiceCommonSpec{
				Project:   os.Getenv("AIVEN_PROJECT_NAME"),
				Plan:      "startup-1",
				CloudName: "google-europe-west1",
				Tags:      map[string]string{"key1": "value1"},
			},
			UserConfig: &grafanauserconfig.GrafanaUserConfig{
				AlertingEnabled: anyPointer(true),
				PublicAccess: &grafanauserconfig.PublicAccess{
					Grafana: anyPointer(true),
				},
				IpFilter: []*grafanauserconfig.IpFilter{
					{
						Network: "10.20.0.0/16",
					},
					{
						Network:     "0.0.0.0",
						Description: anyPointer("whatever"),
					},
				},
			},
			AuthSecretRef: v1alpha1.AuthSecretReference{
				Name: secretRefName,
				Key:  secretRefKey,
			},
		},
	}
}
