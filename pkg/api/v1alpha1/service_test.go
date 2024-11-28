package v1alpha1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1alpha1 Service Suite")
}

var _ = Describe("Service", func() {
	var (
		k8sClient client.Client
		service   *Service
		router    *gin.Engine
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		Expect(srev1alpha1.AddToScheme(scheme)).To(Succeed())

		envBuild := &srev1alpha1.EnvironmentBuild{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-env-build",
				Namespace: "default",
			},
			Spec: srev1alpha1.EnvironmentBuildSpec{
				PodTemplate: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image",
							},
						},
					},
				},
			},
		}

		task := &srev1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-task",
				Namespace: "default",
			},
			Spec: srev1alpha1.TaskSpec{
				UserID:               "anonymous",
				EnvironmentBuildName: envBuild.Name,
				Description:          "say hi",
				Context:              "hello",
			},
			Status: srev1alpha1.TaskStatus{
				State:      srev1alpha1.StateRunning,
				Output:     "hi",
				InternalIP: "127.0.0.1",
			},
		}

		clientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(envBuild, task)
		k8sClient = clientBuilder.Build()
		Expect(k8sClient).NotTo(BeNil())

		service = &Service{
			client: k8sClient,
		}

		router = gin.Default()
		register(router, service)
	})

	Context("GET /api/v1alpha1/healthz", func() {
		It("should return 200", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/healthz", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("should return 404 when the record does not exist", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/not-found/healthz", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("GET /api/v1alpha1/:namespace/environmentbuilds", func() {
		It("should return 200", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/default/environmentbuilds", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			var envBuilds []srev1alpha1.EnvironmentBuild
			Expect(json.Unmarshal(w.Body.Bytes(), &envBuilds)).To(Succeed())
			Expect(envBuilds).To(HaveLen(1))
			Expect(envBuilds[0].Name).To(Equal("test-env-build"))
		})
	})

	Context("GET /api/v1alpha1/:namespace/environmentbuilds/:name", func() {
		It("should return 200 when the record exists", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/default/environmentbuilds/test-env-build", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			var envBuild srev1alpha1.EnvironmentBuild
			Expect(json.Unmarshal(w.Body.Bytes(), &envBuild)).To(Succeed())
			Expect(envBuild.Name).To(Equal("test-env-build"))
		})

		It("should return 404 when the record does not exist", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/default/environmentbuilds/not-found", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("should return 404 when the namespace is not found", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/not-found/environmentbuilds/test-env-build", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("POST /api/v1alpha1/:namespace/environmentbuilds", func() {
		It("should return 201 when the record is created", func() {
			w := httptest.NewRecorder()
			newEnvBuild := &srev1alpha1.EnvironmentBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-env-build-2",
					Namespace: "default",
				},
				Spec: srev1alpha1.EnvironmentBuildSpec{
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
						},
					},
				},
			}
			body, err := json.Marshal(newEnvBuild)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/environmentbuilds", bytes.NewBuffer(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated))

			var createdEnvBuild srev1alpha1.EnvironmentBuild
			Expect(json.Unmarshal(w.Body.Bytes(), &createdEnvBuild)).To(Succeed())
			Expect(createdEnvBuild.Name).To(Equal("test-env-build-2"))

			var builds srev1alpha1.EnvironmentBuildList
			Expect(k8sClient.List(context.Background(), &builds)).To(Succeed())
			Expect(builds.Items).To(HaveLen(2))
		})

		It("should return 400 when the build is invalid", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/environmentbuilds", bytes.NewBufferString(`{"invalid": "body"}`))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the build is gibberish", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/environmentbuilds", bytes.NewBufferString(`abcdefg`))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("PUT /api/v1alpha1/:namespace/environmentbuilds/:name", func() {
		It("should return 200 when the record is updated", func() {
			w := httptest.NewRecorder()
			envBuild := &srev1alpha1.EnvironmentBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-env-build",
					Namespace: "default",
				},
				Spec: srev1alpha1.EnvironmentBuildSpec{
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image-2",
								},
							},
						},
					},
				},
			}
			body, err := json.Marshal(envBuild)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest(http.MethodPut, "/api/v1alpha1/default/environmentbuilds/test-env-build", bytes.NewBuffer(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			var updatedEnvBuild srev1alpha1.EnvironmentBuild
			Expect(json.Unmarshal(w.Body.Bytes(), &updatedEnvBuild)).To(Succeed())
			Expect(updatedEnvBuild.Name).To(Equal("test-env-build"))
		})

		It("should return 404 when the record does not exist", func() {
			envBuild := &srev1alpha1.EnvironmentBuild{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "not-found",
					Namespace: "default",
				},
				Spec: srev1alpha1.EnvironmentBuildSpec{
					PodTemplate: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "test-image-2",
								},
							},
						},
					},
				},
			}
			body, err := json.Marshal(envBuild)
			Expect(err).NotTo(HaveOccurred())

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPut, "/api/v1alpha1/default/environmentbuilds/not-found", bytes.NewBuffer(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		// It("should return 400 when the build is invalid", func() {
		// 	w := httptest.NewRecorder()
		// 	req, err := http.NewRequest(http.MethodPut, "/api/v1alpha1/default/environmentbuilds/test-env-build", bytes.NewBufferString(`{"invalid": "body"}`))
		// 	Expect(err).NotTo(HaveOccurred())

		// 	router.ServeHTTP(w, req)
		// 	Expect(w.Code).To(Equal(http.StatusBadRequest))
		// })
	})

	Context("DELETE /api/v1alpha1/:namespace/environmentbuilds/:name", func() {
		It("should return 200 when the record is deleted", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, "/api/v1alpha1/default/environmentbuilds/test-env-build", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("should return 404 when the record does not exist", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, "/api/v1alpha1/default/environmentbuilds/not-found", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("GET /api/v1alpha1/:namespace/tasks/:name", func() {
		It("should return 200 when the record exists", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/default/tasks/test-task", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			var task srev1alpha1.Task
			Expect(json.Unmarshal(w.Body.Bytes(), &task)).To(Succeed())
			Expect(task.Name).To(Equal("test-task"))
		})

		It("should return 404 when the record does not exist", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/api/v1alpha1/default/tasks/not-found", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})

	Context("POST /api/v1alpha1/:namespace/tasks", func() {
		It("should return 201 when the record is created", func() {
			w := httptest.NewRecorder()
			task := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task-2",
					Namespace: "default",
				},
				Spec: srev1alpha1.TaskSpec{
					UserID:               "anonymous",
					EnvironmentBuildName: "test-env-build",
					Description:          "say hi",
					Context:              "hello",
				},
			}
			body, err := json.Marshal(task)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/tasks", bytes.NewBuffer(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusCreated))

			var createdTask srev1alpha1.Task
			Expect(json.Unmarshal(w.Body.Bytes(), &createdTask)).To(Succeed())
			Expect(createdTask.Name).To(Equal("test-task-2"))

			var tasks srev1alpha1.TaskList
			Expect(k8sClient.List(context.Background(), &tasks)).To(Succeed())
			Expect(tasks.Items).To(HaveLen(2))
		})

		It("should return 400 when the task is invalid", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/tasks", bytes.NewBufferString(`{"invalid": "body"}`))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the task is gibberish", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/tasks", bytes.NewBufferString(`abcdefg`))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return 400 when the task is duplicate", func() {
			task := &srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-task",
					Namespace: "default",
				},
				Spec: srev1alpha1.TaskSpec{
					UserID:               "anonymous",
					EnvironmentBuildName: "test-env-build",
					Description:          "say hi",
					Context:              "hello",
				},
			}
			body, err := json.Marshal(task)
			Expect(err).NotTo(HaveOccurred())

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/v1alpha1/default/tasks", bytes.NewBuffer(body))
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Context("DELETE /api/v1alpha1/:namespace/tasks/:name", func() {
		It("should return 200 when the record is deleted", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, "/api/v1alpha1/default/tasks/test-task", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("should return 404 when the record does not exist", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, "/api/v1alpha1/default/tasks/not-found", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})

		It("should return 404 when the namespace is not found", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, "/api/v1alpha1/not-found/tasks/test-task", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusNotFound))
		})
	})
})
