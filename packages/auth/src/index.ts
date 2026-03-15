export interface TenantClaim {
  tenantId: string;
  roles: string[];
  groups: string[];
}

export function parseTenantClaim(input: unknown): TenantClaim | null {
  if (!input || typeof input !== "object") {
    return null;
  }

  const raw = input as Record<string, unknown>;
  const tenantId = typeof raw.tenant_id === "string" ? raw.tenant_id : null;
  if (!tenantId) {
    return null;
  }

  return {
    tenantId,
    roles: Array.isArray(raw.roles) ? raw.roles.filter((value): value is string => typeof value === "string") : [],
    groups: Array.isArray(raw.groups) ? raw.groups.filter((value): value is string => typeof value === "string") : []
  };
}

export function hasRole(claim: TenantClaim | null, role: string): boolean {
  return Boolean(claim?.roles.includes(role));
}

