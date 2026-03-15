from __future__ import annotations

import json
import os
import pathlib
import sys
import urllib.error
import urllib.request


ROOT = pathlib.Path(__file__).parent
ADMIN_API_URL = os.getenv("ADMIN_API_URL", "http://admin-api:8080").rstrip("/")
TENANT_ID = os.getenv("TENANT_ID", "11111111-1111-1111-1111-111111111111")
PROVIDER_MODEL_ID = os.getenv("PROVIDER_MODEL_ID", "").strip()


def request(method: str, path: str, payload: dict | None = None) -> dict | list:
    data = None
    headers = {"Content-Type": "application/json", "X-Tenant-ID": TENANT_ID}
    if payload is not None:
        data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(f"{ADMIN_API_URL}{path}", data=data, headers=headers, method=method)
    with urllib.request.urlopen(req) as response:
        return json.loads(response.read().decode("utf-8"))


def resolve_provider_model_id() -> str:
    if PROVIDER_MODEL_ID:
        return PROVIDER_MODEL_ID
    models = request("GET", "/api/v1/provider-models")
    if not isinstance(models, list):
        raise RuntimeError("provider models response is not a list")
    llm_models = [item for item in models if item.get("capability") == "llm"]
    if not llm_models:
        raise RuntimeError("no enabled llm provider model found; create one in admin first")
    return str(llm_models[0]["provider_model_id"])


def main() -> int:
    manifest = json.loads((ROOT / "manifest.json").read_text(encoding="utf-8"))
    knowledge = (ROOT / "knowledge" / "tenant-handbook.md").read_text(encoding="utf-8")
    manifest["provider_model_id"] = resolve_provider_model_id()

    try:
        response = request("POST", "/api/v1/agents", manifest)
        agent_id = str(response["agent_id"])
        print(f"created agent: {agent_id}")
    except urllib.error.HTTPError as exc:
        body = exc.read().decode("utf-8", errors="replace")
        if exc.code != 500 or "duplicate key" not in body:
            print(body, file=sys.stderr)
            raise
        agents = request("GET", "/api/v1/agents")
        if not isinstance(agents, list):
            raise RuntimeError("failed to locate existing agent after duplicate response")
        current = next((item for item in agents if item.get("slug") == manifest["slug"]), None)
        if current is None:
            raise RuntimeError("agent already exists but cannot be resolved from agent list")
        agent_id = str(current["agent_id"])
        print(f"using existing agent: {agent_id}")

    response = request(
        "POST",
        "/api/v1/knowledge/index",
        {"agent_id": agent_id, "title": "Tenant handbook", "content": knowledge},
    )
    print(json.dumps({"agent_id": agent_id, "knowledge": response}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
