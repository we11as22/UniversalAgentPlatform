#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

declare -a images=(
  "uap/admin-web:dev apps/admin-web/Dockerfile"
  "uap/chat-web:dev apps/chat-web/Dockerfile"
  "uap/admin-api:dev services/admin-api/Dockerfile"
  "uap/chat-gateway:dev services/chat-gateway/Dockerfile"
  "uap/session-service:dev services/session-service/Dockerfile"
  "uap/conversation-service:dev services/conversation-service/Dockerfile"
  "uap/agent-router:dev services/agent-router/Dockerfile"
  "uap/provider-gateway:dev services/provider-gateway/Dockerfile"
  "uap/voice-gateway:dev services/voice-gateway/Dockerfile"
  "uap/quota-service:dev services/quota-service/Dockerfile"
  "uap/audit-service:dev services/audit-service/Dockerfile"
  "uap/transcript-service:dev services/transcript-service/Dockerfile"
  "uap/agent-runtime:dev services/agent-runtime/Dockerfile"
  "uap/tool-runner:dev services/tool-runner/Dockerfile"
  "uap/rag-service:dev services/rag-service/Dockerfile"
  "uap/indexer:dev services/indexer/Dockerfile"
  "uap/workflow-workers:dev services/workflow-workers/Dockerfile"
)

for spec in "${images[@]}"; do
  image="${spec%% *}"
  dockerfile="${spec#* }"
  echo "Building ${image} from ${dockerfile}"
  docker build --progress=plain -t "${image}" -f "${dockerfile}" .
done
