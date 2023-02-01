package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

var _ = Describe("Generic service integrations test", func() {
	const (
		namespace = "default"
		timeout   = time.Minute * 10
		interval  = time.Second * 10
	)

	ctx := context.Background()
	masterName := "k8s-test-pg-acc-master-" + generateRandomID()
	replicaName := "k8s-test-pg-acc-replica-" + generateRandomID()

	masterSpec := pgSpec(masterName, namespace)
	replicaSpec := pgSpec(replicaName, namespace)
	replicaSpec.Spec.ServiceIntegrations = append(
		replicaSpec.Spec.ServiceIntegrations,
		&v1alpha1.ServiceIntegrationItem{
			IntegrationType:   "read_replica",
			SourceServiceName: masterName,
		},
	)

	// Simple plan is fine here
	masterSpec.Spec.ServiceCommonSpec.Plan = "startup-4"
	replicaSpec.Spec.ServiceCommonSpec.Plan = "startup-4"

	BeforeEach(func() {
		By("Creating replica instance, which will wait master being running")
		Expect(k8sClient.Create(ctx, replicaSpec)).Should(Succeed())

		By("Creating master instance")
		Expect(k8sClient.Create(ctx, masterSpec)).Should(Succeed())

		for _, name := range []string{replicaName, masterName} {
			By(fmt.Sprintf("waiting %s instance to be running", name))
			Eventually(func() bool {
				pg := &v1alpha1.PostgreSQL{}
				err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pg)
				return err == nil && meta.IsStatusConditionTrue(pg.Status.Conditions, conditionTypeRunning)
			}, timeout, interval).Should(BeTrue())
		}
	})

	Context("Validating services", func() {
		It("validates k8s instances", func() {
			By("validating replica is running")
			createdReplica := &v1alpha1.PostgreSQL{}
			replicaLookupKey := types.NamespacedName{Name: replicaName, Namespace: namespace}
			Expect(k8sClient.Get(ctx, replicaLookupKey, createdReplica)).Should(Succeed())
			Expect(createdReplica.Status.State).Should(Equal("RUNNING"))

			By("validating replica's service integrations")
			Expect(len(createdReplica.Spec.ServiceIntegrations)).Should(Equal(1))
			Expect(createdReplica.Spec.ServiceIntegrations[0].IntegrationType).Should(Equal("read_replica"))
			Expect(createdReplica.Spec.ServiceIntegrations[0].SourceServiceName).Should(Equal(masterName))

			By("validating master is running")
			createdMaster := &v1alpha1.PostgreSQL{}
			masterLookupKey := types.NamespacedName{Name: masterName, Namespace: namespace}
			Expect(k8sClient.Get(ctx, masterLookupKey, createdMaster)).Should(Succeed())
			Expect(createdMaster.Status.State).Should(Equal("RUNNING"))

			By("validating mast has no service integrations")
			Expect(len(createdMaster.Spec.ServiceIntegrations)).Should(Equal(0))
		})
	})

	AfterEach(func() {
		By("Ensures that PostgreSQL instance was deleted")
		ensureDelete(ctx, replicaSpec)
		ensureDelete(ctx, masterSpec)
	})
})
