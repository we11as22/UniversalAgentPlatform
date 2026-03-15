#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/versions.env"

export PATH="${HOME}/.local/bin:/usr/local/bin:/usr/bin:${PATH}"

CLUSTER_NAME="${CLUSTER_NAME:-uap}"
KUBECONFIG_CONTEXT="kind-${CLUSTER_NAME}"
EXTERNAL_PORT="${UAP_EXTERNAL_PORT:-8088}"
LIVEKIT_PORT="${UAP_LIVEKIT_PORT:-17880}"
PUBLIC_SCHEME="${UAP_PUBLIC_SCHEME:-http}"
LIVEKIT_SCHEME="${UAP_LIVEKIT_SCHEME:-ws}"
KEYCLOAK_ADMIN_USER="${KEYCLOAK_ADMIN_USER:-admin}"
KEYCLOAK_ADMIN_PASSWORD="${KEYCLOAK_ADMIN_PASSWORD:-admin}"

resolve_base_domain() {
  if [[ -n "${UAP_BASE_DOMAIN:-}" ]]; then
    echo "${UAP_BASE_DOMAIN}"
    return
  fi

  local public_ip=""
  if command -v curl >/dev/null 2>&1; then
    public_ip="$(curl -4fsS --max-time 5 https://api.ipify.org || true)"
  fi

  if [[ -n "${public_ip}" ]]; then
    echo "${public_ip}.sslip.io"
    return
  fi

  echo "uap.localtest.me"
}

BASE_DOMAIN="$(resolve_base_domain)"

if [[ "${EXTERNAL_PORT}" != "8088" ]]; then
  echo "UAP_EXTERNAL_PORT=${EXTERNAL_PORT} requested, but kind host mapping is fixed to 8088. Using 8088." >&2
  EXTERNAL_PORT="8088"
fi

if [[ "${LIVEKIT_PORT}" != "17880" ]]; then
  echo "UAP_LIVEKIT_PORT=${LIVEKIT_PORT} requested, but kind host mapping is fixed to 17880. Using 17880." >&2
  LIVEKIT_PORT="17880"
fi

ensure_cluster() {
  if ! kind get clusters | grep -qx "${CLUSTER_NAME}"; then
    kind create cluster --name "${CLUSTER_NAME}" --config "${ROOT_DIR}/infra/k8s/kind/kind-config.yaml"
  fi
  kubectl cluster-info --context "${KUBECONFIG_CONTEXT}" >/dev/null
}

install_istio() {
  helm repo add istio https://istio-release.storage.googleapis.com/charts >/dev/null 2>&1 || true
  helm repo update >/dev/null
  helm upgrade --install istio-base istio/base --namespace istio-system --create-namespace --version "${ISTIO_VERSION}" --kube-context "${KUBECONFIG_CONTEXT}"
  helm upgrade --install istiod istio/istiod --namespace istio-system --create-namespace --version "${ISTIO_VERSION}" --wait --kube-context "${KUBECONFIG_CONTEXT}"
}

install_envoy_gateway() {
  kubectl apply --context "${KUBECONFIG_CONTEXT}" -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.1/standard-install.yaml"
  helm upgrade --install envoy-gateway oci://docker.io/envoyproxy/gateway-helm --version "v${ENVOY_GATEWAY_VERSION}" --namespace envoy-gateway-system --create-namespace --wait --kube-context "${KUBECONFIG_CONTEXT}"
}

load_images() {
  if [[ "${SKIP_IMAGE_BUILD:-0}" == "1" ]]; then
    echo "Skipping local image build/load because SKIP_IMAGE_BUILD=1"
    return
  fi
  "${ROOT_DIR}/scripts/build-images.sh"
  local images=(
    uap/admin-web:dev
    uap/chat-web:dev
    uap/admin-api:dev
    uap/chat-gateway:dev
    uap/session-service:dev
    uap/conversation-service:dev
    uap/agent-router:dev
    uap/provider-gateway:dev
    uap/voice-gateway:dev
    uap/quota-service:dev
    uap/audit-service:dev
    uap/transcript-service:dev
    uap/agent-runtime:dev
    uap/tool-runner:dev
    uap/rag-service:dev
    uap/indexer:dev
    uap/workflow-workers:dev
  )
  for image in "${images[@]}"; do
    kind load docker-image --name "${CLUSTER_NAME}" "${image}"
  done
}

deploy_platform() {
  helm upgrade --install uap-platform "${ROOT_DIR}/infra/helm/platform" \
    --namespace uap \
    --create-namespace \
    --kube-context "${KUBECONFIG_CONTEXT}" \
    -f "${ROOT_DIR}/infra/helm/platform/values.yaml" \
    -f "${ROOT_DIR}/infra/helm/platform/values-kind.yaml" \
    --set-string global.baseDomain="${BASE_DOMAIN}" \
    --set-string global.publicScheme="${PUBLIC_SCHEME}" \
    --set-string global.livekitScheme="${LIVEKIT_SCHEME}" \
    --set-string global.externalPort="${EXTERNAL_PORT}" \
    --set-string global.livekitPort="${LIVEKIT_PORT}" \
    --wait
}

patch_gateway_service() {
  local gateway_service
  for _ in $(seq 1 60); do
    gateway_service="$(kubectl --context "${KUBECONFIG_CONTEXT}" -n uap get svc -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | grep '^uap-gateway' | head -n1 || true)"
    if [[ -n "${gateway_service}" ]]; then
      break
    fi
    sleep 2
  done

  if [[ -z "${gateway_service}" ]]; then
    echo "Unable to locate Envoy Gateway service for uap-gateway" >&2
    exit 1
  fi

  kubectl --context "${KUBECONFIG_CONTEXT}" -n uap patch service "${gateway_service}" --type merge -p '{"spec":{"type":"NodePort","ports":[{"name":"http","port":80,"nodePort":30080,"protocol":"TCP"}]}}' >/dev/null
}

configure_keycloak_clients() {
  local pod
  pod="$(kubectl --context "${KUBECONFIG_CONTEXT}" -n uap get pods -l app.kubernetes.io/name=keycloak -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  if [[ -z "${pod}" ]]; then
    echo "Skipping Keycloak client update because keycloak pod was not found" >&2
    return
  fi

  local chat_origin="${PUBLIC_SCHEME}://chat.${BASE_DOMAIN}:${EXTERNAL_PORT}"
  local admin_origin="${PUBLIC_SCHEME}://admin.${BASE_DOMAIN}:${EXTERNAL_PORT}"
  local chat_redirect="${chat_origin}/*"
  local admin_redirect="${admin_origin}/*"

  kubectl --context "${KUBECONFIG_CONTEXT}" -n uap exec "${pod}" -- bash -lc "
set -euo pipefail
/opt/keycloak/bin/kcadm.sh config credentials --server http://localhost:8080 --realm master --user '${KEYCLOAK_ADMIN_USER}' --password '${KEYCLOAK_ADMIN_PASSWORD}' >/dev/null
chat_client_id=\$(/opt/keycloak/bin/kcadm.sh get clients -r uap -q clientId=uap-chat-web --fields id | grep -o '\"id\" : \"[^\"]*\"' | head -n1 | cut -d'\"' -f4)
admin_client_id=\$(/opt/keycloak/bin/kcadm.sh get clients -r uap -q clientId=uap-admin-web --fields id | grep -o '\"id\" : \"[^\"]*\"' | head -n1 | cut -d'\"' -f4)
/opt/keycloak/bin/kcadm.sh update clients/\${chat_client_id} -r uap -s 'redirectUris=[\"${chat_redirect}\",\"http://localhost:3200/*\"]' -s 'webOrigins=[\"${chat_origin}\",\"http://localhost:3200\"]' >/dev/null
/opt/keycloak/bin/kcadm.sh update clients/\${admin_client_id} -r uap -s 'redirectUris=[\"${admin_redirect}\",\"http://localhost:3300/*\"]' -s 'webOrigins=[\"${admin_origin}\",\"http://localhost:3300\"]' >/dev/null
"
}

wait_rollouts() {
  local workloads=(
    postgres redis kafka qdrant minio clickhouse
    admin-api chat-gateway session-service conversation-service agent-router provider-gateway voice-gateway quota-service audit-service transcript-service
    agent-runtime tool-runner rag-service indexer workflow-workers
    keycloak temporal temporal-ui livekit otel-collector prometheus loki tempo grafana
    admin-web chat-web
  )

  for workload in "${workloads[@]}"; do
    if kubectl --context "${KUBECONFIG_CONTEXT}" -n uap get deployment "${workload}" >/dev/null 2>&1; then
      kubectl --context "${KUBECONFIG_CONTEXT}" -n uap rollout status deployment/"${workload}" --timeout=10m
      continue
    fi
    if kubectl --context "${KUBECONFIG_CONTEXT}" -n uap get statefulset "${workload}" >/dev/null 2>&1; then
      kubectl --context "${KUBECONFIG_CONTEXT}" -n uap rollout status statefulset/"${workload}" --timeout=10m
    fi
  done
}

ensure_cluster
install_istio
install_envoy_gateway
load_images
deploy_platform
patch_gateway_service
wait_rollouts
configure_keycloak_clients
export UAP_BASE_DOMAIN="${BASE_DOMAIN}"
export UAP_EXTERNAL_PORT="${EXTERNAL_PORT}"
"${ROOT_DIR}/scripts/k8s-smoke.sh"

cat <<EOF
Kubernetes platform is ready.

Panels:
  Chat UI:      ${PUBLIC_SCHEME}://chat.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Admin UI:     ${PUBLIC_SCHEME}://admin.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Chat API:     ${PUBLIC_SCHEME}://api.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Admin API:    ${PUBLIC_SCHEME}://admin-api.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Voice API:    ${PUBLIC_SCHEME}://voice.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Grafana:      ${PUBLIC_SCHEME}://grafana.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Prometheus:   ${PUBLIC_SCHEME}://prometheus.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Keycloak:     ${PUBLIC_SCHEME}://keycloak.${BASE_DOMAIN}:${EXTERNAL_PORT}
  Temporal UI:  ${PUBLIC_SCHEME}://temporal.${BASE_DOMAIN}:${EXTERNAL_PORT}
  MinIO:        ${PUBLIC_SCHEME}://minio.${BASE_DOMAIN}:${EXTERNAL_PORT}
  LiveKit WS:   ${LIVEKIT_SCHEME}://livekit.${BASE_DOMAIN}:${LIVEKIT_PORT}
EOF
