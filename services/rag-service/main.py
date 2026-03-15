from __future__ import annotations

import os
from typing import Any

import httpx
from fastapi import FastAPI
from pydantic import BaseModel

from uap_common import AppConfig, create_app


class RetrieveRequest(BaseModel):
    tenant_id: str
    agent_id: str
    query: str


app: FastAPI = create_app(AppConfig(service_name="rag-service"))


@app.post("/api/v1/retrieve")
async def retrieve(payload: RetrieveRequest) -> dict[str, object]:
    qdrant_url = os.getenv("QDRANT_URL", "http://localhost:16334")
    collection_name = f"tenant_{payload.tenant_id.replace('-', '')}"
    matches: list[dict[str, Any]] = []

    async with httpx.AsyncClient(timeout=20) as client:
        response = await client.post(
            f"{qdrant_url}/collections/{collection_name}/points/scroll",
            json={
                "limit": 200,
                "with_payload": True,
                "with_vector": False,
                "filter": {
                    "must": [
                        {"key": "tenant_id", "match": {"value": payload.tenant_id}},
                        {"key": "agent_id", "match": {"value": payload.agent_id}},
                    ]
                },
            },
        )

    if response.status_code < 300:
        points = response.json().get("result", {}).get("points", [])
        query_tokens = [token for token in payload.query.lower().split() if token]
        ranked = []
        for point in points:
            item = point.get("payload", {})
            text = str(item.get("text", ""))
            title = str(item.get("title", "Knowledge chunk"))
            lowered = text.lower()
            score = sum(lowered.count(token) for token in query_tokens)
            if score > 0:
                ranked.append(
                    {
                        "document_id": item.get("document_id", "unknown"),
                        "score": score,
                        "title": title,
                        "snippet": text[:280],
                    }
                )
        matches = sorted(ranked, key=lambda item: item["score"], reverse=True)[:5]

    return {
        "matches": matches,
        "latency_ms": 12,
    }
