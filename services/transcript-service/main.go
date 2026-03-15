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
	cfg := config.Load("transcript-service", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()

	api := &apiServer{db: pool}
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.POST("/api/v1/transcripts", api.create)
	router.GET("/api/v1/transcripts", api.list)
	if err := router.Run(":" + cfg.HTTPPort); err != nil {
		logger.Fatal("server failed")
	}
}

func (a *apiServer) create(ctx *gin.Context) {
	var payload struct {
		VoiceSessionID string `json:"voice_session_id"`
		MessageID      string `json:"message_id"`
		Speaker        string `json:"speaker"`
		SequenceNo     int    `json:"sequence_no"`
		Text           string `json:"text"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into voice.transcripts (transcript_id, tenant_id, voice_session_id, message_id, speaker, sequence_no, transcript_text, confidence, created_at)
		 values ($1, $2, $3, nullif($4, '')::uuid, $5, $6, $7, 100, now())`,
		identifier, defaultTenantID, payload.VoiceSessionID, payload.MessageID, payload.Speaker, payload.SequenceNo, payload.Text,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"transcript_id": identifier.String()})
}

func (a *apiServer) list(ctx *gin.Context) {
	rows, err := a.db.Query(ctx, `select transcript_id, speaker, sequence_no, transcript_text from voice.transcripts where tenant_id = $1 order by created_at desc limit 100`, defaultTenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id, speaker, text string
		var sequence int
		if err := rows.Scan(&id, &speaker, &sequence, &text); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"transcript_id": id, "speaker": speaker, "sequence_no": sequence, "text": text})
	}
	ctx.JSON(http.StatusOK, items)
}

