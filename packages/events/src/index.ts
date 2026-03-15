export const topics = {
  conversationRunCreated: "conversation.run.created",
  conversationRunCompleted: "conversation.run.completed",
  providerHealthUpdated: "provider.health.updated",
  voiceSessionStarted: "voice.session.started",
  perfRunStarted: "perf.run.started",
  perfRunFinished: "perf.run.finished",
  auditEventRecorded: "audit.event.recorded"
} as const;

export interface EventEnvelope<TPayload> {
  event_id: string;
  event_type: string;
  version: string;
  tenant_id: string;
  trace_id: string;
  occurred_at: string;
  payload: TPayload;
}

