package main

import (
	"net/http"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
)

const version = "0.1.0"

func main() {
	cfg := config.Load("quota-service", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/quotas/status", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"allowed":         true,
			"remaining_calls": 1000,
			"remaining_tokens": 500000,
		})
	})
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

