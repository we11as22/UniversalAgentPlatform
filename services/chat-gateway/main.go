package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/httpclient"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const version = "0.1.0"
const defaultTenantID = "11111111-1111-1111-1111-111111111111"
const defaultUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"

type proxy struct {
	conversationURL string
	agentRouterURL  string
	adminAPIURL     string
	voiceGatewayURL string
	runStore        sync.Map
}

type createConversationRequest struct {
	UserID  string `json:"user_id"`
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
}

type createMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type createRunRequest struct {
	Message string `json:"message"`
	AgentID string `json:"agent_id"`
}

type invokeAgentRequest struct {
	TenantID      string         `json:"tenant_id"`
	UserID        string         `json:"user_id"`
	Message       string         `json:"message"`
	Metadata      map[string]any `json:"metadata"`
	SpeakResponse bool           `json:"speak_response"`
}

type invokeVoiceAgentRequest struct {
	TenantID      string         `json:"tenant_id"`
	UserID        string         `json:"user_id"`
	TextHint      string         `json:"text_hint"`
	AudioBase64   string         `json:"audio_base64"`
	AudioFormat   string         `json:"audio_format"`
	Metadata      map[string]any `json:"metadata"`
	SpeakResponse bool           `json:"speak_response"`
}

type routerRequest struct {
	TenantID       string `json:"tenant_id"`
	AgentID        string `json:"agent_id"`
	AgentVersionID string `json:"agent_version_id"`
	Message        string `json:"message"`
	RAGEnabled     bool   `json:"rag_enabled"`
}

type agentRecord struct {
	AgentID          string `json:"agent_id"`
	CurrentVersionID string `json:"current_version_id"`
	Modality         string `json:"modality"`
	RAGEnabled       bool   `json:"rag_enabled"`
}

type agentResponse struct {
	ProviderName string `json:"provider_name"`
	ProviderKind string `json:"provider_kind"`
	Text         string `json:"text"`
	Retrieval    any    `json:"retrieval,omitempty"`
}

type websocketInvokeRequest struct {
	Type          string         `json:"type"`
	TenantID      string         `json:"tenant_id"`
	UserID        string         `json:"user_id"`
	Message       string         `json:"message"`
	Metadata      map[string]any `json:"metadata"`
	SpeakResponse bool           `json:"speak_response"`
	AgentID       string         `json:"agent_id"`
}

type websocketEvent struct {
	Type      string         `json:"type"`
	Sequence  int            `json:"sequence"`
	Timestamp string         `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	cfg := config.Load("chat-gateway", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	api := &proxy{
		conversationURL: envOrDefault("CONVERSATION_SERVICE_URL", "http://localhost:3240"),
		agentRouterURL:  envOrDefault("AGENT_ROUTER_URL", "http://localhost:3250"),
		adminAPIURL:     envOrDefault("ADMIN_API_URL", "http://localhost:3210"),
		voiceGatewayURL: envOrDefault("VOICE_GATEWAY_URL", "http://localhost:3270"),
	}

	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/agents", api.listAgents)
	router.POST("/api/v1/agents/:agentID/respond", api.invokeAgent)
	router.GET("/api/v1/agents/:agentID/respond/stream", api.streamAgent)
	router.POST("/api/v1/agents/:agentID/respond/stream", api.streamAgent)
	router.GET("/api/v1/agents/:agentID/respond/ws", api.streamAgentWebSocket)
	router.POST("/api/v1/agents/:agentID/respond-from-voice", api.invokeAgentFromVoice)
	router.GET("/api/v1/conversations", api.listConversations)
	router.POST("/api/v1/conversations", api.createConversation)
	router.GET("/api/v1/conversations/search", api.searchConversations)
	router.POST("/api/v1/conversations/:conversationID/clone", api.cloneConversation)
	router.GET("/api/v1/conversations/:conversationID/messages", api.listMessages)
	router.POST("/api/v1/conversations/:conversationID/runs", api.startRun)
	router.GET("/api/v1/conversations/:conversationID/runs/ws", api.streamConversationWebSocket)
	router.POST("/api/v1/files/upload", api.uploadFile)
	router.GET("/api/v1/runs/:runID/events", api.streamRun)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

func (p *proxy) listAgents(ctx *gin.Context) {
	response, err := httpclient.GetJSON[any](ctx, p.adminAPIURL+"/api/v1/agents")
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (p *proxy) listConversations(ctx *gin.Context) {
	url := fmt.Sprintf("%s/api/v1/conversations?tenant_id=%s&user_id=%s", p.conversationURL, defaultTenantID, defaultUserID)
	response, err := httpclient.GetJSON[any](ctx, url)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (p *proxy) createConversation(ctx *gin.Context) {
	var payload createConversationRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.UserID == "" {
		payload.UserID = defaultUserID
	}
	if payload.Title == "" {
		payload.Title = "New chat"
	}
	response, err := httpclient.PostJSON[createConversationRequest, map[string]string](ctx, p.conversationURL+"/api/v1/conversations", payload)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, response)
}

func (p *proxy) searchConversations(ctx *gin.Context) {
	response, err := httpclient.GetJSON[[]map[string]any](ctx, fmt.Sprintf("%s/api/v1/conversations?tenant_id=%s&user_id=%s", p.conversationURL, defaultTenantID, defaultUserID))
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	query := strings.ToLower(strings.TrimSpace(ctx.Query("q")))
	if query == "" {
		ctx.JSON(http.StatusOK, response)
		return
	}
	filtered := make([]map[string]any, 0)
	for _, item := range response {
		title, _ := item["title"].(string)
		if strings.Contains(strings.ToLower(title), query) {
			filtered = append(filtered, item)
		}
	}
	ctx.JSON(http.StatusOK, filtered)
}

func (p *proxy) listMessages(ctx *gin.Context) {
	url := fmt.Sprintf("%s/api/v1/conversations/%s/messages?tenant_id=%s", p.conversationURL, ctx.Param("conversationID"), defaultTenantID)
	response, err := httpclient.GetJSON[any](ctx, url)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, response)
}

func (p *proxy) cloneConversation(ctx *gin.Context) {
	var payload struct {
		AgentID string `json:"agent_id"`
		Title   string `json:"title"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.Title == "" {
		payload.Title = "Cloned chat"
	}
	created, err := httpclient.PostJSON[createConversationRequest, map[string]string](
		ctx,
		p.conversationURL+"/api/v1/conversations",
		createConversationRequest{UserID: defaultUserID, AgentID: payload.AgentID, Title: payload.Title},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	originalMessages, err := httpclient.GetJSON[[]map[string]any](ctx, fmt.Sprintf("%s/api/v1/conversations/%s/messages?tenant_id=%s", p.conversationURL, ctx.Param("conversationID"), defaultTenantID))
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	for _, item := range originalMessages {
		role, _ := item["role"].(string)
		content, _ := item["content"].(string)
		_, _ = httpclient.PostJSON[createMessageRequest, map[string]string](
			ctx,
			fmt.Sprintf("%s/api/v1/conversations/%s/messages?tenant_id=%s", p.conversationURL, created["conversation_id"], defaultTenantID),
			createMessageRequest{Role: role, Content: content},
		)
	}
	ctx.JSON(http.StatusCreated, created)
}

func (p *proxy) startRun(ctx *gin.Context) {
	var payload createRunRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := requestTenantID(ctx, defaultTenantID)
	userID := requestUserID(ctx, defaultUserID)
	runResult, err := p.executeConversationRun(ctx, tenantID, userID, ctx.Param("conversationID"), payload.AgentID, payload.Message)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"run_id":     runResult.RunID,
		"message_id": runResult.AssistantMessageID,
		"text":       runResult.Text,
		"user_id":    userID,
	})
}

func (p *proxy) invokeAgent(ctx *gin.Context) {
	var payload invokeAgentRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.Message) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	tenantID := payload.TenantID
	if tenantID == "" {
		tenantID = requestTenantID(ctx, defaultTenantID)
	}
	userID := payload.UserID
	if userID == "" {
		userID = requestUserID(ctx, defaultUserID)
	}

	agentInfo, err := p.resolveAgent(ctx, ctx.Param("agentID"), tenantID)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	agentResult, err := p.executeAgent(ctx, tenantID, ctx.Param("agentID"), agentInfo.CurrentVersionID, payload.Message, agentInfo.RAGEnabled)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"agent_id":         ctx.Param("agentID"),
		"agent_version_id": agentInfo.CurrentVersionID,
		"modality":         agentInfo.Modality,
		"tenant_id":        tenantID,
		"user_id":          userID,
		"provider_name":    agentResult.ProviderName,
		"provider_kind":    agentResult.ProviderKind,
		"rag_enabled":      agentInfo.RAGEnabled,
		"text":             agentResult.Text,
		"retrieval":        agentResult.Retrieval,
		"metadata":         payload.Metadata,
	})
}

func (p *proxy) streamAgent(ctx *gin.Context) {
	message := strings.TrimSpace(ctx.Query("message"))
	if message == "" && ctx.Request.Method == http.MethodPost {
		var payload invokeAgentRequest
		if err := ctx.ShouldBindJSON(&payload); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		message = strings.TrimSpace(payload.Message)
	}
	if message == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	tenantID := requestTenantID(ctx, defaultTenantID)
	userID := requestUserID(ctx, defaultUserID)
	agentInfo, err := p.resolveAgent(ctx, ctx.Param("agentID"), tenantID)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	agentResult, err := p.executeAgent(ctx, tenantID, ctx.Param("agentID"), agentInfo.CurrentVersionID, message, agentInfo.RAGEnabled)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	p.streamText(ctx, agentResult.Text, gin.H{
		"status":           "completed",
		"agent_id":         ctx.Param("agentID"),
		"agent_version_id": agentInfo.CurrentVersionID,
		"modality":         agentInfo.Modality,
		"tenant_id":        tenantID,
		"user_id":          userID,
		"provider_name":    agentResult.ProviderName,
		"provider_kind":    agentResult.ProviderKind,
		"rag_enabled":      agentInfo.RAGEnabled,
	})
}

func (p *proxy) invokeAgentFromVoice(ctx *gin.Context) {
	var payload invokeVoiceAgentRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.TextHint) == "" && strings.TrimSpace(payload.AudioBase64) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "text_hint or audio_base64 is required"})
		return
	}

	tenantID := payload.TenantID
	if tenantID == "" {
		tenantID = requestTenantID(ctx, defaultTenantID)
	}
	userID := payload.UserID
	if userID == "" {
		userID = requestUserID(ctx, defaultUserID)
	}

	agentInfo, err := p.resolveAgent(ctx, ctx.Param("agentID"), tenantID)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	transcriptResponse, err := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		p.voiceGatewayURL+"/api/v1/voice/transcribe-inline",
		map[string]any{
			"agent_id":     ctx.Param("agentID"),
			"text_hint":    payload.TextHint,
			"audio_base64": payload.AudioBase64,
			"audio_format": payload.AudioFormat,
			"tenant_id":    tenantID,
			"user_id":      userID,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	transcript, _ := transcriptResponse["transcript"].(string)
	agentResult, err := p.executeAgent(ctx, tenantID, ctx.Param("agentID"), agentInfo.CurrentVersionID, transcript, agentInfo.RAGEnabled)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	responsePayload := gin.H{
		"agent_id":         ctx.Param("agentID"),
		"agent_version_id": agentInfo.CurrentVersionID,
		"modality":         agentInfo.Modality,
		"tenant_id":        tenantID,
		"user_id":          userID,
		"transcript":       transcript,
		"provider_name":    agentResult.ProviderName,
		"provider_kind":    agentResult.ProviderKind,
		"rag_enabled":      agentInfo.RAGEnabled,
		"text":             agentResult.Text,
		"retrieval":        agentResult.Retrieval,
		"metadata":         payload.Metadata,
	}
	if payload.SpeakResponse || agentInfo.Modality != "text" {
		ttsResult, err := httpclient.PostJSON[map[string]any, map[string]any](
			ctx,
			p.voiceGatewayURL+"/api/v1/voice/synthesize-inline",
			map[string]any{
				"tenant_id": tenantID,
				"agent_id":  ctx.Param("agentID"),
				"text":      agentResult.Text,
			},
		)
		if err == nil {
			responsePayload["tts"] = ttsResult
		}
	}
	ctx.JSON(http.StatusOK, responsePayload)
}

func (p *proxy) streamRun(ctx *gin.Context) {
	runID := ctx.Param("runID")
	var (
		value any
		ok    bool
	)
	for attempt := 0; attempt < 120; attempt++ {
		value, ok = p.runStore.Load(runID)
		if ok {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	text, _ := value.(string)
	p.streamText(ctx, text, gin.H{"status": "completed"})
}

func (p *proxy) streamAgentWebSocket(ctx *gin.Context) {
	connection, err := websocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	connection.SetReadLimit(1 << 20)
	_ = connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		var payload websocketInvokeRequest
		if err := connection.ReadJSON(&payload); err != nil {
			return
		}
		message := strings.TrimSpace(payload.Message)
		if message == "" {
			_ = p.writeWebsocketEvent(connection, websocketEvent{
				Type:      "run.failed",
				Sequence:  1,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"error": "message is required"},
			})
			continue
		}

		tenantID := payload.TenantID
		if tenantID == "" {
			tenantID = requestTenantID(ctx, defaultTenantID)
		}
		userID := payload.UserID
		if userID == "" {
			userID = requestUserID(ctx, defaultUserID)
		}

		if err := p.streamAgentWebsocketResponse(ctx, connection, ctx.Param("agentID"), tenantID, userID, message, payload.Metadata); err != nil {
			_ = p.writeWebsocketEvent(connection, websocketEvent{
				Type:      "run.failed",
				Sequence:  1,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"error": err.Error()},
			})
		}
	}
}

func (p *proxy) streamConversationWebSocket(ctx *gin.Context) {
	connection, err := websocketUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer connection.Close()

	connection.SetReadLimit(1 << 20)
	_ = connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		var payload websocketInvokeRequest
		if err := connection.ReadJSON(&payload); err != nil {
			return
		}
		message := strings.TrimSpace(payload.Message)
		agentID := strings.TrimSpace(payload.AgentID)
		if message == "" || agentID == "" {
			_ = p.writeWebsocketEvent(connection, websocketEvent{
				Type:      "run.failed",
				Sequence:  1,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"error": "message and agent_id are required"},
			})
			continue
		}

		tenantID := payload.TenantID
		if tenantID == "" {
			tenantID = requestTenantID(ctx, defaultTenantID)
		}
		userID := payload.UserID
		if userID == "" {
			userID = requestUserID(ctx, defaultUserID)
		}

		if err := p.streamConversationWebsocketResponse(ctx, connection, tenantID, userID, ctx.Param("conversationID"), agentID, message, payload.Metadata); err != nil {
			_ = p.writeWebsocketEvent(connection, websocketEvent{
				Type:      "run.failed",
				Sequence:  1,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"error": err.Error()},
			})
		}
	}
}

func (p *proxy) uploadFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	targetDir := filepath.Join(os.TempDir(), "uap-uploads")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	targetPath := filepath.Join(targetDir, filepath.Base(file.Filename))
	if err := ctx.SaveUploadedFile(file, targetPath); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"file_name": file.Filename,
		"size":      file.Size,
		"path":      targetPath,
	})
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func requestTenantID(ctx *gin.Context, fallback string) string {
	if value := strings.TrimSpace(ctx.Query("tenant_id")); value != "" {
		return value
	}
	if value := strings.TrimSpace(ctx.GetHeader("X-Tenant-ID")); value != "" {
		return value
	}
	return fallback
}

func requestUserID(ctx *gin.Context, fallback string) string {
	if value := strings.TrimSpace(ctx.Query("user_id")); value != "" {
		return value
	}
	if value := strings.TrimSpace(ctx.GetHeader("X-User-ID")); value != "" {
		return value
	}
	return fallback
}

func (p *proxy) resolveAgent(ctx *gin.Context, agentID string, tenantID string) (agentRecord, error) {
	agentInfo, err := httpclient.GetJSON[agentRecord](ctx, fmt.Sprintf("%s/api/v1/agents/%s?tenant_id=%s", p.adminAPIURL, agentID, tenantID))
	if err != nil {
		return agentRecord{}, err
	}
	if strings.TrimSpace(agentInfo.CurrentVersionID) == "" {
		return agentRecord{}, fmt.Errorf("agent %s has no current_version_id", agentID)
	}
	return agentInfo, nil
}

func (p *proxy) executeAgent(ctx *gin.Context, tenantID string, agentID string, agentVersionID string, message string, ragEnabled bool) (agentResponse, error) {
	return httpclient.PostJSON[routerRequest, agentResponse](
		ctx,
		p.agentRouterURL+"/api/v1/respond",
		routerRequest{
			TenantID:       tenantID,
			AgentID:        agentID,
			AgentVersionID: agentVersionID,
			Message:        message,
			RAGEnabled:     ragEnabled,
		},
	)
}

type conversationRunResult struct {
	RunID              string
	UserMessageID      string
	AssistantMessageID string
	Text               string
	AgentInfo          agentRecord
	AgentResult        agentResponse
}

type conversationRunDraft struct {
	RunID          string
	UserMessageID  string
	AgentInfo      agentRecord
	ConversationID string
	AgentID        string
	Message        string
}

func (p *proxy) beginConversationRun(ctx *gin.Context, tenantID string, conversationID string, agentID string, message string) (conversationRunDraft, error) {
	agentInfo, err := p.resolveAgent(ctx, agentID, tenantID)
	if err != nil {
		return conversationRunDraft{}, err
	}

	userMessage, err := httpclient.PostJSON[createMessageRequest, map[string]string](
		ctx,
		fmt.Sprintf("%s/api/v1/conversations/%s/messages?tenant_id=%s", p.conversationURL, conversationID, tenantID),
		createMessageRequest{Role: "user", Content: message},
	)
	if err != nil {
		return conversationRunDraft{}, err
	}

	runCreated, err := httpclient.PostJSON[map[string]string, map[string]string](
		ctx,
		p.conversationURL+"/api/v1/runs?tenant_id="+tenantID,
		map[string]string{
			"conversation_id":  conversationID,
			"agent_version_id": agentInfo.CurrentVersionID,
			"user_message_id":  userMessage["message_id"],
		},
	)
	if err != nil {
		return conversationRunDraft{}, err
	}

	return conversationRunDraft{
		RunID:          runCreated["run_id"],
		UserMessageID:  userMessage["message_id"],
		AgentInfo:      agentInfo,
		ConversationID: conversationID,
		AgentID:        agentID,
		Message:        message,
	}, nil
}

func (p *proxy) completeConversationRun(ctx *gin.Context, tenantID string, draft conversationRunDraft) (conversationRunResult, error) {
	agentResult, err := p.executeAgent(ctx, tenantID, draft.AgentID, draft.AgentInfo.CurrentVersionID, draft.Message, draft.AgentInfo.RAGEnabled)
	if err != nil {
		p.failConversationRun(ctx, tenantID, draft.RunID)
		return conversationRunResult{}, err
	}

	assistantMessage, err := httpclient.PostJSON[createMessageRequest, map[string]string](
		ctx,
		fmt.Sprintf("%s/api/v1/conversations/%s/messages?tenant_id=%s", p.conversationURL, draft.ConversationID, tenantID),
		createMessageRequest{Role: "assistant", Content: agentResult.Text},
	)
	if err != nil {
		p.failConversationRun(ctx, tenantID, draft.RunID)
		return conversationRunResult{}, err
	}

	_, _ = httpclient.PostJSON[map[string]string, map[string]any](
		ctx,
		fmt.Sprintf("%s/api/v1/runs/%s/complete?tenant_id=%s", p.conversationURL, draft.RunID, tenantID),
		map[string]string{
			"assistant_message_id": assistantMessage["message_id"],
			"status":               "completed",
		},
	)

	p.runStore.Store(draft.RunID, agentResult.Text)
	return conversationRunResult{
		RunID:              draft.RunID,
		UserMessageID:      draft.UserMessageID,
		AssistantMessageID: assistantMessage["message_id"],
		Text:               agentResult.Text,
		AgentInfo:          draft.AgentInfo,
		AgentResult:        agentResult,
	}, nil
}

func (p *proxy) executeConversationRun(ctx *gin.Context, tenantID string, userID string, conversationID string, agentID string, message string) (conversationRunResult, error) {
	draft, err := p.beginConversationRun(ctx, tenantID, conversationID, agentID, message)
	if err != nil {
		return conversationRunResult{}, err
	}
	_ = userID
	return p.completeConversationRun(ctx, tenantID, draft)
}

func (p *proxy) failConversationRun(ctx *gin.Context, tenantID string, runID string) {
	_, _ = httpclient.PostJSON[map[string]string, map[string]any](
		ctx,
		fmt.Sprintf("%s/api/v1/runs/%s/complete?tenant_id=%s", p.conversationURL, runID, tenantID),
		map[string]string{
			"status": "failed",
		},
	)
}

func (p *proxy) streamText(ctx *gin.Context, text string, completedPayload gin.H) {
	parts := strings.Fields(text)
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	ctx.Stream(func(w io.Writer) bool {
		for index, part := range parts {
			payload := fmt.Sprintf("event: message.delta\ndata: {\"sequence\":%d,\"delta\":\"%s\"}\n\n", index+1, strings.ReplaceAll(part, "\"", "\\\""))
			_, _ = w.Write([]byte(payload))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(120 * time.Millisecond)
		}
		_, _ = w.Write([]byte(fmt.Sprintf("event: run.completed\ndata: %s\n\n", mustJSON(completedPayload))))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return false
	})
}

func (p *proxy) streamAgentWebsocketResponse(ctx *gin.Context, connection *websocket.Conn, agentID string, tenantID string, userID string, message string, metadata map[string]any) error {
	agentInfo, err := p.resolveAgent(ctx, agentID, tenantID)
	if err != nil {
		return err
	}
	agentResult, err := p.executeAgent(ctx, tenantID, agentID, agentInfo.CurrentVersionID, message, agentInfo.RAGEnabled)
	if err != nil {
		return err
	}
	return p.writeTokenStream(connection, agentResult.Text, 1, gin.H{
		"transport":        "websocket",
		"agent_id":         agentID,
		"agent_version_id": agentInfo.CurrentVersionID,
		"modality":         agentInfo.Modality,
		"tenant_id":        tenantID,
		"user_id":          userID,
		"rag_enabled":      agentInfo.RAGEnabled,
		"metadata":         metadata,
	}, gin.H{
		"status":           "completed",
		"agent_id":         agentID,
		"agent_version_id": agentInfo.CurrentVersionID,
		"modality":         agentInfo.Modality,
		"tenant_id":        tenantID,
		"user_id":          userID,
		"provider_name":    agentResult.ProviderName,
		"provider_kind":    agentResult.ProviderKind,
		"rag_enabled":      agentInfo.RAGEnabled,
		"retrieval":        agentResult.Retrieval,
		"metadata":         metadata,
	})
}

func (p *proxy) streamConversationWebsocketResponse(ctx *gin.Context, connection *websocket.Conn, tenantID string, userID string, conversationID string, agentID string, message string, metadata map[string]any) error {
	draft, err := p.beginConversationRun(ctx, tenantID, conversationID, agentID, message)
	if err != nil {
		return err
	}
	if err := p.writeWebsocketEvent(connection, websocketEvent{
		Type:      "run.started",
		Sequence:  1,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]any{
			"transport":        "websocket",
			"transport_mode":   "primary",
			"run_id":           draft.RunID,
			"user_message_id":  draft.UserMessageID,
			"conversation_id":  conversationID,
			"agent_id":         agentID,
			"agent_version_id": draft.AgentInfo.CurrentVersionID,
			"modality":         draft.AgentInfo.Modality,
			"tenant_id":        tenantID,
			"user_id":          userID,
			"rag_enabled":      draft.AgentInfo.RAGEnabled,
		},
	}); err != nil {
		return err
	}
	runResult, err := p.completeConversationRun(ctx, tenantID, draft)
	if err != nil {
		return err
	}
	return p.writeTokenStream(connection, runResult.Text, 2, nil, gin.H{
		"status":               "completed",
		"run_id":               runResult.RunID,
		"message_id":           runResult.AssistantMessageID,
		"conversation_id":      conversationID,
		"agent_id":             agentID,
		"agent_version_id":     runResult.AgentInfo.CurrentVersionID,
		"modality":             runResult.AgentInfo.Modality,
		"tenant_id":            tenantID,
		"user_id":              userID,
		"provider_name":        runResult.AgentResult.ProviderName,
		"provider_kind":        runResult.AgentResult.ProviderKind,
		"rag_enabled":          runResult.AgentInfo.RAGEnabled,
		"retrieval":            runResult.AgentResult.Retrieval,
		"metadata":             metadata,
		"transport_preference": "websocket",
	})
}

func (p *proxy) writeTokenStream(connection *websocket.Conn, text string, sequence int, startedPayload gin.H, completedPayload gin.H) error {
	parts := strings.Fields(text)
	if startedPayload != nil {
		startedAt := time.Now().UTC().Format(time.RFC3339Nano)
		if err := p.writeWebsocketEvent(connection, websocketEvent{
			Type:      "run.started",
			Sequence:  sequence,
			Timestamp: startedAt,
			Payload:   startedPayload,
		}); err != nil {
			return err
		}
		sequence++
	}

	for index, part := range parts {
		if index > 0 && index%6 == 0 {
			if err := p.writeWebsocketEvent(connection, websocketEvent{
				Type:      "stream.heartbeat",
				Sequence:  sequence,
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Payload:   map[string]any{"status": "streaming"},
			}); err != nil {
				return err
			}
			sequence++
		}
		if err := p.writeWebsocketEvent(connection, websocketEvent{
			Type:      "message.delta",
			Sequence:  sequence,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:   map[string]any{"delta": part},
		}); err != nil {
			return err
		}
		sequence++
		time.Sleep(120 * time.Millisecond)
	}

	return p.writeWebsocketEvent(connection, websocketEvent{
		Type:      "run.completed",
		Sequence:  sequence,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   completedPayload,
	})
}

func (p *proxy) writeWebsocketEvent(connection *websocket.Conn, event websocketEvent) error {
	_ = connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return connection.WriteJSON(event)
}

func mustJSON(payload any) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{\"status\":\"completed\"}"
	}
	return string(raw)
}
