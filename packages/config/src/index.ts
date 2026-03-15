export type RuntimeProfile = "local-api" | "gpu" | "dev" | "staging" | "prod";

export interface AppConfig {
  appName: string;
  env: string;
  profile: RuntimeProfile;
  apiBaseUrl: string;
  keycloakUrl: string;
  keycloakRealm: string;
  livekitUrl: string;
}

export function loadConfig(appName: string): AppConfig {
  return {
    appName,
    env: process.env.UAP_ENV ?? "local",
    profile: (process.env.UAP_PROFILE as RuntimeProfile | undefined) ?? "local-api",
    apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:3200",
    keycloakUrl: process.env.KEYCLOAK_URL ?? "http://localhost:18081",
    keycloakRealm: process.env.KEYCLOAK_REALM ?? "uap",
    livekitUrl: process.env.NEXT_PUBLIC_LIVEKIT_URL ?? "ws://localhost:7880"
  };
}
