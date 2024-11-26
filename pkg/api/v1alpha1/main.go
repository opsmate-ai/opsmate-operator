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

func Register(ctx context.Context, router *gin.Engine) error {
	svc, err := NewService(ctx)
	if err != nil {
		return err
	}

	register(router, svc)
	return nil
}

func register(router *gin.Engine, svc *Service) {
	v1alpha1 := router.Group("/api/v1alpha1")

	v1alpha1.GET("/healthz", Observe("/healthz"), svc.Healthz)

	v1alpha1.GET("/:namespace/environmentbuilds", Observe("/:namespace/environmentbuilds"), svc.GetEnvironmentBuilds)
	v1alpha1.GET("/:namespace/environmentbuilds/:name", Observe("/:namespace/environmentbuilds/:name"), svc.GetEnvironmentBuild)
	v1alpha1.POST("/:namespace/environmentbuilds", Observe("/:namespace/environmentbuilds"), svc.CreateEnvironmentBuild)
	v1alpha1.PUT("/:namespace/environmentbuilds/:name", Observe("/:namespace/environmentbuilds/:name"), svc.UpdateEnvironmentBuild)
	v1alpha1.DELETE("/:namespace/environmentbuilds/:name", Observe("/:namespace/environmentbuilds/:name"), svc.DeleteEnvironmentBuild)

	v1alpha1.GET("/:namespace/tasks/:name", Observe("/:namespace/tasks/:name"), svc.GetTask)
	v1alpha1.POST("/:namespace/tasks", Observe("/:namespace/tasks"), svc.CreateTask)
	v1alpha1.DELETE("/:namespace/tasks/:name", Observe("/:namespace/tasks/:name"), svc.DeleteTask)
}
