## 2026-03-15

The platform delivery was hardened across four operational areas.

First, external agent consumption now has a complete invoke matrix for text, voice, and realtime voice agents. `chat-gateway` accepts synchronous invocation, SSE streaming over both `GET` and `POST`, and non-streaming voice-input invocation for every agent modality. Text agents can now be driven from voice input without requiring a realtime voice session.

Second, local edge routing is now a real host-based front proxy instead of a placeholder that assumed a Kubernetes ingress port. Caddy and Nginx route `chat`, `admin`, `api`, `admin-api`, `grafana`, `prometheus`, `keycloak`, `temporal`, `minio`, and `livekit` hostnames to the correct local platform surfaces, so browser testing on a cloud server works without rewriting links.

Third, voice bootstrap was made resilient for pre-chat flows. `voice-gateway` can create a voice session without a pre-existing conversation, resolves user identity more safely, and creates the backing conversation record automatically when needed.

Fourth, the perf subsystem was hardened into an honest end-to-end validation path. `workflow-workers` now normalizes JSON profile config correctly, enforces profile-derived execution timeouts, fails stale running runs on startup, persists k6 plus voice metrics together, and supports a short `validation-short` profile for fast integrity checks before longer smoke/load/stress runs.
