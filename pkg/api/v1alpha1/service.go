package v1alpha1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	"github.com/jingkaihe/opsmate-operator/pkg/logger"
	"k8s.io/apimachinery/pkg/runtime"
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
// @Success 200 {array} srev1alpha1.EnvrionmentBuild
// @Router /environmentbuilds [get]
func (s *Service) GetEnvironmentBuilds(g *gin.Context) {
	ctx := g.Request.Context()
	reglog := logger.G(ctx)
	envBuilds := &srev1alpha1.EnvrionmentBuildList{}
	if err := s.client.List(ctx, envBuilds); err != nil {
		reglog.WithError(err).Error("failed to list environment builds")
		g.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	g.JSON(http.StatusOK, envBuilds.Items)
}
