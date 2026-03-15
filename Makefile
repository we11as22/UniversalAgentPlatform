SHELL := /bin/bash

.PHONY: bootstrap verify up-local-api up-gpu up-edge-caddy up-edge-nginx down-edge up-k8s down down-k8s smoke smoke-k8s test lint build-rag-agent-bundle install-rag-agent-bundle

bootstrap:
	./scripts/bootstrap-tools.sh
	./scripts/verify-prereqs.sh
	pnpm install
	uv sync --all-packages

verify:
	./scripts/verify-prereqs.sh

up-local-api:
	docker compose -f infra/docker-compose/compose.base.yml -f infra/docker-compose/compose.local-api.yml up -d --build

up-gpu:
	docker compose -f infra/docker-compose/compose.base.yml -f infra/docker-compose/compose.gpu.yml up -d --build

up-edge-caddy:
	docker compose -f infra/docker-compose/compose.edge-caddy.yml up -d

up-edge-nginx:
	docker compose -f infra/docker-compose/compose.edge-nginx.yml up -d

up-k8s:
	./scripts/bootstrap-tools.sh
	./scripts/verify-prereqs.sh
	./scripts/k8s-up.sh

down:
	docker compose -f infra/docker-compose/compose.base.yml -f infra/docker-compose/compose.local-api.yml -f infra/docker-compose/compose.gpu.yml down -v

down-edge:
	docker compose -f infra/docker-compose/compose.edge-caddy.yml -f infra/docker-compose/compose.edge-nginx.yml down -v

down-k8s:
	./scripts/k8s-down.sh

smoke:
	./scripts/smoke.sh

smoke-k8s:
	./scripts/k8s-smoke.sh

build-rag-agent-bundle:
	docker build -t uap-rag-agent-bundle -f examples/rag-agent-bundle/Dockerfile .

install-rag-agent-bundle: build-rag-agent-bundle
	docker run --rm --network docker-compose_default -e ADMIN_API_URL=http://admin-api:8080 uap-rag-agent-bundle

lint:
	pnpm lint
	uv run ruff check .
	go test ./...

test:
	pnpm test
	uv run pytest
	go test ./...
