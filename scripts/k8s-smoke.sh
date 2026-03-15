#!/usr/bin/env bash
set -euo pipefail

BASE_DOMAIN="${UAP_BASE_DOMAIN:-uap.localtest.me}"
EXTERNAL_PORT="${UAP_EXTERNAL_PORT:-8088}"

check() {
  local host="$1"
  local path="$2"
  local url="http://${host}:${EXTERNAL_PORT}${path}"
  echo "Checking ${url}"
  curl --resolve "${host}:${EXTERNAL_PORT}:127.0.0.1" -fsS "${url}" >/dev/null
}

check "chat.${BASE_DOMAIN}" "/api/health"
check "admin.${BASE_DOMAIN}" "/api/health"
check "grafana.${BASE_DOMAIN}" "/api/health"
check "prometheus.${BASE_DOMAIN}" "/-/healthy"
check "keycloak.${BASE_DOMAIN}" "/health/ready"
check "temporal.${BASE_DOMAIN}" "/"
check "minio.${BASE_DOMAIN}" "/"
