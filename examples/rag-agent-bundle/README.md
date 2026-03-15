# RAG Agent Bundle

This image packages an installable RAG-enabled agent for UniversalAgentPlatform.

## Build

```bash
docker build -t uap-rag-agent-bundle -f examples/rag-agent-bundle/Dockerfile .
```

## Install Into Local Compose Stack

```bash
docker run --rm \
  --network docker-compose_default \
  -e ADMIN_API_URL=http://admin-api:8080 \
  uap-rag-agent-bundle
```

The container creates the agent if missing and indexes the bundled handbook into Qdrant through `admin-api`.
