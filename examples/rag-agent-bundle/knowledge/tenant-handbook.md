Bundle RAG Agent handbook

This agent answers from its own Qdrant-backed knowledge scope.
The knowledge scope is filtered by tenant_id and agent_id.
The platform indexer splits long documents into chunks and stores them in Qdrant payloads.

Operational facts

The admin API sends indexing requests to the indexer service.
The agent-runtime calls rag-service before provider-gateway generation when rag_enabled is true.
The rag-service performs tenant and agent scoped retrieval and ranks text matches by query token frequency.

Test prompts

Ask:
- Where is the knowledge stored?
- How is retrieval scoped?
- Which service performs retrieval before generation?
