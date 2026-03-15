# Performance Testing

## Perf Planes

The platform has two performance planes:

- `perf/k6` for REST, WebSocket, SSE, and high-rate control/data-path testing
- `perf/voice-bots` for concurrent LiveKit/WebRTC and voice-flow testing

The admin launch path is now backed by `workflow-workers`, which claims queued `perf_runs`, executes real API and voice traffic, and persists result metrics into `perf.perf_run_results`.

## Supported Profiles

- `smoke`
- `validation-short`
- `load`
- `stress`
- `spike`
- `soak`
- `failure-injection-lite`

## Operator Workflow

1. Open Admin UI.
2. Go to `Perf`.
3. Launch a profile.
4. Open `Load Testing Results` in Grafana.
5. Compare latency and error posture with the previous run.

What now actually happens after launch:

- queued run is claimed by `workflow-workers`
- k6 `chat`, `chat_ws`, and `admin` suites are executed
- concurrent voice session and transcript traffic is executed
- metrics are persisted into Postgres
- run is marked `completed` or `failed`
- profile-based execution timeout is enforced
- stale `running` runs are failed on worker startup after the configured timeout window

For the seeded `smoke` profile, expect roughly 60-100 seconds end-to-end because both chat and admin k6 suites run with their own graceful shutdown windows before the voice suite is executed.

For a fast integrity check after deploy or before demos, use `validation-short`. It exercises the same persistence and orchestration path but with a 5-second k6 duration and one voice session.

This validation path is now expected to persist metrics for:

- `chat.*`
- `chat_ws.*`
- `admin.*`
- `voice.*`

## What to Watch

- request rate
- p50/p95/p99 latency
- WebSocket session duration and completion rate
- SSE stream start latency
- queue lag
- DB latency
- cache posture
- provider error rate
- voice session setup latency
- first audio byte latency
- interruption recovery latency

## Local and Kubernetes Paths

### Local

Use Docker Compose for quick functional validation.

Direct validation examples:

```bash
k6 run perf/k6/chat-smoke.js -e BASE_URL=http://localhost:3220 -e VUS=1 -e DURATION=5s
k6 run perf/k6/chat-websocket.js -e BASE_URL=http://localhost:3220 -e VUS=1 -e DURATION=5s
```

### Kubernetes

Use the Admin UI or run the perf tooling inside the cluster for realistic routing, mesh, and ingress behavior.
