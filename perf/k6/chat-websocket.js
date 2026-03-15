import http from "k6/http";
import ws from "k6/ws";
import { check, sleep } from "k6";

const vus = Number(__ENV.VUS || 2);
const duration = __ENV.DURATION || "30s";
const baseUrl = __ENV.BASE_URL || "http://localhost:3220";
const wsBaseUrl = baseUrl.replace(/^http/, "ws");

export const options = {
  vus,
  duration,
  thresholds: {
    checks: ["rate>0.99"],
    ws_session_duration: ["p(95)<3000"]
  }
};

export default function () {
  const agentsResponse = http.get(`${baseUrl}/api/v1/agents`);
  check(agentsResponse, {
    "ws agents listed": (response) => response.status === 200
  });
  const agents = agentsResponse.json();
  const roster = Array.isArray(agents) ? agents : [];
  const textAgent = roster.find((item) => item.modality === "text") || roster[0];
  if (!textAgent || !textAgent.agent_id) {
    sleep(1);
    return;
  }

  const createConversation = http.post(
    `${baseUrl}/api/v1/conversations`,
    JSON.stringify({ agent_id: textAgent.agent_id, title: "k6 websocket chat" }),
    { headers: { "Content-Type": "application/json" } }
  );
  check(createConversation, {
    "ws conversation created": (response) => response.status === 201
  });
  const conversation = createConversation.json();
  if (!conversation || !conversation.conversation_id) {
    sleep(1);
    return;
  }

  let streamOpened = false;
  let deltaSeen = false;
  let completedSeen = false;
  let runId = "";

  const response = ws.connect(`${wsBaseUrl}/api/v1/conversations/${conversation.conversation_id}/runs/ws`, {}, (socket) => {
    socket.on("open", () => {
      streamOpened = true;
      socket.send(
        JSON.stringify({
          type: "run.start",
          agent_id: textAgent.agent_id,
          message: "k6 websocket invoke",
          tenant_id: "11111111-1111-1111-1111-111111111111",
          user_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
        })
      );
    });

    socket.on("message", (raw) => {
      const event = JSON.parse(raw);
      if (event.type === "run.started" && event.payload && event.payload.run_id) {
        runId = event.payload.run_id;
      }
      if (event.type === "message.delta") {
        deltaSeen = true;
      }
      if (event.type === "run.completed") {
        completedSeen = true;
        socket.close();
      }
      if (event.type === "run.failed") {
        socket.close();
      }
    });

    socket.setTimeout(() => {
      socket.close();
    }, 4000);
  });

  check(response, {
    "websocket upgraded": (res) => res && res.status === 101,
    "websocket opened": () => streamOpened,
    "websocket emitted delta": () => deltaSeen,
    "websocket completed": () => completedSeen,
    "websocket exposed run id": () => Boolean(runId)
  });

  sleep(1);
}
