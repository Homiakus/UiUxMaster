# ADR-0001 — Runtime tiers and ownership boundaries

- Status: Accepted
- Date: 2026-09-03

## Context

UiUxMaster targets an edit-to-evidence loop that is usually sub-second and, for warm deterministic local checks, can reach tens-of-milliseconds latency. The system also needs long-running workflow orchestration, epistemic memory, bounded LLM state, research acquisition and high-fidelity cross-browser verification.

Putting all of those capabilities into every validation request would destroy the latency target and blur sources of truth.

A previous planning iteration also over-attributed frontend impact analysis to AutoTraceLab. AutoTraceLab is primarily a block/process-diagram system; UiUxMaster needs frontend-specific source/component/token/runtime-region semantics.

## Decision

### 1. UiUxMaster owns frontend impact semantics

`internal/impact` is a native UiUxMaster subsystem.

It owns the graph and resolver for relationships such as:

```text
source/module/style/token
→ component/variant/instance
→ route/story/page
→ semantic runtime ref
→ render region
→ viewport/theme/scenario
```

External graph algorithms may be compared later, but no external product owns this domain model.

### 2. Runtime verification is tiered

```text
L0 Static/source impact analysis
L1 FastRender — WGGo/go-webengine candidate
L2 FastBrowser — resident Chromium/headless-shell over direct CDP
L3 TruthPath — Playwright clean-state/cross-browser verification
L4 Semantic critique — local-first VLM only when deterministic evidence is insufficient
```

The caller requests evidence intent/fidelity, not a renderer vendor.

### 3. Hot path and control/memory/research planes stay separate

The common hot path is:

```text
change
→ impact
→ fidelity routing
→ L1/L2 evidence
→ deterministic verifier
→ evidence packet
```

Axiom, SncSinCore, SkillState, DeepSearch and full Playwright are not mandatory per-operation dependencies.

- Axiom owns selected multi-step workflow execution/history and retry policy.
- SncSinCore owns admitted long-term epistemic knowledge/evidence.
- SkillState owns bounded typed working projection for LLM reasoning and controlled evolution gates.
- DeepSearch owns optional research/acquisition.
- MCP remains a thin protocol adapter.

### 4. Fast renderer is not truth by default

WGGo or any approximate renderer must expose capability/fidelity metadata and be calibrated against L2/L3 evidence. Unsupported/high-risk feature classes automatically escalate.

### 5. Conservative impact fallback is mandatory

Unknown source ownership, unresolved dynamic imports, stale runtime refs and unsupported framework relationships must expand validation scope rather than silently omit affected UI.

## Consequences

### Positive

- millisecond-scale data-plane work stays in-process and bounded;
- frontend impact logic evolves with UiUxMaster requirements;
- workflow/memory/research libraries can be added without owning renderer state;
- renderer implementations remain replaceable;
- false-negative impact bugs become explicit test targets;
- AutoTraceLab can still contribute isolated algorithms if benchmarks justify reuse.

### Costs

- UiUxMaster must implement and maintain a small native impact kernel;
- multiple evidence tiers require calibration fixtures;
- adapters and evidence provenance add some structural complexity.

These costs are accepted because they directly protect latency, correctness and replaceability.

## Rejected alternatives

### Playwright as the only runtime

Rejected for the hot loop because clean navigation/context orchestration is too expensive for every small edit.

### AutoTraceLab as the impact engine

Rejected because its product/domain semantics are block/process diagrams rather than frontend source/runtime impact analysis.

### Axiom around every renderer primitive

Rejected because durable workflow dispatch around DOM/layout/pixel primitives would add needless latency and coupling.

### SncSinCore as operational state store

Rejected because epistemic memory and current renderer/workflow state have different truth and lifecycle semantics.

## Verification

This ADR is enforced through:

- package ownership tests/architecture review;
- impact false-negative fixtures;
- p50/p95/p99 runtime benchmarks;
- WGGo/CDP/TruthPath parity corpus;
- race tests;
- before/after integration evals.
