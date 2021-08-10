package controllers

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Project Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		namespace             = "default"
		projectCloud          = "aws-eu-west-1"
		projectBillingAddress = "NYC"

		timeout  = time.Minute * 1
		interval = time.Second * 5
	)

	var (
		project     *v1alpha1.Project
		projectName string
		ctx         context.Context
	)

	BeforeEach(func() {
		var src = rand.NewSource(time.Now().UnixNano())
		projectName = "k8s-test-acc-" + strconv.FormatInt(src.Int63(), 10)

		project = &v1alpha1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "aiven.io/v1alpha1",
				Kind:       "Project",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      projectName,
				Namespace: namespace,
			},
			Spec: v1alpha1.ProjectSpec{
				BillingAddress: projectBillingAddress,
				Cloud:          projectCloud,
				AuthSecretRef: v1alpha1.AuthSecretReference{
					Name: secretRefName,
					Key:  secretRefKey,
				},
			},
		}
		ctx = context.Background()

		By("Creating a new Project CR instance")
		Expect(k8sClient.Create(ctx, project)).Should(Succeed())

		projectLookupKey := types.NamespacedName{Name: projectName, Namespace: namespace}
		createdProject := &v1alpha1.Project{}
		// We'll need to retry getting this newly created Project,
		// given that creation may not immediately happen.
		By("by retrieving Project instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, projectLookupKey, createdProject)
			if err == nil {
				return meta.IsStatusConditionTrue(createdProject.Status.Conditions, conditionTypeRunning)
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("by checking finalizers")
		Expect(createdProject.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating Project reconciler behaviour", func() {
		It("should createOrUpdate a new Project", func() {
			createdProject := &v1alpha1.Project{}
			projectLookupKey := types.NamespacedName{Name: projectName, Namespace: namespace}

			Expect(k8sClient.Get(ctx, projectLookupKey, createdProject)).Should(Succeed())

			By("by checking that after creation of a Project is created with a sercret")
			createdSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: projectName, Namespace: namespace}, createdSecret)).Should(Succeed())

			Expect(createdProject.Status.AvailableCredits).NotTo(BeEmpty())
			Expect(createdProject.Status.EstimatedBalance).NotTo(BeEmpty())
			Expect(createdSecret.Data["CA_CERT"]).NotTo(BeEmpty())
		})
	})

	AfterEach(func() {
		By("Ensures that Project instance was deleted")
		ensureDelete(ctx, project)
	})
})
