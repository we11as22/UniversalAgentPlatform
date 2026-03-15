package main

import (
	"context"
	"net/http"

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

type apiServer struct {
	db *pgxpool.Pool
}

func main() {
	cfg := config.Load("audit-service", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()

	api := &apiServer{db: pool}
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/audit-events", api.list)
	router.POST("/api/v1/audit-events", api.create)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

func (a *apiServer) list(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select audit_event_id, action, resource_type, created_at
		   from audit.audit_events
		  where tenant_id = $1
		  order by created_at desc
		  limit 100`,
		defaultTenantID,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id, action, resourceType string
		var createdAt string
		if err := rows.Scan(&id, &action, &resourceType, &createdAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"audit_event_id": id, "action": action, "resource_type": resourceType, "created_at": createdAt})
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) create(ctx *gin.Context) {
	var payload struct {
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into audit.audit_events (audit_event_id, tenant_id, actor_user_id, action, resource_type, resource_id, payload, created_at)
		 values ($1, $2, $3, $4, $5, nullif($6, '')::uuid, '{}'::jsonb, now())`,
		identifier, defaultTenantID, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", payload.Action, payload.ResourceType, payload.ResourceID,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"audit_event_id": identifier.String()})
}

