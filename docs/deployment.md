# Deployment

## Canonical Bootstrap

The canonical deployment flow after clone is:

```bash
git clone <repo>
cd UniversalAgentPlatform
make up-k8s
```

`make up-k8s` performs:

1. tool bootstrap
2. tool verification
3. `kind` cluster creation
4. Istio install
5. Gateway API and Envoy Gateway install
6. image build and `kind load`
7. Helm deployment
8. Keycloak redirect adjustment for the active public domain
9. smoke checks using host resolution on the server itself

## Cloud Server and Browser Testing

This repository is now adapted for the “server in the cloud, tests in the browser” scenario.

### Default behavior

If `UAP_BASE_DOMAIN` is not set, the bootstrap script attempts to discover the server public IP and generates:

```text
<public-ip>.sslip.io
```

Then the platform becomes reachable via:

- `http://chat.<public-ip>.sslip.io:8088`
- `http://admin.<public-ip>.sslip.io:8088`
- `http://grafana.<public-ip>.sslip.io:8088`
- and the rest of the control surfaces

This removes the need for manual DNS when opening the platform from a browser on another machine.

### Override with your own DNS

If you already have a DNS name pointed at the server:

```bash
export UAP_BASE_DOMAIN=agents.example.internal
make up-k8s
```

Then the platform uses:

- `http://chat.agents.example.internal:8088`
- `http://admin.agents.example.internal:8088`
- `http://grafana.agents.example.internal:8088`

## Kubernetes Endpoints

The exact URLs are printed by `make up-k8s`. The standard surface is:

- Chat UI: `http://chat.<base-domain>:8088`
- Admin UI: `http://admin.<base-domain>:8088`
- Chat API: `http://api.<base-domain>:8088`
- Admin API: `http://admin-api.<base-domain>:8088`
- Voice API: `http://voice.<base-domain>:8088`
- Grafana: `http://grafana.<base-domain>:8088`
- Prometheus: `http://prometheus.<base-domain>:8088`
- Keycloak: `http://keycloak.<base-domain>:8088`
- Temporal UI: `http://temporal.<base-domain>:8088`
- MinIO Console: `http://minio.<base-domain>:8088`
- LiveKit: `ws://livekit.<base-domain>:17880`

## Helm Structure

- chart root: [infra/helm/platform](/root/asudakov/projects/UniversalAgentPlatform/infra/helm/platform)
- base values: [values.yaml](/root/asudakov/projects/UniversalAgentPlatform/infra/helm/platform/values.yaml)
- kind values: [values-kind.yaml](/root/asudakov/projects/UniversalAgentPlatform/infra/helm/platform/values-kind.yaml)
- prod values: [values-prod.yaml](/root/asudakov/projects/UniversalAgentPlatform/infra/helm/platform/values-prod.yaml)
- GPU values: [values-gpu.yaml](/root/asudakov/projects/UniversalAgentPlatform/infra/helm/platform/values-gpu.yaml)

## Production Notes

### What `make up-k8s` is for

It is the one-command bootstrap and validation path. It is intentionally opinionated around `kind` and fixed host ports so the platform is reachable immediately after clone.

### What real production should use

For long-lived environments, use:

- the Helm chart directly
- your own storage classes
- your own ingress/load balancer strategy
- your own DNS and certificates
- production secrets management
- production node pools, especially for Triton/GPU

### Production values already included

`values-prod.yaml` now includes:

- multi-replica defaults for stateless services
- HPA config for user-facing and latency-sensitive workloads
- PodDisruptionBudget config for stateless workloads
- persistent volumes for stateful dependencies

## Optional Edge Proxy Layer

Envoy Gateway remains the canonical Kubernetes ingress. Nginx and Caddy are now included as optional outer-edge integrations for cloud-server scenarios.

Available assets:

- Caddy: [Caddyfile](/root/asudakov/projects/UniversalAgentPlatform/infra/caddy/Caddyfile)
- Nginx: [default.conf.template](/root/asudakov/projects/UniversalAgentPlatform/infra/nginx/default.conf.template)
- Caddy compose: [compose.edge-caddy.yml](/root/asudakov/projects/UniversalAgentPlatform/infra/docker-compose/compose.edge-caddy.yml)
- Nginx compose: [compose.edge-nginx.yml](/root/asudakov/projects/UniversalAgentPlatform/infra/docker-compose/compose.edge-nginx.yml)

Use them when you want a simple host-level front proxy on top of the exposed platform ports. Do not treat them as a replacement for Gateway API inside the cluster.

For the Docker/local path they route individual hostnames to the correct local services:

- `chat.<base-domain>` -> `localhost:3200`
- `admin.<base-domain>` -> `localhost:3300`
- `api.<base-domain>` -> `localhost:3220`
- `admin-api.<base-domain>` -> `localhost:3210`
- `grafana.<base-domain>` -> `localhost:13000`
- `prometheus.<base-domain>` -> `localhost:19090`
- `keycloak.<base-domain>` -> `localhost:18081`
- `temporal.<base-domain>` -> `localhost:18088`
- `minio.<base-domain>` -> `localhost:19001`
- `livekit.<base-domain>` -> `localhost:17882`

## GitOps

- Argo CD application: [platform-application.yaml](/root/asudakov/projects/UniversalAgentPlatform/infra/argocd/platform-application.yaml)
- chart remains the source of truth

## Operational Commands

```bash
make up-k8s
make smoke-k8s
make down-k8s
make up-edge-caddy
make up-edge-nginx
make down-edge
```
