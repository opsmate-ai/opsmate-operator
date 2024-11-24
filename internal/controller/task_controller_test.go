package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
)

var _ = Describe("Task Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-task"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		task := &srev1alpha1.Task{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Task")
			err := k8sClient.Get(ctx, typeNamespacedName, task)
			if err != nil && errors.IsNotFound(err) {
				resource := &srev1alpha1.Task{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: srev1alpha1.TaskSpec{
						UserID:               "test-user",
						EnvironmentBuildName: "test-environment-build",
						Instruction:          "echo 'Hello, World!'",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Task")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			// by default it's pending
			Eventually(func() string {
				err := k8sClient.Get(ctx, typeNamespacedName, task)
				Expect(err).NotTo(HaveOccurred())
				return task.Status.State
			}).Should(Equal(srev1alpha1.StatePending))
		})
	})
})
