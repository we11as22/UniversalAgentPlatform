#!/usr/bin/env bash
set -euo pipefail

wait_for_url() {
  local url="$1"
  local attempts="${2:-60}"
  local delay="${3:-2}"

  for ((i=1; i<=attempts; i++)); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      echo "Ready: ${url}"
      return 0
    fi
    echo "Waiting (${i}/${attempts}): ${url}"
    sleep "${delay}"
  done

  echo "Failed readiness: ${url}" >&2
  return 1
}

checks=(
  "http://localhost:18081/realms/uap/.well-known/openid-configuration"
  "http://localhost:19000/minio/health/ready"
  "http://localhost:19090/-/ready"
  "http://localhost:13100/ready"
  "http://localhost:3220/api/health"
  "http://localhost:3210/api/health"
  "http://localhost:3200/api/health"
  "http://localhost:3300/api/health"
)

for url in "${checks[@]}"; do
  wait_for_url "${url}"
done

echo "Validating agent invoke matrix"
curl -fsS http://localhost:3220/api/v1/agents >/tmp/uap-agents.json
TEXT_AGENT_ID="$(jq -r '.[] | select(.modality=="text") | .agent_id' /tmp/uap-agents.json | head -n1)"
VOICE_AGENT_ID="$(jq -r '.[] | select(.modality=="voice") | .agent_id' /tmp/uap-agents.json | head -n1)"
REALTIME_AGENT_ID="$(jq -r '.[] | select(.modality=="realtime_voice") | .agent_id' /tmp/uap-agents.json | head -n1)"

curl -fsS -X POST "http://localhost:3220/api/v1/agents/${TEXT_AGENT_ID}/respond" \
  -H 'Content-Type: application/json' \
  -d '{"message":"smoke direct invoke"}' >/tmp/uap-respond.json
jq -e '.text' /tmp/uap-respond.json >/dev/null

curl -fsS -X POST "http://localhost:3220/api/v1/agents/${TEXT_AGENT_ID}/respond/stream" \
  -H 'Content-Type: application/json' \
  -d '{"message":"smoke stream invoke"}' >/tmp/uap-stream.txt
grep -q 'event: message.delta' /tmp/uap-stream.txt

python3 - <<'PY'
import asyncio
import json
from urllib.request import urlopen

import websockets


async def main() -> None:
    with urlopen("http://localhost:3220/api/v1/agents") as handle:
        agents = json.load(handle)
    text_agent = next(item for item in agents if item.get("modality") == "text")

    payload = json.dumps({"agent_id": text_agent["agent_id"], "title": "smoke websocket chat"}).encode()
    from urllib.request import Request
    request = Request("http://localhost:3220/api/v1/conversations", data=payload, headers={"Content-Type": "application/json"}, method="POST")
    with urlopen(request) as handle:
        conversation = json.load(handle)

    async with websockets.connect(
        f"ws://localhost:3220/api/v1/conversations/{conversation['conversation_id']}/runs/ws",
        open_timeout=5,
        close_timeout=5,
        ping_interval=10,
    ) as socket:
        await socket.send(
            json.dumps(
                {
                    "type": "run.start",
                    "tenant_id": "11111111-1111-1111-1111-111111111111",
                    "user_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1",
                    "agent_id": text_agent["agent_id"],
                    "message": "smoke websocket invoke",
                }
            )
        )
        delta_seen = False
        completed_seen = False
        started_seen = False
        while True:
            event = json.loads(await asyncio.wait_for(socket.recv(), timeout=10))
            if event.get("type") == "run.started":
                started_seen = bool(event.get("payload", {}).get("run_id"))
            if event.get("type") == "message.delta":
                delta_seen = True
            if event.get("type") == "run.completed":
                completed_seen = True
                break
        assert started_seen and delta_seen and completed_seen


asyncio.run(main())
PY

curl -fsS -X POST "http://localhost:3220/api/v1/agents/${TEXT_AGENT_ID}/respond-from-voice" \
  -H 'Content-Type: application/json' \
  -d '{"text_hint":"smoke spoken request","speak_response":true}' >/tmp/uap-voice-text.json
jq -e '.transcript and .tts.audio_url' /tmp/uap-voice-text.json >/dev/null

curl -fsS -X POST "http://localhost:3270/api/v1/voice/sessions" \
  -H 'Content-Type: application/json' \
  -d "{\"agent_id\":\"${REALTIME_AGENT_ID}\",\"user_id\":\"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1\"}" >/tmp/uap-voice-session.json
jq -e '.voice_session_id and .conversation_id' /tmp/uap-voice-session.json >/dev/null

if [[ -n "${VOICE_AGENT_ID}" ]]; then
  curl -fsS -X POST "http://localhost:3220/api/v1/agents/${VOICE_AGENT_ID}/respond-from-voice" \
    -H 'Content-Type: application/json' \
    -d '{"text_hint":"smoke voice agent","speak_response":true}' >/tmp/uap-voice-agent.json
  jq -e '.text and .tts.audio_url' /tmp/uap-voice-agent.json >/dev/null
fi

echo "Smoke checks passed"
