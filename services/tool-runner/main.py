from __future__ import annotations

from fastapi import FastAPI
from pydantic import BaseModel

from uap_common import AppConfig, create_app


class ToolRequest(BaseModel):
    tool_name: str
    arguments: dict[str, object]


app: FastAPI = create_app(AppConfig(service_name="tool-runner"))


@app.get("/api/v1/tools")
async def list_tools() -> list[dict[str, str]]:
    return [
        {"name": "echo", "description": "Returns the submitted payload"},
        {"name": "timestamp", "description": "Returns the current UTC timestamp"},
    ]


@app.post("/api/v1/tools/execute")
async def execute_tool(payload: ToolRequest) -> dict[str, object]:
    return {"tool_name": payload.tool_name, "result": payload.arguments}

