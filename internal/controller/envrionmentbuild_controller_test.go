package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("EnvrionmentBuild Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		envrionmentbuild := &srev1alpha1.EnvrionmentBuild{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind EnvrionmentBuild")
			err := k8sClient.Get(ctx, typeNamespacedName, envrionmentbuild)
			if err != nil && errors.IsNotFound(err) {
				resource := &srev1alpha1.EnvrionmentBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: srev1alpha1.EnvrionmentBuildSpec{
						Template: corev1.PodTemplateSpec{
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

			// create a task as well
			task := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task",
					Namespace: "default",
				},
				Spec: srev1alpha1.TaskSpec{
					EnvironmentBuildName: resourceName,
					Instruction:          "echo hello",
					Context:              "test",
				},
			}
			Expect(k8sClient.Create(ctx, task)).To(Succeed())
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &srev1alpha1.EnvrionmentBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance EnvrionmentBuild")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource with task count set to 1")
			// _, err := envBuildReconciler.Reconcile(ctx, reconcile.Request{
			// 	NamespacedName: typeNamespacedName,
			// })
			// Expect(err).NotTo(HaveOccurred())

			Eventually(func() int {
				envBuild := &srev1alpha1.EnvrionmentBuild{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
				}
				err := k8sClient.Get(ctx, typeNamespacedName, envBuild)
				Expect(err).NotTo(HaveOccurred())
				return envBuild.Status.TaskCount
			}, time.Second*10).Should(Equal(1))
		})
	})
})
