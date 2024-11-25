package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
)

var _ = Describe("Task Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			taskName     = "test-task"
			envBuildName = "test-environment-build"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      taskName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		task := &srev1alpha1.Task{}

		BeforeEach(func() {
			By("creating the environment build")
			envBuild := &srev1alpha1.EnvrionmentBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name:      envBuildName,
					Namespace: "default",
				},
				Spec: srev1alpha1.EnvrionmentBuildSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:    "busybox",
								Image:   "busybox",
								Command: []string{"sleep", "infinity"},
							}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			By("creating the custom resource for the Kind Task")
			resource := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
					Namespace: "default",
				},
				Spec: srev1alpha1.TaskSpec{
					UserID:               "test-user",
					EnvironmentBuildName: envBuildName,
					Instruction:          "echo 'Hello, World!'",
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			By("deleting all resources")
			Expect(k8sClient.DeleteAllOf(ctx, &srev1alpha1.Task{}, client.InNamespace("default"))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &srev1alpha1.EnvrionmentBuild{}, client.InNamespace("default"))).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("it's scheduled")
			Eventually(func() string {
				err := k8sClient.Get(ctx, typeNamespacedName, task)
				Expect(err).NotTo(HaveOccurred())
				return task.Status.State
			}).WithTimeout(5 * time.Second).Should(Equal(srev1alpha1.StateScheduled))
		})
	})
})
