"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ActionCard, AppShell, Badge, Panel, StatusCard, TabBar } from "@uap/ui";

type Agent = {
  agent_id: string;
  slug: string;
  display_name: string;
  description: string;
  modality: string;
  status: string;
  current_version_id: string;
  rag_enabled: boolean;
};

type Provider = {
  provider_id: string;
  name: string;
  kind: string;
  endpoint: string;
  enabled: boolean;
  metadata?: Record<string, unknown>;
};

type ProviderModel = {
  provider_model_id: string;
  provider_id: string;
  provider_name: string;
  capability: string;
  model_slug: string;
  display_name: string;
  streaming: boolean;
  enabled: boolean;
  config: Record<string, unknown>;
};

type PerfProfile = {
  perf_profile_id: string;
  name: string;
  profile_type: string;
};

type PerfRun = {
  perf_run_id: string;
  status: string;
  target_environment: string;
  git_sha?: string;
  build_version?: string;
  started_at?: string;
};

type PerfResult = {
  metric_name: string;
  metric_value: number;
  unit: string;
  metadata?: Record<string, unknown>;
};

type DashboardStats = {
  agents: number;
  providers: number;
  provider_models: number;
  conversations: number;
};

type TabKey = "overview" | "agents" | "providers" | "models" | "knowledge" | "voice" | "perf" | "observability" | "security";

const adminBaseUrl = process.env.NEXT_PUBLIC_ADMIN_API_BASE_URL ?? "http://localhost:3210";
const grafanaBaseUrl = process.env.NEXT_PUBLIC_GRAFANA_URL ?? "http://localhost:13000";
const chatWebUrl = process.env.NEXT_PUBLIC_CHAT_WEB_URL ?? "http://localhost:3200";
const keycloakAdminUrl = process.env.NEXT_PUBLIC_KEYCLOAK_ADMIN_URL ?? "http://localhost:18081/admin";
const temporalUiUrl = process.env.NEXT_PUBLIC_TEMPORAL_UI_URL ?? "http://localhost:18088";
const prometheusUrl = process.env.NEXT_PUBLIC_PROMETHEUS_URL ?? "http://localhost:19090";
const minioConsoleUrl = process.env.NEXT_PUBLIC_MINIO_CONSOLE_URL ?? "http://localhost:19001";
const livekitTransportUrl = process.env.NEXT_PUBLIC_LIVEKIT_HTTP_URL ?? "http://localhost:17880";

const initialProviderForm = {
  name: "",
  kind: "demo",
  endpoint: "http://provider-gateway:8080",
  credential_ref_type: "env",
  credential_ref_locator: "OPENAI_API_KEY",
  metadata: '{ "capabilities": ["llm"] }'
};

const initialProviderModelForm = {
  provider_id: "",
  capability: "llm",
  model_slug: "",
  display_name: "",
  streaming: true,
  config: '{ "max_tokens": 1024 }'
};

const initialAgentForm = {
  slug: "",
  display_name: "",
  description: "",
  modality: "text",
  system_prompt: "You are a production enterprise agent.",
  prompt_template: "Answer using the selected provider and available knowledge.",
  provider_model_id: "",
  asr_provider_model_id: "",
  tts_provider_model_id: "",
  tools: "tenant_knowledge_search",
  rag_enabled: false,
  config: '{ "temperature": 0.2 }',
  policies: '{ "retention_days": 30, "rate_limit_per_minute": 60 }'
};

const initialKnowledgeForm = {
  agent_id: "",
  title: "Tenant handbook",
  content: "Add tenant-specific product, process, and policy knowledge here."
};

const tabs: Array<{ key: TabKey; label: string; hint: string; href: string }> = [
  { key: "overview", label: "Overview", hint: "Platform posture", href: "/" },
  { key: "agents", label: "Agents", hint: "Registry and rollout", href: "/agents" },
  { key: "providers", label: "Providers", hint: "Self-hosted and external", href: "/providers" },
  { key: "models", label: "Models", hint: "Capability catalog", href: "/models" },
  { key: "knowledge", label: "Knowledge", hint: "Qdrant and RAG", href: "/knowledge" },
  { key: "voice", label: "Voice", hint: "Realtime readiness", href: "/voice" },
  { key: "perf", label: "Perf", hint: "k6 and voice load", href: "/perf" },
  { key: "observability", label: "Observability", hint: "Grafana and ops", href: "/observability" },
  { key: "security", label: "Security", hint: "Isolation and control", href: "/security" }
];

function parseJSONField(value: string) {
  if (!value.trim()) {
    return {};
  }
  return JSON.parse(value) as Record<string, unknown>;
}

function Field({
  label,
  children,
  hint
}: {
  label: string;
  children: React.ReactNode;
  hint?: string;
}) {
  return (
    <label className="block space-y-2">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium text-slate-100">{label}</span>
        {hint ? <span className="text-xs text-slate-500">{hint}</span> : null}
      </div>
      {children}
    </label>
  );
}

function inputClassName() {
  return "w-full rounded-[1.15rem] border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-white outline-none transition duration-200 placeholder:text-slate-500 focus:border-cyan-300/40 focus:ring-2 focus:ring-cyan-300/15";
}

function textAreaClassName() {
  return `${inputClassName()} min-h-28 resize-y`;
}

function PrimaryButton({
  children,
  onClick,
  tone = "cyan"
}: {
  children: React.ReactNode;
  onClick: () => void;
  tone?: "cyan" | "emerald" | "amber";
}) {
  const toneClass =
    tone === "emerald"
      ? "bg-emerald-300 text-slate-950 hover:bg-emerald-200"
      : tone === "amber"
        ? "bg-amber-300 text-slate-950 hover:bg-amber-200"
        : "bg-cyan-300 text-slate-950 hover:bg-cyan-200";

  return (
    <button className={`rounded-full px-4 py-2 text-sm font-medium transition duration-200 ${toneClass}`} onClick={onClick} type="button">
      {children}
    </button>
  );
}

export function AdminDashboard({ initialTab = "overview" }: { initialTab?: TabKey }) {
  const router = useRouter();
  const [activeTab, setActiveTab] = useState<TabKey>(initialTab);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [providerModels, setProviderModels] = useState<ProviderModel[]>([]);
  const [dashboard, setDashboard] = useState<DashboardStats | null>(null);
  const [profiles, setProfiles] = useState<PerfProfile[]>([]);
  const [runs, setRuns] = useState<PerfRun[]>([]);
  const [selectedPerfRunId, setSelectedPerfRunId] = useState<string | null>(null);
  const [perfResults, setPerfResults] = useState<PerfResult[]>([]);
  const [providerForm, setProviderForm] = useState(initialProviderForm);
  const [providerModelForm, setProviderModelForm] = useState(initialProviderModelForm);
  const [agentForm, setAgentForm] = useState(initialAgentForm);
  const [knowledgeForm, setKnowledgeForm] = useState(initialKnowledgeForm);
  const [editingProviderId, setEditingProviderId] = useState<string | null>(null);
  const [editingProviderModelId, setEditingProviderModelId] = useState<string | null>(null);
  const [editingAgentId, setEditingAgentId] = useState<string | null>(null);
  const [activity, setActivity] = useState("Control plane synced. Ready for agent rollout.");

  const voiceEnabledAgents = useMemo(() => agents.filter((agent) => agent.modality !== "text"), [agents]);
  const ragAgents = useMemo(() => agents.filter((agent) => agent.rag_enabled), [agents]);
  const llmModels = useMemo(() => providerModels.filter((model) => model.capability === "llm"), [providerModels]);
  const asrModels = useMemo(() => providerModels.filter((model) => model.capability === "asr"), [providerModels]);
  const ttsModels = useMemo(() => providerModels.filter((model) => model.capability === "tts"), [providerModels]);

  useEffect(() => {
    setActiveTab(initialTab);
  }, [initialTab]);

  async function refreshData() {
    const [agentsResponse, providersResponse, providerModelsResponse, dashboardResponse, profilesResponse, runsResponse] = await Promise.all([
      fetch(`${adminBaseUrl}/api/v1/agents`).then((response) => response.json()),
      fetch(`${adminBaseUrl}/api/v1/providers`).then((response) => response.json()),
      fetch(`${adminBaseUrl}/api/v1/provider-models`).then((response) => response.json()),
      fetch(`${adminBaseUrl}/api/v1/dashboard`).then((response) => response.json()),
      fetch(`${adminBaseUrl}/api/v1/perf/profiles`).then((response) => response.json()),
      fetch(`${adminBaseUrl}/api/v1/perf/runs`).then((response) => response.json())
    ]);

    setAgents(agentsResponse);
    setProviders(providersResponse);
    setProviderModels(providerModelsResponse);
    setDashboard(dashboardResponse);
    setProfiles(profilesResponse);
    setRuns(runsResponse);
    setSelectedPerfRunId((current) => current || runsResponse[0]?.perf_run_id || null);

    setProviderModelForm((current) => ({
      ...current,
      provider_id: current.provider_id || providersResponse[0]?.provider_id || ""
    }));
    setAgentForm((current) => ({
      ...current,
      provider_model_id: current.provider_model_id || providerModelsResponse.find((item: ProviderModel) => item.capability === "llm")?.provider_model_id || "",
      asr_provider_model_id: current.asr_provider_model_id || providerModelsResponse.find((item: ProviderModel) => item.capability === "asr")?.provider_model_id || "",
      tts_provider_model_id: current.tts_provider_model_id || providerModelsResponse.find((item: ProviderModel) => item.capability === "tts")?.provider_model_id || ""
    }));
    setKnowledgeForm((current) => ({
      ...current,
      agent_id: current.agent_id || agentsResponse[0]?.agent_id || ""
    }));
  }

  async function loadPerfResults(perfRunId: string) {
    const results = await fetch(`${adminBaseUrl}/api/v1/perf/runs/${perfRunId}/results`).then((response) => response.json());
    setPerfResults(results);
    setSelectedPerfRunId(perfRunId);
  }

  useEffect(() => {
    void refreshData();
  }, []);

  useEffect(() => {
    if (!selectedPerfRunId) {
      setPerfResults([]);
      return;
    }
    void loadPerfResults(selectedPerfRunId);
  }, [selectedPerfRunId]);

  async function createProvider() {
    try {
      await fetch(`${adminBaseUrl}/api/v1/providers`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...providerForm,
          metadata: parseJSONField(providerForm.metadata)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "provider create failed");
        }
      });
      setActivity(`Provider ${providerForm.name} registered.`);
      setProviderForm(initialProviderForm);
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Provider create failed");
    }
  }

  async function updateProvider() {
    if (!editingProviderId) {
      return;
    }
    try {
      await fetch(`${adminBaseUrl}/api/v1/providers/${editingProviderId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...providerForm,
          metadata: parseJSONField(providerForm.metadata)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "provider update failed");
        }
      });
      setActivity(`Provider ${providerForm.name} updated.`);
      setEditingProviderId(null);
      setProviderForm(initialProviderForm);
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Provider update failed");
    }
  }

  async function createProviderModel() {
    try {
      await fetch(`${adminBaseUrl}/api/v1/provider-models`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...providerModelForm,
          config: parseJSONField(providerModelForm.config)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "provider model create failed");
        }
      });
      setActivity(`Provider model ${providerModelForm.display_name} registered.`);
      setProviderModelForm((current) => ({ ...initialProviderModelForm, provider_id: current.provider_id }));
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Provider model create failed");
    }
  }

  async function updateProviderModel() {
    if (!editingProviderModelId) {
      return;
    }
    try {
      await fetch(`${adminBaseUrl}/api/v1/provider-models/${editingProviderModelId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...providerModelForm,
          config: parseJSONField(providerModelForm.config)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "provider model update failed");
        }
      });
      setActivity(`Provider model ${providerModelForm.display_name} updated.`);
      setEditingProviderModelId(null);
      setProviderModelForm((current) => ({ ...initialProviderModelForm, provider_id: current.provider_id }));
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Provider model update failed");
    }
  }

  async function createAgent() {
    try {
      await fetch(`${adminBaseUrl}/api/v1/agents`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...agentForm,
          config: parseJSONField(agentForm.config),
          policies: parseJSONField(agentForm.policies),
          tools: agentForm.tools
            .split(",")
            .map((tool) => tool.trim())
            .filter(Boolean)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "agent create failed");
        }
      });
      setActivity(`Agent ${agentForm.display_name} created and bound to its current version.`);
      setAgentForm((current) => ({
        ...initialAgentForm,
        provider_model_id: current.provider_model_id,
        asr_provider_model_id: current.asr_provider_model_id,
        tts_provider_model_id: current.tts_provider_model_id
      }));
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Agent create failed");
    }
  }

  async function updateAgent() {
    if (!editingAgentId) {
      return;
    }
    try {
      await fetch(`${adminBaseUrl}/api/v1/agents/${editingAgentId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...agentForm,
          config: parseJSONField(agentForm.config),
          policies: parseJSONField(agentForm.policies),
          tools: agentForm.tools
            .split(",")
            .map((tool) => tool.trim())
            .filter(Boolean)
        })
      }).then(async (response) => {
        if (!response.ok) {
          throw new Error((await response.json()).error ?? "agent update failed");
        }
      });
      setActivity(`Agent ${agentForm.display_name} updated through a new immutable version.`);
      setEditingAgentId(null);
      setAgentForm((current) => ({
        ...initialAgentForm,
        provider_model_id: current.provider_model_id,
        asr_provider_model_id: current.asr_provider_model_id,
        tts_provider_model_id: current.tts_provider_model_id
      }));
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Agent update failed");
    }
  }

  async function loadAgentForEdit(agentId: string) {
    try {
      const response = await fetch(`${adminBaseUrl}/api/v1/agents/${agentId}`).then((value) => value.json());
      const llmBinding = Array.isArray(response.bindings)
        ? response.bindings.find((binding: { capability?: string }) => binding.capability === "llm")
        : null;
      setAgentForm({
        slug: response.slug ?? "",
        display_name: response.display_name ?? "",
        description: response.description ?? "",
        modality: response.modality ?? "text",
        system_prompt: response.system_prompt ?? "",
        prompt_template: response.prompt_template ?? "",
        provider_model_id: response.llm_provider_model_id ?? llmBinding?.provider_model_id ?? "",
        asr_provider_model_id: response.asr_provider_model_id ?? "",
        tts_provider_model_id: response.tts_provider_model_id ?? "",
        tools: Array.isArray(response.tools) ? response.tools.join(", ") : "",
        rag_enabled: Boolean(response.rag_enabled),
        config: JSON.stringify(response.config ?? {}, null, 2),
        policies: JSON.stringify(response.policies ?? {}, null, 2)
      });
      setEditingAgentId(agentId);
      setActivity(`Loaded agent ${response.display_name ?? agentId} into the rollout form.`);
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Unable to load agent details");
    }
  }

  function loadProviderForEdit(provider: Provider) {
    setProviderForm({
      name: provider.name,
      kind: provider.kind,
      endpoint: provider.endpoint,
      credential_ref_type: "env",
      credential_ref_locator: "OPENAI_API_KEY",
      metadata: JSON.stringify(provider.metadata ?? {}, null, 2)
    });
    setEditingProviderId(provider.provider_id);
    setActivity(`Loaded provider ${provider.name} into the registry form.`);
  }

  function loadProviderModelForEdit(model: ProviderModel) {
    setProviderModelForm({
      provider_id: model.provider_id,
      capability: model.capability,
      model_slug: model.model_slug,
      display_name: model.display_name,
      streaming: model.streaming,
      config: JSON.stringify(model.config ?? {}, null, 2)
    });
    setEditingProviderModelId(model.provider_model_id);
    setActivity(`Loaded provider model ${model.display_name} into the binding form.`);
  }

  function clearAgentEditor() {
    setEditingAgentId(null);
    setAgentForm((current) => ({
      ...initialAgentForm,
      provider_model_id: current.provider_model_id,
      asr_provider_model_id: current.asr_provider_model_id,
      tts_provider_model_id: current.tts_provider_model_id
    }));
  }

  function clearProviderEditor() {
    setEditingProviderId(null);
    setProviderForm(initialProviderForm);
  }

  function clearProviderModelEditor() {
    setEditingProviderModelId(null);
    setProviderModelForm((current) => ({ ...initialProviderModelForm, provider_id: current.provider_id }));
  }

  async function installExampleRAGAgent() {
    try {
      const response = await fetch(`${adminBaseUrl}/api/v1/agents/install/rag-example`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({})
      }).then(async (value) => {
        if (!value.ok) {
          throw new Error((await value.json()).error ?? "example install failed");
        }
        return value.json();
      });
      setActivity(`Example RAG agent installed: ${response.agent_id as string}.`);
      await refreshData();
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Example install failed");
    }
  }

  async function ingestKnowledge() {
    try {
      const response = await fetch(`${adminBaseUrl}/api/v1/knowledge/index`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(knowledgeForm)
      }).then(async (value) => {
        if (!value.ok) {
          throw new Error((await value.json()).error ?? "knowledge ingest failed");
        }
        return value.json();
      });
      setActivity(`Indexed ${String(response.indexed_chunks)} chunks for agent ${knowledgeForm.agent_id}.`);
    } catch (error) {
      setActivity(error instanceof Error ? error.message : "Knowledge ingest failed");
    }
  }

  async function launchPerfRun(profileId: string) {
    const response = await fetch(`${adminBaseUrl}/api/v1/perf/runs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ perf_profile_id: profileId, target_environment: "local-api" })
    }).then((value) => value.json());
    setRuns((current) => [{ perf_run_id: response.perf_run_id as string, status: "queued", target_environment: "local-api" }, ...current]);
    setSelectedPerfRunId(response.perf_run_id as string);
    setActivity(`Perf run queued for profile ${profileId}.`);
  }

  const monitoringLinks = [
    { title: "Platform Overview", href: `${grafanaBaseUrl}/d/uap-platform-overview/platform-overview`, description: "Traffic, service health, baseline platform throughput.", tone: "cyan" as const },
    { title: "Chat Pipeline", href: `${grafanaBaseUrl}/d/uap-chat-pipeline/chat-pipeline`, description: "SSE startup latency, run flow, chat service health.", tone: "emerald" as const },
    { title: "Voice Pipeline", href: `${grafanaBaseUrl}/d/uap-voice-pipeline/voice-pipeline`, description: "Voice setup, transcripts, interruption recovery path.", tone: "amber" as const },
    { title: "Provider Health", href: `${grafanaBaseUrl}/d/uap-provider-health/provider-health`, description: "Provider routing, error rate, fallback and health posture.", tone: "rose" as const },
    { title: "Triton Inference", href: `${grafanaBaseUrl}/d/uap-triton-inference/triton-inference`, description: "Self-hosted inference plane overview and model latency.", tone: "cyan" as const },
    { title: "Data Plane", href: `${grafanaBaseUrl}/d/uap-data-plane/data-plane`, description: "Kafka, Redis, Postgres, Qdrant, object storage.", tone: "emerald" as const },
    { title: "Agent Overview", href: `${grafanaBaseUrl}/d/uap-agent-overview/agent-overview`, description: "Per-agent latency, throughput, retrieval, voice mix.", tone: "amber" as const },
    { title: "Tenant Overview", href: `${grafanaBaseUrl}/d/uap-tenant-overview/tenant-overview`, description: "Tenant isolation, usage shape, quotas and engagement.", tone: "rose" as const },
    { title: "Load Testing Results", href: `${grafanaBaseUrl}/d/uap-load-testing-results/load-testing-results`, description: "Smoke, load, stress, spike and soak histories.", tone: "emerald" as const },
    { title: "Cost / Usage / Latency", href: `${grafanaBaseUrl}/d/uap-cost-usage-latency/cost-usage-latency`, description: "Spend posture, provider efficiency and latency mix.", tone: "cyan" as const }
  ];

  const operationsLinks = [
    { title: "Grafana Home", href: grafanaBaseUrl, description: "Dashboards, explore, traces and product analytics.", tone: "cyan" as const },
    { title: "Keycloak Admin", href: keycloakAdminUrl, description: "SSO, realm config, clients, roles and identity flows.", tone: "rose" as const },
    { title: "Temporal UI", href: temporalUiUrl, description: "Workflow state, retries, long-running orchestration.", tone: "amber" as const },
    { title: "Prometheus", href: prometheusUrl, description: "Raw metrics and rule evaluation entry point.", tone: "emerald" as const },
    { title: "MinIO Console", href: minioConsoleUrl, description: "Attachments, artifacts and object-storage checks.", tone: "cyan" as const },
    { title: "Chat UI", href: chatWebUrl, description: "Open the user workspace and validate live chat flows.", tone: "emerald" as const }
  ];

  const sidebar = (
    <div className="flex h-full flex-col gap-5 px-4 py-5">
      <div className="rounded-[1.8rem] border border-white/10 bg-white/[0.04] p-4">
        <p className="font-display text-[11px] uppercase tracking-[0.28em] text-cyan-200/80">UniversalAgentPlatform</p>
        <h1 className="mt-3 font-display text-2xl font-semibold text-white">Admin cockpit</h1>
        <p className="mt-3 text-sm leading-6 text-slate-300">Operate agents, providers, RAG, voice and observability from one tenant-scoped workspace.</p>
      </div>

      <div className="grid grid-cols-3 gap-2 lg:grid-cols-1">
        {tabs.map((tab) => (
          <Link
            key={tab.key}
            href={tab.href}
            className={`rounded-[1.25rem] border px-4 py-3 text-left transition duration-200 ${
              activeTab === tab.key
                ? "border-cyan-300/25 bg-cyan-300/12 text-white"
                : "border-white/10 bg-white/[0.03] text-slate-300 hover:border-white/20 hover:bg-white/[0.05]"
            }`}
          >
            <div className="text-sm font-medium">{tab.label}</div>
            <div className="mt-1 hidden text-xs text-slate-500 lg:block">{tab.hint}</div>
          </Link>
        ))}
      </div>

      <div className="mt-auto space-y-3">
        <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
          <p className="text-xs uppercase tracking-[0.24em] text-slate-400">Environment</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Badge tone="emerald">local-api</Badge>
            <Badge tone="cyan">tenant acme</Badge>
          </div>
          <p className="mt-3 text-sm text-slate-300">This cockpit is now environment-driven and can target localhost, kind, or a public cloud hostname without rewriting links.</p>
        </div>
        <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
          <p className="text-xs uppercase tracking-[0.24em] text-slate-400">Activity</p>
          <p className="mt-3 text-sm leading-6 text-slate-300">{activity}</p>
        </div>
      </div>
    </div>
  );

  const header = (
    <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <p className="text-[11px] uppercase tracking-[0.3em] text-cyan-200/80">Trust and authority control plane</p>
        <h2 className="font-display mt-2 text-3xl font-semibold tracking-tight text-white lg:text-4xl">Operate chat, voice, providers and monitoring as one system.</h2>
        <p className="mt-3 max-w-4xl text-sm leading-6 text-slate-300">
          This workspace is structured by real operator use-cases: rollout new agents, bind models, load tenant knowledge, verify voice readiness, inspect Grafana dashboards, then launch perf runs without switching tooling.
        </p>
      </div>
      <div className="flex flex-wrap gap-3">
        <a className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={chatWebUrl} target="_blank" rel="noreferrer">
          Open chat
        </a>
        <a className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={grafanaBaseUrl} target="_blank" rel="noreferrer">
          Open Grafana
        </a>
        <PrimaryButton onClick={() => void refreshData()}>Refresh workspace</PrimaryButton>
      </div>
    </div>
  );

  return (
    <AppShell sidebar={sidebar} header={header}>
      <div className="space-y-6">
        <section className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
          <Panel
            kicker="Mission control"
            title="Everything required for enterprise operations is surfaced by domain"
            description="Provider registry, model catalog, agent rollout, Qdrant knowledge management, voice readiness, load testing and observability are separated into operational zones so the admin path matches how platform teams actually work."
          >
            <div className="flex flex-wrap gap-2">
              <Badge tone="cyan">Agent registry</Badge>
              <Badge tone="emerald">Provider routing</Badge>
              <Badge tone="amber">Realtime voice</Badge>
              <Badge tone="rose">Security posture</Badge>
              <Badge tone="slate">Grafana launchers</Badge>
            </div>
          </Panel>
          <Panel kicker="Operator rhythm" title="Fast path" description="Launch common actions without leaving the home strip.">
            <div className="grid gap-3 sm:grid-cols-2">
              <ActionCard title="Install example RAG agent" description="Provision a grounded Qdrant-backed agent with seeded knowledge." onClick={() => void installExampleRAGAgent()} meta={<Badge tone="emerald">one click</Badge>} />
              <ActionCard title="Launch smoke perf" description="Queue the smoke profile and verify the platform baseline." onClick={() => void launchPerfRun(profiles.find((profile) => profile.profile_type === "smoke")?.perf_profile_id ?? profiles[0]?.perf_profile_id ?? "")} meta={<Badge tone="amber">perf</Badge>} />
            </div>
          </Panel>
        </section>

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <StatusCard title="Agents" value={String(dashboard?.agents ?? 0)} hint="Versioned agents available for tenant-scoped chats and voice sessions." />
          <StatusCard title="Providers" value={String(dashboard?.providers ?? 0)} hint="External, BYO and Triton-backed providers under one registry." />
          <StatusCard title="Models" value={String(dashboard?.provider_models ?? 0)} hint="LLM, ASR, TTS and embedding capability bindings." />
          <StatusCard title="Conversations" value={String(dashboard?.conversations ?? 0)} hint="Persisted chat history and execution traces currently stored." />
        </section>

        <Panel
          kicker="Workspace"
          title="Choose an operations domain"
          description="Tabs are organized around day-two platform work: onboarding, tuning, observing and validating."
        >
          <TabBar items={tabs} activeKey={activeTab} onChange={(key) => router.push(tabs.find((tab) => tab.key === key)?.href ?? "/")} />
        </Panel>

        {activeTab === "overview" ? (
          <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
            <Panel kicker="Current posture" title="Platform overview" description="A quick operational read across agent mix, provider shape, RAG coverage and voice readiness.">
              <div className="grid gap-4 md:grid-cols-2">
                <ActionCard title="RAG-enabled agents" description={`${ragAgents.length} agents can answer from tenant knowledge in Qdrant.`} meta={<Badge tone="emerald">{ragAgents.length}</Badge>} />
                <ActionCard title="Voice-capable agents" description={`${voiceEnabledAgents.length} agents expose voice or realtime voice modalities.`} meta={<Badge tone="amber">{voiceEnabledAgents.length}</Badge>} />
                <ActionCard title="LLM bindings ready" description={`${llmModels.length} LLM models are available for agent binding.`} meta={<Badge tone="cyan">{llmModels.length}</Badge>} />
                <ActionCard title="Recent perf runs" description={`${runs.length} recent perf runs are retained in the admin history.`} meta={<Badge tone="rose">{runs.length}</Badge>} />
              </div>
            </Panel>
            <Panel kicker="Jump points" title="Operations launchers" description="Open the main control surfaces immediately.">
              <div className="grid gap-3">
                {operationsLinks.map((item) => (
                  <ActionCard key={item.title} title={item.title} description={item.description} href={item.href} meta={<Badge tone={item.tone}>{item.title.split(" ")[0]}</Badge>} />
                ))}
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "agents" ? (
          <div className="grid gap-6 xl:grid-cols-[1.18fr_0.82fr]">
            <Panel kicker="Registry" title="Agent inventory" description="Versioned agents with immutable conversation binding and explicit modality.">
              <div className="grid gap-4 md:grid-cols-2">
                {agents.map((agent) => (
                  <div key={agent.agent_id} className="rounded-[1.75rem] border border-white/10 bg-white/[0.03] p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h3 className="font-display text-lg font-semibold text-white">{agent.display_name}</h3>
                        <p className="mt-1 text-sm text-slate-400">{agent.slug}</p>
                      </div>
                      <Badge tone={agent.rag_enabled ? "emerald" : "slate"}>{agent.rag_enabled ? "rag" : "plain"}</Badge>
                    </div>
                    <p className="mt-3 text-sm leading-6 text-slate-300">{agent.description}</p>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Badge tone="cyan">{agent.modality}</Badge>
                      <Badge tone={agent.status === "active" ? "emerald" : "rose"}>{agent.status}</Badge>
                    </div>
                    <div className="mt-4">
                      <button
                        className="rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10"
                        onClick={() => void loadAgentForEdit(agent.agent_id)}
                        type="button"
                      >
                        Edit agent
                      </button>
                    </div>
                    <p className="font-mono-ui mt-4 break-all text-xs text-slate-500">{agent.current_version_id}</p>
                  </div>
                ))}
              </div>
            </Panel>
            <Panel
              kicker="Rollout"
              title={editingAgentId ? "Edit agent" : "Create agent"}
              description="Bind the agent to an LLM model immediately and optionally enable Qdrant-backed retrieval."
            >
              <div className="space-y-4">
                <Field label="Slug">
                  <input className={inputClassName()} value={agentForm.slug} onChange={(event) => setAgentForm({ ...agentForm, slug: event.target.value })} placeholder="compliance-copilot" />
                </Field>
                <Field label="Display name">
                  <input className={inputClassName()} value={agentForm.display_name} onChange={(event) => setAgentForm({ ...agentForm, display_name: event.target.value })} placeholder="Compliance Copilot" />
                </Field>
                <Field label="Description">
                  <textarea className={textAreaClassName()} value={agentForm.description} onChange={(event) => setAgentForm({ ...agentForm, description: event.target.value })} placeholder="What this agent is for, who uses it, and what it can access." />
                </Field>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field label="Modality">
                    <select className={inputClassName()} value={agentForm.modality} onChange={(event) => setAgentForm({ ...agentForm, modality: event.target.value })}>
                      <option value="text">text</option>
                      <option value="voice">voice</option>
                      <option value="realtime_voice">realtime_voice</option>
                    </select>
                  </Field>
                  <Field label="LLM binding">
                    <select className={inputClassName()} value={agentForm.provider_model_id} onChange={(event) => setAgentForm({ ...agentForm, provider_model_id: event.target.value })}>
                      <option value="">Select LLM model</option>
                      {llmModels.map((model) => (
                        <option key={model.provider_model_id} value={model.provider_model_id}>
                          {model.provider_name} / {model.display_name}
                        </option>
                      ))}
                    </select>
                  </Field>
                </div>
                {agentForm.modality !== "text" ? (
                  <div className="grid gap-4 md:grid-cols-2">
                    <Field label="ASR binding">
                      <select className={inputClassName()} value={agentForm.asr_provider_model_id} onChange={(event) => setAgentForm({ ...agentForm, asr_provider_model_id: event.target.value })}>
                        <option value="">Select ASR model</option>
                        {asrModels.map((model) => (
                          <option key={model.provider_model_id} value={model.provider_model_id}>
                            {model.provider_name} / {model.display_name}
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Field label="TTS binding">
                      <select className={inputClassName()} value={agentForm.tts_provider_model_id} onChange={(event) => setAgentForm({ ...agentForm, tts_provider_model_id: event.target.value })}>
                        <option value="">Select TTS model</option>
                        {ttsModels.map((model) => (
                          <option key={model.provider_model_id} value={model.provider_model_id}>
                            {model.provider_name} / {model.display_name}
                          </option>
                        ))}
                      </select>
                    </Field>
                  </div>
                ) : null}
                <Field label="System prompt">
                  <textarea className={textAreaClassName()} value={agentForm.system_prompt} onChange={(event) => setAgentForm({ ...agentForm, system_prompt: event.target.value })} />
                </Field>
                <Field label="Prompt template">
                  <textarea className={textAreaClassName()} value={agentForm.prompt_template} onChange={(event) => setAgentForm({ ...agentForm, prompt_template: event.target.value })} />
                </Field>
                <Field label="Tools" hint="comma separated">
                  <input className={inputClassName()} value={agentForm.tools} onChange={(event) => setAgentForm({ ...agentForm, tools: event.target.value })} placeholder="tenant_knowledge_search,web_search,crm_lookup" />
                </Field>
                <Field label="Runtime config" hint="JSON">
                  <textarea className={textAreaClassName()} value={agentForm.config} onChange={(event) => setAgentForm({ ...agentForm, config: event.target.value })} />
                </Field>
                <Field label="Policies" hint="JSON">
                  <textarea className={textAreaClassName()} value={agentForm.policies} onChange={(event) => setAgentForm({ ...agentForm, policies: event.target.value })} />
                </Field>
                <label className="flex items-center gap-3 rounded-[1.15rem] border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white">
                  <input type="checkbox" checked={agentForm.rag_enabled} onChange={(event) => setAgentForm({ ...agentForm, rag_enabled: event.target.checked })} />
                  Enable Qdrant-backed RAG retrieval for this agent
                </label>
                <div className="flex flex-wrap gap-3">
                  <PrimaryButton onClick={() => void (editingAgentId ? updateAgent() : createAgent())}>
                    {editingAgentId ? "Publish new version" : "Create agent"}
                  </PrimaryButton>
                  {editingAgentId ? (
                    <PrimaryButton tone="amber" onClick={clearAgentEditor}>
                      Cancel edit
                    </PrimaryButton>
                  ) : null}
                  <PrimaryButton tone="emerald" onClick={() => void installExampleRAGAgent()}>
                    Install example RAG agent
                  </PrimaryButton>
                </div>
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "providers" ? (
          <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
            <Panel kicker="Provider registry" title="Routing endpoints" description="Manage demo, Triton, openai-compatible and BYO providers through one contract.">
              <div className="space-y-3">
                {providers.map((provider) => (
                  <div key={provider.provider_id} className="flex flex-col gap-4 rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4 md:flex-row md:items-center md:justify-between">
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="font-display text-lg font-semibold text-white">{provider.name}</h3>
                        <Badge tone={provider.enabled ? "emerald" : "rose"}>{provider.enabled ? "enabled" : "disabled"}</Badge>
                        <Badge tone="cyan">{provider.kind}</Badge>
                      </div>
                      <p className="font-mono-ui mt-3 text-xs text-slate-500">{provider.endpoint}</p>
                    </div>
                    <div className="flex items-center gap-3">
                      <div className="text-sm text-slate-400">Unified provider gateway contract</div>
                      <button
                        className="rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10"
                        onClick={() => loadProviderForEdit(provider)}
                        type="button"
                      >
                        Edit
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </Panel>
            <Panel kicker="Create provider" title={editingProviderId ? "Edit provider" : "Register provider"} description="Keep credentials indirect through refs, never in database plaintext.">
              <div className="space-y-4">
                <Field label="Name">
                  <input className={inputClassName()} value={providerForm.name} onChange={(event) => setProviderForm({ ...providerForm, name: event.target.value })} placeholder="acme-openai-compatible" />
                </Field>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field label="Kind">
                    <select className={inputClassName()} value={providerForm.kind} onChange={(event) => setProviderForm({ ...providerForm, kind: event.target.value })}>
                      <option value="demo">demo</option>
                      <option value="triton">triton</option>
                      <option value="openai-compatible">openai-compatible</option>
                      <option value="byo">byo</option>
                    </select>
                  </Field>
                  <Field label="Endpoint">
                    <input className={inputClassName()} value={providerForm.endpoint} onChange={(event) => setProviderForm({ ...providerForm, endpoint: event.target.value })} />
                  </Field>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field label="Credential ref type">
                    <select className={inputClassName()} value={providerForm.credential_ref_type} onChange={(event) => setProviderForm({ ...providerForm, credential_ref_type: event.target.value })}>
                      <option value="env">env</option>
                      <option value="file">file</option>
                      <option value="k8s-secret">k8s-secret</option>
                      <option value="vault">vault</option>
                    </select>
                  </Field>
                  <Field label="Credential locator">
                    <input className={inputClassName()} value={providerForm.credential_ref_locator} onChange={(event) => setProviderForm({ ...providerForm, credential_ref_locator: event.target.value })} />
                  </Field>
                </div>
                <Field label="Metadata" hint="JSON">
                  <textarea className={textAreaClassName()} value={providerForm.metadata} onChange={(event) => setProviderForm({ ...providerForm, metadata: event.target.value })} />
                </Field>
                <div className="flex flex-wrap gap-3">
                  <PrimaryButton tone="emerald" onClick={() => void (editingProviderId ? updateProvider() : createProvider())}>
                    {editingProviderId ? "Update provider" : "Register provider"}
                  </PrimaryButton>
                  {editingProviderId ? (
                    <PrimaryButton tone="amber" onClick={clearProviderEditor}>
                      Cancel edit
                    </PrimaryButton>
                  ) : null}
                </div>
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "models" ? (
          <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
            <Panel kicker="Model catalog" title="Provider models" description="A single list for LLM, ASR, TTS and embeddings across self-hosted and external providers.">
              <div className="grid gap-4 md:grid-cols-2">
                {providerModels.map((model) => (
                  <div key={model.provider_model_id} className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <h3 className="font-display text-lg font-semibold text-white">{model.display_name}</h3>
                        <p className="mt-1 text-sm text-slate-400">{model.provider_name}</p>
                      </div>
                      <Badge tone={model.capability === "llm" ? "cyan" : model.capability === "tts" ? "amber" : model.capability === "asr" ? "emerald" : "rose"}>
                        {model.capability}
                      </Badge>
                    </div>
                    <p className="font-mono-ui mt-4 text-xs text-slate-500">{model.model_slug}</p>
                    <div className="mt-4 flex flex-wrap gap-2">
                      <Badge tone={model.streaming ? "emerald" : "slate"}>{model.streaming ? "streaming" : "batch"}</Badge>
                      <Badge tone={model.enabled ? "emerald" : "rose"}>{model.enabled ? "enabled" : "disabled"}</Badge>
                    </div>
                    <div className="mt-4">
                      <button
                        className="rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10"
                        onClick={() => loadProviderModelForEdit(model)}
                        type="button"
                      >
                        Edit model
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </Panel>
            <Panel kicker="Binding surface" title={editingProviderModelId ? "Edit provider model" : "Register provider model"} description="Use this to expose concrete capabilities without leaking vendor internals into agent logic.">
              <div className="space-y-4">
                <Field label="Provider">
                  <select className={inputClassName()} value={providerModelForm.provider_id} onChange={(event) => setProviderModelForm({ ...providerModelForm, provider_id: event.target.value })}>
                    <option value="">Select provider</option>
                    {providers.map((provider) => (
                      <option key={provider.provider_id} value={provider.provider_id}>
                        {provider.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <div className="grid gap-4 md:grid-cols-2">
                  <Field label="Capability">
                    <select className={inputClassName()} value={providerModelForm.capability} onChange={(event) => setProviderModelForm({ ...providerModelForm, capability: event.target.value })}>
                      <option value="llm">llm</option>
                      <option value="asr">asr</option>
                      <option value="tts">tts</option>
                      <option value="embedding">embedding</option>
                    </select>
                  </Field>
                  <Field label="Streaming">
                    <label className="flex items-center gap-3 rounded-[1.15rem] border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white">
                      <input type="checkbox" checked={providerModelForm.streaming} onChange={(event) => setProviderModelForm({ ...providerModelForm, streaming: event.target.checked })} />
                      Streaming enabled
                    </label>
                  </Field>
                </div>
                <Field label="Model slug">
                  <input className={inputClassName()} value={providerModelForm.model_slug} onChange={(event) => setProviderModelForm({ ...providerModelForm, model_slug: event.target.value })} placeholder="llama3-70b-instruct" />
                </Field>
                <Field label="Display name">
                  <input className={inputClassName()} value={providerModelForm.display_name} onChange={(event) => setProviderModelForm({ ...providerModelForm, display_name: event.target.value })} placeholder="Llama 3 70B Instruct" />
                </Field>
                <Field label="Config" hint="JSON">
                  <textarea className={textAreaClassName()} value={providerModelForm.config} onChange={(event) => setProviderModelForm({ ...providerModelForm, config: event.target.value })} />
                </Field>
                <div className="flex flex-wrap gap-3">
                  <PrimaryButton onClick={() => void (editingProviderModelId ? updateProviderModel() : createProviderModel())}>
                    {editingProviderModelId ? "Update model" : "Register model"}
                  </PrimaryButton>
                  {editingProviderModelId ? (
                    <PrimaryButton tone="amber" onClick={clearProviderModelEditor}>
                      Cancel edit
                    </PrimaryButton>
                  ) : null}
                </div>
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "knowledge" ? (
          <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <Panel kicker="RAG workspace" title="Knowledge indexing" description="Chunk and index tenant or agent-specific documents into Qdrant through the platform indexer.">
              <div className="space-y-4">
                <Field label="Target agent">
                  <select className={inputClassName()} value={knowledgeForm.agent_id} onChange={(event) => setKnowledgeForm({ ...knowledgeForm, agent_id: event.target.value })}>
                    <option value="">Select agent</option>
                    {agents.map((agent) => (
                      <option key={agent.agent_id} value={agent.agent_id}>
                        {agent.display_name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="Document title">
                  <input className={inputClassName()} value={knowledgeForm.title} onChange={(event) => setKnowledgeForm({ ...knowledgeForm, title: event.target.value })} />
                </Field>
                <Field label="Knowledge content">
                  <textarea className={`${textAreaClassName()} min-h-[18rem]`} value={knowledgeForm.content} onChange={(event) => setKnowledgeForm({ ...knowledgeForm, content: event.target.value })} />
                </Field>
                <PrimaryButton tone="amber" onClick={() => void ingestKnowledge()}>
                  Index knowledge
                </PrimaryButton>
              </div>
            </Panel>
            <Panel kicker="Use cases" title="What operators do here" description="The knowledge zone is designed for the common enterprise RAG loop.">
              <div className="grid gap-3">
                <ActionCard title="Seed a tenant handbook" description="Index operating procedures, domain lexicons and policy rules for a specific agent." meta={<Badge tone="emerald">rag</Badge>} />
                <ActionCard title="Install demo RAG agent" description="Provision a working example if the tenant needs a known-good baseline." onClick={() => void installExampleRAGAgent()} meta={<Badge tone="amber">guided</Badge>} />
                <ActionCard title="Validate retrieval in chat" description="Open Chat UI, create a new conversation with the agent, and ask grounded questions from the indexed text." href={chatWebUrl} meta={<Badge tone="cyan">chat</Badge>} />
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "voice" ? (
          <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <Panel kicker="Voice posture" title="Voice-ready agents" description="Use this zone to validate which agents expose voice or realtime voice experiences.">
              <div className="grid gap-4">
                {voiceEnabledAgents.map((agent) => (
                  <div key={agent.agent_id} className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <h3 className="font-display text-lg font-semibold text-white">{agent.display_name}</h3>
                        <p className="mt-1 text-sm text-slate-400">{agent.description}</p>
                      </div>
                      <Badge tone={agent.modality === "realtime_voice" ? "amber" : "emerald"}>{agent.modality}</Badge>
                    </div>
                    <p className="font-mono-ui mt-4 text-xs text-slate-500">{agent.current_version_id}</p>
                  </div>
                ))}
                {voiceEnabledAgents.length === 0 ? <p className="text-sm text-slate-400">No voice-capable agents are currently configured.</p> : null}
              </div>
            </Panel>
            <Panel kicker="Monitoring" title="Voice operations" description="Jump straight into the systems operators use when tuning voice experience.">
              <div className="grid gap-3">
                <ActionCard title="Voice Pipeline dashboard" href={`${grafanaBaseUrl}/d/uap-voice-pipeline/voice-pipeline`} description="Session setup, transcript latency, first-audio-byte timing." meta={<Badge tone="amber">grafana</Badge>} />
                <ActionCard title="Provider Health dashboard" href={`${grafanaBaseUrl}/d/uap-provider-health/provider-health`} description="ASR/TTS provider health, degradation and fallback chain." meta={<Badge tone="rose">grafana</Badge>} />
                <ActionCard title="LiveKit transport" href={livekitTransportUrl} description="Validate signalling endpoint availability for WebRTC sessions." meta={<Badge tone="cyan">livekit</Badge>} />
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "perf" ? (
          <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <Panel kicker="Perf launchpad" title="Test profiles" description="Run smoke, load, stress, spike and soak flows directly from the admin workspace.">
              <div className="space-y-3">
                {profiles.map((profile) => (
                  <div key={profile.perf_profile_id} className="flex items-center justify-between gap-4 rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
                    <div>
                      <h3 className="font-display text-lg font-semibold text-white">{profile.name}</h3>
                      <p className="mt-1 text-sm text-slate-400">{profile.profile_type}</p>
                    </div>
                    <PrimaryButton tone="emerald" onClick={() => void launchPerfRun(profile.perf_profile_id)}>
                      Run
                    </PrimaryButton>
                  </div>
                ))}
              </div>
            </Panel>
            <Panel kicker="Run history" title="Recent perf runs" description="Queue state and environment binding are visible here before you drill into Grafana.">
              <div className="space-y-3">
                {runs.map((run) => (
                  <button
                    key={run.perf_run_id}
                    className={`w-full rounded-[1.5rem] border bg-white/[0.03] p-4 text-left transition ${
                      selectedPerfRunId === run.perf_run_id ? "border-cyan-300/30 bg-cyan-300/10" : "border-white/10 hover:border-white/20"
                    }`}
                    onClick={() => setSelectedPerfRunId(run.perf_run_id)}
                    type="button"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <p className="font-mono-ui break-all text-xs text-slate-400">{run.perf_run_id}</p>
                      <Badge tone={run.status === "queued" ? "amber" : "emerald"}>{run.status}</Badge>
                    </div>
                    <p className="mt-3 text-sm text-slate-300">{run.target_environment}</p>
                    {run.started_at ? <p className="mt-2 text-xs text-slate-500">{new Date(run.started_at).toLocaleString()}</p> : null}
                  </button>
                ))}
                <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.03] p-4">
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="font-display text-lg font-semibold text-white">Selected run metrics</h3>
                    {selectedPerfRunId ? <Badge tone="cyan">live results</Badge> : null}
                  </div>
                  <div className="mt-4 space-y-2">
                    {perfResults.length === 0 ? <p className="text-sm text-slate-400">Select a run to inspect persisted metrics from k6 and voice suites.</p> : null}
                    {perfResults.map((result) => (
                      <div key={result.metric_name} className="flex items-center justify-between gap-3 rounded-[1rem] border border-white/10 px-3 py-2">
                        <div>
                          <p className="text-sm text-slate-100">{result.metric_name}</p>
                          {result.metadata ? <p className="text-xs text-slate-500">{JSON.stringify(result.metadata)}</p> : null}
                        </div>
                        <p className="font-mono-ui text-sm text-cyan-200">
                          {result.metric_value.toFixed(2)} {result.unit}
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="pt-2">
                  <a className="inline-flex rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={`${grafanaBaseUrl}/d/uap-load-testing-results/load-testing-results`} target="_blank" rel="noreferrer">
                    Open load-testing dashboard
                  </a>
                </div>
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "observability" ? (
          <div className="space-y-6">
            <Panel kicker="Monitoring launchers" title="Open Grafana by domain" description="Every major domain in the platform now has a dedicated entry point. Operators should not hunt for dashboards.">
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {monitoringLinks.map((link) => (
                  <ActionCard key={link.title} title={link.title} href={link.href} description={link.description} meta={<Badge tone={link.tone}>grafana</Badge>} />
                ))}
              </div>
            </Panel>
            <Panel kicker="Operations surfaces" title="Supporting control systems" description="Identity, workflow, object storage and raw metrics are one click away.">
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {operationsLinks.map((item) => (
                  <ActionCard key={item.title} title={item.title} href={item.href} description={item.description} meta={<Badge tone={item.tone}>ops</Badge>} />
                ))}
              </div>
            </Panel>
          </div>
        ) : null}

        {activeTab === "security" ? (
          <div className="grid gap-6 xl:grid-cols-[1fr_1fr]">
            <Panel kicker="Security baseline" title="Enterprise controls" description="This view is designed to keep the operator focused on the baseline guarantees already wired into the platform.">
              <div className="grid gap-3">
                <ActionCard title="Tenant isolation" description="Tenant-first schema, scoped IDs and per-tenant control plane actions." meta={<Badge tone="emerald">active</Badge>} />
                <ActionCard title="Secret indirection" description="Provider credentials stay behind `CredentialRef` and never land as plaintext in DB rows." meta={<Badge tone="emerald">active</Badge>} />
                <ActionCard title="Immutable versioning" description="Agents execute using `current_version_id`; chat history stays pinned to the chosen agent." meta={<Badge tone="cyan">invariant</Badge>} />
                <ActionCard title="Observability discipline" description="Metrics, traces, logs and perf telemetry are exposed through dedicated dashboards and operational links." meta={<Badge tone="amber">visible</Badge>} />
              </div>
            </Panel>
            <Panel kicker="Next operator actions" title="Security operations" description="Use the linked control systems to drill into auth, workflow and platform state.">
              <div className="grid gap-3">
                <ActionCard title="Open Keycloak admin" href={keycloakAdminUrl} description="Inspect roles, clients, groups and login flows." meta={<Badge tone="rose">identity</Badge>} />
                <ActionCard title="Open Platform dashboard" href={`${grafanaBaseUrl}/d/uap-platform-overview/platform-overview`} description="Verify service health, traffic and error budget posture." meta={<Badge tone="cyan">grafana</Badge>} />
                <ActionCard title="Open Data Plane dashboard" href={`${grafanaBaseUrl}/d/uap-data-plane/data-plane`} description="Watch the stateful dependencies behind tenant isolation and durability." meta={<Badge tone="amber">data</Badge>} />
              </div>
            </Panel>
          </div>
        ) : null}
      </div>
    </AppShell>
  );
}
