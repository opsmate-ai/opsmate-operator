package v1alpha1

import (
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

		envBuild := &srev1alpha1.EnvrionmentBuild{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-env-build",
				Namespace: "default",
			},
			Spec: srev1alpha1.EnvrionmentBuildSpec{
				Template: corev1.PodTemplateSpec{
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

		clientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(envBuild)
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
			req, err := http.NewRequest("GET", "/api/v1alpha1/healthz", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Context("GET /api/v1alpha1/environmentbuilds", func() {
		It("should return 200", func() {
			w := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/api/v1alpha1/environmentbuilds", nil)
			Expect(err).NotTo(HaveOccurred())

			router.ServeHTTP(w, req)
			Expect(w.Code).To(Equal(http.StatusOK))

			var envBuilds []srev1alpha1.EnvrionmentBuild
			Expect(json.Unmarshal(w.Body.Bytes(), &envBuilds)).To(Succeed())
			Expect(envBuilds).To(HaveLen(1))
			Expect(envBuilds[0].Name).To(Equal("test-env-build"))
		})
	})
})
