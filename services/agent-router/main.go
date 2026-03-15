package main

import (
	"net/http"
	"os"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/httpclient"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
)

const version = "0.1.0"

type request struct {
	TenantID       string `json:"tenant_id"`
	AgentID        string `json:"agent_id"`
	AgentVersionID string `json:"agent_version_id"`
	Message        string `json:"message"`
	RAGEnabled     bool   `json:"rag_enabled"`
}

type response struct {
	ProviderName string `json:"provider_name"`
	ProviderKind string `json:"provider_kind"`
	Text         string `json:"text"`
	Retrieval    any    `json:"retrieval,omitempty"`
}

func main() {
	cfg := config.Load("agent-router", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	agentRuntimeURL := os.Getenv("AGENT_RUNTIME_URL")
	if agentRuntimeURL == "" {
		agentRuntimeURL = "http://localhost:18001"
	}

	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.POST("/api/v1/respond", func(ctx *gin.Context) {
		var payload request
		if err := ctx.ShouldBindJSON(&payload); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := httpclient.PostJSON[request, response](ctx, agentRuntimeURL+"/api/v1/execute", payload)
		if err != nil {
			ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, result)
	})
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}
