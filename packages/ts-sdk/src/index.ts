export interface AgentSummary {
  agent_id: string;
  slug?: string;
  display_name: string;
  description: string;
  modality: "text" | "voice" | "realtime_voice";
  rag_enabled?: boolean;
}

export interface ConversationSummary {
  conversation_id: string;
  agent_id: string;
  title: string;
}

export interface HealthStatus {
  service: string;
  status: string;
  version: string;
}

export interface AgentInvokeRequest {
  tenant_id?: string;
  user_id?: string;
  message: string;
  metadata?: Record<string, unknown>;
  speak_response?: boolean;
}

export interface AgentVoiceInvokeRequest {
  tenant_id?: string;
  user_id?: string;
  text_hint?: string;
  audio_base64?: string;
  audio_format?: string;
  metadata?: Record<string, unknown>;
  speak_response?: boolean;
}

export interface AgentInvokeResponse {
  agent_id: string;
  agent_version_id: string;
  modality?: "text" | "voice" | "realtime_voice";
  tenant_id: string;
  user_id?: string;
  provider_name: string;
  provider_kind: string;
  rag_enabled: boolean;
  text: string;
  retrieval?: Record<string, unknown> | null;
  metadata?: Record<string, unknown>;
  transcript?: string;
  tts?: Record<string, unknown>;
}

export interface AgentStreamEvent {
  type: string;
  sequence: number;
  timestamp: string;
  payload?: Record<string, unknown>;
}

export interface PerfRunResult {
  metric_name: string;
  metric_value: number;
  unit: string;
  metadata?: Record<string, unknown>;
  created_at?: string;
}

export interface CreateConversationRequest {
  user_id?: string;
  agent_id: string;
  title?: string;
}

export interface CreateConversationResponse {
  conversation_id: string;
}

export interface MessageRecord {
  message_id: string;
  role: string;
  content: string;
}

export interface CreateRunRequest {
  message: string;
  agent_id: string;
}

export interface CreateRunResponse {
  run_id: string;
  message_id: string;
  text: string;
}

async function getJson<T>(url: string): Promise<T> {
  const response = await fetch(url, {
    headers: {
      "content-type": "application/json"
    },
    cache: "no-store"
  });

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return response.json() as Promise<T>;
}

async function postJson<TRequest, TResponse>(url: string, body: TRequest): Promise<TResponse> {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "content-type": "application/json"
    },
    body: JSON.stringify(body),
    cache: "no-store"
  });

  if (!response.ok) {
    throw new Error(`Request failed: ${response.status}`);
  }

  return response.json() as Promise<TResponse>;
}

export const api = {
  health: (baseUrl: string) => getJson<HealthStatus>(`${baseUrl}/api/health`),
  agents: (baseUrl: string) => getJson<AgentSummary[]>(`${baseUrl}/api/v1/agents`),
  invokeAgent: (baseUrl: string, agentId: string, body: AgentInvokeRequest) =>
    postJson<AgentInvokeRequest, AgentInvokeResponse>(`${baseUrl}/api/v1/agents/${agentId}/respond`, body),
  streamAgent: (baseUrl: string, agentId: string, body: AgentInvokeRequest) =>
    fetch(`${baseUrl}/api/v1/agents/${agentId}/respond/stream`, {
      method: "POST",
      headers: {
        "content-type": "application/json"
      },
      body: JSON.stringify(body),
      cache: "no-store"
    }),
  streamAgentUrl: (baseUrl: string, agentId: string, message: string) =>
    `${baseUrl}/api/v1/agents/${agentId}/respond/stream?message=${encodeURIComponent(message)}`,
  streamAgentWebSocketUrl: (baseUrl: string, agentId: string) =>
    `${baseUrl.replace(/^http/, "ws")}/api/v1/agents/${agentId}/respond/ws`,
  streamConversationWebSocketUrl: (baseUrl: string, conversationId: string) =>
    `${baseUrl.replace(/^http/, "ws")}/api/v1/conversations/${conversationId}/runs/ws`,
  invokeAgentFromVoice: (baseUrl: string, agentId: string, body: AgentVoiceInvokeRequest) =>
    postJson<AgentVoiceInvokeRequest, AgentInvokeResponse>(`${baseUrl}/api/v1/agents/${agentId}/respond-from-voice`, body),
  conversations: (baseUrl: string) => getJson<ConversationSummary[]>(`${baseUrl}/api/v1/conversations`),
  createConversation: (baseUrl: string, body: CreateConversationRequest) =>
    postJson<CreateConversationRequest, CreateConversationResponse>(`${baseUrl}/api/v1/conversations`, body),
  messages: (baseUrl: string, conversationId: string) =>
    getJson<MessageRecord[]>(`${baseUrl}/api/v1/conversations/${conversationId}/messages`),
  startRun: (baseUrl: string, conversationId: string, body: CreateRunRequest) =>
    postJson<CreateRunRequest, CreateRunResponse>(`${baseUrl}/api/v1/conversations/${conversationId}/runs`, body),
  perfRunResults: (baseUrl: string, perfRunId: string) =>
    getJson<PerfRunResult[]>(`${baseUrl}/api/v1/perf/runs/${perfRunId}/results`)
};
