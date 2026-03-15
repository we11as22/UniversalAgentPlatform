package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/db"
	"github.com/asudakov/universal-agent-platform/packages/go-common/httpclient"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0"
const defaultTenantID = "11111111-1111-1111-1111-111111111111"
const defaultUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"

type apiServer struct {
	db            *pgxpool.Pool
	livekitURL    string
	providerURL   string
	transcriptURL string
}

func main() {
	cfg := config.Load("voice-gateway", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()
	api := &apiServer{
		db:            pool,
		livekitURL:    envOrDefault("LIVEKIT_URL", "ws://localhost:7880"),
		providerURL:   envOrDefault("PROVIDER_GATEWAY_URL", "http://localhost:3260"),
		transcriptURL: envOrDefault("TRANSCRIPT_SERVICE_URL", "http://localhost:3291"),
	}

	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.POST("/api/v1/voice/sessions", api.createSession)
	router.POST("/api/v1/voice/transcribe", api.transcribe)
	router.POST("/api/v1/voice/transcribe-inline", api.transcribeInline)
	router.POST("/api/v1/voice/synthesize-inline", api.synthesizeInline)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

func (a *apiServer) createSession(ctx *gin.Context) {
	var payload struct {
		TenantID       string `json:"tenant_id"`
		UserID         string `json:"user_id"`
		ConversationID string `json:"conversation_id"`
		AgentID        string `json:"agent_id"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.AgentID) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "agent_id is required"})
		return
	}
	tenantID := strings.TrimSpace(payload.TenantID)
	if tenantID == "" {
		tenantID = defaultTenantID
	}
	userID, err := a.resolveUserID(ctx, strings.TrimSpace(payload.UserID))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	conversationID := strings.TrimSpace(payload.ConversationID)
	if conversationID == "" {
		conversationID, err = a.createBootstrapConversation(ctx, tenantID, userID, payload.AgentID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	id := uuid.New()
	roomName := "voice-" + id.String()
	if _, err := a.db.Exec(
		ctx,
		`insert into voice.voice_sessions (voice_session_id, tenant_id, conversation_id, agent_id, livekit_room, status, started_at, metrics)
		 values ($1, $2, $3, $4, $5, 'created', now(), '{}'::jsonb)`,
		id, tenantID, conversationID, payload.AgentID, roomName,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"voice_session_id": id.String(),
		"conversation_id":  conversationID,
		"room_name":        roomName,
		"livekit_url":      a.livekitURL,
		"token":            "dev-token",
	})
}

func (a *apiServer) transcribe(ctx *gin.Context) {
	var payload struct {
		VoiceSessionID string `json:"voice_session_id"`
		TextHint       string `json:"text_hint"`
		AudioBase64    string `json:"audio_base64"`
		AudioFormat    string `json:"audio_format"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var agentID string
	_ = a.db.QueryRow(ctx, `select agent_id::text from voice.voice_sessions where voice_session_id = $1::uuid`, payload.VoiceSessionID).Scan(&agentID)
	transcribed, err := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.providerURL+"/api/v1/asr/transcribe",
		map[string]any{
			"agent_id":     agentID,
			"text_hint":    payload.TextHint,
			"audio_base64": payload.AudioBase64,
			"audio_format": payload.AudioFormat,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	_, _ = httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.transcriptURL+"/api/v1/transcripts",
		map[string]any{
			"voice_session_id": payload.VoiceSessionID,
			"speaker":          "user",
			"sequence_no":      1,
			"text":             transcribed["transcript"],
		},
	)
	ctx.JSON(http.StatusOK, gin.H{"transcript": transcribed["transcript"]})
}

func (a *apiServer) transcribeInline(ctx *gin.Context) {
	var payload struct {
		TenantID    string `json:"tenant_id"`
		AgentID     string `json:"agent_id"`
		TextHint    string `json:"text_hint"`
		AudioBase64 string `json:"audio_base64"`
		AudioFormat string `json:"audio_format"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	transcribed, err := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.providerURL+"/api/v1/asr/transcribe",
		map[string]any{
			"tenant_id":    payload.TenantID,
			"agent_id":     payload.AgentID,
			"text_hint":    payload.TextHint,
			"audio_base64": payload.AudioBase64,
			"audio_format": payload.AudioFormat,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"transcript": transcribed["transcript"]})
}

func (a *apiServer) synthesizeInline(ctx *gin.Context) {
	var payload struct {
		TenantID     string `json:"tenant_id"`
		AgentID      string `json:"agent_id"`
		Text         string `json:"text"`
		VoiceProfile string `json:"voice_profile"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	synthesized, err := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.providerURL+"/api/v1/tts/synthesize",
		map[string]any{
			"tenant_id":     payload.TenantID,
			"agent_id":      payload.AgentID,
			"text":          payload.Text,
			"voice_profile": payload.VoiceProfile,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, synthesized)
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func (a *apiServer) resolveUserID(ctx context.Context, raw string) (string, error) {
	if raw == "" {
		return defaultUserID, nil
	}
	if _, err := uuid.Parse(raw); err == nil {
		return raw, nil
	}

	var resolved string
	err := a.db.QueryRow(
		ctx,
		`select user_id::text
		   from iam.users
		  where external_subject = $1 or email = $1 or lower(display_name) = lower($1)
		  limit 1`,
		raw,
	).Scan(&resolved)
	if err == nil {
		return resolved, nil
	}

	return defaultUserID, nil
}

func (a *apiServer) createBootstrapConversation(ctx context.Context, tenantID string, userID string, agentID string) (string, error) {
	conversationID := uuid.New().String()
	title := fmt.Sprintf("Voice session for %s", agentID)
	_, err := a.db.Exec(
		ctx,
		`insert into conversation.conversations (conversation_id, tenant_id, user_id, agent_id, title, archived, metadata, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, false, jsonb_build_object('origin', 'voice-bootstrap'), now(), now())`,
		conversationID, tenantID, userID, agentID, title,
	)
	if err != nil {
		return "", err
	}
	return conversationID, nil
}
