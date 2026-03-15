package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type HealthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Version string `json:"version"`
}

func NewRouter(service string, version string, logger *zap.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(ctx *gin.Context) {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, Last-Event-ID")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})
	router.Use(func(ctx *gin.Context) {
		logger.Info("request", zap.String("method", ctx.Request.Method), zap.String("path", ctx.Request.URL.Path))
		ctx.Next()
	})
	router.GET("/api/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, HealthResponse{
			Service: service,
			Status:  "ok",
			Version: version,
		})
	})
	router.GET("/health/ready", func(ctx *gin.Context) {
		ctx.Status(http.StatusOK)
	})
	return router
}
