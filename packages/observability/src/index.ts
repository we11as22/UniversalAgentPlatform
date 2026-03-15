export interface LogContext {
  traceId?: string;
  tenantId?: string;
  conversationId?: string;
  agentId?: string;
}

export function logInfo(message: string, context: LogContext = {}): void {
  console.info(JSON.stringify({ level: "info", message, ...context }));
}

export function logError(message: string, context: LogContext = {}): void {
  console.error(JSON.stringify({ level: "error", message, ...context }));
}

