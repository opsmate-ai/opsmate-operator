package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("EnvironmentBuild Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		environmentbuild := &srev1alpha1.EnvironmentBuild{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind EnvironmentBuild")
			err := k8sClient.Get(ctx, typeNamespacedName, environmentbuild)
			if err != nil && errors.IsNotFound(err) {
				resource := &srev1alpha1.EnvironmentBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: srev1alpha1.EnvironmentBuildSpec{
						PodTemplate: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:  "test",
										Image: "busybox",
									},
								},
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			By("Deleting the resource")
			Expect(
				k8sClient.DeleteAllOf(
					ctx,
					&srev1alpha1.EnvironmentBuild{},
					client.InNamespace("default"),
				),
			).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource with task count set to 1")
			// create a task as well
			task := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task",
					Namespace: "default",
				},
				Spec: srev1alpha1.TaskSpec{
					EnvironmentBuildName: resourceName,
					Description:          "echo hello",
					Context:              "test",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			Eventually(func() int {
				envBuild := &srev1alpha1.EnvironmentBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
				}
				Expect(k8sClient.Get(ctx, typeNamespacedName, envBuild)).To(Succeed())

				return envBuild.Status.TaskCount
			}, time.Second*5).Should(Equal(1))

			By("Deleting the task the task count should be 0")
			Expect(k8sClient.Delete(ctx, task)).To(Succeed())
			Eventually(func() int {
				envBuild := &srev1alpha1.EnvironmentBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
				}
				err := k8sClient.Get(ctx, typeNamespacedName, envBuild)
				Expect(err).NotTo(HaveOccurred())
				return envBuild.Status.TaskCount
			}, time.Second*10).Should(Equal(0))
		})
	})
})
