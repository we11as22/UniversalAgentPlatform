insert into iam.tenants (tenant_id, slug, name, status, metadata)
values
  ('11111111-1111-1111-1111-111111111111', 'acme', 'Acme Corp', 'active', '{"region":"us"}'),
  ('22222222-2222-2222-2222-222222222222', 'globex', 'Globex', 'active', '{"region":"eu"}')
on conflict do nothing;

insert into iam.users (user_id, external_subject, email, display_name, status)
values
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', 'kc-admin', 'admin@acme.test', 'Acme Admin', 'active'),
  ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2', 'kc-analyst', 'analyst@acme.test', 'Acme Analyst', 'active'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1', 'kc-owner', 'owner@globex.test', 'Globex Owner', 'active'),
  ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2', 'kc-viewer', 'viewer@globex.test', 'Globex Viewer', 'active')
on conflict do nothing;

insert into iam.roles (role_id, tenant_id, name, description, permissions)
values
  ('31000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'tenant-admin', 'Acme tenant admin', '["agents:*","providers:*","perf:*"]'),
  ('31000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'analyst', 'Acme analyst', '["chat:read","chat:write"]'),
  ('32000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'tenant-owner', 'Globex owner', '["agents:*","providers:*","chat:*"]'),
  ('32000000-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'viewer', 'Globex viewer', '["chat:read"]')
on conflict do nothing;

insert into iam.memberships (membership_id, tenant_id, user_id, role_id, group_name, status)
values
  ('41000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', '31000000-0000-0000-0000-000000000001', 'platform-admins', 'active'),
  ('41000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2', '31000000-0000-0000-0000-000000000002', 'analysts', 'active'),
  ('42000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1', '32000000-0000-0000-0000-000000000001', 'owners', 'active'),
  ('42000000-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb2', '32000000-0000-0000-0000-000000000002', 'viewers', 'active')
on conflict do nothing;

insert into control.providers (provider_id, tenant_id, name, kind, endpoint, enabled, metadata)
values
  ('51000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'acme-demo-provider', 'demo', 'http://provider-gateway:8080', true, '{"capabilities":["llm","asr","tts"]}'),
  ('51000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'acme-triton', 'triton', 'http://triton:8000', false, '{"capabilities":["llm","asr","tts"]}'),
  ('52000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'globex-demo-provider', 'demo', 'http://provider-gateway:8080', true, '{"capabilities":["llm"]}')
on conflict do nothing;

insert into control.provider_models (provider_model_id, tenant_id, provider_id, capability, model_slug, display_name, streaming, config, enabled)
values
  ('61000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '51000000-0000-0000-0000-000000000001', 'llm', 'demo-llm', 'Demo LLM', true, '{"max_tokens":1024}', true),
  ('61000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '51000000-0000-0000-0000-000000000001', 'asr', 'demo-asr', 'Demo ASR', true, '{"language":"en"}', true),
  ('61000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '51000000-0000-0000-0000-000000000001', 'tts', 'demo-tts', 'Demo TTS', true, '{"voice":"alloy"}', true),
  ('62000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', '52000000-0000-0000-0000-000000000001', 'llm', 'demo-llm', 'Demo LLM', true, '{"max_tokens":1024}', true)
on conflict do nothing;

insert into control.agents (agent_id, tenant_id, slug, display_name, description, modality, status, created_by)
values
  ('71000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'platform-assistant', 'Platform Assistant', 'General enterprise assistant', 'text', 'active', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('71000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'voice-analyst', 'Voice Analyst', 'Voice-enabled analyst', 'voice', 'active', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('71000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'realtime-coach', 'Realtime Coach', 'Realtime voice coaching agent', 'realtime_voice', 'active', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1'),
  ('72000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', 'globex-assistant', 'Globex Assistant', 'Tenant assistant', 'text', 'active', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1')
on conflict do nothing;

insert into control.agent_versions (agent_version_id, tenant_id, agent_id, version_number, system_prompt, prompt_template, config, policies, signed_by)
values
  ('81000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '71000000-0000-0000-0000-000000000001', 1, 'You are a concise enterprise assistant.', 'Answer as the selected agent.', '{"temperature":0.2}', '{"retention_days":30}', 'bootstrap'),
  ('81000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '71000000-0000-0000-0000-000000000002', 1, 'You are a voice analyst.', 'Answer in a voice-friendly style.', '{"temperature":0.3}', '{"voice_profile":"analyst"}', 'bootstrap'),
  ('81000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '71000000-0000-0000-0000-000000000003', 1, 'You are a realtime coach.', 'Respond with short spoken turns.', '{"temperature":0.4}', '{"voice_profile":"coach"}', 'bootstrap'),
  ('82000000-0000-0000-0000-000000000001', '22222222-2222-2222-2222-222222222222', '72000000-0000-0000-0000-000000000001', 1, 'You are a tenant-specific assistant.', 'Answer according to tenant policy.', '{"temperature":0.2}', '{"retention_days":30}', 'bootstrap')
on conflict do nothing;

update control.agents
set current_version_id = case agent_id
  when '71000000-0000-0000-0000-000000000001' then '81000000-0000-0000-0000-000000000001'
  when '71000000-0000-0000-0000-000000000002' then '81000000-0000-0000-0000-000000000002'
  when '71000000-0000-0000-0000-000000000003' then '81000000-0000-0000-0000-000000000003'
  when '72000000-0000-0000-0000-000000000001' then '82000000-0000-0000-0000-000000000001'
  else current_version_id
end;

insert into control.agent_model_bindings (agent_model_binding_id, tenant_id, agent_version_id, provider_model_id, priority, capability)
values
  ('91000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000001', '61000000-0000-0000-0000-000000000001', 10, 'llm'),
  ('91000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000002', '61000000-0000-0000-0000-000000000001', 10, 'llm'),
  ('91000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000002', '61000000-0000-0000-0000-000000000002', 10, 'asr'),
  ('91000000-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000002', '61000000-0000-0000-0000-000000000003', 10, 'tts'),
  ('91000000-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000003', '61000000-0000-0000-0000-000000000001', 10, 'llm'),
  ('91000000-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000003', '61000000-0000-0000-0000-000000000002', 10, 'asr'),
  ('91000000-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', '81000000-0000-0000-0000-000000000003', '61000000-0000-0000-0000-000000000003', 10, 'tts')
on conflict do nothing;

insert into perf.perf_profiles (perf_profile_id, tenant_id, name, profile_type, config)
values
  ('a1000000-0000-0000-0000-000000000007', '11111111-1111-1111-1111-111111111111', 'validation-short', 'smoke', '{"vus":1,"duration":"5s","voice_concurrency":1}'),
  ('a1000000-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'smoke', 'smoke', '{"vus":2,"duration":"30s","voice_concurrency":2}'),
  ('a1000000-0000-0000-0000-000000000002', '11111111-1111-1111-1111-111111111111', 'load', 'load', '{"vus":20,"duration":"5m","voice_concurrency":10}'),
  ('a1000000-0000-0000-0000-000000000003', '11111111-1111-1111-1111-111111111111', 'stress', 'stress', '{"vus":100,"duration":"10m","voice_concurrency":25}'),
  ('a1000000-0000-0000-0000-000000000004', '11111111-1111-1111-1111-111111111111', 'spike', 'spike', '{"vus":200,"duration":"2m","voice_concurrency":40}'),
  ('a1000000-0000-0000-0000-000000000005', '11111111-1111-1111-1111-111111111111', 'soak', 'soak', '{"vus":30,"duration":"1h","voice_concurrency":12}'),
  ('a1000000-0000-0000-0000-000000000006', '11111111-1111-1111-1111-111111111111', 'failure-injection-lite', 'failure-injection-lite', '{"vus":10,"duration":"10m","voice_concurrency":6,"faults":["provider-timeout"]}')
on conflict do nothing;
