#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/versions.env"

LOCAL_BIN="${HOME}/.local/bin"
LOCAL_OPT="${HOME}/.local/opt"

mkdir -p "${LOCAL_BIN}" "${LOCAL_OPT}"
export PATH="${LOCAL_BIN}:/usr/local/bin:/usr/bin:${PATH}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1
}

download() {
  local url="$1"
  local output="$2"
  curl -fsSL "${url}" -o "${output}"
}

install_with_go() {
  local pkg="$1"
  GOBIN="${LOCAL_BIN}" go install "${pkg}"
}

install_go() {
  local archive="/tmp/go${GO_VERSION}.linux-amd64.tar.gz"
  download "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" "${archive}"
  rm -rf "${LOCAL_OPT}/go"
  tar -C "${LOCAL_OPT}" -xzf "${archive}"
  ln -sf "${LOCAL_OPT}/go/bin/go" "${LOCAL_BIN}/go"
  ln -sf "${LOCAL_OPT}/go/bin/gofmt" "${LOCAL_BIN}/gofmt"
}

install_node() {
  local archive="/tmp/node-v${NODE_VERSION}-linux-x64.tar.xz"
  local install_dir="${LOCAL_OPT}/node-v${NODE_VERSION}"
  download "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" "${archive}"
  rm -rf "${install_dir}"
  mkdir -p "${install_dir}"
  tar -xJf "${archive}" -C "${install_dir}" --strip-components=1
  ln -sf "${install_dir}/bin/node" "${LOCAL_BIN}/node"
  ln -sf "${install_dir}/bin/npm" "${LOCAL_BIN}/npm"
  ln -sf "${install_dir}/bin/npx" "${LOCAL_BIN}/npx"
  ln -sf "${install_dir}/bin/corepack" "${LOCAL_BIN}/corepack"
}

install_kubectl() {
  download "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl" "${LOCAL_BIN}/kubectl"
  chmod +x "${LOCAL_BIN}/kubectl"
}

install_helm() {
  local archive="/tmp/helm-v${HELM_VERSION}-linux-amd64.tar.gz"
  local unpack_dir="/tmp/linux-amd64"
  download "https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz" "${archive}"
  rm -rf "${unpack_dir}"
  tar -xzf "${archive}" -C /tmp
  install -m 0755 "${unpack_dir}/helm" "${LOCAL_BIN}/helm"
}

install_protoc() {
  local archive="/tmp/protoc-${PROTOC_VERSION}.zip"
  local target_dir="${LOCAL_OPT}/protoc-${PROTOC_VERSION}"
  download "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" "${archive}"
  rm -rf "${target_dir}"
  mkdir -p "${target_dir}"
  PROTOC_ARCHIVE="${archive}" PROTOC_TARGET="${target_dir}" python3 - <<'PY'
import os
import zipfile

archive = os.environ["PROTOC_ARCHIVE"]
target = os.environ["PROTOC_TARGET"]

with zipfile.ZipFile(archive) as zf:
    zf.extractall(target)
PY
  ln -sf "${target_dir}/bin/protoc" "${LOCAL_BIN}/protoc"
}

install_uv() {
  curl -LsSf https://astral.sh/uv/install.sh | env UV_INSTALL_DIR="${LOCAL_BIN}" INSTALLER_NO_MODIFY_PATH=1 sh
}

if ! need_cmd go; then
  install_go
fi

if ! need_cmd node; then
  install_node
fi

if ! need_cmd kubectl; then
  install_kubectl
fi

if ! need_cmd helm; then
  install_helm
fi

if ! need_cmd protoc; then
  install_protoc
fi

if ! need_cmd uv; then
  install_uv
fi

if ! need_cmd pnpm; then
  corepack enable >/dev/null 2>&1 || true
  corepack prepare "pnpm@${PNPM_VERSION}" --activate
fi

if ! need_cmd buf; then
  install_with_go "github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"
fi

if ! need_cmd task; then
  install_with_go "github.com/go-task/task/v3/cmd/task@v${TASK_VERSION}"
fi

if ! need_cmd golangci-lint; then
  curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh" | sh -s -- -b "${LOCAL_BIN}" "v${GOLANGCI_LINT_VERSION}"
fi

if ! need_cmd sqlc; then
  install_with_go "github.com/sqlc-dev/sqlc/cmd/sqlc@v${SQLC_VERSION}"
fi

if ! need_cmd mockery; then
  install_with_go "github.com/vektra/mockery/v2@v${MOCKERY_VERSION}"
fi

if ! need_cmd trivy; then
  if ! curl -sfL "https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh" | sh -s -- -b "${LOCAL_BIN}" "v${TRIVY_VERSION}"; then
    echo "Warning: unable to install trivy automatically; continuing without it." >&2
  fi
fi

if ! need_cmd k6; then
  curl -fsSL "https://github.com/grafana/k6/releases/download/v${K6_VERSION}/k6-v${K6_VERSION}-linux-amd64.tar.gz" | tar -xz --strip-components=1 -C /tmp "k6-v${K6_VERSION}-linux-amd64/k6"
  install -m 0755 /tmp/k6 "${LOCAL_BIN}/k6"
fi

if ! need_cmd kind; then
  curl -Lo "${LOCAL_BIN}/kind" "https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-linux-amd64"
  chmod +x "${LOCAL_BIN}/kind"
fi

if ! need_cmd k3d; then
  if ! curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | TAG="v${K3D_VERSION}" PATH="${LOCAL_BIN}:${PATH}" USE_SUDO=false bash; then
    echo "Warning: unable to install k3d automatically; continuing without it." >&2
  fi
fi

if ! need_cmd argocd; then
  if curl -sSL -o "${LOCAL_BIN}/argocd" "https://github.com/argoproj/argo-cd/releases/download/v${ARGOCD_VERSION}/argocd-linux-amd64"; then
    chmod +x "${LOCAL_BIN}/argocd"
  else
    echo "Warning: unable to install argocd CLI automatically; continuing without it." >&2
  fi
fi

if ! need_cmd shadcn; then
  pnpm dlx "shadcn@${SHADCN_CLI_VERSION}" --help >/dev/null 2>&1 || true
fi

if ! uv tool list | grep -q "ruff"; then
  uv tool install ruff
fi

if ! uv tool list | grep -q "mypy"; then
  uv tool install mypy
fi

echo "Bootstrap complete. Ensure ${LOCAL_BIN} is on PATH."
