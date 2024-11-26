package v1alpha1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	"github.com/jingkaihe/opsmate-operator/pkg/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	goscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Service struct {
	client client.Client
}

func NewService(ctx context.Context) (*Service, error) {
	scheme := runtime.NewScheme()

	if err := goscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	if err := srev1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config := ctrl.GetConfigOrDie()

	ca, err := cache.New(config, cache.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	go func() {
		if err := ca.Start(ctx); err != nil {
			logger.G(ctx).WithError(err).Error("cache error")
		}
	}()

	if waited := ca.WaitForCacheSync(ctx); !waited {
		return nil, errors.New("cache sync failed")
	}

	c, err := client.New(config, client.Options{
		Cache: &client.CacheOptions{
			Reader: ca,
		},
	})
	if err != nil {
		return nil, err
	}

	return &Service{client: c}, nil
}

// @Summary Get health status of the API server
// @Description get health check endpoint
// @Accept  json
// @Produce  json
// @Success 200 {string} ok
// @Router /healthz [get]
func (s *Service) Healthz(g *gin.Context) {
	g.JSON(http.StatusOK, "ok")
}

// @Summary Get EnvironmentBuilds
// @Description get environment builds
// @Accept  json
// @Produce  json
// @Success 200 {array} srev1alpha1.EnvironmentBuild
// @Failure 404 {object} error
// @Failure 500 {object} error
// @Router /:namespace/environmentbuilds [get]
func (s *Service) GetEnvironmentBuilds(g *gin.Context) {
	ctx := g.Request.Context()
	reglog := logger.G(ctx)
	envBuilds := &srev1alpha1.EnvironmentBuildList{}
	if err := s.client.List(ctx, envBuilds, client.InNamespace(g.Param("namespace"))); err != nil {
		reglog.WithError(err).Error("failed to list environment builds")
		if apierrors.IsNotFound(err) {
			g.JSON(http.StatusNotFound, "not found")
		} else {
			g.JSON(http.StatusInternalServerError, "internal server error")
		}
		return
	}
	g.JSON(http.StatusOK, envBuilds.Items)
}

// @Summary Get EnvironmentBuild
// @Description get environment build
// @Accept  json
// @Produce  json
// @Success 200 {object} srev1alpha1.EnvironmentBuild
// @Failure 404 {object} error
// @Failure 500 {object} error
// @Router /:namespace/environmentbuilds/:name [get]
func (s *Service) GetEnvironmentBuild(g *gin.Context) {
	var (
		ctx       = g.Request.Context()
		reqLog    = logger.G(ctx)
		namespace = g.Param("namespace")
		name      = g.Param("name")
	)
	envBuild := &srev1alpha1.EnvironmentBuild{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, envBuild); err != nil {
		reqLog.WithError(err).Error("failed to get environment build")
		if apierrors.IsNotFound(err) {
			g.JSON(http.StatusNotFound, "not found")
		} else {
			g.JSON(http.StatusInternalServerError, "internal server error")
		}
		return
	}
	g.JSON(http.StatusOK, envBuild)
}

// @Summary Create EnvironmentBuild
// @Description create environment build
// @Param environmentBuild body srev1alpha1.EnvironmentBuild true "EnvironmentBuild"
// @Produce  json
// @Success 201 {object} srev1alpha1.EnvironmentBuild
// @Failure 400 {object} error
// @Router /:namespace/environmentbuilds [post]
func (s *Service) CreateEnvironmentBuild(g *gin.Context) {
	var (
		namespace = g.Param("namespace")
		envBuild  = srev1alpha1.EnvironmentBuild{}
		ctx       = g.Request.Context()
		reqLog    = logger.G(ctx)
	)
	if err := g.ShouldBindJSON(&envBuild); err != nil {
		reqLog.WithError(err).Error("failed to bind environment build")
		g.JSON(http.StatusBadRequest, err.Error())
		return
	}
	envBuild.Namespace = namespace
	if err := s.client.Create(ctx, &envBuild); err != nil {
		reqLog.WithError(err).Error("failed to create environment build")
		g.JSON(http.StatusBadRequest, err.Error())
		return
	}
	g.JSON(http.StatusCreated, envBuild)
}

// @Summary Update EnvironmentBuild
// @Description update environment build
// @Accept  json
// @Produce  json
// @Success 200 {object} srev1alpha1.EnvironmentBuild
// @Failure 404 {object} error
// @Failure 400 {object} error
// @Router /:namespace/environmentbuilds/:name [put]
func (s *Service) UpdateEnvironmentBuild(g *gin.Context) {
	var (
		namespace = g.Param("namespace")
		name      = g.Param("name")
		envBuild  = srev1alpha1.EnvironmentBuild{}
		ctx       = g.Request.Context()
		reqLog    = logger.G(ctx)
	)

	if err := g.ShouldBindJSON(&envBuild); err != nil {
		reqLog.WithError(err).Error("failed to bind environment build")
		g.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// get existing env build
	existingEnvBuild := &srev1alpha1.EnvironmentBuild{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existingEnvBuild); err != nil {
		reqLog.WithError(err).Error("failed to get environment build")
		if apierrors.IsNotFound(err) {
			g.JSON(http.StatusNotFound, "not found")
		} else {
			g.JSON(http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// update env build
	existingEnvBuild.Spec = envBuild.Spec
	if err := s.client.Update(ctx, existingEnvBuild); err != nil {
		reqLog.WithError(err).Error("failed to update environment build")
		g.JSON(http.StatusBadRequest, err.Error())
		return
	}
	g.JSON(http.StatusOK, existingEnvBuild)
}

// @Summary Delete EnvironmentBuild
// @Description delete environment build
// @Produce  json
// @Success 200 {string} string
// @Failure 404 {object} error
// @Failure 400 {object} error
// @Router /:namespace/environmentbuilds/:name [delete]
func (s *Service) DeleteEnvironmentBuild(g *gin.Context) {
	var (
		namespace = g.Param("namespace")
		name      = g.Param("name")
		ctx       = g.Request.Context()
		reqLog    = logger.G(ctx)
	)

	if err := s.client.Delete(ctx, &srev1alpha1.EnvironmentBuild{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}); err != nil {
		reqLog.WithError(err).Error("failed to delete environment build")
		if apierrors.IsNotFound(err) {
			g.JSON(http.StatusNotFound, "not found")
		} else {
			g.JSON(http.StatusBadRequest, err.Error())
		}
		return
	}
	g.JSON(http.StatusOK, "deleted")
}
