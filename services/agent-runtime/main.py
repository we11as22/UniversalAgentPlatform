from __future__ import annotations

import os

import httpx
from fastapi import FastAPI
from pydantic import BaseModel

from uap_common import AppConfig, create_app


class ExecuteRequest(BaseModel):
    tenant_id: str
    agent_id: str
    agent_version_id: str
    message: str
    rag_enabled: bool = False


class ExecuteResponse(BaseModel):
    provider_name: str
    provider_kind: str
    text: str
    retrieval: dict[str, object]


app: FastAPI = create_app(AppConfig(service_name="agent-runtime"))


@app.post("/api/v1/execute")
async def execute(payload: ExecuteRequest) -> ExecuteResponse:
    provider_gateway_url = os.getenv("PROVIDER_GATEWAY_URL", "http://localhost:3260")
    rag_service_url = os.getenv("RAG_SERVICE_URL", "http://localhost:8003")
    retrieval_payload: dict[str, object] = {"matches": [], "latency_ms": 0}

    async with httpx.AsyncClient(timeout=20) as client:
        if payload.rag_enabled:
            retrieval = await client.post(
                f"{rag_service_url}/api/v1/retrieve",
                json={"tenant_id": payload.tenant_id, "agent_id": payload.agent_id, "query": payload.message},
            )
            retrieval.raise_for_status()
            retrieval_payload = retrieval.json()

        provider = await client.post(
            f"{provider_gateway_url}/api/v1/generate",
            json={
                "tenant_id": payload.tenant_id,
                "agent_id": payload.agent_id,
                "agent_version_id": payload.agent_version_id,
                "message": payload.message,
            },
        )

    provider.raise_for_status()
    provider_payload = provider.json()

    citations = retrieval_payload.get("matches", [])
    citation_text = ""
    if citations:
        top = citations[0]
        citation_text = f"RAG context: {top['title']} :: {top['snippet']}"

    final_text = provider_payload["text"]
    if citation_text:
        final_text = f"{final_text}\n\n{citation_text}"

    return ExecuteResponse(
        provider_name=provider_payload["provider_name"],
        provider_kind=provider_payload["provider_kind"],
        text=final_text,
        retrieval=retrieval_payload,
    )
