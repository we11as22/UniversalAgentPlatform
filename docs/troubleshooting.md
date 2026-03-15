# Troubleshooting

## `make up-k8s` finishes but panels do not open

1. run `make smoke-k8s`
2. check `kubectl -n uap get pods`
3. check `kubectl -n uap get httproutes`
4. verify the printed domain resolves from your browser machine
5. if needed, set `UAP_BASE_DOMAIN` explicitly and re-run

## Browser on another machine cannot reach the server

Use the printed `sslip.io` or your own DNS name. The platform is meant to be accessed with:

- `chat.<base-domain>:8088`
- `admin.<base-domain>:8088`

If your browser machine cannot resolve the hostname, set `UAP_BASE_DOMAIN` to a DNS name you control.

## Keycloak login redirects incorrectly

The bootstrap script updates client redirect URIs after deploy. If you changed the domain after deploy, rerun:

```bash
make down-k8s
UAP_BASE_DOMAIN=<your-domain> make up-k8s
```

## Agent exists but does not answer with RAG

Check:

1. the agent has `rag_enabled`
2. knowledge was indexed in the admin `Knowledge` tab
3. the chat was created with that agent
4. Qdrant is healthy

## WebSocket text streaming fails but sync invoke works

Check:

1. `GET /api/v1/conversations/{conversation_id}/runs/ws` or `GET /api/v1/agents/{agent_id}/respond/ws` is reachable
2. browser or edge proxy is not stripping `Upgrade` headers
3. fallback to `GET /api/v1/runs/{run_id}/events` still works after `run.started`
4. `make smoke` passes, because it now validates both SSE and WebSocket text streaming

If you front the Docker path with Nginx or Caddy, verify the edge config still routes WebSocket upgrades correctly.

## Grafana opens but dashboards are missing

Check Grafana provisioning config and verify dashboards were mounted from `infra/observability/dashboards`.
