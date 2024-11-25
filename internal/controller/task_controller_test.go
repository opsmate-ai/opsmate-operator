package controller

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
)

var _ = Describe("Task Controller", func() {
	Context("When reconciling a resource", func() {
		var (
			taskName        string
			envBuildName    string
			namespace       string
			ensureTaskEvent = func(ctx context.Context, taskName, message string) func() bool {
				return func() bool {
					var events corev1.EventList
					Expect(
						k8sClient.List(
							ctx,
							&events,
							client.InNamespace(namespace),
							client.MatchingFields{"involvedObject.name": taskName},
						),
					).To(Succeed())
					return slices.ContainsFunc(events.Items, func(event corev1.Event) bool {
						return strings.Contains(event.Message, message)
					})
				}
			}
			ensureTaskStateTransition = func(ctx context.Context, taskName string, expectedState string) func() bool {
				return ensureTaskEvent(ctx, taskName, fmt.Sprintf("Task status updated to %s", expectedState))
			}

			ensureTaskState = func(ctx context.Context, taskName string, expectedState string) func() bool {
				return func() bool {
					var task srev1alpha1.Task
					return k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: namespace}, &task) == nil && task.Status.State == expectedState
				}
			}

			ensureTaskRemoved = func(ctx context.Context, taskName string) func() bool {
				return func() bool {
					return apierrors.IsNotFound(
						k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: namespace}, &srev1alpha1.Task{}),
					)
				}
			}

			podExists = func(ctx context.Context, taskName string) func() bool {
				return func() bool {
					var pod corev1.Pod
					return k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: namespace}, &pod) == nil
				}
			}
		)

		ctx := context.Background()

		BeforeEach(func() {
			namespace = "default"
			envBuildName = "test-environment-build"
			taskName = "test-task-" + uuid.New().String()
		})

		AfterEach(func() {
			By("deleting all resources")
			Expect(k8sClient.DeleteAllOf(ctx, &srev1alpha1.Task{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &srev1alpha1.EnvrionmentBuild{}, client.InNamespace(namespace))).To(Succeed())
			Expect(k8sClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(namespace))).To(Succeed())
		})

		It("should successfully reconcile a correctly configured task", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("it's eventually running")
			Eventually(ensureTaskState(ctx, taskName, srev1alpha1.StateRunning)).WithTimeout(5 * time.Second).Should(BeTrue())
		})

		It("should have the the task status updated when the task is running", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("it's eventually running")
			Eventually(ensureTaskState(ctx, taskName, srev1alpha1.StateRunning)).WithTimeout(5 * time.Second).Should(BeTrue())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: namespace}, task)).To(Succeed())
			var pod corev1.Pod
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: taskName, Namespace: namespace}, &pod)).To(Succeed())
			Expect(task.Status.Pod.Name).To(Equal(pod.Name))
			Expect(task.Status.Pod.Namespace).To(Equal(pod.Namespace))
			Expect(task.Status.InternalIP).To(Equal(pod.Status.PodIP))
			Expect(task.Status.AllocatedAt).NotTo(BeNil())
			Expect(task.Status.Conditions).To(HaveLen(2))
			Expect(task.Status.Conditions[0].Type).To(Equal(srev1alpha1.ConditionTaskPodScheduled))
			Expect(task.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(task.Status.Conditions[1].Type).To(Equal(srev1alpha1.ConditionTaskPodRunning))
			Expect(task.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
		})

		It("should remove the pod when the task is removed", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("the pod is eventually created")
			Eventually(podExists(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())

			// remove the task
			Expect(k8sClient.Delete(ctx, task)).To(Succeed())

			By("the pod is eventually deleted")
			Eventually(podExists(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeFalse())
		})

		It("should remove the task when the environment build is invalid", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			envBuild.Spec.Template.Spec.Containers[0].Command = []string{"invalid-command"}
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("it's eventually failed")
			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateTerminating)).WithTimeout(5 * time.Second).Should(BeTrue())

			By("the task is eventually deleted")
			Eventually(ensureTaskRemoved(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())

			By("the pod is eventually deleted")
			Eventually(podExists(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeFalse())
		})

		It("should remove the task when the pod is partially failing", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			envBuild.Spec.Template.Spec.Containers = append(envBuild.Spec.Template.Spec.Containers, corev1.Container{
				Name:    "failing-container",
				Image:   "busybox",
				Command: []string{"exit", "1"},
			})
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			By("it's eventually failed")
			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateError)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskEvent(ctx, taskName, "pod container error")).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateTerminating)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskRemoved(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())
		})

		It("should remove the task when the pod is exit prematurely", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			envBuild.Spec.Template.Spec.Containers[0].Command = []string{"echo", "hello"}
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateError)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskEvent(ctx, taskName, "pod completed prematurely")).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateTerminating)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskRemoved(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())
		})

		It("should remove the task when the build is malformed", func() {
			envBuild := newEnvBuild(envBuildName, namespace)
			envBuild.Spec.Template.Spec.Containers[0].Name = "abcEFG" // invalid name
			Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

			task := newTask(taskName, namespace, envBuildName)
			Expect(k8sClient.Create(ctx, task)).To(Succeed())

			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateError)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskEvent(ctx, taskName, "failed to create pod")).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateTerminating)).WithTimeout(5 * time.Second).Should(BeTrue())
			Eventually(ensureTaskRemoved(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())
		})

		// slow test
		// It("should terminate the task when the pod is terminated", func() {
		// 	envBuild := newEnvBuild(envBuildName, namespace)
		// 	Expect(k8sClient.Create(ctx, envBuild)).To(Succeed())

		// 	task := newTask(taskName, namespace, envBuildName)
		// 	Expect(k8sClient.Create(ctx, task)).To(Succeed())

		// 	// the task is running
		// 	Eventually(ensureTaskState(ctx, taskName, srev1alpha1.StateRunning)).WithTimeout(5 * time.Second).Should(BeTrue())

		// 	By("the pod is eventually created")
		// 	Eventually(podExists(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())

		// 	// remove the pod
		// 	Expect(k8sClient.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: taskName, Namespace: namespace}})).To(Succeed())

		// 	// pod is deleted. it can take a while sometimes...
		// 	Eventually(podExists(ctx, taskName)).WithTimeout(time.Minute).Should(BeFalse())

		// 	By("the task is eventually terminated")
		// 	Eventually(ensureTaskStateTransition(ctx, taskName, srev1alpha1.StateTerminating)).WithTimeout(5 * time.Second).Should(BeTrue())

		// 	By("the task is eventually deleted")
		// 	Eventually(ensureTaskRemoved(ctx, taskName)).WithTimeout(5 * time.Second).Should(BeTrue())
		// })
	})
})

func newEnvBuild(name, namespace string) *srev1alpha1.EnvrionmentBuild {
	return &srev1alpha1.EnvrionmentBuild{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
}

func newTask(name, namespace, envBuildName string) *srev1alpha1.Task {
	return &srev1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: srev1alpha1.TaskSpec{
			UserID:               "test-user",
			EnvironmentBuildName: envBuildName,
			Instruction:          "echo 'Hello, World!'",
		},
	}
}
