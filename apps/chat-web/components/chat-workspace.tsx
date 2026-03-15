"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";
import { AppShell, Badge, ChatLayout, Panel } from "@uap/ui";

type Agent = {
  agent_id: string;
  slug?: string;
  display_name: string;
  description: string;
  modality: "text" | "voice" | "realtime_voice";
  rag_enabled?: boolean;
};

type Conversation = {
  conversation_id: string;
  agent_id: string;
  title: string;
};

type Message = {
  message_id: string;
  role: string;
  content: string;
};

const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:3220";
const voiceApiBaseUrl = process.env.NEXT_PUBLIC_VOICE_API_BASE_URL ?? "http://localhost:3270";
const grafanaBaseUrl = process.env.NEXT_PUBLIC_GRAFANA_URL ?? "http://localhost:13000";
const adminWebUrl = process.env.NEXT_PUBLIC_ADMIN_WEB_URL ?? "http://localhost:3300";
const exampleTenantId = "11111111-1111-1111-1111-111111111111";
const exampleUserId = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1";

type WorkspaceSection = "chat" | "voice" | "search" | "settings" | "api";
type StreamTransport = "websocket" | "sse";

const workspaceSections: Array<{ key: WorkspaceSection; label: string; href: string }> = [
  { key: "chat", label: "Chat", href: "/" },
  { key: "voice", label: "Voice", href: "/voice" },
  { key: "search", label: "Search", href: "/search" },
  { key: "settings", label: "Settings", href: "/settings" },
  { key: "api", label: "API", href: "/api-usage" }
];

function agentTone(modality: string) {
  if (modality === "realtime_voice") {
    return "amber" as const;
  }
  if (modality === "voice") {
    return "emerald" as const;
  }
  return "cyan" as const;
}

export function ChatWorkspace({ initialSection = "chat" }: { initialSection?: WorkspaceSection }) {
  const router = useRouter();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [selectedConversation, setSelectedConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [composer, setComposer] = useState("");
  const [selectedAgent, setSelectedAgent] = useState("");
  const [streamedText, setStreamedText] = useState("");
  const [voiceSession, setVoiceSession] = useState<string>("");
  const [search, setSearch] = useState("");
  const [uploadStatus, setUploadStatus] = useState("");
  const [streamTransport, setStreamTransport] = useState<StreamTransport>("websocket");
  const [workspaceMessage, setWorkspaceMessage] = useState("Chat workspace online. Pick an agent and start a tenant-scoped conversation.");
  const streamSocketRef = useRef<WebSocket | null>(null);

  const selectedAgentRecord = useMemo(
    () => agents.find((agent) => agent.agent_id === (selectedConversation?.agent_id || selectedAgent)) ?? null,
    [agents, selectedAgent, selectedConversation]
  );

  useEffect(() => {
    void Promise.all([
      fetch(`${apiBaseUrl}/api/v1/agents`).then((response) => response.json()),
      fetch(`${apiBaseUrl}/api/v1/conversations`).then((response) => response.json())
    ]).then(([agentsResponse, conversationsResponse]) => {
      setAgents(agentsResponse);
      setConversations(conversationsResponse);
      const firstConversation = conversationsResponse[0] ?? null;
      setSelectedConversation(firstConversation);
      setSelectedAgent(firstConversation?.agent_id ?? agentsResponse[0]?.agent_id ?? "");
    });
  }, []);

  useEffect(() => {
    if (!selectedConversation) {
      return;
    }
    void reloadMessages(selectedConversation.conversation_id);
  }, [selectedConversation]);

  useEffect(() => {
    return () => {
      streamSocketRef.current?.close();
      streamSocketRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (initialSection === "voice") {
      setWorkspaceMessage("Voice-focused workspace loaded. Start or inspect a realtime or push-to-talk session.");
      return;
    }
    if (initialSection === "search") {
      setWorkspaceMessage("Search-focused workspace loaded. Filter conversation history and jump into an existing thread.");
      return;
    }
    if (initialSection === "settings") {
      setWorkspaceMessage("Settings-focused workspace loaded. Review invariants, observability links and environment posture.");
      return;
    }
    if (initialSection === "api") {
      setWorkspaceMessage("API-focused workspace loaded. Use the public agent invoke endpoint to integrate this platform into other applications.");
    }
  }, [initialSection]);

  async function createConversation() {
    if (!selectedAgent) {
      setWorkspaceMessage("Choose an agent before creating a new conversation.");
      return;
    }
    const response = await fetch(`${apiBaseUrl}/api/v1/conversations`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ agent_id: selectedAgent, title: "New chat" })
    }).then((value) => value.json());

    const nextConversation = {
      conversation_id: response.conversation_id as string,
      agent_id: selectedAgent,
      title: "New chat"
    };
    setConversations((current) => [nextConversation, ...current]);
    setSelectedConversation(nextConversation);
    setMessages([]);
    setWorkspaceMessage(`Conversation created for agent ${selectedAgentRecord?.display_name ?? selectedAgent}.`);
  }

  async function searchConversations() {
    const response = await fetch(`${apiBaseUrl}/api/v1/conversations/search?q=${encodeURIComponent(search)}`).then((value) => value.json());
    setConversations(response);
    setWorkspaceMessage(search ? `Filtered conversations by “${search}”.` : "Showing all conversations.");
  }

  async function sendMessage() {
    if (!composer.trim()) {
      return;
    }
    const pending = composer;
    setComposer("");
    await submitConversationInput(pending, `Streaming response from ${selectedAgentRecord?.display_name ?? "selected agent"}...`);
  }

  async function submitConversationInput(input: string, pendingStatus: string) {
    if (!selectedConversation || !input.trim()) {
      return;
    }
    setMessages((current) => [...current, { message_id: crypto.randomUUID(), role: "user", content: input }]);
    setWorkspaceMessage(pendingStatus);
    await streamRunWithPreferredTransport(input);
  }

  async function startVoiceSession() {
    if (!selectedConversation) {
      return;
    }
    const result = await fetch(`${voiceApiBaseUrl}/api/v1/voice/sessions`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ conversation_id: selectedConversation.conversation_id, agent_id: selectedConversation.agent_id })
    }).then((response) => response.json());
    setVoiceSession(result.voice_session_id as string);
    setWorkspaceMessage("Voice session bootstrapped. You can now push a transcript sample into the conversation.");
  }

  async function sendVoiceHint() {
    if (!voiceSession) {
      setWorkspaceMessage("Start a voice session before sending a push-to-talk sample.");
      return;
    }
    const result = await fetch(`${voiceApiBaseUrl}/api/v1/voice/transcribe`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ voice_session_id: voiceSession, text_hint: "spoken request from browser" })
    }).then((response) => response.json());
    await submitConversationInput(result.transcript as string, "Voice transcript captured. Running the agent on the same conversation.");
  }

  async function cloneConversation() {
    if (!selectedConversation || !selectedAgent) {
      return;
    }
    const response = await fetch(`${apiBaseUrl}/api/v1/conversations/${selectedConversation.conversation_id}/clone`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ agent_id: selectedAgent, title: `Clone of ${selectedConversation.title}` })
    }).then((value) => value.json());
    const cloned = {
      conversation_id: response.conversation_id as string,
      agent_id: selectedAgent,
      title: `Clone of ${selectedConversation.title}`
    };
    setConversations((current) => [cloned, ...current]);
    setSelectedConversation(cloned);
    setWorkspaceMessage("Conversation cloned into a new agent workspace. Agent binding remains immutable per chat.");
  }

  async function uploadAttachment(file: File) {
    const formData = new FormData();
    formData.append("file", file);
    const response = await fetch(`${apiBaseUrl}/api/v1/files/upload`, {
      method: "POST",
      body: formData
    }).then((value) => value.json());
    setUploadStatus(`${response.file_name as string} uploaded (${String(response.size)} bytes)`);
    setWorkspaceMessage("Attachment uploaded to the platform file endpoint.");
  }

  async function reloadMessages(conversationId: string) {
    const data = await fetch(`${apiBaseUrl}/api/v1/conversations/${conversationId}/messages`).then((response) => response.json());
    setMessages(data);
  }

  async function streamRunWithPreferredTransport(input: string) {
    setStreamedText("");
    const websocketUrl = `${apiBaseUrl.replace(/^http/, "ws")}/api/v1/conversations/${selectedConversation?.conversation_id}/runs/ws`;

    if (selectedConversation) {
      try {
        await streamRunViaWebSocket(websocketUrl, input);
        setStreamTransport("websocket");
        return;
      } catch (error) {
        console.error("websocket stream failed, falling back to SSE", error);
        setWorkspaceMessage("WebSocket streaming degraded. Falling back to SSE for this run.");
        const failedStream = error as Error & { runId?: string; messageId?: string };
        if (failedStream.runId) {
          await streamRunViaSSE(failedStream.runId);
          setStreamTransport("sse");
          return;
        }
      }
    }

    const run = await fetch(`${apiBaseUrl}/api/v1/conversations/${selectedConversation?.conversation_id}/runs`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ message: input, agent_id: selectedConversation?.agent_id })
    }).then((response) => response.json());

    await streamRunViaSSE(run.run_id as string);
    setStreamTransport("sse");
  }

  async function streamRunViaWebSocket(websocketUrl: string, input: string) {
    await new Promise<void>((resolve, reject) => {
      if (!selectedConversation) {
        reject(new Error("conversation is required"));
        return;
      }

      const socket = new WebSocket(websocketUrl);
      streamSocketRef.current = socket;
      let builtText = "";
      let runId = "";
      let assistantMessageId = "";
      let resolved = false;

      socket.onopen = () => {
        socket.send(JSON.stringify({
          type: "run.start",
          tenant_id: exampleTenantId,
          user_id: exampleUserId,
          agent_id: selectedConversation.agent_id,
          message: input
        }));
      };

      socket.onmessage = (event) => {
        const payload = JSON.parse(event.data) as { type: string; payload?: { delta?: string; message_id?: string; run_id?: string; error?: string } };
        if (payload.type === "run.started") {
          runId = typeof payload.payload?.run_id === "string" ? payload.payload.run_id : runId;
          assistantMessageId = typeof payload.payload?.message_id === "string" ? payload.payload.message_id : assistantMessageId;
          setWorkspaceMessage("WebSocket stream established. Response can resume over SSE if the connection degrades.");
          return;
        }
        if (payload.type === "message.delta" && payload.payload?.delta) {
          builtText = `${builtText}${builtText ? " " : ""}${payload.payload.delta}`;
          setStreamedText(builtText);
          return;
        }
        if (payload.type === "run.completed") {
          runId = typeof payload.payload?.run_id === "string" ? payload.payload.run_id : runId;
          assistantMessageId = typeof payload.payload?.message_id === "string" ? payload.payload.message_id : assistantMessageId;
          reloadMessages(selectedConversation.conversation_id)
            .then(() => {
              setStreamedText("");
              setWorkspaceMessage("Response completed over WebSocket and persisted to the conversation timeline.");
              resolved = true;
              socket.close();
              resolve();
            })
            .catch((error) => {
              const streamError = error as Error & { runId?: string; messageId?: string };
              streamError.runId = runId;
              streamError.messageId = assistantMessageId;
              reject(streamError);
            });
          return;
        }
        if (payload.type === "run.failed") {
          const message = typeof payload.payload?.error === "string" ? payload.payload.error : "WebSocket stream failed";
          if (!resolved) {
            resolved = true;
            socket.close();
            const streamError = new Error(message) as Error & { runId?: string; messageId?: string };
            streamError.runId = runId;
            streamError.messageId = assistantMessageId;
            reject(streamError);
          }
        }
      };

      socket.onerror = () => {
        if (!resolved) {
          resolved = true;
          const streamError = new Error("WebSocket error") as Error & { runId?: string; messageId?: string };
          streamError.runId = runId;
          streamError.messageId = assistantMessageId;
          reject(streamError);
        }
      };

      socket.onclose = () => {
        streamSocketRef.current = null;
        if (!resolved) {
          const streamError = new Error("WebSocket closed before completion") as Error & { runId?: string; messageId?: string };
          streamError.runId = runId;
          streamError.messageId = assistantMessageId;
          reject(streamError);
        }
      };
    });
  }

  async function streamRunViaSSE(runId: string) {
    await new Promise<void>((resolve, reject) => {
      let builtText = "";
      const eventSource = new EventSource(`${apiBaseUrl}/api/v1/runs/${runId}/events`);
      eventSource.addEventListener("message.delta", (event) => {
        const payload = JSON.parse((event as MessageEvent).data) as { delta: string };
        builtText = `${builtText}${builtText ? " " : ""}${payload.delta}`;
        setStreamedText(builtText);
      });
      eventSource.addEventListener("run.completed", () => {
        if (!selectedConversation) {
          eventSource.close();
          resolve();
          return;
        }
        reloadMessages(selectedConversation.conversation_id)
          .then(() => {
            setStreamedText("");
            setWorkspaceMessage("Response completed over SSE and persisted to the conversation timeline.");
            eventSource.close();
            resolve();
          })
          .catch((error) => {
            eventSource.close();
            reject(error);
          });
      });
      eventSource.onerror = () => {
        eventSource.close();
        reject(new Error("SSE stream failed"));
      };
    });
  }

  const sidebar = (
    <div className="flex h-full flex-col gap-5 px-4 py-5">
      <div className="rounded-[1.8rem] border border-white/10 bg-white/[0.04] p-4">
        <p className="font-display text-[11px] uppercase tracking-[0.28em] text-cyan-200/80">UniversalAgentPlatform</p>
        <h1 className="mt-3 font-display text-2xl font-semibold text-white">Chat cockpit</h1>
        <p className="mt-3 text-sm leading-6 text-slate-300">Multi-agent chat with streaming text, voice bootstrap, transcript projection and immutable chat-to-agent binding.</p>
      </div>

      <div className="grid grid-cols-2 gap-2 lg:grid-cols-1">
        {workspaceSections.map((section) => (
          <Link
            key={section.key}
            href={section.href}
            className={`rounded-[1.25rem] border px-4 py-3 text-left transition duration-200 ${
              initialSection === section.key ? "border-cyan-300/25 bg-cyan-300/12 text-white" : "border-white/10 bg-white/[0.03] text-slate-300 hover:border-white/20 hover:bg-white/[0.05]"
            }`}
          >
            <div className="text-sm font-medium">{section.label}</div>
          </Link>
        ))}
      </div>

      <Panel kicker="Conversation rail" title="Chats" description={`${conversations.length} persisted conversations for this user.`} className="p-4">
        <div className="space-y-3">
          {conversations.map((conversation) => {
            const isActive = selectedConversation?.conversation_id === conversation.conversation_id;
            const agent = agents.find((item) => item.agent_id === conversation.agent_id);
            return (
              <button
                key={conversation.conversation_id}
                className={`w-full rounded-[1.4rem] border px-4 py-3 text-left transition duration-200 ${
                  isActive ? "border-cyan-300/30 bg-cyan-300/10" : "border-white/10 bg-white/[0.03] hover:border-white/20 hover:bg-white/[0.05]"
                }`}
                onClick={() => setSelectedConversation(conversation)}
                type="button"
              >
                <p className="font-medium text-white">{conversation.title}</p>
                <div className="mt-3 flex flex-wrap gap-2">
                  <Badge tone="slate">{agent?.display_name ?? "Agent"}</Badge>
                  <Badge tone={agentTone(agent?.modality ?? "text")}>{agent?.modality ?? "text"}</Badge>
                </div>
              </button>
            );
          })}
        </div>
      </Panel>
    </div>
  );

  const header = (
    <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p className="text-[11px] uppercase tracking-[0.28em] text-cyan-200/80">Operator-grade chat experience</p>
        <h2 className="font-display mt-2 text-3xl font-semibold tracking-tight text-white">Grounded conversation workspace with live streaming and voice controls.</h2>
        <p className="mt-3 max-w-4xl text-sm leading-6 text-slate-300">{workspaceMessage}</p>
      </div>
      <div className="flex flex-wrap gap-3">
        <input
          className="w-full rounded-full border border-white/10 bg-slate-950/70 px-4 py-2 text-sm text-white outline-none transition focus:border-cyan-300/40 focus:ring-2 focus:ring-cyan-300/15 sm:w-64"
          placeholder="Search chats"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
        />
        <button className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" onClick={searchConversations} type="button">
          Search
        </button>
        <select
          className="rounded-full border border-white/10 bg-slate-950/70 px-4 py-2 text-sm text-white outline-none transition focus:border-cyan-300/40 focus:ring-2 focus:ring-cyan-300/15"
          value={selectedAgent}
          onChange={(event) => setSelectedAgent(event.target.value)}
        >
          {agents.map((agent) => (
            <option key={agent.agent_id} value={agent.agent_id}>
              {agent.display_name}
            </option>
          ))}
        </select>
        <button className="rounded-full bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950 transition hover:bg-cyan-200" onClick={createConversation} type="button">
          New chat
        </button>
      </div>
    </div>
  );

  const sectionFocus =
    initialSection === "voice" ? (
      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel kicker="Voice mode" title="Realtime and push-to-talk validation" description="Use this route when testing browser audio, transcript projection and fallback from voice to text.">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Flow</p>
              <p className="mt-3 text-sm leading-6 text-slate-200">Start a voice session, send a push-to-talk sample, then verify the transcript is projected into the active conversation timeline.</p>
            </div>
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Ops link</p>
              <a className="mt-3 inline-flex text-sm text-cyan-200 transition hover:text-cyan-100" href={`${grafanaBaseUrl}/d/uap-voice-pipeline/voice-pipeline`} target="_blank" rel="noreferrer">
                Open Voice Pipeline dashboard
              </a>
            </div>
          </div>
        </Panel>
        <Panel kicker="Voice invariants" title="What should always hold true" description="These are the checks operators and QA should use on every voice-capable agent.">
          <ul className="space-y-3 text-sm leading-6 text-slate-300">
            <li>Realtime-capable agents should establish a voice session without blocking the chat UI.</li>
            <li>Push-to-talk should persist the transcript into the same conversation record.</li>
            <li>Barge-in and fallback should keep the agent reachable through text even when voice is degraded.</li>
          </ul>
        </Panel>
      </section>
    ) : initialSection === "search" ? (
      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel kicker="Search mode" title="Conversation retrieval and operator QA" description="This route narrows the workspace around history search so users can jump back into the correct thread quickly.">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">How to use</p>
              <p className="mt-3 text-sm leading-6 text-slate-200">Use the top search box to filter by conversation title, then pick the exact thread from the left rail. This keeps agent binding immutable while still making large chat histories operable.</p>
            </div>
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Design goal</p>
              <p className="mt-3 text-sm leading-6 text-slate-200">Search is separated from generation so power users can navigate history without disturbing the active stream.</p>
            </div>
          </div>
        </Panel>
      </section>
    ) : initialSection === "settings" ? (
      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel kicker="Settings mode" title="Environment, invariants and operational links" description="Keep the chat workspace understandable in local Docker, kind, or a public cloud hostname.">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Chat API</p>
              <p className="mt-3 break-all text-sm text-slate-200">{apiBaseUrl}</p>
            </div>
            <div className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Voice API</p>
              <p className="mt-3 break-all text-sm text-slate-200">{voiceApiBaseUrl}</p>
            </div>
          </div>
        </Panel>
      </section>
    ) : initialSection === "api" ? (
      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Panel kicker="External API" title="Use agents outside the chat application" description="The platform exposes a public agent invocation endpoint so any other internal application can call the same agent registry and provider-routing plane.">
          <div className="space-y-4">
            <div className="rounded-[1.4rem] border border-white/10 bg-slate-950/70 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">cURL</p>
              <pre className="mt-3 overflow-x-auto text-xs leading-6 text-slate-200">{`curl -X POST ${apiBaseUrl}/api/v1/agents/<agent_id>/respond \\
  -H 'Content-Type: application/json' \\
  -H 'X-Tenant-ID: ${exampleTenantId}' \\
  -d '{
    "message": "Summarise the tenant handbook",
    "user_id": "${exampleUserId}"
  }'`}</pre>
            </div>
            <div className="rounded-[1.4rem] border border-white/10 bg-slate-950/70 p-4">
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Response shape</p>
              <pre className="mt-3 overflow-x-auto text-xs leading-6 text-slate-200">{`{
  "agent_id": "...",
  "agent_version_id": "...",
  "tenant_id": "...",
  "provider_name": "acme-demo-provider",
  "provider_kind": "demo",
  "rag_enabled": true,
  "text": "..."
}`}</pre>
            </div>
          </div>
        </Panel>
        <Panel kicker="Integration path" title="What this API gives you" description="External apps can use the same agents, policies, provider routing and RAG without embedding chat-specific UI concerns.">
          <ul className="space-y-3 text-sm leading-6 text-slate-300">
            <li>Use the public invoke endpoint for portals, backoffice tools, knowledge widgets and automation entry points.</li>
            <li>Keep agent definitions in the admin cockpit so prompt/model/policy changes are centralized.</li>
            <li>Open the admin cockpit to add agents, bind providers and inspect usage before exposing the agent to more applications.</li>
          </ul>
        </Panel>
      </section>
    ) : null;

  return (
    <AppShell sidebar={sidebar} header={header}>
      <div className="grid gap-6">
        <section className="grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
          <Panel kicker="Selected agent" title={selectedAgentRecord?.display_name ?? "Pick an agent"} description={selectedAgentRecord?.description ?? "Select an agent to open a new chat or clone an existing conversation into it."}>
            <div className="flex flex-wrap gap-2">
              {selectedAgentRecord ? (
                <>
                  <Badge tone={agentTone(selectedAgentRecord.modality)}>{selectedAgentRecord.modality}</Badge>
                  {selectedAgentRecord.rag_enabled ? <Badge tone="emerald">rag enabled</Badge> : null}
                  {selectedAgentRecord.slug ? <Badge tone="slate">{selectedAgentRecord.slug}</Badge> : null}
                </>
              ) : (
                <Badge tone="slate">no agent selected</Badge>
              )}
            </div>
          </Panel>
          <Panel kicker="Live session" title="Workspace status" description="Keep an eye on voice state, file uploads and observability without leaving the conversation.">
            <div className="grid gap-3 sm:grid-cols-4">
              <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.03] p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Voice session</p>
                <p className="font-mono-ui mt-3 break-all text-xs text-slate-200">{voiceSession || "not started"}</p>
              </div>
              <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.03] p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Attachment</p>
                <p className="mt-3 text-sm text-slate-200">{uploadStatus || "none uploaded"}</p>
              </div>
              <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.03] p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Streaming transport</p>
                <p className="mt-3 text-sm text-slate-200">{streamTransport === "websocket" ? "websocket preferred" : "sse fallback"}</p>
              </div>
              <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.03] p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Observability</p>
                <a className="mt-3 inline-flex text-sm text-cyan-200 transition hover:text-cyan-100" href={`${grafanaBaseUrl}/d/uap-chat-pipeline/chat-pipeline`} target="_blank" rel="noreferrer">
                  Open chat dashboard
                </a>
              </div>
            </div>
          </Panel>
        </section>

        {sectionFocus}

        <ChatLayout
          chatList={
            <div className="space-y-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-[11px] uppercase tracking-[0.24em] text-cyan-200/80">Agent roster</p>
                  <h3 className="font-display mt-1 text-xl font-semibold text-white">Available agents</h3>
                </div>
                <a className="rounded-full border border-white/10 px-3 py-1.5 text-xs text-slate-200 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={adminWebUrl} target="_blank" rel="noreferrer">
                  Open admin
                </a>
              </div>
              <div className="space-y-3">
                {agents.map((agent) => (
                  <div key={agent.agent_id} className="rounded-[1.4rem] border border-white/10 bg-white/[0.03] p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="font-medium text-white">{agent.display_name}</p>
                        <p className="mt-1 text-sm leading-6 text-slate-300">{agent.description}</p>
                      </div>
                      <Badge tone={agentTone(agent.modality)}>{agent.modality}</Badge>
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {agent.rag_enabled ? <Badge tone="emerald">knowledge</Badge> : null}
                      {agent.slug ? <Badge tone="slate">{agent.slug}</Badge> : null}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          }
          transcript={
            <div className="flex h-full flex-col">
              <div className="mb-4 flex flex-wrap gap-2">
                <Badge tone="cyan">{selectedConversation ? "conversation active" : "no conversation"}</Badge>
                {selectedAgentRecord ? <Badge tone={agentTone(selectedAgentRecord.modality)}>{selectedAgentRecord.display_name}</Badge> : null}
                {selectedAgentRecord?.rag_enabled ? <Badge tone="emerald">grounded retrieval</Badge> : null}
              </div>
              <div className="flex-1 space-y-4 overflow-y-auto pr-2">
                {messages.map((message) => (
                  <div key={message.message_id} className={`max-w-[88%] rounded-[1.8rem] px-4 py-4 shadow-[0_12px_40px_rgba(0,0,0,0.16)] ${message.role === "user" ? "ml-auto bg-cyan-300 text-slate-950" : "bg-white/[0.05] text-slate-100"}`}>
                    <div className="mb-2 flex items-center gap-2">
                      <Badge tone={message.role === "user" ? "slate" : "emerald"} className={message.role === "user" ? "!border-slate-900/10 !bg-slate-900/10 !text-slate-950" : ""}>
                        {message.role}
                      </Badge>
                    </div>
                    <p className="whitespace-pre-wrap text-sm leading-7">{message.content}</p>
                  </div>
                ))}
                {streamedText ? (
                  <div className="max-w-[88%] rounded-[1.8rem] bg-white/[0.06] px-4 py-4 text-slate-100 shadow-[0_12px_40px_rgba(0,0,0,0.16)]">
                    <div className="mb-2 flex items-center gap-2">
                      <Badge tone="emerald">streaming</Badge>
                    </div>
                    <p className="whitespace-pre-wrap text-sm leading-7">{streamedText}</p>
                  </div>
                ) : null}
              </div>
            </div>
          }
          composer={
            <div className="space-y-4">
              <textarea
                className="min-h-32 w-full rounded-[1.8rem] border border-white/10 bg-slate-950/70 px-4 py-4 text-sm leading-7 text-slate-100 outline-none transition focus:border-cyan-300/40 focus:ring-2 focus:ring-cyan-300/15"
                placeholder="Send a tenant-scoped message to the selected agent..."
                value={composer}
                onChange={(event) => setComposer(event.target.value)}
              />
              <div className="flex flex-wrap items-center gap-3">
                <button className="rounded-full bg-cyan-300 px-4 py-2 font-medium text-slate-950 transition hover:bg-cyan-200" onClick={sendMessage} type="button">
                  Stream response
                </button>
                <button className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" onClick={cloneConversation} type="button">
                  Clone to agent
                </button>
                <button className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" onClick={startVoiceSession} type="button">
                  Start voice session
                </button>
                <button className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" onClick={sendVoiceHint} type="button">
                  Push-to-talk sample
                </button>
                <label className="cursor-pointer rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm text-slate-100 transition hover:border-cyan-300/20 hover:bg-cyan-300/10">
                  Upload attachment
                  <input
                    className="hidden"
                    type="file"
                    onChange={(event) => {
                      const file = event.target.files?.[0];
                      if (file) {
                        void uploadAttachment(file);
                      }
                    }}
                  />
                </label>
              </div>
            </div>
          }
          sidePanel={
            <div className="space-y-4">
              <Panel kicker="Voice" title="Realtime session" description="Bootstrap voice, send a sample transcript and verify the projection into chat history." className="p-4">
                <div className="space-y-3">
                  <div className="rounded-[1.25rem] border border-white/10 bg-white/[0.03] p-4">
                    <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Current session</p>
                    <p className="font-mono-ui mt-3 break-all text-xs text-slate-200">{voiceSession || "No active voice session"}</p>
                  </div>
                </div>
              </Panel>
              <Panel kicker="Operator notes" title="Conversation invariants" description="Critical product rules stay visible inside the user workspace." className="p-4">
                <ul className="space-y-3 text-sm leading-6 text-slate-300">
                  <li>Each conversation is pinned to one `agent_id`.</li>
                  <li>Changing agent means creating or cloning into a new chat.</li>
                  <li>Streaming text arrives over SSE and persists after completion.</li>
                  <li>Voice transcripts are projected back into the conversation timeline.</li>
                </ul>
              </Panel>
              <Panel kicker="Operations" title="Monitoring and control" description="Jump to the platform surfaces you need while validating a chat flow." className="p-4">
                <div className="grid gap-3">
                  <a className="rounded-[1.2rem] border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-slate-200 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={`${grafanaBaseUrl}/d/uap-chat-pipeline/chat-pipeline`} target="_blank" rel="noreferrer">
                    Open Chat Pipeline dashboard
                  </a>
                  <a className="rounded-[1.2rem] border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-slate-200 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" href={adminWebUrl} target="_blank" rel="noreferrer">
                    Open Admin cockpit
                  </a>
                  <button className="rounded-[1.2rem] border border-white/10 bg-white/[0.03] px-4 py-3 text-left text-sm text-slate-200 transition hover:border-cyan-300/20 hover:bg-cyan-300/10" onClick={() => router.push("/api-usage")} type="button">
                    Open API integration guide
                  </button>
                </div>
              </Panel>
            </div>
          }
        />
      </div>
    </AppShell>
  );
}
