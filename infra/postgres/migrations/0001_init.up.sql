create schema if not exists iam;
create schema if not exists control;
create schema if not exists conversation;
create schema if not exists voice;
create schema if not exists audit;
create schema if not exists perf;

create table if not exists iam.tenants (
  tenant_id uuid primary key,
  slug text not null unique,
  name text not null,
  status text not null default 'active',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists iam.users (
  user_id uuid primary key,
  external_subject text not null unique,
  email text not null unique,
  display_name text not null,
  status text not null default 'active',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists iam.roles (
  role_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  name text not null,
  description text not null default '',
  permissions jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now(),
  unique (tenant_id, name)
);

create table if not exists iam.memberships (
  membership_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  user_id uuid not null references iam.users(user_id),
  role_id uuid references iam.roles(role_id),
  group_name text not null default '',
  status text not null default 'active',
  created_at timestamptz not null default now(),
  unique (tenant_id, user_id, role_id, group_name)
);

create table if not exists control.providers (
  provider_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  name text not null,
  kind text not null check (kind in ('triton', 'openai-compatible', 'byo', 'demo')),
  endpoint text not null,
  enabled boolean not null default true,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (tenant_id, name)
);

create table if not exists control.provider_credentials_refs (
  credential_ref_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  provider_id uuid not null references control.providers(provider_id) on delete cascade,
  ref_type text not null check (ref_type in ('k8s-secret', 'vault', 'file', 'env')),
  ref_locator text not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists control.provider_models (
  provider_model_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  provider_id uuid not null references control.providers(provider_id) on delete cascade,
  capability text not null check (capability in ('llm', 'asr', 'tts', 'embedding')),
  model_slug text not null,
  display_name text not null,
  streaming boolean not null default false,
  config jsonb not null default '{}'::jsonb,
  enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (tenant_id, provider_id, model_slug, capability)
);

create table if not exists control.provider_health (
  provider_health_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  provider_id uuid not null references control.providers(provider_id) on delete cascade,
  status text not null check (status in ('unknown', 'healthy', 'degraded', 'unhealthy')),
  latency_ms integer not null default 0,
  error_rate numeric(5,2) not null default 0,
  checked_at timestamptz not null default now(),
  details jsonb not null default '{}'::jsonb
);

create table if not exists control.routing_policies (
  routing_policy_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  name text not null,
  capability text not null,
  policy jsonb not null,
  enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (tenant_id, name)
);

create table if not exists control.cost_policies (
  cost_policy_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  name text not null,
  policy jsonb not null,
  enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (tenant_id, name)
);

create table if not exists control.agents (
  agent_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  slug text not null,
  display_name text not null,
  description text not null default '',
  modality text not null check (modality in ('text', 'voice', 'realtime_voice')),
  status text not null check (status in ('draft', 'active', 'disabled')),
  current_version_id uuid,
  created_by uuid references iam.users(user_id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (tenant_id, slug)
);

create table if not exists control.agent_versions (
  agent_version_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  agent_id uuid not null references control.agents(agent_id) on delete cascade,
  version_number integer not null,
  system_prompt text not null,
  prompt_template text not null,
  config jsonb not null default '{}'::jsonb,
  policies jsonb not null default '{}'::jsonb,
  signed_by text not null default 'bootstrap',
  created_at timestamptz not null default now(),
  unique (agent_id, version_number)
);

alter table control.agents
  add constraint fk_agents_current_version
  foreign key (current_version_id) references control.agent_versions(agent_version_id);

create table if not exists control.agent_model_bindings (
  agent_model_binding_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  agent_version_id uuid not null references control.agent_versions(agent_version_id) on delete cascade,
  provider_model_id uuid not null references control.provider_models(provider_model_id),
  priority integer not null default 100,
  capability text not null,
  created_at timestamptz not null default now()
);

create table if not exists control.agent_tool_bindings (
  agent_tool_binding_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  agent_version_id uuid not null references control.agent_versions(agent_version_id) on delete cascade,
  tool_name text not null,
  config jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (agent_version_id, tool_name)
);

create table if not exists conversation.conversations (
  conversation_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  user_id uuid not null references iam.users(user_id),
  agent_id uuid not null references control.agents(agent_id),
  title text not null,
  archived boolean not null default false,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists conversation.messages (
  message_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  conversation_id uuid not null references conversation.conversations(conversation_id) on delete cascade,
  role text not null check (role in ('system', 'user', 'assistant', 'tool')),
  status text not null default 'complete',
  content text not null default '',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists conversation.message_parts (
  message_part_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  message_id uuid not null references conversation.messages(message_id) on delete cascade,
  part_type text not null check (part_type in ('text', 'markdown', 'code', 'file', 'citation', 'tool_event', 'transcript')),
  sequence_no integer not null,
  mime_type text not null default 'text/plain',
  body text not null default '',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (message_id, sequence_no)
);

create table if not exists conversation.files (
  file_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  conversation_id uuid references conversation.conversations(conversation_id) on delete cascade,
  message_id uuid references conversation.messages(message_id) on delete set null,
  object_key text not null,
  file_name text not null,
  content_type text not null,
  size_bytes bigint not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists conversation.runs (
  run_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  conversation_id uuid not null references conversation.conversations(conversation_id) on delete cascade,
  agent_version_id uuid not null references control.agent_versions(agent_version_id),
  user_message_id uuid references conversation.messages(message_id),
  assistant_message_id uuid references conversation.messages(message_id),
  status text not null check (status in ('queued', 'running', 'completed', 'failed', 'cancelled')),
  started_at timestamptz not null default now(),
  completed_at timestamptz,
  metadata jsonb not null default '{}'::jsonb
);

create table if not exists conversation.run_steps (
  run_step_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  run_id uuid not null references conversation.runs(run_id) on delete cascade,
  step_type text not null,
  sequence_no integer not null,
  status text not null default 'completed',
  input jsonb not null default '{}'::jsonb,
  output jsonb not null default '{}'::jsonb,
  started_at timestamptz not null default now(),
  completed_at timestamptz,
  unique (run_id, sequence_no)
);

create table if not exists conversation.tool_invocations (
  tool_invocation_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  run_step_id uuid not null references conversation.run_steps(run_step_id) on delete cascade,
  tool_name text not null,
  request jsonb not null default '{}'::jsonb,
  status text not null default 'completed',
  created_at timestamptz not null default now()
);

create table if not exists conversation.tool_results (
  tool_result_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  tool_invocation_id uuid not null references conversation.tool_invocations(tool_invocation_id) on delete cascade,
  response jsonb not null default '{}'::jsonb,
  success boolean not null default true,
  created_at timestamptz not null default now()
);

create table if not exists voice.voice_sessions (
  voice_session_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  conversation_id uuid not null references conversation.conversations(conversation_id) on delete cascade,
  agent_id uuid not null references control.agents(agent_id),
  livekit_room text not null,
  status text not null check (status in ('created', 'active', 'ended', 'failed')),
  started_at timestamptz not null default now(),
  ended_at timestamptz,
  metrics jsonb not null default '{}'::jsonb
);

create table if not exists voice.voice_events (
  voice_event_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  voice_session_id uuid not null references voice.voice_sessions(voice_session_id) on delete cascade,
  event_type text not null,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists voice.transcripts (
  transcript_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  voice_session_id uuid not null references voice.voice_sessions(voice_session_id) on delete cascade,
  message_id uuid references conversation.messages(message_id) on delete set null,
  speaker text not null check (speaker in ('user', 'agent')),
  sequence_no integer not null,
  transcript_text text not null,
  confidence numeric(5,2) not null default 0,
  created_at timestamptz not null default now(),
  unique (voice_session_id, sequence_no)
);

create table if not exists conversation.documents (
  document_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  file_id uuid references conversation.files(file_id) on delete set null,
  agent_id uuid references control.agents(agent_id) on delete set null,
  title text not null,
  status text not null default 'indexed',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists conversation.document_chunks (
  chunk_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  document_id uuid not null references conversation.documents(document_id) on delete cascade,
  sequence_no integer not null,
  chunk_text text not null,
  token_count integer not null default 0,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (document_id, sequence_no)
);

create table if not exists conversation.indexes (
  index_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  document_id uuid not null references conversation.documents(document_id) on delete cascade,
  backend text not null default 'qdrant',
  collection_name text not null,
  status text not null default 'ready',
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists audit.usage_events (
  usage_event_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  user_id uuid references iam.users(user_id),
  agent_id uuid references control.agents(agent_id),
  provider_id uuid references control.providers(provider_id),
  event_type text not null,
  quantity numeric(12,2) not null default 0,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists audit.audit_events (
  audit_event_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  actor_user_id uuid references iam.users(user_id),
  action text not null,
  resource_type text not null,
  resource_id uuid,
  payload jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists perf.perf_profiles (
  perf_profile_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  name text not null,
  profile_type text not null check (profile_type in ('smoke', 'load', 'stress', 'spike', 'soak', 'failure-injection-lite')),
  config jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (tenant_id, name)
);

create table if not exists perf.perf_runs (
  perf_run_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  perf_profile_id uuid not null references perf.perf_profiles(perf_profile_id),
  status text not null check (status in ('queued', 'running', 'completed', 'failed')),
  target_environment text not null,
  git_sha text not null,
  build_version text not null,
  started_at timestamptz not null default now(),
  completed_at timestamptz,
  metadata jsonb not null default '{}'::jsonb
);

create table if not exists perf.perf_run_results (
  perf_run_result_id uuid primary key,
  tenant_id uuid not null references iam.tenants(tenant_id),
  perf_run_id uuid not null references perf.perf_runs(perf_run_id) on delete cascade,
  metric_name text not null,
  metric_value numeric(14,4) not null,
  unit text not null,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists idx_memberships_tenant_user on iam.memberships (tenant_id, user_id);
create index if not exists idx_providers_tenant_kind on control.providers (tenant_id, kind);
create index if not exists idx_provider_models_tenant_capability on control.provider_models (tenant_id, capability, enabled);
create index if not exists idx_provider_health_tenant_provider_checked on control.provider_health (tenant_id, provider_id, checked_at desc);
create index if not exists idx_agents_tenant_status on control.agents (tenant_id, status, modality);
create index if not exists idx_agent_versions_tenant_agent on control.agent_versions (tenant_id, agent_id, version_number desc);
create index if not exists idx_conversations_tenant_user_updated on conversation.conversations (tenant_id, user_id, updated_at desc);
create index if not exists idx_messages_tenant_conversation_created on conversation.messages (tenant_id, conversation_id, created_at);
create index if not exists idx_message_parts_tenant_message_sequence on conversation.message_parts (tenant_id, message_id, sequence_no);
create index if not exists idx_runs_tenant_conversation_started on conversation.runs (tenant_id, conversation_id, started_at desc);
create index if not exists idx_run_steps_tenant_run_sequence on conversation.run_steps (tenant_id, run_id, sequence_no);
create index if not exists idx_voice_sessions_tenant_conversation on voice.voice_sessions (tenant_id, conversation_id, started_at desc);
create index if not exists idx_transcripts_tenant_session_sequence on voice.transcripts (tenant_id, voice_session_id, sequence_no);
create index if not exists idx_documents_tenant_agent on conversation.documents (tenant_id, agent_id, created_at desc);
create index if not exists idx_chunks_tenant_document on conversation.document_chunks (tenant_id, document_id, sequence_no);
create index if not exists idx_usage_events_tenant_created on audit.usage_events (tenant_id, created_at desc);
create index if not exists idx_audit_events_tenant_created on audit.audit_events (tenant_id, created_at desc);
create index if not exists idx_perf_runs_tenant_started on perf.perf_runs (tenant_id, started_at desc);
create index if not exists idx_perf_results_tenant_run_metric on perf.perf_run_results (tenant_id, perf_run_id, metric_name);

