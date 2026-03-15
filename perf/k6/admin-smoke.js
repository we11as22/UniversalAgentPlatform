import http from "k6/http";
import { check, sleep } from "k6";

const vus = Number(__ENV.VUS || 1);
const duration = __ENV.DURATION || "20s";
const baseUrl = __ENV.ADMIN_BASE_URL || "http://localhost:3210";

export const options = {
  vus,
  duration,
  thresholds: {
    http_req_duration: ["p(95)<1200"],
    checks: ["rate>0.99"]
  }
};

export default function () {
  const dashboard = http.get(`${baseUrl}/api/v1/dashboard`);
  check(dashboard, {
    "admin dashboard healthy": (value) => value.status === 200
  });

  const agents = http.get(`${baseUrl}/api/v1/agents`);
  check(agents, {
    "agents list reachable": (value) => value.status === 200
  });

  const providers = http.get(`${baseUrl}/api/v1/providers`);
  check(providers, {
    "providers list reachable": (value) => value.status === 200
  });

  const models = http.get(`${baseUrl}/api/v1/provider-models`);
  check(models, {
    "provider models reachable": (value) => value.status === 200
  });

  const profiles = http.get(`${baseUrl}/api/v1/perf/profiles`);
  check(profiles, {
    "perf profiles reachable": (value) => value.status === 200
  });

  sleep(1);
}
