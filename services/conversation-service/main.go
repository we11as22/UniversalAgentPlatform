package main

import (
	"context"
	"net/http"
	"time"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/db"
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
	db *pgxpool.Pool
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
	ConversationID    string `json:"conversation_id"`
	AgentVersionID    string `json:"agent_version_id"`
	UserMessageID     string `json:"user_message_id"`
	AssistantMessageID string `json:"assistant_message_id,omitempty"`
}

func main() {
	cfg := config.Load("conversation-service", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()

	api := &apiServer{db: pool}
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/conversations", api.listConversations)
	router.POST("/api/v1/conversations", api.createConversation)
	router.GET("/api/v1/conversations/:conversationID/messages", api.listMessages)
	router.POST("/api/v1/conversations/:conversationID/messages", api.createMessage)
	router.POST("/api/v1/runs", api.createRun)
	router.POST("/api/v1/runs/:runID/complete", api.completeRun)
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

func (a *apiServer) listConversations(ctx *gin.Context) {
	userID := ctx.Query("user_id")
	if userID == "" {
		userID = defaultUserID
	}
	rows, err := a.db.Query(
		ctx,
		`select conversation_id, agent_id, title, updated_at
		   from conversation.conversations
		  where tenant_id = $1 and user_id = $2
		  order by updated_at desc`,
		tenantID(ctx), userID,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type response struct {
		ConversationID string    `json:"conversation_id"`
		AgentID        string    `json:"agent_id"`
		Title          string    `json:"title"`
		UpdatedAt      time.Time `json:"updated_at"`
	}

	items := make([]response, 0)
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.ConversationID, &item.AgentID, &item.Title, &item.UpdatedAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) createConversation(ctx *gin.Context) {
	var payload createConversationRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.UserID == "" {
		payload.UserID = defaultUserID
	}
	if payload.Title == "" {
		payload.Title = "New conversation"
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into conversation.conversations (conversation_id, tenant_id, user_id, agent_id, title, archived, metadata, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, false, '{}'::jsonb, now(), now())`,
		identifier, tenantID(ctx), payload.UserID, payload.AgentID, payload.Title,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"conversation_id": identifier.String()})
}

func (a *apiServer) listMessages(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select message_id, role, status, content, created_at
		   from conversation.messages
		  where tenant_id = $1 and conversation_id = $2
		  order by created_at asc`,
		tenantID(ctx), ctx.Param("conversationID"),
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type response struct {
		MessageID  string    `json:"message_id"`
		Role       string    `json:"role"`
		Status     string    `json:"status"`
		Content    string    `json:"content"`
		CreatedAt  time.Time `json:"created_at"`
	}

	items := make([]response, 0)
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.MessageID, &item.Role, &item.Status, &item.Content, &item.CreatedAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) createMessage(ctx *gin.Context) {
	var payload createMessageRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into conversation.messages (message_id, tenant_id, conversation_id, role, status, content, metadata, created_at)
		 values ($1, $2, $3, $4, 'complete', $5, '{}'::jsonb, now())`,
		identifier, tenantID(ctx), ctx.Param("conversationID"), payload.Role, payload.Content,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := a.db.Exec(
		ctx,
		`update conversation.conversations set updated_at = now() where conversation_id = $1 and tenant_id = $2`,
		ctx.Param("conversationID"), tenantID(ctx),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message_id": identifier.String()})
}

func (a *apiServer) createRun(ctx *gin.Context) {
	var payload createRunRequest
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	runID := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into conversation.runs (run_id, tenant_id, conversation_id, agent_version_id, user_message_id, assistant_message_id, status, started_at, metadata)
		 values ($1, $2, $3, $4, $5, nullif($6, '')::uuid, 'running', now(), '{}'::jsonb)`,
		runID, tenantID(ctx), payload.ConversationID, payload.AgentVersionID, payload.UserMessageID, payload.AssistantMessageID,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"run_id": runID.String()})
}

func (a *apiServer) completeRun(ctx *gin.Context) {
	var payload struct {
		AssistantMessageID string `json:"assistant_message_id"`
		Status             string `json:"status"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.Status == "" {
		payload.Status = "completed"
	}
	if _, err := a.db.Exec(
		ctx,
		`update conversation.runs
		    set assistant_message_id = nullif($2, '')::uuid,
		        status = $3,
		        completed_at = now()
		  where run_id = $1 and tenant_id = $4`,
		ctx.Param("runID"), payload.AssistantMessageID, payload.Status, tenantID(ctx),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"run_id": ctx.Param("runID"), "status": payload.Status})
}

