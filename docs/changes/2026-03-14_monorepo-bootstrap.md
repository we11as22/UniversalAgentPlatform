# Monorepo Bootstrap and Architecture Baseline

**Date:** 2026-03-14
**Type:** architecture

## What Changed

The repository now has a defined architecture baseline for a production-grade enterprise agent platform. The change establishes the platform planes, mandatory technology choices, core invariants, local profiles, and the initial monorepo operating model.

## Why

The project started from an empty repository. Without a strict architecture and bootstrap baseline, later service and infrastructure work would drift into incompatible assumptions.

## What This Replaces

None.

## Watch Out For

All future implementation must preserve the Triton-only self-hosted inference boundary and the immutable `conversation.agent_id` rule.

## Related

- `ARCHITECTURE.md`
- `docs/adr/0002-triton-only-inference.md`

