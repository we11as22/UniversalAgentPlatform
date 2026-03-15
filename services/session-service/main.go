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
	cfg := config.Load("session-service", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/session", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"user_id":    "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
			"tenant_id":  "11111111-1111-1111-1111-111111111111",
			"roles":      []string{"tenant-admin"},
			"display_name": "Acme Admin",
		})
	})
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

