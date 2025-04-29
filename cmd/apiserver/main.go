package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opsmate-ai/opsmate-operator/pkg/api/v1alpha1"
	"github.com/opsmate-ai/opsmate-operator/pkg/logger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Config struct {
	Port               int
	ShowDoc            bool
	CORSAllowedOrigins []string
}

func loadConfig() *Config {
	var (
		port               int
		showDoc            bool
		corsAllowedOrigins string
	)
	flag.IntVar(&port, "port", 8080, "port to listen on")
	flag.BoolVar(&showDoc, "show-doc", false, "show doc")
	flag.StringVar(&corsAllowedOrigins, "cors-allowed-origins", "*", "cors allowed origins")
	flag.Parse()

	return &Config{
		Port:               port,
		ShowDoc:            showDoc,
		CORSAllowedOrigins: strings.Split(corsAllowedOrigins, ","),
	}
}
func main() {
	var (
		ctx    = context.Background()
		config = loadConfig()
	)

	// context with signal
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	v1alpha1.InitMetricsServer(ctx)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	if err := v1alpha1.Register(ctx, router); err != nil {
		logger.G(ctx).WithError(err).Fatal("failed to register v1alpha1 api")
	}

	if config.ShowDoc {
		router.GET("/swagger/v1alpha1/*any", ginSwagger.WrapHandler(
			swaggerFiles.NewHandler(),
			ginSwagger.InstanceName("v1alpha1"),
		))
	}

	// handle CROS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	server := &http.Server{Addr: fmt.Sprintf(":%d", config.Port), Handler: router}

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			logger.G(ctx).WithError(err).Error("failed to gracefully shutdown api server")
		}
	}()

	logger.G(ctx).Info("Starting api server on :8080")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.G(ctx).WithError(err).Fatal("failed to run api server")
	}
}
