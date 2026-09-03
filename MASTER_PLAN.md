# MASTER PLAN — UiUxMaster

This is the living execution plan. Do not create a competing roadmap.

## Mission

Build a production-grade system that gives coding agents a closed UI/UX improvement loop:

```text
intent → code → incremental render → evidence → critique → repair → independent verification → learned design knowledge
```

The final delivery is an MCP server with a stable tool/resource surface, while all core design and verification logic remains protocol-independent.

The normal design loop must target **sub-second latency**, and common warm deterministic checks should target the **tens-of-milliseconds range** whenever the changed scope allows it. High-fidelity browser automation remains a verification path, not the default inner loop.

---

# Non-negotiable principles

1. **Code is a hypothesis. Render is evidence. Interaction is the result.**
2. **Hierarchy before pixels.** Understand page → section → component → element semantics before pixel-level repair.
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
13. **Thin MCP adapter.** The protocol must not own design rules, renderer logic, browser logic or model logic.
14. **Measurement over intuition.** Latency, visual-diff stability, fidelity and agent success rate need benchmarks/evals.
15. **Browser launch/navigation are exceptional work.** The common edit loop must reuse warm render state.
16. **Incremental invalidation before global verification.** Validate only the affected component/region unless the change is global.
17. **Fast renderer is not automatically truth.** Approximate/in-process renderers require fidelity-risk routing and periodic calibration against browser TruthPath.
18. **Never encode/decode images unnecessarily.** If a renderer produces RGBA in-process, diff/crop/analysis should operate directly on memory buffers.
19. **Smallest reset wins.** Recover component → page → context → browser, not browser-first.
20. **Latency is part of agent UX.** Every evidence run must expose where time was spent.

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
                ▼
      ┌──────────────────────┐
      │ Validation Router    │
      │ scope + fidelity +   │
      │ confidence + latency │
      └─────────┬────────────┘
                │
    ┌───────────┼───────────────────┬─────────────────────┐
    ▼           ▼                   ▼                     ▼
 L0 Static   L1 FastRender       L2 FastBrowser        L3 TruthPath
 analysis    WGGo/go-webengine   resident Chromium     Playwright
             in-process Go       direct CDP            clean/cross-browser
    │           │                   │                     │
    └───────────┴──────────┬────────┴──────────┬──────────┘
                           ▼                   ▼
                  deterministic          semantic critic
                     verifiers          local VLM first
                           │                   │
                           └─────────┬─────────┘
                                     ▼
                                Design memory
```

## Runtime tiers

### L0 — Static / source-level preflight

Use when the defect can be detected without rendering:

- CSS/token changes;
- unsupported feature detection;
- component ownership mapping;
- dependency/invalidation graph;
- obvious duplicate IDs/invalid values when available from source;
- changed-file → changed-component resolution.

Target: microseconds to low milliseconds.

### L1 — WGGo FastRender

Candidate implementation: WGGo / `go-webengine/engine`-class pure-Go renderer.

Purpose:

- in-process approximate DOM/CSS/layout/paint;
- geometry and overflow analysis;
- simple flex/grid/table/positioning verification;
- direct `image.RGBA` output for pixel/region diff without PNG round-trip;
- speculative rendering while the real browser is still processing HMR;
- rapid region localization before a browser screenshot/VLM call.

WGGo is **not** assumed to be browser-perfect. The system must maintain a feature/fidelity scanner and automatically escalate unsupported or high-risk cases.

### L2 — FastBrowser

Resident `chrome-headless-shell` / Chromium target controlled over direct CDP from Go.

Characteristics:

- browser process launched once;
- warm contexts/pages leased from a bounded pool;
- normal source edit uses HMR and does not navigate;
- direct `DOMSnapshot.captureSnapshot` for flattened DOM/layout/selected computed styles;
- ROI-first `Page.captureScreenshot` rather than full-page capture;
- explicit render/HMR epoch instead of sleeps/network-idle;
- direct CDP driver selected by benchmark: raw CDP baseline vs `chromedp/cdproto` vs Rod.

Purpose: browser-accurate Blink evidence with warm tens-of-milliseconds targets where possible.

### L3 — TruthPath

Playwright against real Chromium/Chrome and selected WebKit/Firefox environments.

Purpose:

- clean-state verification;
- browser isolation;
- high-fidelity screenshot baselines;
- complex scenarios/interactions;
- cross-browser release verification;
- FastRender/FastBrowser calibration.

TruthPath is deliberately not the latency-critical inner loop.

### L4 — Semantic visual critique

Local-first VLM/hierarchical critic.

Only invoke for unresolved questions such as hierarchy, typography quality, crop/composition, generic-template smell, color balance or art direction.

Use page → section → component → element crops, not full-screen VLM calls by default.

---

# Core packages

- `internal/design` — canonical rubric, design vocabulary, principles, relative judgement contracts.
- `internal/evidence` — normalized evidence packet independent of renderer/browser/VLM vendors.
- `internal/engine` — progressive validation/orchestration and next-step decisions.
- `internal/mcpserver` — MCP tool/resource adapter only.
- `internal/invalidation` — changed source/token/module → affected component/region/page graph.
- `internal/fidelity` — feature scanner, renderer capability model, fidelity-risk/confidence routing.
- `internal/runtime/fastrender` — renderer-neutral FastRender interface.
- `internal/runtime/wggo` — WGGo/go-webengine candidate adapter after benchmark acceptance.
- `internal/runtime/fastcdp` — resident Chromium/headless-shell direct-CDP path.
- `internal/runtime/playwright` — TruthPath protocol adapter.
- `internal/visualdiff` — memory/region pixel diff and semantic DOM-region correlation.
- `internal/critic` — deterministic/relative/hierarchical critic orchestration.
- `internal/vlm` — local VLM providers; vendor-neutral interface.
- `internal/memory` — candidate lessons, validated heuristics and replay evidence.
- `internal/eval` — adversarial visual evals, mutation tests and benchmark corpus.

---

# Canonical renderer contracts

The design engine must not depend on WGGo, Chromium, Playwright or any specific vendor.

```text
Render(ctx, RenderRequest) → RenderEvidence
Inspect(ctx, InspectRequest) → StructuralEvidence
CaptureRegion(ctx, RegionRequest) → PixelBuffer / ArtifactRef
RunScenario(ctx, ScenarioRequest) → ScenarioEvidence
Capabilities() → RendererCapabilities
```

`RenderEvidence` should be able to represent:

- hierarchy/semantic refs;
- element geometry;
- selected computed styles;
- runtime issues when available;
- raw RGBA or image artifact reference;
- renderer identity/version;
- fidelity/confidence metadata;
- latency breakdown.

---

# Fidelity / capability routing

Before choosing WGGo FastRender, inspect the relevant source/runtime features.

## Example risk classes

### LOW

- static or lightly dynamic HTML;
- ordinary block/flex/grid/table layout;
- basic typography;
- common positioning;
- standard images/SVG supported by renderer;
- no advanced browser APIs required for the changed region.

Preferred path: L1 FastRender, with periodic L2 calibration.

### MEDIUM

- React/Vue-generated DOM;
- complex nested layout;
- custom fonts;
- SVG-heavy components;
- moderate transforms/animations;
- renderer supports most but not all features.

Preferred path: L1 speculative + L2 FastBrowser verification.

### HIGH

- canvas/WebGL;
- CSS filters/masks/advanced paint features unsupported by FastRender;
- shadow DOM/custom element semantics not correctly modeled;
- browser API-dependent layout/state;
- complex pseudo-elements;
- animation/interaction is part of the intended result;
- known FastRender divergence class.

Preferred path: skip L1 as verifier; use L2 and/or L3.

## Feature scanner

Track at minimum:

- unsupported CSS properties/functions;
- canvas/WebGL;
- SVG features;
- custom elements/shadow DOM;
- complex pseudo selectors/elements;
- browser APIs used by the changed component;
- custom font dependencies;
- transforms/filters/masks;
- dynamic measurement code;
- hydration/runtime requirements.

The scanner produces **fidelity risk**, not a binary supported/unsupported verdict.

---

# Incremental invalidation architecture

The agent already knows what it changed. UiUxMaster must exploit that information.

```text
source diff
   ↓
module/token dependency graph
   ↓
component ownership
   ↓
runtime semantic refs
   ↓
affected layout regions
   ↓
targeted validation
```

Examples:

### Local component CSS

```text
Button.module.css
→ Button
→ hero CTA + pricing CTA + footer CTA
→ validate only those instances
```

### Shared design token

```text
--font-size-h1
→ many components/pages
→ escalate to representative-page set
```

### Layout primitive / reset / typography base

Treat as broad invalidation and skip narrow-pass assumptions.

The invalidation system should be conservative: false-positive scope expansion is preferable to silent missed regressions.

---

# Warm render lifecycle

The common edit loop must not perform browser startup/navigation.

```text
UiUxMaster startup
    ↓
start renderer/browser pools once
    ↓
open/warm representative pages/stories once
    ↓
source change
    ↓
HMR / targeted state update
    ↓
render epoch changes
    ↓
L1/L2 targeted evidence
```

Introduce an explicit app/render synchronization mechanism such as:

```text
window.__UIUX_RENDER_EPOCH__
```

or an adapter-specific equivalent.

Do not use arbitrary sleeps. Do not use generic `networkidle` as the normal HMR readiness condition.

---

# In-memory pixel path

When FastRender returns `image.RGBA` or equivalent:

```text
render
→ RGBA
→ crop/subimage
→ diff/statistics
→ optional VLM encoding only for selected region
```

Avoid:

```text
render → PNG encode → filesystem → PNG decode → diff
```

For L2 browser capture, prefer ROI screenshots; full-page screenshots are milestone/debug artifacts, not the default hot-path signal.

---

# Speculative parallel rendering

WGGo FastRender may run concurrently with browser HMR.

```text
source changed
    ├── L1 FastRender → early geometry/pixel suspicion
    └── L2 browser HMR → browser-truth evidence when required
```

If L1 finds a high-confidence deterministic defect, the agent can begin repair without waiting for L2.

If L1 passes and fidelity risk is low, routing policy may stop early subject to calibration policy.

---

# Latency telemetry

Every run must expose timing fields such as:

```text
invalidation_ms
fast_render_ms
hmr_wait_ms
browser_snapshot_ms
roi_capture_ms
pixel_diff_ms
vlm_ms
synthesis_ms
total_ms
```

Also record renderer/browser cold vs warm state.

Do not report system speedup without p50/p95/p99 distributions and scenario definition.

---

# Benchmark program — mandatory before locking implementations

## Drivers

- raw CDP minimal baseline;
- `chromedp/cdproto`;
- Rod;
- Playwright attached to existing browser;
- Playwright component/gallery/context-reuse modes where applicable.

## Renderers/engines

- WGGo / `go-webengine/engine` candidate;
- `chrome-headless-shell`;
- modern Chromium Headless;
- branded Chrome Headless where useful;
- Playwright Chromium/Firefox/WebKit TruthPath;
- Lightpanda only for structural-only experiments, never pixel truth.

## Fixture classes

1. simple static marketing page;
2. flex/grid-heavy landing page;
3. large dashboard;
4. React/Vue SPA;
5. custom-font page;
6. SVG-heavy page;
7. 100 / 1k / 10k DOM nodes;
8. table/data-dense professional UI;
9. complex interactive component;
10. unsupported-feature fixtures for fidelity scanner.

## Measured operations

- cold engine/process start;
- warm page/story acquire;
- CSS change → ready;
- component JS HMR change → ready;
- layout-only render/snapshot;
- RGBA render without PNG;
- small ROI render/capture;
- viewport capture;
- full-page capture;
- pixel diff;
- click → local state → evidence;
- resize → evidence.

## Metrics

- p50/p95/p99 latency;
- CPU;
- RSS/peak memory;
- allocations where practical;
- protocol bytes/round trips;
- screenshot/image encode cost;
- fidelity against TruthPath;
- false PASS / false FAIL by evidence class.

---

# Initial latency goals — hypotheses to validate

These are engineering targets, not promises.

### L0 static/invalidation

- common changed-file scope analysis: <5 ms target.

### L1 FastRender

- simple layout-only render: low-ms to tens-of-ms target;
- deterministic geometry checks: 1–10 ms target;
- in-memory ROI pixel diff: 1–10 ms target;
- ordinary simple component validation: target <30 ms where feasible.

### L2 resident Chromium

- structural ROI snapshot: 2–20 ms target;
- local deterministic geometry checks: 1–10 ms target;
- small ROI screenshot: 5–30 ms target;
- common warm deterministic validation: target <50 ms where feasible; p95 <100 ms aspiration.

### L3 TruthPath

No artificial low-latency requirement; optimize responsibly but prioritize clean-state/cross-browser correctness.

The sought orders-of-magnitude improvement must come from eliminating cold launch/navigation/full-page capture/VLM from most iterations, not from marketing claims about a single renderer.

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

The caller should request evidence intent/fidelity, not a vendor. The engine selects FastRender/FastBrowser/TruthPath.

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

**Status: MOSTLY COMPLETE**

### Goals

- [x] Initialize Go module.
- [x] Depend on official MCP Go SDK line supporting MCP `2026-07-28`.
- [x] Define canonical design rubric.
- [x] Define normalized evidence packet.
- [x] Create deterministic evidence synthesis engine.
- [x] Expose first MCP tools over stdio.
- [x] Add unit tests for rubric/evidence synthesis.
- [x] Add CI (`go test`, `go vet`).
- [ ] Add staticcheck if pinned/reproducible.
- [ ] Add tool schema contract tests.
- [ ] Add ADR format and architecture decisions.

### Exit gate

`go test ./...` and `go vet ./...` pass; core packages do not import WGGo/Chromium/Playwright/VLM/MCP implementation details except their adapter packages.

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

# Phase 2 — Ultra-Fast Runtime Evidence Stack

This phase supersedes the previous assumption that Playwright is the primary inner-loop worker.

## Phase 2A — Benchmark harness + fidelity scanner

### Goals

- [ ] Build reproducible benchmark fixtures and runner.
- [ ] Implement renderer capability model.
- [ ] Implement source/runtime feature scanner.
- [ ] Define LOW/MEDIUM/HIGH fidelity risk.
- [ ] Measure WGGo vs direct CDP/chromedp/Rod/warm Playwright on all fixture classes.
- [ ] Persist p50/p95/p99 and fidelity metrics in repo artifacts/docs.

### Exit gate

No renderer/driver becomes default without measured evidence.

## Phase 2B — WGGo FastRender adapter

### Goals

- [ ] Add renderer-neutral `internal/runtime/fastrender` contract.
- [ ] Add WGGo/go-webengine candidate adapter if benchmarks justify it.
- [ ] Expose layout/geometry/selected styles.
- [ ] Expose direct RGBA where available.
- [ ] Implement ROI crop/diff without PNG encode/decode.
- [ ] Add FastRender fidelity metadata to evidence packet.
- [ ] Add automatic escalation on unsupported/high-risk features.
- [ ] Add TruthPath parity fixtures.

### Exit gate

FastRender is demonstrably useful for one or more fixture classes, and it never silently claims browser truth outside its calibrated capability envelope.

## Phase 2C — Resident Chromium FastBrowser

### Goals

- [ ] Browser daemon/process pool; no launch per MCP call.
- [ ] `chrome-headless-shell` first candidate; benchmark modern headless Chromium too.
- [ ] Direct-CDP Go adapter selected by benchmark.
- [ ] Warm context/page leasing.
- [ ] HMR/render epoch synchronization.
- [ ] `DOMSnapshot.captureSnapshot` with verifier-specific computed-style whitelist.
- [ ] ROI-first screenshot capture and speed-oriented settings where safe.
- [ ] Smallest-reset recovery ladder.
- [ ] Stale-state detection and bounded pools.

### Exit gate

Normal local source edit requires no navigation and produces browser-accurate targeted evidence in the measured warm latency budget.

## Phase 2D — Incremental invalidation

### Goals

- [ ] changed file/module/token → component ownership graph;
- [ ] component → semantic runtime refs mapping;
- [ ] region-level invalidation;
- [ ] global token/baseline change expansion policy;
- [ ] representative-page selection for broad invalidation;
- [ ] conservative fallback when ownership is uncertain.

### Exit gate

Local changes no longer trigger whole-site verification by default.

## Phase 2E — Playwright TruthPath

### Goal

Get trustworthy clean-state/cross-browser evidence deterministically.

### Responsibilities

- screenshot/full-page and region screenshots;
- ARIA/accessibility representation;
- semantic element refs;
- bounding boxes/selected computed styles when needed;
- page/console errors and failed requests;
- font readiness;
- focus order / active element;
- interaction scenarios;
- optional trace;
- Chromium/Firefox/WebKit release verification.

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

Same fixture produces stable evidence across repeated runs within defined tolerance and calibrates L1/L2 evidence classes.

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

These checks should generate grounded findings **without VLM** and should run on the cheapest evidence tier capable of proving the condition.

---

# Phase 4 — Visual Regression & Region Localization

### Fast path

Prefer in-memory RGBA/ROI diff when FastRender can represent the region faithfully.

### Browser path

Use direct browser ROI capture for Blink-accurate changed regions.

### TruthPath baseline

Use Playwright-native screenshot assertions for protected cross-browser/clean-state regression because they provide stable orchestration and baseline workflows.

### Required output

Not just:

`1500 pixels changed`

but:

```text
region hero/actions
→ intersects element refs button:publish, nav:primary
→ 18px overlap
→ diff density 0.23
→ renderer/fidelity source identified
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
- baseline/candidate images for relative judgement;
- renderer/fidelity provenance.

### Critic output

Structured and grounded: axis, severity, confidence, region, elements, evidence and direction. Free-form prose is secondary.

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

FastBrowser may handle warm/local interactions; TruthPath handles clean-state and cross-browser confidence.

---

# Phase 8 — Cross-browser / Perturbation Verification

### Fast loop

L1/L2 current target + current viewport.

### Milestone loop

Representative responsive matrix through L2 plus selected TruthPath checks.

### Release loop

Browser × viewport × theme × selected perturbations through TruthPath.

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

Also inject renderer-fidelity traps:

- unsupported CSS features;
- custom fonts;
- SVG edge cases;
- dynamic browser API layout;
- canvas/WebGL;
- shadow DOM/custom elements;
- transforms/filter/mask cases.

Measure:

- defect detection recall;
- false positives;
- localization accuracy;
- axis classification;
- repair success;
- regression rate;
- FastRender false PASS/FAIL;
- FastBrowser/TruthPath parity;
- cost and latency per validation level.

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
- OpenTelemetry for structured remote observability;
- tasks extension only for genuinely long-running operations when client support justifies it.

### Routing abstraction

MCP callers should express intent such as:

```text
fast structural check
browser-accurate ROI verification
clean-state screenshot baseline
cross-browser release check
semantic design critique
```

They should not be required to choose WGGo/chromedp/Playwright directly.

### MCP resources

Expose large artifacts as resources rather than stuffing them into tool output:

- screenshots;
- RGBA/diff artifacts when persisted;
- traces;
- full evidence packets;
- critique reports;
- design memory entries;
- benchmark reports.

---

# Phase 12 — Safety, privacy, legal/provenance

Engineering invariants:

- local-first processing of screenshots/DOM;
- purpose limitation and data minimization;
- redact secrets/tokens/PII before optional external-model calls;
- explicit retention policy for screenshots/traces;
- audit external model calls;
- only interact with targets the operator is authorized to test;
- track licenses/provenance for renderer/browser/model weights and dependencies;
- use reference designs for abstract principles, not unauthorized reproduction of protected text/assets/distinctive compositions.

---

# Definition of Done for a UI polish run

A run cannot be called complete only because a screenshot matches a baseline, FastRender passes, or a VLM says “looks good”. Completion requires, as applicable:

- no blocking runtime/layout/accessibility findings;
- intended interactions pass;
- responsive target set passes;
- current result is not worse than baseline on protected axes;
- visual critic findings above the configured severity threshold are resolved or explicitly accepted;
- no unexplained visual regression;
- evidence packet retains renderer/fidelity provenance;
- hidden/perturbed verification passes at milestone/release gates;
- FastRender-only PASS is within a calibrated evidence class or escalated before final acceptance.

---

# Definition of Done for the ultra-fast loop

- reproducible benchmark harness is committed;
- p50/p95/p99 are recorded for each supported evidence tier;
- browser/renderer processes are reused across requests where applicable;
- normal local source edit requires no navigation;
- incremental invalidation limits the inspected scope;
- WGGo FastRender is enabled only for measured/calibrated classes;
- in-memory RGBA path avoids unnecessary PNG round trips;
- resident Chromium provides browser-accurate warm evidence;
- Playwright TruthPath exists for clean-state/cross-browser calibration;
- each evidence result reports latency + renderer/fidelity provenance;
- `go test ./...` and `go vet ./...` remain green.

---

# Immediate next execution slice

1. Add ADR for the four-tier runtime architecture: L0 Static → L1 WGGo FastRender → L2 resident Chromium/CDP → L3 Playwright TruthPath.
2. Build benchmark harness before locking WGGo/chromedp/Rod choices.
3. Add `internal/fidelity` capability/risk model and feature scanner.
4. Add renderer-neutral `internal/runtime/fastrender` interface.
5. Benchmark WGGo/go-webengine on static, grid/flex, dashboard, SPA, SVG and 100/1k/10k-node fixtures.
6. Implement direct RGBA crop/diff path if WGGo benchmark/fidelity is acceptable.
7. Implement resident `chrome-headless-shell` process pool and direct-CDP benchmark path.
8. Add HMR/render-epoch synchronization and eliminate normal-loop `page.goto()/reload()/networkidle` waits.
9. Implement incremental invalidation from changed files/tokens to semantic runtime refs.
10. Route `uiux_capture` / `uiux_inspect_layout` through the cheapest sufficient tier automatically.
11. Build FastRender/FastBrowser/TruthPath parity corpus.
12. Only after deterministic runtime tiers are stable, add local VLM integration.
