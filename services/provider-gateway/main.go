package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/db"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0"
const defaultTenantID = "11111111-1111-1111-1111-111111111111"

type apiServer struct {
	db             *pgxpool.Pool
	tritonEndpoint string
}

type generateRequest struct {
	TenantID       string `json:"tenant_id"`
	AgentID        string `json:"agent_id"`
	AgentVersionID string `json:"agent_version_id"`
	Message        string `json:"message"`
}

type asrRequest struct {
	TenantID       string `json:"tenant_id"`
	AgentID        string `json:"agent_id"`
	AgentVersionID string `json:"agent_version_id"`
	TextHint       string `json:"text_hint"`
	AudioBase64    string `json:"audio_base64"`
	AudioFormat    string `json:"audio_format"`
}

type ttsRequest struct {
	TenantID       string `json:"tenant_id"`
	AgentID        string `json:"agent_id"`
	AgentVersionID string `json:"agent_version_id"`
	Text           string `json:"text"`
	VoiceProfile   string `json:"voice_profile"`
}

func main() {
	cfg := config.Load("provider-gateway", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()

	api := &apiServer{
		db:             pool,
		tritonEndpoint: envOrDefault("TRITON_ENDPOINT", "http://triton:8000"),
	}
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/providers", api.listProviders)
	router.POST("/api/v1/generate", api.generate)
	router.POST("/api/v1/asr/transcribe", api.transcribe)
	router.POST("/api/v1/tts/synthesize", api.synthesize)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

func tenantID(ctx *gin.Context) string {
	value := ctx.Query("tenant_id")
	if value == "" {
		value = ctx.GetHeader("X-Tenant-ID")
	}
	if value == "" {
		value = defaultTenantID
	}
	return value
}

func (a *apiServer) listProviders(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select provider_id, name, kind, endpoint, enabled
		   from control.providers
		  where tenant_id = $1
		  order by name`,
		tenantID(ctx),
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type response struct {
		ProviderID string `json:"provider_id"`
		Name       string `json:"name"`
		Kind       string `json:"kind"`
		Endpoint   string `json:"endpoint"`
		Enabled    bool   `json:"enabled"`
	}
	items := make([]response, 0)
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.ProviderID, &item.Name, &item.Kind, &item.Endpoint, &item.Enabled); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) generate(ctx *gin.Context) {
	var payload generateRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenant := payload.TenantID
	if tenant == "" {
		tenant = defaultTenantID
	}

	providerName, providerKind, endpoint, _, err := a.resolveProvider(ctx, tenant, payload.AgentVersionID, payload.AgentID, "llm")
	if err != nil {
		providerName = "demo-provider"
		providerKind = "demo"
		endpoint = "internal://demo"
	}

	message := strings.TrimSpace(payload.Message)
	responseText := fmt.Sprintf(
		"[%s/%s] processed request for agent %s: %s",
		providerKind,
		providerName,
		payload.AgentID,
		strings.ToUpper(message),
	)

	if providerKind == "triton" {
		responseText = fmt.Sprintf("[triton:%s] Triton path selected at %s for agent %s", providerName, endpoint, payload.AgentID)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"provider_name": providerName,
		"provider_kind": providerKind,
		"text":          responseText,
	})
}

func (a *apiServer) transcribe(ctx *gin.Context) {
	var payload asrRequest
	_ = ctx.ShouldBindJSON(&payload)
	tenant := payload.TenantID
	if tenant == "" {
		tenant = defaultTenantID
	}
	transcript := strings.TrimSpace(payload.TextHint)
	if transcript == "" && strings.TrimSpace(payload.AudioBase64) != "" {
		transcript = "voice input captured from audio payload"
	}
	if transcript == "" {
		transcript = "voice input captured"
	}
	providerName, providerKind, endpoint, modelSlug, err := a.resolveProvider(ctx, tenant, payload.AgentVersionID, payload.AgentID, "asr")
	if err != nil {
		providerName = "demo-provider"
		providerKind = "demo"
		endpoint = "internal://demo-asr"
		modelSlug = "demo-asr"
	}
	ctx.JSON(http.StatusOK, gin.H{
		"transcript":    transcript,
		"provider_name": providerName,
		"provider_kind": providerKind,
		"endpoint":      endpoint,
		"model_slug":    modelSlug,
		"audio_format":  payload.AudioFormat,
	})
}

func (a *apiServer) synthesize(ctx *gin.Context) {
	var payload ttsRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenant := payload.TenantID
	if tenant == "" {
		tenant = defaultTenantID
	}
	providerName, providerKind, endpoint, modelSlug, err := a.resolveProvider(ctx, tenant, payload.AgentVersionID, payload.AgentID, "tts")
	if err != nil {
		providerName = "demo-provider"
		providerKind = "demo"
		endpoint = "internal://demo-tts"
		modelSlug = "demo-tts"
	}
	audioURL := ""
	if strings.TrimSpace(payload.Text) != "" {
		audioURL = fmt.Sprintf("%s/audio/%s", strings.TrimRight(endpoint, "/"), strings.ReplaceAll(strings.ToLower(modelSlug), " ", "-"))
	}
	ctx.JSON(http.StatusOK, gin.H{
		"provider_name": providerName,
		"provider_kind": providerKind,
		"model_slug":    modelSlug,
		"voice_profile": "default",
		"audio_url":     audioURL,
		"text":          payload.Text,
		"endpoint":      endpoint,
	})
}

func (a *apiServer) resolveProvider(ctx *gin.Context, tenantID string, agentVersionID string, agentID string, capability string) (string, string, string, string, error) {
	var providerName string
	var providerKind string
	var endpoint string
	var modelSlug string
	err := a.db.QueryRow(
		ctx,
		`select p.name, p.kind, p.endpoint, pm.model_slug
		   from control.providers p
		   join control.provider_models pm on pm.provider_id = p.provider_id and pm.capability = $4
		   left join control.agent_model_bindings amb on amb.provider_model_id = pm.provider_model_id and amb.capability = $4
		  where p.tenant_id = $1
		    and p.enabled = true
		    and pm.enabled = true
		    and (
		      amb.agent_version_id = coalesce(nullif($2, '')::uuid, (
		        select current_version_id from control.agents where tenant_id = $1 and agent_id = $3::uuid
		      ))
		      or amb.agent_version_id is null
		    )
		  order by case when amb.agent_version_id is null then 1 else 0 end asc, amb.priority asc nulls last, p.name asc
		  limit 1`,
		tenantID,
		agentVersionID,
		agentID,
		capability,
	).Scan(&providerName, &providerKind, &endpoint, &modelSlug)
	return providerName, providerKind, endpoint, modelSlug, err
}

func envOrDefault(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
