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

var _ = Describe("ServiceUser Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PG
		su          *v1alpha1.ServiceUser
		serviceName string
		userName    string
		ctx         context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		serviceName = "k8s-test-pg-user-acc-" + generateRandomID()
		userName = "k8s-user-acc-" + generateRandomID()
		pg = pgSpec(serviceName, namespace)
		su = serviceUserSpec(serviceName, userName, namespace)

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		By("Waiting PG service status to become RUNNING")
		Eventually(func() string {
			pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: namespace}
			createdPG := &v1alpha1.PG{}
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)
			if err == nil {
				return createdPG.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("Creating a new ServiceUser CR instance")
		Expect(k8sClient.Create(ctx, su)).Should(Succeed())

		// We'll need to retry getting this newly created instance,
		// given that creation may not immediately happen.
		By("by retrieving ServiceUser instance from k8s")
		Eventually(func() bool {
			suLookupKey := types.NamespacedName{Name: userName, Namespace: namespace}
			createdUser := &v1alpha1.ServiceUser{}
			err := k8sClient.Get(ctx, suLookupKey, createdUser)

			return err == nil && createdUser.Status.Type != ""
		}, timeout, interval).Should(BeTrue())

		time.Sleep(10 * time.Second)
	})

	Context("Validating ServiceUser reconciler behaviour", func() {
		It("should create a new ServiceUser instance", func() {
			createdUser := &v1alpha1.ServiceUser{}
			lookupKey := types.NamespacedName{Name: userName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, lookupKey, createdUser)).Should(Succeed())

			// Let's make sure our instance status was properly populated.
			By("by checking that after creation ServiceUser status fields were properly populated")
			Expect(createdUser.Status.Project).Should(Equal(os.Getenv("AIVEN_PROJECT_NAME")))
			Expect(createdUser.Status.Authentication).Should(Equal("caching_sha2_password"))
			Expect(createdUser.Status.Type).ToNot(BeEmpty())
			Expect(createdUser.Status.ServiceName).Should(Equal(serviceName))
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
		ensureDelete(ctx, pg)
	})
})

func serviceUserSpec(service, user, namespace string) *v1alpha1.ServiceUser {
	return &v1alpha1.ServiceUser{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "k8s-operator.aiven.io/v1alpha1",
			Kind:       "ServiceUser",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      user,
			Namespace: namespace,
		},
		Spec: v1alpha1.ServiceUserSpec{
			Project:        os.Getenv("AIVEN_PROJECT_NAME"),
			ServiceName:    service,
			Authentication: "caching_sha2_password",
		},
	}
}
