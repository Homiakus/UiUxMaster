# MASTER PLAN — UiUxMaster

This is the living execution plan. Do not create a competing roadmap.

## Mission

Build a production-grade system that gives coding agents a closed UI/UX improvement loop:

```text
intent → code → real render → evidence → critique → repair → independent verification → learned design knowledge
```

The final delivery is an MCP server with a stable tool/resource surface, while all core design and verification logic remains protocol-independent.

## Non-negotiable principles

1. **Code is a hypothesis. Render is evidence. Interaction is the result.**
2. **Hierarchy before pixels.** Understand page/section/component semantics before pixel-level repair.
3. **Progressive visual attention.** Whole page → section → component → element → pixels/crop.
4. **Deterministic evidence before expensive VLM inference.** Runtime/DOM/A11y/geometry/pixel diff first; local VLM only for unresolved semantic judgement.
5. **Relative comparison before absolute aesthetic score.** Prefer baseline vs candidate A/B ranking.
6. **Localize before repair.** Findings must point to regions/elements/evidence when possible.
7. **Independent verifiers.** Runtime correctness, accessibility, interaction, visual regression and aesthetic critique must not collapse into one score.
8. **Step-level verification.** Re-render after meaningful groups of changes instead of waiting for a giant final diff.
9. **Anti-reward-hacking.** Visible tests are not the specification. Add hidden/perturbed scenarios and real interaction playthroughs.
10. **History-aware critique.** Carry previous patch, critique, resolved and unresolved defects into the next pass.
11. **Validated design memory.** Store generalized lessons only after replay/evaluation; do not self-modify canonical rules from one VLM opinion.
12. **Local-first privacy.** Raw screenshots/DOM stay local by default. Cloud escalation, if ever enabled, must be explicit, minimized and auditable.
13. **Thin MCP adapter.** The protocol must not own design rules, browser logic or model logic.
14. **Measurement over intuition.** Latency, visual-diff stability and agent success rate need benchmarks/evals.

---

# Target architecture

```text
┌──────────────────────────── Agent / MCP Host ────────────────────────────┐
│                                                                          │
│   uiux.* tools/resources                                                 │
└───────────────────────────────┬──────────────────────────────────────────┘
                                │
                                ▼
                     ┌─────────────────────┐
                     │ MCP Adapter (Go)    │
                     └──────────┬──────────┘
                                │
                ┌───────────────┴────────────────┐
                ▼                                ▼
      ┌────────────────────┐          ┌────────────────────┐
      │ Design Engine      │          │ Evidence Store     │
      │ / Orchestrator     │          │ / Run History      │
      └─────────┬──────────┘          └────────────────────┘
                │
       ┌────────┼───────────┬───────────────┬──────────────────┐
       ▼        ▼           ▼               ▼                  ▼
 Deterministic  Pixel      Browser        Visual             Design
 verifiers      diff       runtime        critic adapters    memory
                           worker          (local-first)
       │                    │
       └────────────┬───────┘
                    ▼
              Playwright sidecar
```

## Core packages

- `internal/design` — canonical rubric, design vocabulary, principles, relative judgement contracts.
- `internal/evidence` — normalized evidence packet independent of Playwright/VLM vendors.
- `internal/engine` — progressive validation/orchestration and next-step decisions.
- `internal/mcpserver` — MCP tool/resource adapter only.

## Planned adapters

- `internal/runtime/playwright` — Go-side client/protocol for the Playwright worker.
- `worker/playwright` — Node/TypeScript Playwright worker: screenshots, ARIA snapshots, DOM geometry, computed styles, interactions, traces.
- `internal/visualdiff` — pixel/region diff analysis and DOM-region correlation.
- `internal/critic` — critic interface + deterministic/relative/hierarchical critic orchestration.
- `internal/vlm` — local VLM providers; Ollama/OpenAI-compatible local endpoint first, vendor-neutral interface.
- `internal/memory` — candidate lessons, validated heuristics and replay evidence.
- `internal/eval` — adversarial visual evals, mutation tests and benchmark corpus.

---

# MCP tool surface — target

The first release should converge toward these tool families.

## Discovery / intent

- `uiux_get_rubric`
- `uiux_analyze_brief`
- `uiux_plan_validation`

## Runtime evidence

- `uiux_capture`
- `uiux_inspect_layout`
- `uiux_inspect_accessibility`
- `uiux_run_scenario`

## Visual comparison

- `uiux_compare_baseline`
- `uiux_compare_candidates`
- `uiux_localize_visual_change`

## Semantic critique

- `uiux_critique_page`
- `uiux_critique_region`
- `uiux_rank_candidates`

## Synthesis

- `uiux_evaluate_evidence` (already scaffolded)
- `uiux_recommend_repairs`
- `uiux_verify_completion`

## Memory / evaluation

- `uiux_record_lesson`
- `uiux_replay_lesson`
- `uiux_run_design_eval`

Do not expose every internal helper as an MCP tool. Tool count and schemas are part of agent UX.

---

# Phase 0 — Foundation & invariants

**Status: IN PROGRESS**

### Goals

- [x] Initialize Go module.
- [x] Depend on official MCP Go SDK line supporting MCP `2026-07-28`.
- [x] Define canonical design rubric.
- [x] Define normalized evidence packet.
- [x] Create deterministic evidence synthesis engine.
- [x] Expose first MCP tools over stdio.
- [ ] Add unit tests for rubric/evidence synthesis.
- [ ] Add CI (`go test`, `go vet`, staticcheck if pinned).
- [ ] Add tool schema contract tests.
- [ ] Add ADR format and first architecture decisions.

### Exit gate

`go test ./...` and `go vet ./...` pass; core packages do not import Playwright/VLM/MCP implementation details except the adapter package.

---

# Phase 1 — Design Intelligence Core

### Goal

Turn the design prompts/research into machine-usable structured knowledge rather than one giant prompt.

### Work

- [ ] Convert premium editorial principles into versioned rubric/rule objects.
- [ ] Add rule categories: identity, composition, typography, color, imagery, interaction, responsive, accessibility, motion, micro-craft.
- [ ] Model `Finding`, `Evidence`, `RepairHypothesis`, `CritiquePass`, `CandidateComparison`.
- [ ] Add hierarchy model: page → section → component → element.
- [ ] Add relative pairwise comparison contract and confidence.
- [ ] Separate hard constraints from preferences.
- [ ] Add “generic-template smell” heuristics without making them universal style rules.
- [ ] Support product-mode profiles: marketing/editorial, SaaS dashboard, CMS, docs/editor, data-heavy professional UI.
- [ ] Preserve original master-prompt material under `knowledge/` with traceability to structured rules.

### Tests

- rule IDs unique/stable;
- no contradictory hard rules within one profile;
- serialization round-trip;
- representative design fixtures map to expected axes.

---

# Phase 2 — Playwright Evidence Worker

### Goal

Get trustworthy real-browser evidence cheaply and deterministically.

### Worker output

- screenshot/full-page and region screenshots;
- `ariaSnapshot()` / accessibility representation;
- stable semantic element refs;
- bounding boxes;
- selected `getComputedStyle` properties;
- page errors and console errors;
- failed requests;
- font readiness;
- page-level horizontal overflow;
- focus order / active element where requested;
- optional Playwright trace.

### Determinism controls

- pinned browser versions;
- viewport/DPR;
- locale/timezone;
- light/dark theme;
- clock/time control;
- disable/freeze animations and caret for snapshot tests;
- explicit app-ready signal;
- font readiness;
- dynamic-region masks.

### Exit gate

Same fixture produces stable evidence across repeated runs within defined tolerance.

---

# Phase 3 — Deterministic Layout & Accessibility Verifiers

### Checks

- page horizontal overflow;
- clipped important controls/text;
- overlapping actionable elements;
- elements outside viewport/container;
- fixed/sticky obstruction;
- target size;
- contrast where computable;
- missing accessible names;
- focus traps / obscured focus;
- duplicate IDs;
- unexpected hidden/zero-size controls;
- text truncation anomalies;
- responsive breakpoint/container failures.

### Principle

These checks should generate grounded findings **without VLM**.

---

# Phase 4 — Visual Regression & Region Localization

### Baseline path

Use Playwright-native screenshot assertions for ordinary regression because they already handle screenshot stabilization and pixel comparison.

### Low-level diff path

Use direct pixel diff only when UiUxMaster needs:

- diff masks;
- region clustering;
- changed-pixel density;
- candidate region extraction;
- DOM-bounding-box correlation.

### Required output

Not just:

`1500 pixels changed`

but:

```text
region hero/actions
→ intersects element refs button:publish, nav:primary
→ 18px overlap
→ diff density 0.23
```

---

# Phase 5 — Progressive Local Visual Critic

### Architecture

```text
whole page thumbnail
  ↓ uncertainty / suspicious region
section crop
  ↓
component crop + element metadata
  ↓
optional stronger local model
```

### Provider strategy

- local-first provider interface;
- Ollama/OpenAI-compatible local endpoint initially;
- fast edge VLM profile first;
- stronger model only on uncertainty;
- no mandatory cloud dependency.

### Critic input

- user/design intent;
- current screenshot/crop;
- hierarchy path;
- element refs and geometry;
- relevant computed style summary;
- previous critique and applied patch;
- baseline/candidate images for relative judgement.

### Critic output

Must be structured and grounded: axis, severity, confidence, region, elements, evidence and direction. Free-form prose is secondary.

---

# Phase 6 — Relative Design Search

### Goal

Move from “is it beautiful?” to comparative optimization.

### Loop

1. Capture baseline.
2. Produce candidate A/B when the design decision is ambiguous and cost is justified.
3. Compare candidates per rubric axis.
4. Reject candidates violating hard correctness/accessibility constraints.
5. Merge or select the stronger direction.
6. Re-render and independently verify.

Absolute aggregate score must never be the only completion gate.

---

# Phase 7 — Interaction Playthrough Verification

Static screenshots cannot prove interactive UX.

Add scenarios for:

- navigation;
- dropdown/menu;
- modal/dialog;
- forms and validation;
- loading/progress/error states;
- hover/focus/touch alternatives;
- responsive resize during a flow;
- theme switching;
- keyboard-only flow.

Each step may capture targeted evidence.

---

# Phase 8 — Cross-browser / Perturbation Verification

### Fast loop

Current Chromium + current viewport.

### Milestone loop

Representative responsive matrix.

### Release loop

Browser × viewport × theme × selected perturbations.

### Perturbations / hidden checks

- arbitrary widths between named breakpoints;
- long German/Russian titles;
- missing/slow image;
- delayed font;
- 200% zoom;
- reduced motion;
- high-saturation accent;
- empty/huge datasets;
- slow network/error state.

Purpose: prevent “building to the visible test”.

---

# Phase 9 — Design Defect Memory

### Memory levels

```text
Observation
→ Candidate heuristic
→ Validated heuristic
→ Invariant
```

Never promote a one-run VLM opinion directly into canonical design rules.

### Promotion gate

- evidence from multiple unrelated pages;
- replay succeeds;
- no regression on counterexamples;
- provenance retained.

---

# Phase 10 — Adversarial Design Evals

Build a benchmark corpus by deliberately injecting defects:

- shift CTA;
- reduce contrast;
- break heading scale;
- equalize all spacing and flatten hierarchy;
- clip images;
- break focus outline;
- introduce horizontal overflow;
- collapse button targets;
- mismatch dark theme;
- create card soup / excessive chrome.

Measure:

- defect detection recall;
- false positives;
- localization accuracy;
- axis classification;
- repair success;
- regression rate;
- latency/cost per validation level.

---

# Phase 11 — MCP Productization

### Requirements

- stable JSON Schema contracts;
- bounded tool outputs;
- deterministic ordering;
- explicit artifact/resource references instead of huge payloads;
- stdio for local agents;
- stateless HTTP transport later;
- cacheable catalogs where supported;
- OpenTelemetry for remote/structured observability rather than deprecated MCP logging primitives;
- tasks extension only for genuinely long-running operations when client support justifies it.

### MCP resources (planned)

Expose large artifacts as resources rather than stuffing them into tool output:

- screenshots;
- diff images;
- traces;
- full evidence packets;
- critique reports;
- design memory entries.

---

# Phase 12 — Safety, privacy, legal/provenance

Engineering invariants:

- local-first processing of screenshots/DOM;
- purpose limitation and data minimization;
- redact secrets/tokens/PII before optional external-model calls;
- explicit retention policy for screenshots/traces;
- audit external model calls;
- only interact with targets the operator is authorized to test;
- track licenses/provenance for model weights and dependencies;
- use reference designs for abstract principles, not unauthorized reproduction of protected text/assets/distinctive compositions.

---

# Definition of Done for a UI polish run

A run cannot be called complete only because a screenshot matches a baseline or a VLM says “looks good”. Completion requires, as applicable:

- no blocking runtime/layout/accessibility findings;
- intended interactions pass;
- responsive target set passes;
- current result is not worse than baseline on protected axes;
- visual critic findings above the configured severity threshold are resolved or explicitly accepted;
- no unexplained visual regression;
- evidence packet is retained for the run;
- hidden/perturbed verification passes at milestone/release gates.

---

# Immediate next execution slice

1. Finish tests + CI for Phase 0.
2. Add browser-worker protocol contract.
3. Implement Playwright worker with `capture`, `aria`, geometry and runtime error collection.
4. Add `uiux_capture` MCP tool backed by the worker.
5. Add deterministic overflow/overlap checks.
6. Add fixture web app and golden evidence tests.
7. Only then add local VLM integration.
