#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/versions.env"

export PATH="${HOME}/.local/bin:/usr/local/bin:/usr/bin:${PATH}"

assert_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing command: ${cmd}" >&2
    exit 1
  fi
}

for cmd in go protoc node pnpm uv docker kubectl helm kind curl tar python3; do
  assert_cmd "${cmd}"
done

echo "Toolchain summary"
go version
protoc --version
node --version
pnpm --version
uv --version
docker --version
docker compose version
kubectl version --client
helm version --short
