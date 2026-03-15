package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asudakov/universal-agent-platform/packages/go-common/config"
	"github.com/asudakov/universal-agent-platform/packages/go-common/db"
	"github.com/asudakov/universal-agent-platform/packages/go-common/httpclient"
	"github.com/asudakov/universal-agent-platform/packages/go-common/observability"
	"github.com/asudakov/universal-agent-platform/packages/go-common/server"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version = "0.1.0"
const defaultTenantID = "11111111-1111-1111-1111-111111111111"
const defaultUserID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"

type apiServer struct {
	db         *pgxpool.Pool
	indexerURL string
}

type agentPayload struct {
	Slug               string          `json:"slug"`
	DisplayName        string          `json:"display_name"`
	Description        string          `json:"description"`
	Modality           string          `json:"modality"`
	SystemPrompt       string          `json:"system_prompt"`
	PromptTemplate     string          `json:"prompt_template"`
	Config             json.RawMessage `json:"config"`
	Policies           json.RawMessage `json:"policies"`
	ProviderModelID    string          `json:"provider_model_id"`
	ASRProviderModelID string          `json:"asr_provider_model_id"`
	TTSProviderModelID string          `json:"tts_provider_model_id"`
	Tools              []string        `json:"tools"`
	RAGEnabled         bool            `json:"rag_enabled"`
}

type providerPayload struct {
	Name                 string          `json:"name"`
	Kind                 string          `json:"kind"`
	Endpoint             string          `json:"endpoint"`
	Metadata             json.RawMessage `json:"metadata"`
	CredentialRefType    string          `json:"credential_ref_type"`
	CredentialRefLocator string          `json:"credential_ref_locator"`
}

type providerModelPayload struct {
	ProviderID  string          `json:"provider_id"`
	Capability  string          `json:"capability"`
	ModelSlug   string          `json:"model_slug"`
	DisplayName string          `json:"display_name"`
	Streaming   bool            `json:"streaming"`
	Config      json.RawMessage `json:"config"`
}

type knowledgePayload struct {
	AgentID string `json:"agent_id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type agentCreateInput struct {
	TenantID           string
	Slug               string
	DisplayName        string
	Description        string
	Modality           string
	SystemPrompt       string
	PromptTemplate     string
	Config             json.RawMessage
	Policies           json.RawMessage
	ProviderModelID    string
	ASRProviderModelID string
	TTSProviderModelID string
	Tools              []string
	RAGEnabled         bool
}

func main() {
	cfg := config.Load("admin-api", "8080")
	logger := observability.MustLogger(cfg.ServiceName)
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("db connect failed")
	}
	defer pool.Close()

	api := &apiServer{
		db:         pool,
		indexerURL: envOrDefault("INDEXER_URL", "http://indexer:8000"),
	}
	router := server.NewRouter(cfg.ServiceName, version, logger)
	router.GET("/api/v1/agents", api.listAgents)
	router.GET("/api/v1/agents/:agentID", api.getAgent)
	router.POST("/api/v1/agents", api.createAgent)
	router.PUT("/api/v1/agents/:agentID", api.updateAgent)
	router.POST("/api/v1/agents/install/rag-example", api.installExampleRAGAgent)
	router.GET("/api/v1/providers", api.listProviders)
	router.POST("/api/v1/providers", api.createProvider)
	router.PUT("/api/v1/providers/:providerID", api.updateProvider)
	router.GET("/api/v1/provider-models", api.listProviderModels)
	router.POST("/api/v1/provider-models", api.createProviderModel)
	router.PUT("/api/v1/provider-models/:providerModelID", api.updateProviderModel)
	router.POST("/api/v1/knowledge/index", api.indexKnowledge)
	router.GET("/api/v1/dashboard", api.dashboard)
	router.GET("/api/v1/perf/profiles", api.listPerfProfiles)
	router.GET("/api/v1/perf/runs", api.listPerfRuns)
	router.GET("/api/v1/perf/runs/:perfRunID/results", api.listPerfRunResults)
	router.POST("/api/v1/perf/runs", api.createPerfRun)
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

func (a *apiServer) listAgents(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select a.agent_id,
		        a.slug,
		        a.display_name,
		        a.description,
		        a.modality,
		        a.status,
		        coalesce(a.current_version_id::text, ''),
		        exists(
		          select 1
		            from control.agent_tool_bindings tb
		           where tb.agent_version_id = a.current_version_id
		             and tb.tool_name = 'tenant_knowledge_search'
		        ) as rag_enabled
		   from control.agents a
		  where a.tenant_id = $1
		  order by a.display_name`,
		tenantID(ctx),
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type response struct {
		AgentID          string `json:"agent_id"`
		Slug             string `json:"slug"`
		DisplayName      string `json:"display_name"`
		Description      string `json:"description"`
		Modality         string `json:"modality"`
		Status           string `json:"status"`
		CurrentVersionID string `json:"current_version_id"`
		RAGEnabled       bool   `json:"rag_enabled"`
	}

	items := make([]response, 0)
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.AgentID, &item.Slug, &item.DisplayName, &item.Description, &item.Modality, &item.Status, &item.CurrentVersionID, &item.RAGEnabled); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) getAgent(ctx *gin.Context) {
	tenant := tenantID(ctx)

	var agentID, slug, displayName, description, modality, status, currentVersionID string
	var systemPrompt, promptTemplate string
	var configJSON []byte
	var policiesJSON []byte
	err := a.db.QueryRow(
		ctx,
		`select a.agent_id,
		        a.slug,
		        a.display_name,
		        a.description,
		        a.modality,
		        a.status,
		        a.current_version_id::text,
		        av.system_prompt,
		        av.prompt_template,
		        av.config,
		        av.policies
		   from control.agents a
		   join control.agent_versions av on av.agent_version_id = a.current_version_id
		  where a.tenant_id = $1 and a.agent_id = $2`,
		tenant,
		ctx.Param("agentID"),
	).Scan(&agentID, &slug, &displayName, &description, &modality, &status, &currentVersionID, &systemPrompt, &promptTemplate, &configJSON, &policiesJSON)
	if err != nil {
		if err == pgx.ErrNoRows {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	bindingsRows, err := a.db.Query(
		ctx,
		`select amb.capability,
		        pm.provider_model_id::text,
		        pm.model_slug,
		        pm.display_name,
		        p.name
		   from control.agent_model_bindings amb
		   join control.provider_models pm on pm.provider_model_id = amb.provider_model_id
		   join control.providers p on p.provider_id = pm.provider_id
		  where amb.tenant_id = $1 and amb.agent_version_id = $2::uuid
		  order by amb.priority asc`,
		tenant,
		currentVersionID,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer bindingsRows.Close()

	bindings := make([]gin.H, 0)
	for bindingsRows.Next() {
		var capability, providerModelID, modelSlug, modelDisplayName, providerName string
		if err := bindingsRows.Scan(&capability, &providerModelID, &modelSlug, &modelDisplayName, &providerName); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		bindings = append(bindings, gin.H{
			"capability":        capability,
			"provider_model_id": providerModelID,
			"model_slug":        modelSlug,
			"display_name":      modelDisplayName,
			"provider_name":     providerName,
		})
	}

	toolsRows, err := a.db.Query(
		ctx,
		`select tool_name
		   from control.agent_tool_bindings
		  where tenant_id = $1 and agent_version_id = $2::uuid
		  order by tool_name`,
		tenant,
		currentVersionID,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer toolsRows.Close()

	tools := make([]string, 0)
	ragEnabled := false
	for toolsRows.Next() {
		var tool string
		if err := toolsRows.Scan(&tool); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tools = append(tools, tool)
		if tool == "tenant_knowledge_search" {
			ragEnabled = true
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"agent_id":              agentID,
		"slug":                  slug,
		"display_name":          displayName,
		"description":           description,
		"modality":              modality,
		"status":                status,
		"current_version_id":    currentVersionID,
		"system_prompt":         systemPrompt,
		"prompt_template":       promptTemplate,
		"config":                json.RawMessage(configJSON),
		"policies":              json.RawMessage(policiesJSON),
		"tools":                 tools,
		"rag_enabled":           ragEnabled,
		"llm_provider_model_id": firstBindingByCapability(bindings, "llm"),
		"asr_provider_model_id": firstBindingByCapability(bindings, "asr"),
		"tts_provider_model_id": firstBindingByCapability(bindings, "tts"),
		"bindings":              bindings,
	})
}

func (a *apiServer) createAgent(ctx *gin.Context) {
	var payload agentPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := tenantID(ctx)
	tx, err := a.db.Begin(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	agentID, versionID, err := a.createAgentRecord(ctx, tx, agentCreateInput{
		TenantID:           tenant,
		Slug:               payload.Slug,
		DisplayName:        payload.DisplayName,
		Description:        payload.Description,
		Modality:           payload.Modality,
		SystemPrompt:       payload.SystemPrompt,
		PromptTemplate:     payload.PromptTemplate,
		Config:             payload.Config,
		Policies:           payload.Policies,
		ProviderModelID:    payload.ProviderModelID,
		ASRProviderModelID: payload.ASRProviderModelID,
		TTSProviderModelID: payload.TTSProviderModelID,
		Tools:              payload.Tools,
		RAGEnabled:         payload.RAGEnabled,
	})
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"agent_id": agentID.String(), "agent_version_id": versionID.String()})
}

func (a *apiServer) updateAgent(ctx *gin.Context) {
	var payload agentPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := tenantID(ctx)
	agentIdentifier, err := uuid.Parse(ctx.Param("agentID"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent id"})
		return
	}

	tx, err := a.db.Begin(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	var nextVersionNumber int
	if err := tx.QueryRow(
		ctx,
		`select coalesce(max(version_number), 0) + 1
		   from control.agent_versions
		  where tenant_id = $1 and agent_id = $2`,
		tenant,
		agentIdentifier,
	).Scan(&nextVersionNumber); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if _, err := tx.Exec(
		ctx,
		`update control.agents
		    set slug = $3,
		        display_name = $4,
		        description = $5,
		        modality = $6,
		        updated_at = now()
		  where tenant_id = $1 and agent_id = $2`,
		tenant,
		agentIdentifier,
		payload.Slug,
		payload.DisplayName,
		payload.Description,
		payload.Modality,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	versionID := uuid.New()
	configJSON := defaultJSON(payload.Config, `{}`)
	policiesJSON := defaultJSON(payload.Policies, `{"retention_days":30}`)

	if _, err := tx.Exec(
		ctx,
		`insert into control.agent_versions (agent_version_id, tenant_id, agent_id, version_number, system_prompt, prompt_template, config, policies, signed_by)
		 values ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, 'admin-api')`,
		versionID,
		tenant,
		agentIdentifier,
		nextVersionNumber,
		defaultIfEmpty(payload.SystemPrompt, "You are a production enterprise agent."),
		defaultIfEmpty(payload.PromptTemplate, "Answer using the selected provider and available knowledge."),
		configJSON,
		policiesJSON,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(payload.ProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, tenant, versionID, payload.ProviderModelID, "llm", 10); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if strings.TrimSpace(payload.ASRProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, tenant, versionID, payload.ASRProviderModelID, "asr", 20); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if strings.TrimSpace(payload.TTSProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, tenant, versionID, payload.TTSProviderModelID, "tts", 30); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	toolSet := make(map[string]struct{})
	for _, tool := range payload.Tools {
		trimmed := strings.TrimSpace(tool)
		if trimmed != "" {
			toolSet[trimmed] = struct{}{}
		}
	}
	if payload.RAGEnabled {
		toolSet["tenant_knowledge_search"] = struct{}{}
	}
	for tool := range toolSet {
		if _, err := tx.Exec(
			ctx,
			`insert into control.agent_tool_bindings (agent_tool_binding_id, tenant_id, agent_version_id, tool_name, config)
			 values ($1, $2, $3, $4, '{}'::jsonb)`,
			uuid.New(),
			tenant,
			versionID,
			tool,
		); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if _, err := tx.Exec(
		ctx,
		`update control.agents set current_version_id = $3, updated_at = now() where tenant_id = $1 and agent_id = $2`,
		tenant,
		agentIdentifier,
		versionID,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"agent_id": agentIdentifier.String(), "agent_version_id": versionID.String(), "version_number": nextVersionNumber})
}

func (a *apiServer) createAgentRecord(ctx context.Context, tx pgx.Tx, input agentCreateInput) (uuid.UUID, uuid.UUID, error) {
	if strings.TrimSpace(input.Slug) == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("slug is required")
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("display_name is required")
	}
	if input.Modality == "" {
		input.Modality = "text"
	}
	if input.SystemPrompt == "" {
		input.SystemPrompt = "You are a production enterprise agent."
	}
	if input.PromptTemplate == "" {
		input.PromptTemplate = "Answer using the agent configuration and the available knowledge."
	}

	agentID := uuid.New()
	versionID := uuid.New()
	configJSON := defaultJSON(input.Config, `{}`)
	policiesJSON := defaultJSON(input.Policies, `{"retention_days":30}`)

	if _, err := tx.Exec(
		ctx,
		`insert into control.agents (agent_id, tenant_id, slug, display_name, description, modality, status, created_by, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, $6, 'active', $7::uuid, now(), now())`,
		agentID, input.TenantID, input.Slug, input.DisplayName, input.Description, input.Modality, defaultUserID,
	); err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	if _, err := tx.Exec(
		ctx,
		`insert into control.agent_versions (agent_version_id, tenant_id, agent_id, version_number, system_prompt, prompt_template, config, policies, signed_by)
		 values ($1, $2, $3, 1, $4, $5, $6::jsonb, $7::jsonb, 'admin-api')`,
		versionID, input.TenantID, agentID, input.SystemPrompt, input.PromptTemplate, configJSON, policiesJSON,
	); err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	if _, err := tx.Exec(ctx, `update control.agents set current_version_id = $2 where agent_id = $1`, agentID, versionID); err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	if strings.TrimSpace(input.ProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, input.TenantID, versionID, input.ProviderModelID, "llm", 10); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}
	if strings.TrimSpace(input.ASRProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, input.TenantID, versionID, input.ASRProviderModelID, "asr", 20); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}
	if strings.TrimSpace(input.TTSProviderModelID) != "" {
		if err := insertAgentBinding(ctx, tx, input.TenantID, versionID, input.TTSProviderModelID, "tts", 30); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}

	toolSet := make(map[string]struct{})
	for _, tool := range input.Tools {
		trimmed := strings.TrimSpace(tool)
		if trimmed != "" {
			toolSet[trimmed] = struct{}{}
		}
	}
	if input.RAGEnabled {
		toolSet["tenant_knowledge_search"] = struct{}{}
	}
	for tool := range toolSet {
		if _, err := tx.Exec(
			ctx,
			`insert into control.agent_tool_bindings (agent_tool_binding_id, tenant_id, agent_version_id, tool_name, config)
			 values ($1, $2, $3, $4, '{}'::jsonb)`,
			uuid.New(), input.TenantID, versionID, tool,
		); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}

	return agentID, versionID, nil
}

func (a *apiServer) listProviders(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select provider_id, name, kind, endpoint, enabled, metadata
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
		ProviderID string          `json:"provider_id"`
		Name       string          `json:"name"`
		Kind       string          `json:"kind"`
		Endpoint   string          `json:"endpoint"`
		Enabled    bool            `json:"enabled"`
		Metadata   json.RawMessage `json:"metadata"`
	}

	items := make([]response, 0)
	for rows.Next() {
		var item response
		if err := rows.Scan(&item.ProviderID, &item.Name, &item.Kind, &item.Endpoint, &item.Enabled, &item.Metadata); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, item)
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) createProvider(ctx *gin.Context) {
	var payload providerPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := tenantID(ctx)
	tx, err := a.db.Begin(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	providerID := uuid.New()
	if _, err := tx.Exec(
		ctx,
		`insert into control.providers (provider_id, tenant_id, name, kind, endpoint, enabled, metadata, created_at, updated_at)
		 values ($1, $2, $3, $4, $5, true, $6::jsonb, now(), now())`,
		providerID, tenant, payload.Name, payload.Kind, payload.Endpoint, defaultJSON(payload.Metadata, `{}`),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(payload.CredentialRefType) != "" && strings.TrimSpace(payload.CredentialRefLocator) != "" {
		if _, err := tx.Exec(
			ctx,
			`insert into control.provider_credentials_refs (credential_ref_id, tenant_id, provider_id, ref_type, ref_locator, metadata)
			 values ($1, $2, $3, $4, $5, '{}'::jsonb)`,
			uuid.New(), tenant, providerID, payload.CredentialRefType, payload.CredentialRefLocator,
		); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"provider_id": providerID.String()})
}

func (a *apiServer) updateProvider(ctx *gin.Context) {
	var payload providerPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant := tenantID(ctx)
	providerID, err := uuid.Parse(ctx.Param("providerID"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider id"})
		return
	}

	tx, err := a.db.Begin(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`update control.providers
		    set name = $3,
		        kind = $4,
		        endpoint = $5,
		        metadata = $6::jsonb,
		        updated_at = now()
		  where tenant_id = $1 and provider_id = $2`,
		tenant,
		providerID,
		payload.Name,
		payload.Kind,
		payload.Endpoint,
		defaultJSON(payload.Metadata, `{}`),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if _, err := tx.Exec(ctx, `delete from control.provider_credentials_refs where tenant_id = $1 and provider_id = $2`, tenant, providerID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.CredentialRefType) != "" && strings.TrimSpace(payload.CredentialRefLocator) != "" {
		if _, err := tx.Exec(
			ctx,
			`insert into control.provider_credentials_refs (credential_ref_id, tenant_id, provider_id, ref_type, ref_locator, metadata)
			 values ($1, $2, $3, $4, $5, '{}'::jsonb)`,
			uuid.New(),
			tenant,
			providerID,
			payload.CredentialRefType,
			payload.CredentialRefLocator,
		); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"provider_id": providerID.String()})
}

func (a *apiServer) listProviderModels(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select pm.provider_model_id,
		        pm.provider_id,
		        pm.capability,
		        pm.model_slug,
		        pm.display_name,
		        pm.streaming,
		        pm.enabled,
		        pm.config,
		        p.name
		   from control.provider_models pm
		   join control.providers p on p.provider_id = pm.provider_id
		  where pm.tenant_id = $1
		  order by p.name, pm.capability, pm.display_name`,
		tenantID(ctx),
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var providerModelID, providerID, capability, modelSlug, displayName, providerName string
		var streaming, enabled bool
		var configJSON []byte
		if err := rows.Scan(&providerModelID, &providerID, &capability, &modelSlug, &displayName, &streaming, &enabled, &configJSON, &providerName); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{
			"provider_model_id": providerModelID,
			"provider_id":       providerID,
			"provider_name":     providerName,
			"capability":        capability,
			"model_slug":        modelSlug,
			"display_name":      displayName,
			"streaming":         streaming,
			"enabled":           enabled,
			"config":            json.RawMessage(configJSON),
		})
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) createProviderModel(ctx *gin.Context) {
	var payload providerModelPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into control.provider_models (provider_model_id, tenant_id, provider_id, capability, model_slug, display_name, streaming, config, enabled, created_at, updated_at)
		 values ($1, $2, $3::uuid, $4, $5, $6, $7, $8::jsonb, true, now(), now())`,
		identifier, tenantID(ctx), payload.ProviderID, payload.Capability, payload.ModelSlug, payload.DisplayName, payload.Streaming, defaultJSON(payload.Config, `{}`),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"provider_model_id": identifier.String()})
}

func (a *apiServer) updateProviderModel(ctx *gin.Context) {
	var payload providerModelPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	identifier, err := uuid.Parse(ctx.Param("providerModelID"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider model id"})
		return
	}
	if _, err := a.db.Exec(
		ctx,
		`update control.provider_models
		    set provider_id = $3::uuid,
		        capability = $4,
		        model_slug = $5,
		        display_name = $6,
		        streaming = $7,
		        config = $8::jsonb,
		        updated_at = now()
		  where tenant_id = $1 and provider_model_id = $2`,
		tenantID(ctx),
		identifier,
		payload.ProviderID,
		payload.Capability,
		payload.ModelSlug,
		payload.DisplayName,
		payload.Streaming,
		defaultJSON(payload.Config, `{}`),
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"provider_model_id": identifier.String()})
}

func (a *apiServer) indexKnowledge(ctx *gin.Context) {
	var payload knowledgePayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.AgentID) == "" || strings.TrimSpace(payload.Content) == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "agent_id and content are required"})
		return
	}

	tenant := tenantID(ctx)
	chunks := splitKnowledge(payload.Content, 900)
	documentID := uuid.New().String()
	response, err := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.indexerURL+"/api/v1/index",
		map[string]any{
			"tenant_id":   tenant,
			"agent_id":    payload.AgentID,
			"document_id": documentID,
			"title":       defaultIfEmpty(payload.Title, "Knowledge document"),
			"chunks":      chunks,
		},
	)
	if err != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"agent_id":       payload.AgentID,
		"document_id":    documentID,
		"indexed_chunks": response["indexed_chunks"],
		"collection":     response["collection_name"],
	})
}

func (a *apiServer) installExampleRAGAgent(ctx *gin.Context) {
	tenant := tenantID(ctx)

	var existingAgentID string
	var existingVersionID string
	err := a.db.QueryRow(
		ctx,
		`select agent_id::text, current_version_id::text
		   from control.agents
		  where tenant_id = $1 and slug = 'qdrant-rag-search'`,
		tenant,
	).Scan(&existingAgentID, &existingVersionID)
	if err != nil && err != pgx.ErrNoRows {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	agentID := existingAgentID
	versionID := existingVersionID
	created := false

	if err == pgx.ErrNoRows {
		var providerModelID string
		err = a.db.QueryRow(
			ctx,
			`select pm.provider_model_id::text
			   from control.provider_models pm
			   join control.providers p on p.provider_id = pm.provider_id
			  where pm.tenant_id = $1
			    and pm.capability = 'llm'
			    and pm.enabled = true
			    and p.enabled = true
			  order by p.kind = 'demo' desc, p.name asc
			  limit 1`,
			tenant,
		).Scan(&providerModelID)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "no enabled llm provider model found for tenant"})
			return
		}

		tx, beginErr := a.db.Begin(ctx)
		if beginErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": beginErr.Error()})
			return
		}
		defer tx.Rollback(ctx)

		newAgentID, newVersionID, createErr := a.createAgentRecord(ctx, tx, agentCreateInput{
			TenantID:        tenant,
			Slug:            "qdrant-rag-search",
			DisplayName:     "Qdrant Knowledge Agent",
			Description:     "Answers from tenant knowledge using Qdrant-backed full-text retrieval.",
			Modality:        "text",
			SystemPrompt:    "You are a RAG agent. Prefer retrieved tenant knowledge over generic phrasing.",
			PromptTemplate:  "Answer using retrieved knowledge. If knowledge is missing, state that clearly.",
			ProviderModelID: providerModelID,
			RAGEnabled:      true,
			Tools:           []string{"tenant_knowledge_search"},
			Config:          json.RawMessage(`{"answer_style":"grounded","retrieval_scope":"tenant+agent"}`),
			Policies:        json.RawMessage(`{"retention_days":30}`),
		})
		if createErr != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": createErr.Error()})
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": commitErr.Error()})
			return
		}
		agentID = newAgentID.String()
		versionID = newVersionID.String()
		created = true
	}

	knowledge := strings.TrimSpace(`
UniversalAgentPlatform operational handbook

The Qdrant Knowledge Agent answers by searching its tenant knowledge base.
Knowledge lives in Qdrant and is scoped by tenant_id and agent_id.
When a user asks a question, the agent-runtime calls rag-service first and then routes generation through provider-gateway.
The admin panel can install this example agent and upload additional documents into its knowledge base.

Support notes

To test the agent, open Chat Web, create a new chat with Qdrant Knowledge Agent, and ask:
- Where does this agent keep its knowledge?
- How is retrieval scoped?
- Which service calls retrieval before provider generation?

Expected grounded facts

Knowledge is stored in Qdrant.
Retrieval is filtered by both tenant_id and agent_id.
agent-runtime invokes rag-service before provider-gateway generation.
`)

	indexResponse, indexErr := httpclient.PostJSON[map[string]any, map[string]any](
		ctx,
		a.indexerURL+"/api/v1/index",
		map[string]any{
			"tenant_id":   tenant,
			"agent_id":    agentID,
			"document_id": uuid.New().String(),
			"title":       "UniversalAgentPlatform handbook",
			"chunks":      splitKnowledge(knowledge, 900),
		},
	)
	if indexErr != nil {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": indexErr.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"agent_id":         agentID,
		"agent_version_id": versionID,
		"created":          created,
		"indexed_chunks":   indexResponse["indexed_chunks"],
		"collection":       indexResponse["collection_name"],
	})
}

func (a *apiServer) dashboard(ctx *gin.Context) {
	tenant := tenantID(ctx)
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var agents int
	var providers int
	var providerModels int
	var conversations int
	_ = a.db.QueryRow(timeoutCtx, `select count(*) from control.agents where tenant_id = $1`, tenant).Scan(&agents)
	_ = a.db.QueryRow(timeoutCtx, `select count(*) from control.providers where tenant_id = $1`, tenant).Scan(&providers)
	_ = a.db.QueryRow(timeoutCtx, `select count(*) from control.provider_models where tenant_id = $1`, tenant).Scan(&providerModels)
	_ = a.db.QueryRow(timeoutCtx, `select count(*) from conversation.conversations where tenant_id = $1`, tenant).Scan(&conversations)

	ctx.JSON(http.StatusOK, gin.H{
		"tenant_id":       tenant,
		"agents":          agents,
		"providers":       providers,
		"provider_models": providerModels,
		"conversations":   conversations,
		"build_version":   version,
		"current_time":    time.Now().UTC(),
		"hostname":        hostname(),
	})
}

func (a *apiServer) listPerfProfiles(ctx *gin.Context) {
	rows, err := a.db.Query(ctx, `select perf_profile_id, name, profile_type, config from perf.perf_profiles where tenant_id = $1 order by name`, tenantID(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id, name, profileType string
		var configJSON []byte
		if err := rows.Scan(&id, &name, &profileType, &configJSON); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{"perf_profile_id": id, "name": name, "profile_type": profileType, "config": string(configJSON)})
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) listPerfRuns(ctx *gin.Context) {
	rows, err := a.db.Query(ctx, `select perf_run_id, status, target_environment, git_sha, build_version, started_at from perf.perf_runs where tenant_id = $1 order by started_at desc limit 20`, tenantID(ctx))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := make([]gin.H, 0)
	for rows.Next() {
		var id, status, environment, gitSHA, buildVersion string
		var startedAt time.Time
		if err := rows.Scan(&id, &status, &environment, &gitSHA, &buildVersion, &startedAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{
			"perf_run_id":        id,
			"status":             status,
			"target_environment": environment,
			"git_sha":            gitSHA,
			"build_version":      buildVersion,
			"started_at":         startedAt,
		})
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) listPerfRunResults(ctx *gin.Context) {
	rows, err := a.db.Query(
		ctx,
		`select metric_name, metric_value, unit, metadata, created_at
		   from perf.perf_run_results
		  where tenant_id = $1 and perf_run_id = $2::uuid
		  order by metric_name asc`,
		tenantID(ctx),
		ctx.Param("perfRunID"),
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var metricName string
		var metricValue float64
		var unit string
		var metadata []byte
		var createdAt time.Time
		if err := rows.Scan(&metricName, &metricValue, &unit, &metadata, &createdAt); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		items = append(items, gin.H{
			"metric_name":  metricName,
			"metric_value": metricValue,
			"unit":         unit,
			"metadata":     json.RawMessage(metadata),
			"created_at":   createdAt,
		})
	}
	ctx.JSON(http.StatusOK, items)
}

func (a *apiServer) createPerfRun(ctx *gin.Context) {
	var payload struct {
		PerfProfileID string `json:"perf_profile_id"`
		TargetEnv     string `json:"target_environment"`
		GitSHA        string `json:"git_sha"`
		BuildVersion  string `json:"build_version"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.TargetEnv == "" {
		payload.TargetEnv = "local-api"
	}
	if payload.GitSHA == "" {
		payload.GitSHA = "working-tree"
	}
	if payload.BuildVersion == "" {
		payload.BuildVersion = version
	}
	identifier := uuid.New()
	if _, err := a.db.Exec(
		ctx,
		`insert into perf.perf_runs (perf_run_id, tenant_id, perf_profile_id, status, target_environment, git_sha, build_version, started_at, metadata)
		 values ($1, $2, $3, 'queued', $4, $5, $6, now(), '{}'::jsonb)`,
		identifier, tenantID(ctx), payload.PerfProfileID, payload.TargetEnv, payload.GitSHA, payload.BuildVersion,
	); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"perf_run_id": identifier.String(), "status": "queued"})
}

func splitKnowledge(content string, maxChunkSize int) []string {
	parts := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n")
	chunks := make([]string, 0)
	var current strings.Builder
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if current.Len() > 0 && current.Len()+2+len(trimmed) > maxChunkSize {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(trimmed)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	if len(chunks) == 0 {
		return []string{strings.TrimSpace(content)}
	}
	return chunks
}

func defaultJSON(raw json.RawMessage, fallback string) []byte {
	if len(bytes.TrimSpace(raw)) == 0 {
		return []byte(fallback)
	}
	return raw
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return value
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func insertAgentBinding(ctx context.Context, tx pgx.Tx, tenantID string, agentVersionID uuid.UUID, providerModelID string, capability string, priority int) error {
	var actualCapability string
	if err := tx.QueryRow(
		ctx,
		`select capability
		   from control.provider_models
		  where tenant_id = $1 and provider_model_id = $2::uuid and enabled = true`,
		tenantID,
		providerModelID,
	).Scan(&actualCapability); err != nil {
		return fmt.Errorf("provider model lookup failed: %w", err)
	}
	if actualCapability != capability {
		return fmt.Errorf("provider model must have capability %s, got %s", capability, actualCapability)
	}
	_, err := tx.Exec(
		ctx,
		`insert into control.agent_model_bindings (agent_model_binding_id, tenant_id, agent_version_id, provider_model_id, priority, capability)
		 values ($1, $2, $3, $4::uuid, $5, $6)`,
		uuid.New(),
		tenantID,
		agentVersionID,
		providerModelID,
		priority,
		capability,
	)
	return err
}

func firstBindingByCapability(bindings []gin.H, capability string) string {
	for _, binding := range bindings {
		value, _ := binding["capability"].(string)
		if value != capability {
			continue
		}
		providerModelID, _ := binding["provider_model_id"].(string)
		return providerModelID
	}
	return ""
}
