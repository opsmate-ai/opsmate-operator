package v1alpha1

import (
	"context"

	"github.com/gin-gonic/gin"
)

// @title Opsmate Operator API
// @version 0.1.0
// @description This is Opsmate Operator API.

// @contact.name Opsmate engineering team
// @contact.email jingkai@hey.com

// @BasePath /api/v1alpha1

func Register(router *gin.Engine, config context.Context) error {
	svc, err := NewService(config)
	if err != nil {
		return err
	}

	register(router, svc)
	return nil
}

func register(router *gin.Engine, svc *Service) {
	v1alpha1 := router.Group("/api/v1alpha1")

	v1alpha1.GET("/healthz", svc.Healthz)
	v1alpha1.GET("/:namespace/environmentbuilds", svc.GetEnvironmentBuilds)
	v1alpha1.GET("/:namespace/environmentbuilds/:name", svc.GetEnvironmentBuild)
	v1alpha1.POST("/:namespace/environmentbuilds", svc.CreateEnvironmentBuild)
	v1alpha1.PUT("/:namespace/environmentbuilds/:name", svc.UpdateEnvironmentBuild)
	v1alpha1.DELETE("/:namespace/environmentbuilds/:name", svc.DeleteEnvironmentBuild)
}
