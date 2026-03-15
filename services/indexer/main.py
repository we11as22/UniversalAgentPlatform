from __future__ import annotations

import os
import uuid

import httpx
from fastapi import FastAPI
from pydantic import BaseModel

from uap_common import AppConfig, create_app


class IndexRequest(BaseModel):
    tenant_id: str
    agent_id: str
    document_id: str
    title: str = "Knowledge document"
    chunks: list[str]


app: FastAPI = create_app(AppConfig(service_name="indexer"))


@app.post("/api/v1/index")
async def index_document(payload: IndexRequest) -> dict[str, object]:
    qdrant_url = os.getenv("QDRANT_URL", "http://localhost:16334")
    collection_name = f"tenant_{payload.tenant_id.replace('-', '')}"

    async with httpx.AsyncClient(timeout=20) as client:
        await client.put(
            f"{qdrant_url}/collections/{collection_name}",
            json={"vectors": {"size": 1, "distance": "Cosine"}},
        )
        points = [
            {
                "id": str(uuid.uuid4()),
                "vector": [0.0],
                "payload": {
                    "tenant_id": payload.tenant_id,
                    "agent_id": payload.agent_id,
                    "document_id": payload.document_id,
                    "chunk_no": index + 1,
                    "title": payload.title,
                    "text": chunk,
                },
            }
            for index, chunk in enumerate(payload.chunks)
        ]
        await client.put(
            f"{qdrant_url}/collections/{collection_name}/points?wait=true",
            json={"points": points},
        )

    return {"document_id": payload.document_id, "indexed_chunks": len(payload.chunks), "status": "ready", "collection_name": collection_name}
