from __future__ import annotations

import asyncio
import contextlib
import json
import logging
import os
import statistics
import subprocess
import tempfile
import time
import uuid
from pathlib import Path
from typing import Any

import asyncpg
import httpx
from fastapi import FastAPI

from uap_common import AppConfig, create_app


ROOT_DIR = Path(__file__).resolve().parents[2]
PERF_DIR = ROOT_DIR / "perf"
DEFAULT_DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://uap:uap@localhost:5432/uap")
CHAT_GATEWAY_URL = os.getenv("CHAT_GATEWAY_URL", "http://chat-gateway:8080")
ADMIN_API_URL = os.getenv("ADMIN_API_URL", "http://admin-api:8080")
VOICE_GATEWAY_URL = os.getenv("VOICE_GATEWAY_URL", "http://voice-gateway:8080")
POLL_INTERVAL_SECONDS = float(os.getenv("PERF_POLL_INTERVAL_SECONDS", "5"))
PERF_TIMEOUT_GRACE_SECONDS = int(os.getenv("PERF_TIMEOUT_GRACE_SECONDS", "90"))
PERF_STALE_RUN_TIMEOUT_SECONDS = int(os.getenv("PERF_STALE_RUN_TIMEOUT_SECONDS", "7200"))
DEFAULT_TENANT_ID = os.getenv("PERF_TENANT_ID", "11111111-1111-1111-1111-111111111111")
DEFAULT_USER_ID = os.getenv("PERF_USER_ID", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1")
DEFAULT_TEXT_AGENT_ID = os.getenv("PERF_TEXT_AGENT_ID", "71000000-0000-0000-0000-000000000001")
DEFAULT_VOICE_AGENT_ID = os.getenv("PERF_VOICE_AGENT_ID", "71000000-0000-0000-0000-000000000002")
DEFAULT_REALTIME_AGENT_ID = os.getenv("PERF_REALTIME_AGENT_ID", "71000000-0000-0000-0000-000000000003")

app: FastAPI = create_app(AppConfig(service_name="workflow-workers"))
logger = logging.getLogger("workflow-workers")


@app.on_event("startup")
async def startup() -> None:
    app.state.database = await asyncpg.create_pool(DEFAULT_DATABASE_URL, min_size=1, max_size=4)
    await fail_stale_runs(app.state.database)
    app.state.perf_task = asyncio.create_task(poll_perf_runs(app))


@app.on_event("shutdown")
async def shutdown() -> None:
    perf_task = getattr(app.state, "perf_task", None)
    if perf_task is not None:
        perf_task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await perf_task
    database = getattr(app.state, "database", None)
    if database is not None:
        await database.close()


@app.get("/api/v1/workflows")
async def list_workflows() -> list[dict[str, str]]:
    return [
        {"name": "document-indexing", "state": "ready"},
        {"name": "perf-orchestration", "state": "running"},
    ]


@app.get("/api/v1/perf/status")
async def perf_status() -> dict[str, str]:
    return {"status": "polling", "database_url": DEFAULT_DATABASE_URL.split("@")[-1]}


async def poll_perf_runs(fastapi_app: FastAPI) -> None:
    while True:
        try:
            claimed = await claim_run(fastapi_app.state.database)
            if claimed is None:
                await asyncio.sleep(POLL_INTERVAL_SECONDS)
                continue
            logger.info("claimed perf run %s", claimed["perf_run_id"])
            await execute_perf_run(fastapi_app.state.database, claimed)
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            logger.exception("perf polling loop failed: %s", exc)
            await asyncio.sleep(POLL_INTERVAL_SECONDS)


async def claim_run(pool: asyncpg.Pool) -> asyncpg.Record | None:
    async with pool.acquire() as connection:
        async with connection.transaction():
            return await connection.fetchrow(
                """
                with next_run as (
                  select pr.perf_run_id,
                         pr.tenant_id,
                         pr.target_environment,
                         pr.git_sha,
                         pr.build_version,
                         pp.name as profile_name,
                         pp.profile_type,
                         pp.config
                    from perf.perf_runs pr
                    join perf.perf_profiles pp on pp.perf_profile_id = pr.perf_profile_id
                   where pr.status = 'queued'
                   order by pr.started_at asc
                   limit 1
                   for update skip locked
                )
                update perf.perf_runs pr
                   set status = 'running',
                       started_at = now(),
                       metadata = coalesce(pr.metadata, '{}'::jsonb) || jsonb_build_object('runner', 'workflow-workers')
                  from next_run nr
                 where pr.perf_run_id = nr.perf_run_id
                returning nr.perf_run_id,
                          nr.tenant_id,
                          nr.target_environment,
                          nr.git_sha,
                          nr.build_version,
                          nr.profile_name,
                          nr.profile_type,
                          nr.config
                """
            )


async def execute_perf_run(pool: asyncpg.Pool, run: asyncpg.Record) -> None:
    tenant_id = str(run["tenant_id"])
    config = normalize_config(run["config"])
    metrics: list[dict[str, Any]] = []
    metadata: dict[str, Any] = {
        "profile_name": run["profile_name"],
        "profile_type": run["profile_type"],
        "target_environment": run["target_environment"],
    }

    try:
        timeout_seconds = profile_timeout_seconds(config)
        logger.info("starting perf run %s with timeout=%ss", run["perf_run_id"], timeout_seconds)
        metrics.extend(await asyncio.wait_for(run_perf_suites(config), timeout=timeout_seconds))
        await persist_results(pool, str(run["perf_run_id"]), tenant_id, metrics)
        await complete_run(pool, str(run["perf_run_id"]), status="completed", metadata=metadata)
        logger.info("completed perf run %s with %d metrics", run["perf_run_id"], len(metrics))
    except asyncio.TimeoutError:
        metadata["error"] = f"perf run exceeded timeout of {profile_timeout_seconds(config)}s"
        await complete_run(pool, str(run["perf_run_id"]), status="failed", metadata=metadata)
        logger.error("timed out perf run %s", run["perf_run_id"])
    except Exception as exc:
        metadata["error"] = str(exc)
        await complete_run(pool, str(run["perf_run_id"]), status="failed", metadata=metadata)
        logger.exception("perf run %s failed: %s", run["perf_run_id"], exc)


async def run_perf_suites(config: dict[str, Any]) -> list[dict[str, Any]]:
    metrics: list[dict[str, Any]] = []
    metrics.extend(await run_k6_suite(config))
    metrics.extend(await run_voice_suite(config))
    return metrics


async def run_k6_suite(config: dict[str, Any]) -> list[dict[str, Any]]:
    vus = str(config.get("vus", 2))
    duration = str(config.get("duration", "30s"))
    metrics: list[dict[str, Any]] = []
    scripts = [
        ("chat", PERF_DIR / "k6" / "chat-smoke.js", {"BASE_URL": CHAT_GATEWAY_URL}),
        ("chat_ws", PERF_DIR / "k6" / "chat-websocket.js", {"BASE_URL": CHAT_GATEWAY_URL}),
        ("admin", PERF_DIR / "k6" / "admin-smoke.js", {"ADMIN_BASE_URL": ADMIN_API_URL}),
    ]

    for suite, script, extra_env in scripts:
        if not script.exists():
            continue
        with tempfile.NamedTemporaryFile(suffix=".json", delete=False) as temp_file:
            summary_path = temp_file.name
        env = os.environ | extra_env | {"VUS": vus, "DURATION": duration}
        command = ["k6", "run", str(script), "--summary-export", summary_path]
        completed = await asyncio.to_thread(subprocess.run, command, env=env, capture_output=True, text=True)
        if completed.returncode != 0:
            raise RuntimeError(f"k6 {suite} failed: {completed.stderr.strip()}")
        with open(summary_path, "r", encoding="utf-8") as handle:
            summary = json.load(handle)
        metrics.extend(extract_k6_metrics(summary, suite))
        os.unlink(summary_path)

    return metrics


def extract_k6_metrics(summary: dict[str, Any], suite: str) -> list[dict[str, Any]]:
    metrics = summary.get("metrics", {})
    extracted: list[dict[str, Any]] = []
    http_duration = metric_values(metrics.get("http_req_duration", {}))
    http_failed = metric_values(metrics.get("http_req_failed", {}))
    checks = metric_values(metrics.get("checks", {}))
    iterations = metric_values(metrics.get("iterations", {}))
    ws_session = metric_values(metrics.get("ws_session_duration", {}))

    if "p(95)" in http_duration:
        extracted.append(metric(f"{suite}.http_req_duration_p95", http_duration["p(95)"], "ms", {"suite": suite}))
    if "avg" in http_duration:
        extracted.append(metric(f"{suite}.http_req_duration_avg", http_duration["avg"], "ms", {"suite": suite}))
    if "rate" in http_failed:
        extracted.append(metric(f"{suite}.http_req_failed_rate", http_failed["rate"], "ratio", {"suite": suite}))
    elif "value" in http_failed:
        extracted.append(metric(f"{suite}.http_req_failed_rate", http_failed["value"], "ratio", {"suite": suite}))
    if "rate" in checks:
        extracted.append(metric(f"{suite}.checks_rate", checks["rate"], "ratio", {"suite": suite}))
    elif "value" in checks:
        extracted.append(metric(f"{suite}.checks_rate", checks["value"], "ratio", {"suite": suite}))
    if "rate" in iterations:
        extracted.append(metric(f"{suite}.iterations_rate", iterations["rate"], "ops_per_s", {"suite": suite}))
    if "avg" in ws_session:
        extracted.append(metric(f"{suite}.ws_session_duration_avg", ws_session["avg"], "ms", {"suite": suite}))
    if "p(95)" in ws_session:
        extracted.append(metric(f"{suite}.ws_session_duration_p95", ws_session["p(95)"], "ms", {"suite": suite}))
    return extracted


async def run_voice_suite(config: dict[str, Any]) -> list[dict[str, Any]]:
    concurrency = int(config.get("voice_concurrency", 2))
    latencies: list[float] = []
    transcription_latencies: list[float] = []
    synthesis_latencies: list[float] = []
    invocation_latencies: list[float] = []
    successes = 0
    voice_agent_id = str(config.get("voice_agent_id", DEFAULT_VOICE_AGENT_ID))
    realtime_agent_id = str(config.get("realtime_agent_id", DEFAULT_REALTIME_AGENT_ID))

    async def exercise(index: int) -> None:
        nonlocal successes
        async with httpx.AsyncClient(timeout=15.0) as client:
            create_conversation = await client.post(
                f"{CHAT_GATEWAY_URL}/api/v1/conversations",
                json={
                    "user_id": DEFAULT_USER_ID,
                    "agent_id": realtime_agent_id if index % 2 else voice_agent_id,
                    "title": f"perf voice chat {index}",
                },
            )
            create_conversation.raise_for_status()
            conversation_id = create_conversation.json()["conversation_id"]

            started = time.perf_counter()
            session = await client.post(
                f"{VOICE_GATEWAY_URL}/api/v1/voice/sessions",
                json={
                    "tenant_id": DEFAULT_TENANT_ID,
                    "user_id": DEFAULT_USER_ID,
                    "conversation_id": conversation_id,
                    "agent_id": realtime_agent_id if index % 2 else voice_agent_id,
                },
            )
            session.raise_for_status()
            latencies.append((time.perf_counter() - started) * 1000)

            voice_session_id = session.json()["voice_session_id"]
            transcript_started = time.perf_counter()
            transcribe = await client.post(
                f"{VOICE_GATEWAY_URL}/api/v1/voice/transcribe",
                json={"voice_session_id": voice_session_id, "text_hint": f"perf voice input {index}"},
            )
            transcribe.raise_for_status()
            transcription_latencies.append((time.perf_counter() - transcript_started) * 1000)

            invocation_started = time.perf_counter()
            invoke = await client.post(
                f"{CHAT_GATEWAY_URL}/api/v1/agents/{realtime_agent_id if index % 2 else voice_agent_id}/respond-from-voice",
                json={
                    "tenant_id": DEFAULT_TENANT_ID,
                    "user_id": DEFAULT_USER_ID,
                    "text_hint": f"perf voice invoke {index}",
                    "speak_response": True,
                },
            )
            invoke.raise_for_status()
            invocation_latencies.append((time.perf_counter() - invocation_started) * 1000)

            synthesis_started = time.perf_counter()
            synthesize = await client.post(
                f"{VOICE_GATEWAY_URL}/api/v1/voice/synthesize-inline",
                json={
                    "tenant_id": DEFAULT_TENANT_ID,
                    "agent_id": realtime_agent_id if index % 2 else voice_agent_id,
                    "text": f"perf synthesized output {index}",
                },
            )
            synthesize.raise_for_status()
            synthesis_latencies.append((time.perf_counter() - synthesis_started) * 1000)
            successes += 1

    await asyncio.gather(*(exercise(index) for index in range(concurrency)))

    metrics = [
        metric("voice.session_setup_avg", statistics.fmean(latencies) if latencies else 0, "ms", {"concurrency": concurrency}),
        metric("voice.session_setup_p95", percentile(latencies, 95), "ms", {"concurrency": concurrency}),
        metric("voice.transcript_avg", statistics.fmean(transcription_latencies) if transcription_latencies else 0, "ms", {"concurrency": concurrency}),
        metric("voice.transcript_p95", percentile(transcription_latencies, 95), "ms", {"concurrency": concurrency}),
        metric("voice.tts_avg", statistics.fmean(synthesis_latencies) if synthesis_latencies else 0, "ms", {"concurrency": concurrency}),
        metric("voice.tts_p95", percentile(synthesis_latencies, 95), "ms", {"concurrency": concurrency}),
        metric("voice.invoke_from_voice_avg", statistics.fmean(invocation_latencies) if invocation_latencies else 0, "ms", {"concurrency": concurrency}),
        metric("voice.invoke_from_voice_p95", percentile(invocation_latencies, 95), "ms", {"concurrency": concurrency}),
        metric("voice.success_rate", successes / concurrency if concurrency else 1, "ratio", {"concurrency": concurrency}),
    ]
    return metrics


def percentile(values: list[float], percentile_value: int) -> float:
    if not values:
        return 0
    ordered = sorted(values)
    index = max(0, min(len(ordered) - 1, round((percentile_value / 100) * (len(ordered) - 1))))
    return ordered[index]


def metric(name: str, value: float, unit: str, metadata: dict[str, Any]) -> dict[str, Any]:
    return {"metric_name": name, "metric_value": float(value), "unit": unit, "metadata": metadata}


def normalize_config(raw: Any) -> dict[str, Any]:
    if raw is None:
        return {}
    if isinstance(raw, dict):
        return raw
    if isinstance(raw, str):
        return json.loads(raw)
    return dict(raw)


def metric_values(raw: Any) -> dict[str, Any]:
    if isinstance(raw, dict) and "values" in raw and isinstance(raw["values"], dict):
        return raw["values"]
    if isinstance(raw, dict):
        return raw
    return {}


def profile_timeout_seconds(config: dict[str, Any]) -> int:
    duration = str(config.get("duration", "30s"))
    return max(60, parse_duration_seconds(duration) * 2 + PERF_TIMEOUT_GRACE_SECONDS)


def parse_duration_seconds(raw: str) -> int:
    raw = raw.strip().lower()
    if raw.endswith("ms"):
        return max(1, int(float(raw[:-2]) / 1000))
    if raw.endswith("s"):
        return int(float(raw[:-1]))
    if raw.endswith("m"):
        return int(float(raw[:-1]) * 60)
    if raw.endswith("h"):
        return int(float(raw[:-1]) * 3600)
    return int(float(raw))


async def persist_results(pool: asyncpg.Pool, perf_run_id: str, tenant_id: str, metrics: list[dict[str, Any]]) -> None:
    async with pool.acquire() as connection:
        async with connection.transaction():
            await connection.execute("delete from perf.perf_run_results where perf_run_id = $1", uuid.UUID(perf_run_id))
            for item in metrics:
                await connection.execute(
                    """
                    insert into perf.perf_run_results (perf_run_result_id, tenant_id, perf_run_id, metric_name, metric_value, unit, metadata, created_at)
                    values ($1, $2::uuid, $3::uuid, $4, $5, $6, $7::jsonb, now())
                    """,
                    uuid.uuid4(),
                    tenant_id,
                    perf_run_id,
                    item["metric_name"],
                    item["metric_value"],
                    item["unit"],
                    json.dumps(item["metadata"]),
                )


async def complete_run(pool: asyncpg.Pool, perf_run_id: str, status: str, metadata: dict[str, Any]) -> None:
    async with pool.acquire() as connection:
        await connection.execute(
            """
            update perf.perf_runs
               set status = $2,
                   completed_at = now(),
                   metadata = coalesce(metadata, '{}'::jsonb) || $3::jsonb
             where perf_run_id = $1::uuid
            """,
            perf_run_id,
            status,
            json.dumps(metadata),
        )


async def fail_stale_runs(pool: asyncpg.Pool) -> None:
    async with pool.acquire() as connection:
        await connection.execute(
            """
            update perf.perf_runs
               set status = 'failed',
                   completed_at = now(),
                   metadata = coalesce(metadata, '{}'::jsonb) || jsonb_build_object('error', 'marked failed after stale running timeout')
             where status = 'running'
               and started_at < now() - make_interval(secs => $1::int)
            """,
            PERF_STALE_RUN_TIMEOUT_SECONDS,
        )
