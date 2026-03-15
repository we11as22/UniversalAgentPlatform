#!/usr/bin/env bash
set -euo pipefail

export PATH="${HOME}/.local/bin:/usr/local/bin:/usr/bin:${PATH}"

CLUSTER_NAME="${CLUSTER_NAME:-uap}"

if kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  kind delete cluster --name "${CLUSTER_NAME}"
fi
