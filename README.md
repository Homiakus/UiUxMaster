# UiUxMaster

UiUxMaster is an evidence-driven UI/UX engineering system for AI coding agents.

The end state is an MCP server that lets an agent **design, render, inspect, critique, compare, repair, and verify** web interfaces using deterministic browser evidence plus optional local vision models.

## Core thesis

```text
CODE IS A HYPOTHESIS
RENDER IS EVIDENCE
INTERACTION IS THE RESULT
```

A frontend task is not complete because CSS looks plausible or a visible test passes. UiUxMaster is built around a closed loop:

```text
Design intent
    ↓
Code change
    ↓
Real browser render
    ↓
Runtime + DOM + accessibility + geometry + screenshot evidence
    ↓
Deterministic checks
    ↓
Hierarchical visual critique
    ↓
Relative comparison against baseline/candidates
    ↓
Targeted repair
    ↓
Independent verification
    ↺
```

## Architecture pillars

1. **Design Intelligence** — editorial hierarchy, typography, composition, color, motion, information density, accessibility and responsive art direction.
2. **Visual Runtime Verification** — Playwright render evidence, accessibility snapshots, computed styles, geometry, pixel diffing, interaction playthroughs and cross-browser checks.
3. **Progressive Visual Attention** — whole page → section → component → element → pixels; expensive VLM calls are used only when cheaper evidence is insufficient.
4. **Independent Verifiers** — deterministic layout/runtime checks, accessibility, visual regression, semantic design critique and interaction checks remain separate signals.
5. **Relative Design Judgement** — prefer before/A/B ranking over poorly calibrated absolute “beauty scores”.
6. **Design Defect Memory** — retain validated reusable design lessons, not raw one-off model opinions.
7. **MCP Surface** — stable tools/resources for coding agents, with the core kept independent from the protocol adapter.

## Initial repository layout

```text
cmd/uiuxmaster-mcp/     MCP server entry point
internal/design/        design rubric and critique domain
internal/evidence/      normalized visual/runtime evidence model
internal/engine/        audit and decision orchestration
internal/mcpserver/     MCP adapter and tool registration
docs/                   architecture and research-derived principles
MASTER_PLAN.md           living execution plan
```

## MCP baseline

UiUxMaster targets the official Tier-1 Go SDK and MCP specification `2026-07-28`. The protocol adapter is intentionally thin so the design engine can also be used from CLI, tests, CI and other hosts.

## Status

Early foundation. The first milestone establishes the domain model, evidence contracts, deterministic critique engine and MCP skeleton before browser/VLM adapters are added.

See [MASTER_PLAN.md](MASTER_PLAN.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
