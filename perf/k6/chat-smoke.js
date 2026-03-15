import http from "k6/http";
import { check, sleep } from "k6";

const vus = Number(__ENV.VUS || 2);
const duration = __ENV.DURATION || "30s";
const baseUrl = __ENV.BASE_URL || "http://localhost:3220";

export const options = {
  vus,
  duration,
  thresholds: {
    http_req_duration: ["p(95)<1500"],
    checks: ["rate>0.99"]
  }
};

export default function () {
  const health = http.get(`${baseUrl}/api/health`);
  check(health, {
    "chat gateway healthy": (response) => response.status === 200
  });

  const agentsResponse = http.get(`${baseUrl}/api/v1/agents`);
  check(agentsResponse, {
    "agents listed": (response) => response.status === 200
  });
  const agents = agentsResponse.json();
  const roster = Array.isArray(agents) ? agents : [];
  const textAgent = roster.find((item) => item.modality === "text") || roster[0];
  const voiceAgent = roster.find((item) => item.modality === "voice");
  const realtimeAgent = roster.find((item) => item.modality === "realtime_voice");
  if (!textAgent || !textAgent.agent_id) {
    sleep(1);
    return;
  }

  const createConversation = http.post(
    `${baseUrl}/api/v1/conversations`,
    JSON.stringify({ agent_id: textAgent.agent_id, title: "k6 smoke chat" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(createConversation, {
    "conversation created": (response) => response.status === 201
  });

  const conversation = createConversation.json();
  if (!conversation || !conversation.conversation_id) {
    sleep(1);
    return;
  }

  const runResponse = http.post(
    `${baseUrl}/api/v1/conversations/${conversation.conversation_id}/runs`,
    JSON.stringify({ agent_id: textAgent.agent_id, message: "k6 synthetic request" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(runResponse, {
    "run created": (response) => response.status === 201
  });

  const syncInvoke = http.post(
    `${baseUrl}/api/v1/agents/${textAgent.agent_id}/respond`,
    JSON.stringify({ message: "k6 direct invoke" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(syncInvoke, {
    "text agent direct invoke works": (response) => response.status === 200
  });

  const streamResponse = http.post(
    `${baseUrl}/api/v1/agents/${textAgent.agent_id}/respond/stream`,
    JSON.stringify({ message: "k6 streamed invoke" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(streamResponse, {
    "stream endpoint reachable": (response) => response.status === 200,
    "stream emits deltas": (response) => response.body.includes("event: message.delta")
  });

  if (voiceAgent && voiceAgent.agent_id) {
    const voiceInvoke = http.post(
      `${baseUrl}/api/v1/agents/${voiceAgent.agent_id}/respond-from-voice`,
      JSON.stringify({ text_hint: "k6 voice invoke", speak_response: true }),
      { headers: { "Content-Type": "application/json" } }
    );
    check(voiceInvoke, {
      "voice agent invoke works": (response) => response.status === 200,
      "voice agent returns transcript": (response) => {
        const payload = response.json();
        return Boolean(payload.transcript);
      }
    });
  }

  if (realtimeAgent && realtimeAgent.agent_id) {
    const realtimeInvoke = http.post(
      `${baseUrl}/api/v1/agents/${realtimeAgent.agent_id}/respond-from-voice`,
      JSON.stringify({ text_hint: "k6 realtime invoke", speak_response: true }),
      { headers: { "Content-Type": "application/json" } }
    );
    check(realtimeInvoke, {
      "realtime agent api invoke works": (response) => response.status === 200
    });
  }

  const searchResponse = http.get(`${baseUrl}/api/v1/conversations/search?q=k6`);
  check(searchResponse, {
    "conversation search works": (response) => response.status === 200
  });

  sleep(1);
}
