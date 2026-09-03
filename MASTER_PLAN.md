# MASTER PLAN — UiUxMaster

This is the single living execution plan for UiUxMaster. Do not create a competing roadmap.

Detailed subordinate specifications:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — ownership and evidence architecture;
- [`docs/ULTRA_FAST_VISUAL_LOOP.md`](docs/ULTRA_FAST_VISUAL_LOOP.md) — low-latency rendering/verification architecture;
- [`docs/ECOSYSTEM_INTEGRATION_PROGRAM.md`](docs/ECOSYSTEM_INTEGRATION_PROGRAM.md) — Axiom/SncSinCore/SkillState/DeepSearch integration contracts and optional ecosystem reuse gates;
- [`docs/RESEARCH_FOUNDATIONS.md`](docs/RESEARCH_FOUNDATIONS.md) — research → invariant traceability.

---

# 1. Mission

Build a production-grade system that gives coding agents a closed UI/UX engineering loop:

```text
intent
→ code change
→ incremental impact analysis
→ fastest sufficient real render/evidence
→ deterministic verification
→ semantic design critique when needed
→ repair
→ independent verification
→ evidence-backed memory
→ controlled skill evolution
```

The final product is an MCP server with a stable tool/resource surface, while all core design, validation, orchestration, memory and rendering logic remains protocol-independent.

The normal edit/polish loop must target **sub-second latency**. Common warm deterministic checks should target the **tens-of-milliseconds range** whenever the changed scope and fidelity requirements permit it.

High-fidelity browser automation, deep research, durable orchestration and long-term memory must never become mandatory work on every local CSS/layout edit.

---

# 2. Non-negotiable principles

1. **Code is a hypothesis. Render is evidence. Interaction is the result.**
2. **Hierarchy before pixels.** Understand page → section → component → element before pixel repair.
3. **Progressive visual attention.** Whole page → section → component → element → crop/pixels.
4. **Incremental invalidation before global verification.** Validate only the affected scope unless the change is broad or uncertain.
5. **Deterministic evidence before VLM inference.** Runtime/DOM/A11y/geometry/pixel checks precede semantic visual judgement.
6. **Relative comparison before absolute aesthetic score.** Prefer baseline/candidate A/B comparisons by rubric axis.
7. **Localize before repair.** Findings should resolve to region/element/evidence whenever possible.
8. **Independent verifiers.** Correctness, accessibility, interaction, visual regression and aesthetics do not collapse into one score.
9. **Step-level verification.** Verify meaningful groups of edits instead of waiting for a giant final diff.
10. **Anti-reward-hacking.** Visible tests are not the specification; use perturbed/hidden scenarios and real playthroughs.
11. **History-aware refinement.** Previous patch, resolved/unresolved findings and evidence digests accompany the next reasoning step.
12. **Validated memory only.** One VLM opinion never becomes a canonical design rule.
13. **Local-first privacy.** Raw screenshots/DOM stay local by default; optional external-model transmission is minimized and auditable.
14. **Thin MCP adapter.** MCP does not own domain, renderer, browser, memory, orchestration or model logic.
15. **Measurement over intuition.** Latency, fidelity, repair success and regression rate require benchmarks/evals.
16. **Browser launch/navigation are exceptional work.** Warm renderer/browser state is reused.
17. **Fast renderer is not automatically truth.** Approximate renderers require capability/fidelity routing and calibration.
18. **Never encode/decode images unnecessarily.** Operate directly on RGBA/crops when available.
19. **Smallest reset wins.** Recover component → page → context → browser.
20. **Latency is part of agent UX.** Every evidence run exposes a latency breakdown.
21. **Fast data plane != control plane != memory plane != research plane.** Do not merge these responsibilities.
22. **No second scheduler.** SkillState never owns retry/scheduling; Axiom/host does.
23. **Operational state != epistemic memory.** SncSinCore stores admitted knowledge/evidence, not current renderer/workflow state.
24. **Evidence before promotion.** Skill/rule evolution requires replay/shadow/non-regression proof and rollback.
25. **Integration must earn its complexity.** Every external library requires a before/after benchmark or eval showing value.
26. **UiUxMaster owns frontend impact semantics.** Source/component/token/runtime-region impact analysis is a UiUxMaster domain capability, not an AutoTraceLab responsibility.
27. **Reuse algorithms, not accidental product coupling.** External graph implementations may be reused only behind local ports after semantic fit, benchmark and maintenance review.

---

# 3. Target architecture

```text
                          Coding Agent / MCP Host
                                   │
                                   ▼
                           UiUxMaster MCP
                                   │
                                   ▼
                         Design / Policy Engine
                                   │
                  ┌────────────────┴────────────────┐
                  ▼                                 ▼
          Axiom Control Plane                 SkillState Σ
       (selected long workflows)       (bounded LLM working state)
                  │                                 │
                  └──────────────┬──────────────────┘
                                 ▼
                    UiUxMaster Impact Engine
                  source/component/token/runtime
                                 │
                                 ▼
                         Validation Router
                  scope + fidelity + confidence
                         + remaining budget
                                 │
              ┌──────────────────┼───────────────────┐
              ▼                  ▼                   ▼
          L1 FastRender      L2 FastBrowser      L3 TruthPath
          WGGo / Go          resident Chromium  Playwright
          in-process         direct CDP          clean/cross-browser
              │                  │                   │
              └──────────────────┼───────────────────┘
                                 ▼
                           Evidence Packet
                                 │
                 ┌───────────────┼────────────────┐
                 ▼               ▼                ▼
          impact/scope       deterministic    local visual
          correlation       verifiers        critic/VLM
                 │               │                │
                 └───────────────┴───────┬────────┘
                                         ▼
                                  Repair decision
                                         │
                                         ▼
                                     Axiom/host
                                         │
                         evidence/admission/provenance
                                         ▼
                                    SncSinCore
                                         ▲
                                         │
                              DeepSearch (optional)
                                         │
                                research evidence
                                         │
                                         ▼
                              controlled SkillState
                                   evolution gate
```

The hottest path is deliberately short:

```text
change
→ native impact engine
→ fidelity router
→ WGGo or resident CDP
→ deterministic verifier
→ evidence packet
```

Axiom durable workflows, SncSinCore retrieval, SkillState evolution, DeepSearch and full Playwright are invoked only when their capability is required.

---

# 4. Runtime evidence tiers

## L0 — Static/source preflight

Use for:

- changed-file/module/token detection;
- source dependency graph updates;
- unsupported feature detection;
- component ownership mapping;
- obvious static violations;
- validation-scope calculation.

Target: microseconds to low milliseconds.

## L1 — WGGo FastRender

Candidate implementation: WGGo / `go-webengine/engine`-class pure-Go renderer.

Use for calibrated low/medium-risk evidence classes:

- approximate DOM/CSS/layout/paint;
- flex/grid/table/positioning checks;
- geometry/overflow;
- direct `image.RGBA` output;
- in-memory crop/pixel diff;
- speculative render while Chromium HMR completes;
- early region localization.

WGGo is never assumed to be browser-perfect. `internal/fidelity` decides whether it may prove a condition or is speculative only.

## L2 — FastBrowser

Resident `chrome-headless-shell` / Chromium controlled through direct CDP from Go.

Required properties:

- process started once;
- bounded warm context/page pool;
- HMR instead of normal reload/navigation;
- explicit render epoch/readiness;
- `DOMSnapshot.captureSnapshot` with verifier-specific computed-style whitelist;
- ROI-first screenshot capture;
- raw CDP vs `chromedp/cdproto` vs Rod chosen by benchmark;
- smallest-reset recovery;
- stale-state detection.

Purpose: Blink-accurate warm evidence with tens-of-ms targets where feasible.

## L3 — TruthPath

Playwright against real Chromium/Chrome and selected Firefox/WebKit environments.

Use for:

- clean-state verification;
- isolation-sensitive scenarios;
- protected screenshot baselines;
- complex interactions;
- cross-browser milestone/release verification;
- L1/L2 fidelity calibration.

TruthPath is not the latency-critical inner loop.

## L4 — Semantic visual critique

Local-first VLM/hierarchical critic only for unresolved semantic questions:

- hierarchy;
- typography relationships;
- composition/crop;
- color balance;
- generic-template/card-soup smell;
- art direction;
- subtle visual polish.

Prefer targeted page → section → component crops and structural context rather than full 4K screenshots.

---

# 5. Core packages and ownership

```text
internal/design        canonical rubric/rules/profiles
internal/evidence      normalized evidence contracts
internal/engine        validation policy/orchestration decisions
internal/mcpserver     MCP adapter only
internal/impact        native frontend dependency/impact model and resolver
internal/invalidation  ImpactSet → validation-scope policy
internal/fidelity      capability scanner + fidelity risk/router
internal/runtime/fastrender  renderer-neutral L1 contract
internal/runtime/wggo        WGGo adapter
internal/runtime/fastcdp     resident Chromium direct-CDP adapter
internal/runtime/playwright  TruthPath adapter
internal/visualdiff     RGBA/region diff + DOM-region mapping
internal/critic         deterministic/relative/hierarchical critique
internal/vlm            model-neutral local VLM providers
internal/control        Axiom control-plane adapter/workflows
internal/knowledge      SncSinCore memory/admission/retrieval adapter
internal/skillruntime   SkillState bounded projection/evolution adapter
internal/research       DeepSearch optional research adapter
internal/memory         design-memory domain policy
internal/eval           adversarial/replay/benchmark corpus
```

Domain packages depend on local interfaces. External library types do not leak across ownership boundaries.

`internal/impact` is a **native UiUxMaster subsystem**. It must not expose AutoTraceLab types or assume AutoTraceLab process/block semantics.

---

# 6. Canonical renderer contracts

```text
Render(ctx, RenderRequest) → RenderEvidence
Inspect(ctx, InspectRequest) → StructuralEvidence
CaptureRegion(ctx, RegionRequest) → PixelBuffer / ArtifactRef
RunScenario(ctx, ScenarioRequest) → ScenarioEvidence
Capabilities() → RendererCapabilities
```

Evidence records:

- hierarchy/semantic refs;
- geometry;
- selected styles;
- runtime issues when available;
- raw RGBA/artifact ref;
- renderer/version;
- fidelity/confidence;
- latency breakdown;
- provenance/digest.

---

# 7. Fidelity/capability routing

## LOW risk

Examples: static/lightly dynamic HTML, ordinary block/flex/grid/table, basic typography/positioning, supported SVG/images.

Preferred route: L1 FastRender with periodic L2 calibration.

## MEDIUM risk

Examples: React/Vue runtime DOM, complex nested layout, custom fonts, SVG-heavy components, moderate transforms/animations.

Preferred route: L1 speculative + L2 confirmation as policy requires.

## HIGH risk

Examples: canvas/WebGL, unsupported CSS filters/masks/paint features, shadow DOM/custom elements with uncertain support, browser-API-dependent layout, interaction/animation as the result.

Preferred route: L2/L3; L1 may only assist localization.

Feature scanner tracks unsupported CSS/functions, canvas/WebGL, SVG features, shadow DOM/custom elements, pseudo elements/selectors, browser APIs, custom fonts, transforms/filters/masks, dynamic measurements and hydration/runtime dependencies.

---

# 8. UiUxMaster Incremental Impact Engine (P0)

The impact engine is a **UiUxMaster-native frontend analysis subsystem**. Its job is to answer one product-specific question:

> Given this source/design/runtime change, what is the smallest conservative set of UI states and rendered regions that must be revalidated?

It is not a generic process-flow editor and must not inherit process-simulation semantics from AutoTraceLab.

Target graph:

```text
SourceFile
→ Module / StyleSheet / DesignToken
→ Component / Variant
→ ComponentInstance
→ Route / Story / Page
→ SemanticElementRef
→ RenderRegion
→ Viewport / Theme / Scenario
```

## 8.1 Native graph kernel

Implement the minimum graph machinery required by this domain directly under UiUxMaster:

```text
internal/impact/
  model.go
  graph.go
  builder.go
  reverse_index.go
  scc.go
  dirty.go
  query.go
  snapshot.go
  resolver.go
```

Required properties:

- frontend-specific typed node/edge kinds;
- deterministic ordering;
- forward/reverse adjacency;
- SCC + condensation DAG for cyclic module/component dependencies;
- bounded dirty propagation;
- immutable/read-optimized snapshot + cheap delta/update path;
- stable IDs;
- versioned serialization only if persistence proves useful;
- no renderer/browser/React runtime dependency in the graph kernel itself;
- 100/1k/10k/100k-node benchmarks.

Do not create a generic graph framework beyond what UiUxMaster needs.

## 8.2 UiUxMaster port

```go
type ImpactResolver interface {
    ApplyChanges(context.Context, ChangeSet) (ImpactSet, error)
    ResolveComponent(context.Context, string) (ImpactSet, error)
    ResolveToken(context.Context, string) (ImpactSet, error)
}
```

Inputs start pragmatic rather than waiting for a perfect compiler:

1. agent/git changed-file set;
2. JS/TS/template import graph;
3. CSS module/style imports;
4. CSS custom-property definitions/references;
5. explicit component/test/design IDs;
6. route/story registry;
7. runtime semantic refs learned from L1/L2;
8. optional framework adapters for React/Vue/Svelte when measured value justifies them.

## 8.3 Invalidation rules

- component-local CSS → affected instances;
- shared component → all known instances/representative pages;
- local token → consumers;
- global reset/typography/theme token → broad representative-page set;
- SCC/cycle → invalidate SCC as one unit;
- unknown dynamic import/runtime ownership → conservative expansion;
- uncertain ownership → conservative expansion;
- user-declared critical routes may force wider validation regardless of graph locality.

## 8.4 Performance/quality gates

Initial targets to validate:

- 1k-node local impact p95 <1 ms;
- 10k-node p95 <5 ms;
- 100k-node bounded local change p95 <20 ms;
- no full traversal on known leaf edit;
- incremental/full recompute parity;
- false-negative mutation suite passes;
- framework adapters cannot silently reduce conservative coverage.

Exit: local edits no longer cause whole-site verification by default.

## 8.5 AutoTraceLab optional reuse gate — P3/reference only

AutoTraceLab is primarily a system for building, tracing and analysing block/process diagrams. It is **not** the source of truth or mandatory dependency for UiUxMaster frontend impact analysis.

After the native `internal/impact` contracts, fixtures and benchmarks exist, individual algorithms may be reviewed for reuse, for example:

- deterministic SCC implementation;
- condensation DAG construction;
- generic reverse-adjacency helpers;
- dirty/incremental recomputation primitives;
- graph snapshot or deterministic ordering utilities.

Reuse is allowed only if all gates pass:

1. the primitive is genuinely domain-neutral;
2. it can be isolated without importing AutoTraceLab application/process/UI semantics;
3. its API fits UiUxMaster local ports without leaking foreign types;
4. benchmark shows equal or better latency/allocation/maintainability than the native implementation;
5. correctness fixtures and mutation tests remain green;
6. license/provenance is compatible;
7. dependency cost is lower than copying/maintaining a tiny stable primitive where licensing permits.

Otherwise keep the UiUxMaster-native implementation.

---

# 9. Warm render lifecycle

```text
UiUxMaster startup
→ start renderer/browser pools once
→ open/warm representative pages/stories once
→ source change
→ native impact scope
→ HMR/targeted state update
→ render epoch changes
→ targeted L1/L2 evidence
```

Use an explicit app/render synchronization signal such as `window.__UIUX_RENDER_EPOCH__` or adapter equivalent.

Do not use arbitrary sleeps or generic `networkidle` as the normal HMR readiness condition.

---

# 10. In-memory visual path

When L1 returns RGBA:

```text
render → RGBA → crop/subimage → diff/statistics → optional VLM encoding of selected crop
```

Avoid:

```text
render → PNG → filesystem → decode → diff
```

For L2, prefer ROI screenshots. Full-page captures are milestone/debug artifacts rather than hot-path defaults.

WGGo may run speculatively in parallel with Chromium HMR. A high-confidence deterministic L1 failure may start a repair before L2 finishes; L1 PASS is accepted only inside calibrated fidelity classes.

---

# 11. Latency telemetry

Every evidence run records at minimum:

```text
impact_ms
fidelity_scan_ms
fast_render_ms
hmr_wait_ms
browser_snapshot_ms
roi_capture_ms
pixel_diff_ms
memory_query_ms
vlm_ms
synthesis_ms
total_ms
```

Also record cold/warm state and evidence tier.

Never claim system speedup without p50/p95/p99 distributions and a defined scenario.

---

# 12. Benchmark program

## Drivers

- raw CDP baseline;
- `chromedp/cdproto`;
- Rod;
- Playwright attached to existing browser;
- Playwright component/context-reuse modes when applicable.

## Engines

- WGGo/go-webengine candidate;
- `chrome-headless-shell`;
- modern Chromium Headless;
- branded Chrome when useful;
- Playwright Chromium/Firefox/WebKit TruthPath;
- Lightpanda structural-only experiments, never pixel truth.

## Impact-engine implementations

Benchmark the native implementation first. Optional external graph primitives are compared only after the native baseline is stable.

Metrics include:

- full build;
- one-leaf update;
- shared-component update;
- global-token invalidation;
- SCC update;
- 1k/10k/100k node graphs;
- allocations;
- false-negative mutation coverage.

## UI fixtures

1. static marketing page;
2. flex/grid-heavy landing;
3. large dashboard;
4. React/Vue SPA;
5. custom fonts;
6. SVG-heavy UI;
7. 100/1k/10k DOM nodes;
8. data-dense professional UI;
9. complex interactive component;
10. unsupported-feature/fidelity traps.

## Runtime operations

- cold start;
- warm acquire;
- CSS HMR → ready;
- component JS HMR → ready;
- layout snapshot;
- RGBA render without PNG;
- ROI render/capture;
- viewport/full-page capture;
- pixel diff;
- click → evidence;
- resize → evidence.

## Metrics

p50/p95/p99, CPU, RSS, allocations, protocol bytes/round trips, image encode cost, fidelity vs TruthPath, false PASS/FAIL.

---

# 13. Initial latency targets — hypotheses, not promises

- L0/impact common local scope: <5 ms.
- L1 simple component validation: <30 ms where feasible.
- L1 geometry/diff operations: 1–10 ms targets.
- L2 structural ROI snapshot: 2–20 ms target.
- L2 small ROI screenshot: 5–30 ms target.
- L2 common warm deterministic validation: <50 ms where feasible, p95 <100 ms aspiration.
- L3 TruthPath prioritizes clean/cross-browser correctness, not artificial low latency.

Orders-of-magnitude speedup comes from eliminating cold launch/navigation/full-page capture/VLM from most iterations, not from one renderer marketing claim.

---

# 14. Axiom control-plane integration (P0)

Axiom is used for **selected long-running/explainable workflows**, never around each CDP/render primitive.

Current prerequisite: Axiom requires Go 1.26+, so direct integration begins only after UiUxMaster upgrades from Go 1.25 and CI/race/vet remain green.

Use the Axiom declarative `model` frontend for new workflow definitions and typed activities for effects.

## 14.1 Workflow candidates

- `DesignPolishRun`;
- `CandidateComparisonRun`;
- `TruthPathCalibrationRun`;
- `CrossBrowserReleaseRun`;
- `DesignEvalRun`;
- `SkillPromotionRun`.

Do not use Axiom for individual layout/screenshot/diff operations.

## 14.2 DesignPolishRun model

State includes run ID, goal, phase, iteration, impact/evidence digests, open/resolved findings, candidates, budget and status.

Events:

```text
Start
ImpactResolved
EvidenceCollected
FindingRaised
RepairProposed
RepairApplied
CandidateCompared
EscalateFidelity
Verify
Complete
Fail
Cancel
```

Claims/invariants include:

- no complete while blocking findings remain;
- no verified TruthPath state without matching evidence digest;
- iteration/budget bounds;
- inaccessible/broken candidate cannot win aesthetically;
- collector failure != PASS;
- activity IDs/idempotency are stable.

## 14.3 Activities

Typed activities wrap existing local ports:

- request L1/L2 evidence;
- request TruthPath scenario;
- semantic critic;
- host repair application;
- evidence persistence;
- SncSinCore query/admission;
- SkillState projection/evolution evaluation.

Axiom owns retry/timeout/idempotency for these workflow activities where needed.

## 14.4 Explicit budgets

```text
max_iterations
max_total_latency
max_vlm_calls
max_truthpath_runs
max_candidates
max_repair_attempts
```

Remaining budget + uncertainty participates in routing.

## 14.5 Storage

Start in-memory. Enable Pebble/durable execution only when crash-resume has measured product value. Large evidence blobs stay in artifact/evidence storage by digest/URI.

Exit: multi-step polish/eval runs are reproducible/explainable without measurable L0–L2 hot-path regression.

---

# 15. SncSinCore epistemic design memory (P0/P1)

Use SncSinCore for **admitted long-term evidence-backed knowledge**, not operational state or raw conversation history.

Start with embedded `epmemory`; activate segmented `memoryv2` only after corpus/memory/latency thresholds justify it.

## 15.1 Ontology

Node classes:

- DesignFinding;
- DesignRule;
- RepairPattern;
- Counterexample;
- ComponentPattern;
- ProductProfile;
- EvidenceArtifact;
- RenderEnvironment;
- EvaluationResult;
- ResearchSource.

Relations:

```text
evidence_for
refutes
generalizes
observed_on
caused_by
repaired_by
improves_axis
regresses_axis
applicable_to
counterexample_to
derived_from
```

## 15.2 Admission

```text
runtime observation
→ evidence.Packet
→ candidate memory atoms
→ provenance/scope/time validation
→ SncSinCore admission
```

Every admitted fact retains run ID, evidence digest, renderer/fidelity, environment, critic/rule version, scope and outcome.

## 15.3 Retrieval port

```go
type DesignMemory interface {
    Query(context.Context, MemoryQuery, MemoryBudget) (MemoryView, error)
    Propose(context.Context, []MemoryCandidate) error
}
```

Queries are requirement/target driven, e.g. validated repairs + counterexamples for weak dark-SaaS hero typography, rather than generic top-k similarity.

## 15.4 Minimal ContextPack

Semantic critic receives a bounded sufficient view, for example:

- <=5 validated similar cases;
- <=3 counterexamples/refutations;
- applicable invariants;
- unresolved conflicts;
- provenance refs.

## 15.5 Namespace firewall

```text
knowledge/global-design
knowledge/project/<id>
evidence/project/<id>
research/global
skillmeta/<skill-id>
```

Project-private content never leaks into global memory. Conflicts are preserved rather than averaged away.

Exit: held-out design evals measurably improve with memory enabled without context explosion or scope leakage.

---

# 16. SkillState bounded state + controlled evolution (P1)

SkillState owns the bounded typed **working projection** seen by the reasoning model. It is not canonical truth and not a scheduler.

## 16.1 Working state

```text
run_id
goal_summary
current_phase
iteration
active_region_ids
active_finding_ids
resolved_finding_ids
protected_axes
remaining_budget
last_evidence_digest
last_patch_digest
oscillation_flags
```

Large screenshots, DOM, traces and histories are references/digests only.

## 16.2 Typed patches and CAS

Model output mutates working state only through typed patches with expected revision/digest. Stale/rejected patches mutate nothing.

## 16.3 SncSinCore MemoryPort

Reasoning input becomes:

```text
immutable Spec P
+ bounded state Σ
+ latest observation O
+ verified MemoryView on demand
```

No implicit replay of long chat/history.

## 16.4 Oscillation detection

Track compact hypothesis/outcome IDs and detect loops such as:

```text
repair A → breaks B → repair B → restores A → repeat
```

On oscillation, require higher-level redesign rather than another identical local patch.

## 16.5 Evolution pipeline

```text
Observation
→ CandidateHeuristic
→ CandidateSkillVersion
→ ReplayEval
→ ShadowEval
→ NonRegressionGate
→ AuthorizedPromotion
→ rollback-capable ActiveVersion
```

Security/privacy/authorization policy is never self-modifying.

Promotion requires multiple unrelated fixtures, target-axis improvement, correctness/accessibility non-regression, acceptable false positives, measured token/latency cost, counterexamples, provenance and validated rollback.

Exit: 50+ iteration trajectories maintain bounded context and candidate rules cannot become active without reproducible evidence.

---

# 17. DeepSearch research plane (P2, optional)

DeepSearch remains an optional sidecar/provider. UiUxMaster must work fully for local rendering/verification without Python or DeepSearch installed.

Trigger only for:

- new domain/product profile;
- current standards/browser/WCAG questions;
- unfamiliar UI/UX pattern;
- periodic research refresh;
- benchmark/source verification;
- explicit evidence/research request.

## Adapter

```go
type Researcher interface {
    Research(context.Context, ResearchRequest) (ResearchBundle, error)
    InspectSource(context.Context, string) (SourceAssessment, error)
}
```

Local REST or stdio sidecar may implement it.

Admission:

```text
DeepSearch bundle
→ provenance/source validation
→ claim extraction
→ memory candidates
→ SncSinCore gates
→ validated knowledge
```

Research output never mutates active design rules directly.

Use content-addressable caching, explicit budgets/staleness rules, timeout/cancellation and prompt-injection-safe handling of web content.

Exit: research improves knowledge freshness while the local edit loop remains independent of research availability.

---

# 18. Optional ecosystem reuse and operational gates

## 18.1 AutoTraceLab graph primitives — P3/reference only

AutoTraceLab is **not** a runtime dependency or impact-analysis owner for UiUxMaster.

Use it only as a source to review individual domain-neutral graph techniques after native UiUxMaster contracts and tests exist. See §8.5.

Do not import the AutoTraceLab React application, process simulator, block-diagram domain, scheduling semantics or scene model.

## 18.2 IRIS patterns — P2

Align a small compatibility schema around:

```text
Claim / Evidence / Artifact / Provenance / Confidence / Scope
```

Do not import the entire IRIS Studio application. Create a shared compatibility package only when at least two projects need the same stable schema and duplication is a measurable maintenance problem.

## 18.3 RepoArk — P3

No runtime dependency. Consider later for benchmark/release artifact archival, backup and reproducibility snapshots.

Activation gate: losing/reproducing benchmark/release history has become an operational problem.

## 18.4 WebGate — P3

No local-runtime dependency now. Consider only when remote browser/device workers become a committed feature.

Future topology may support authenticated remote workers for Windows Chrome/Linux Chromium/macOS Safari/Android. Require authorization, authenticated transport, bounded retries, idempotency and evidence provenance.

---

# 19. MCP tool surface

## Discovery/intent

- `uiux_get_rubric`
- `uiux_analyze_brief`
- `uiux_plan_validation`

## Runtime evidence

- `uiux_capture`
- `uiux_inspect_layout`
- `uiux_inspect_accessibility`
- `uiux_run_scenario`

Caller requests evidence intent/fidelity, not WGGo/chromedp/Playwright vendor selection.

## Visual comparison

- `uiux_compare_baseline`
- `uiux_compare_candidates`
- `uiux_localize_visual_change`

## Semantic critique

- `uiux_critique_page`
- `uiux_critique_region`
- `uiux_rank_candidates`

## Synthesis

- `uiux_evaluate_evidence` — already scaffolded;
- `uiux_recommend_repairs`;
- `uiux_verify_completion`.

## Memory/evolution/research

Expose only high-value operations, not every internal helper:

- `uiux_record_lesson`;
- `uiux_replay_lesson`;
- `uiux_run_design_eval`;
- optional explicit research operation if host composition cannot directly call DeepSearch.

Large screenshots/diffs/traces/evidence/memory/benchmark artifacts are MCP resources/references, not huge inline payloads.

---

# 20. Execution phases

## Phase 0 — Foundation & invariants

**Status: MOSTLY COMPLETE**

- [x] Go module initialized.
- [x] MCP Go SDK integrated.
- [x] canonical rubric/evidence packet/engine.
- [x] first MCP tools over stdio.
- [x] unit tests.
- [x] `go test` + `go vet` CI.
- [ ] upgrade to Go 1.26+ and requalify.
- [ ] add `go test -race ./...` CI.
- [ ] staticcheck if pinned/reproducible.
- [ ] MCP schema contract tests.
- [ ] ADR format + runtime/ecosystem ADRs.

Exit: all core tests/vet/race green; external vendor types remain behind adapters.

## Phase 1 — Design Intelligence Core

- [ ] convert premium editorial/motion/responsive rules into versioned structured rules;
- [ ] stable rule IDs/categories;
- [ ] `Finding`, `Evidence`, `RepairHypothesis`, `CritiquePass`, `CandidateComparison`;
- [ ] page→section→component→element hierarchy;
- [ ] hard constraints vs preferences;
- [ ] product profiles;
- [ ] original prompt material under `knowledge/` with traceability.

## Phase 2 — Ultra-Fast Runtime

### 2A Benchmark + fidelity scanner

- [ ] benchmark harness;
- [ ] capability model;
- [ ] LOW/MEDIUM/HIGH fidelity risk;
- [ ] WGGo/CDP/Rod/chromedp/warm Playwright measurements;
- [ ] p50/p95/p99 + fidelity artifacts.

### 2B UiUxMaster Incremental Impact Engine

- [ ] define native `ImpactResolver`/`ImpactSet` contracts and fixtures;
- [ ] implement frontend-specific node/edge model;
- [ ] source/import/CSS-token analyzers;
- [ ] reverse index/SCC/dirty propagation;
- [ ] route/story/component ownership graph;
- [ ] runtime semantic-ref binding;
- [ ] conservative fallback for uncertain dynamic relationships;
- [ ] false-negative mutation suite;
- [ ] feed `ImpactSet` to router;
- [ ] benchmark native implementation before considering any external graph primitive.

### 2C WGGo FastRender

- [ ] renderer-neutral interface;
- [ ] WGGo adapter if benchmark justifies;
- [ ] geometry/styles/RGBA;
- [ ] ROI direct diff;
- [ ] fidelity metadata/escalation;
- [ ] parity fixtures.

### 2D Resident Chromium FastBrowser

- [ ] browser daemon/pool;
- [ ] `chrome-headless-shell` candidate;
- [ ] direct-CDP driver benchmark/selection;
- [ ] HMR/render epoch;
- [ ] `DOMSnapshot` + style whitelist;
- [ ] ROI capture;
- [ ] stale-state detection;
- [ ] smallest-reset ladder.

### 2E Playwright TruthPath

- [ ] clean-state screenshots/ARIA/errors/fonts;
- [ ] complex scenarios;
- [ ] browser matrix;
- [ ] deterministic baseline controls;
- [ ] L1/L2 calibration corpus.

Exit: normal local edit does not navigate/relaunch and uses the cheapest sufficient evidence tier.

## Phase 3 — Deterministic verifiers

- overflow;
- clipping;
- overlap;
- offscreen/invalid geometry;
- fixed/sticky obstruction;
- target size;
- computable contrast;
- accessible names;
- focus traps/obstruction;
- duplicate IDs;
- hidden/zero-size controls;
- truncation anomalies;
- responsive failures.

Run on the cheapest evidence tier capable of proving the condition.

## Phase 4 — Visual regression/localization

- in-memory RGBA/ROI diff where faithful;
- browser ROI for Blink truth;
- Playwright baseline for protected clean/cross-browser regression;
- region clustering;
- changed-pixel density;
- DOM box intersection;
- semantic findings instead of raw pixel counts.

## Phase 5 — Axiom control plane

- [ ] Go 1.26 compatibility complete;
- [ ] `DesignPolishRun` model/state/events/claims;
- [ ] typed evidence/critic/repair/memory activities;
- [ ] explicit budgets;
- [ ] cancellation/retry/idempotency;
- [ ] in-memory workflow first;
- [ ] durability only after proven need;
- [ ] explain/history diagnostics.

## Phase 6 — SncSinCore memory

- [ ] ontology/namespaces;
- [ ] evidence→candidate admission mapper;
- [ ] `epmemory` embedded start;
- [ ] bounded ContextPack retrieval;
- [ ] provenance/conflict/retraction/scope tests;
- [ ] memory on/off held-out eval;
- [ ] `memoryv2` only after scale threshold.

## Phase 7 — SkillState bounded reasoning

- [ ] typed UiUx skill state/patch schema;
- [ ] project Axiom/domain state into bounded Σ;
- [ ] externalize large artifacts by digest;
- [ ] SncSinCore MemoryPort;
- [ ] CAS/stale/oscillation gates;
- [ ] remove implicit long-history replay;
- [ ] token/task-success benchmarks.

## Phase 8 — Progressive local visual critic

- page→section→component crops;
- model-neutral local provider;
- edge model first;
- stronger model only on uncertainty;
- structured grounded output;
- relevant SncSinCore memory only when useful;
- renderer/fidelity provenance retained.

## Phase 9 — Relative design search

- baseline;
- candidate A/B when justified;
- per-axis pairwise comparison;
- hard correctness/accessibility constraints;
- select/merge;
- re-render and independently verify.

Absolute aggregate score is never the sole completion gate.

## Phase 10 — Interaction playthrough

Scenarios cover navigation, menus, dialogs, forms, loading/error, hover/focus/touch, resize, theme and keyboard-only flows. FastBrowser handles warm/local flows; TruthPath proves clean/cross-browser flows.

## Phase 11 — Cross-browser/perturbation

Fast loop: current L1/L2 target.

Milestone: representative responsive matrix + selected TruthPath.

Release: browser × viewport × theme × perturbations.

Perturbations include arbitrary widths, long RU/DE text, missing/slow media/font, 200% zoom, reduced motion, extreme accent, empty/large data, slow/error network.

## Phase 12 — Controlled evolution with SkillState

- [ ] collect candidate heuristics only from admitted evidence;
- [ ] immutable candidate skill versions;
- [ ] replay corpus;
- [ ] shadow/current-vs-candidate eval;
- [ ] design improvement + non-regression + latency/token gates;
- [ ] authorized promotion;
- [ ] rollback validation.

## Phase 13 — DeepSearch research plane

- [ ] optional sidecar/feature flag;
- [ ] bounded `Researcher` port;
- [ ] source/provenance admission;
- [ ] DeepSearch→SncSinCore path;
- [ ] cache/staleness policy;
- [ ] optional periodic/manual Axiom research workflow;
- [ ] no dependency of hot loop on research availability.

## Phase 14 — Adversarial evals

Inject controlled defects:

- CTA shift;
- contrast reduction;
- heading hierarchy collapse;
- spacing flattening;
- image clipping/crop damage;
- focus loss;
- horizontal overflow;
- tiny targets;
- dark-theme mismatch;
- card soup/chrome excess;
- WGGo fidelity traps: unsupported CSS, fonts, SVG, browser APIs, canvas/WebGL, shadow DOM, filters/masks;
- impact-engine traps: dynamic import, re-export cycle, shared token, CSS cascade, runtime-only component instance, route alias.

Measure detection recall, false positives, localization, severity, repair success, regression, impact false-negative rate, FastRender false PASS/FAIL, parity, latency/cost.

## Phase 15 — MCP productization

- stable JSON Schemas;
- bounded deterministic tool outputs;
- artifacts as resources;
- stdio local transport;
- stateless HTTP later;
- cacheable catalogs;
- OpenTelemetry;
- tasks only for genuine long operations with client support.

## Phase 16 — Safety/privacy/legal/provenance

- local-first screenshots/DOM;
- purpose limitation/data minimization;
- redact secrets/tokens/PII before optional external model;
- explicit retention;
- audit external calls;
- authorized targets only;
- license/provenance inventory for dependencies/models/renderers;
- reference designs used for abstract principles, not unauthorized reproduction.

## Phase 17 — Optional ecosystem/operational expansion

Only after activation gates:

- inspect isolated AutoTraceLab graph primitives only if native impact benchmarks reveal a real gap;
- RepoArk for benchmark/release artifact archival/mirroring;
- WebGate for authenticated resilient remote browser/device workers.

None is required for the first production UiUxMaster.

---

# 21. Ecosystem implementation sequence E0→E7

This sequence is mandatory because later layers depend on evidence/contracts from earlier layers.

## E0 — Compatibility baseline

1. Freeze benchmark/CI baseline.
2. Upgrade Go to 1.26+.
3. Add race CI.
4. Add dependency/license inventory.
5. Pin Axiom/SncSinCore/SkillState versions/commits when integration starts.
6. ADR ecosystem ownership boundaries.
7. Record that AutoTraceLab is not a required dependency; only isolated graph primitives may later pass an optional reuse gate.

## E1 — Native UiUxMaster Impact Engine

1. Define `ImpactResolver`, `ImpactSet`, node/edge kinds and fixtures.
2. Implement native graph kernel under `internal/impact`.
3. Add source/import/CSS-token analyzers.
4. Add reverse indexes + SCC + dirty propagation.
5. Add route/story/component ownership.
6. Bind runtime semantic refs.
7. Integrate router.
8. Benchmark 1k/10k/100k graphs.
9. Build false-negative mutation suite.
10. Only after a stable native baseline, optionally compare isolated AutoTraceLab graph primitives; adopt none unless they clearly win on the defined gates.

## E2 — Axiom control plane

1. `DesignPolishRun` state/events/claims.
2. Typed activities over existing ports.
3. Keep renderer/data primitives direct.
4. Budgets/cancellation/idempotency.
5. In-memory representative workflows.
6. Durable Pebble only after proven crash-resume need.
7. Explain/history diagnostics.

## E3 — SncSinCore memory

1. Ontology/namespaces.
2. Admission mapper.
3. Embedded `epmemory`.
4. Minimal requirement-driven ContextPacks.
5. Feed semantic critic/repair planning only where useful.
6. Conflict/scope/retraction tests.
7. Memory on/off eval.
8. `memoryv2` only after measured threshold.

## E4 — SkillState bounded state

1. Typed state/patch.
2. Projection from Axiom/domain state.
3. Large artifacts by reference.
4. SncSinCore MemoryPort.
5. CAS/oscillation protections.
6. Replace long-history replay.
7. Token/task-success benchmark.

## E5 — Controlled evolution

1. Candidate heuristics from repeated admitted evidence.
2. Immutable candidate skill versions.
3. Replay corpus.
4. A/B/shadow evals.
5. Non-regression + latency/token gates.
6. Authorized promotion.
7. Validated rollback.

## E6 — DeepSearch adapter

1. Optional feature/sidecar.
2. Bounded research port.
3. Provenance/source admission.
4. SncSinCore ingestion only after gates.
5. Cache/staleness.
6. Periodic/manual research workflow only if useful.

## E7 — Ecosystem hardening

1. Full race suite.
2. Fault injection across activities/memory/render/research.
3. Prove no L0–L2 p95 regression.
4. Dependency-upgrade compatibility suite.
5. Privacy/scope isolation.
6. License/provenance report.
7. End-to-end bare-vs-integrated held-out eval.
8. Re-review every optional external dependency and remove any that does not still justify its complexity.

Full detailed contracts and tests live in [`docs/ECOSYSTEM_INTEGRATION_PROGRAM.md`](docs/ECOSYSTEM_INTEGRATION_PROGRAM.md); that document must preserve the same ownership boundary: the UiUxMaster impact engine is native, while AutoTraceLab is optional/reference-only.

---

# 22. Success metrics

## Impact engine

- % edits avoiding whole-page/site scan;
- impact p50/p95/p99;
- false-negative scope rate;
- average affected regions;
- full vs incremental recompute parity;
- allocations/update;
- dynamic/framework fallback frequency.

## Fast runtime

- p50/p95/p99 by tier;
- warm validation rate;
- cold launches/navigations per edit;
- FastRender/TruthPath parity;
- false PASS/FAIL.

## Axiom

- replay/recovery success;
- duplicate side effects = 0;
- explain/history completeness;
- no measurable hot-path regression.

## SncSinCore

- repair-success uplift held-out;
- context tokens saved;
- scope leakage = 0;
- conflict/retraction correctness.

## SkillState

- bounded context over 50+ iterations;
- stale patch correctness;
- oscillation detection;
- promotion regression rate.

## DeepSearch

- research freshness/coverage;
- provenance completeness;
- ordinary local validation remains available when sidecar is down.

## Whole system

- UI polish success/preference vs baseline;
- regression escape rate;
- VLM calls per solved defect;
- TruthPath escalations per edit;
- tokens/cost per solved design defect;
- time from source patch to actionable evidence.

---

# 23. Anti-overengineering guards

- No Axiom dispatch around each WGGo/CDP primitive.
- No second scheduler inside SkillState.
- No operational state stored as SncSinCore truth.
- No DeepSearch dependency in local hot path.
- No AutoTraceLab runtime dependency for UiUxMaster impact analysis.
- No AutoTraceLab process/block-diagram/React domain imported into `internal/impact`.
- No generic graph framework unless the native frontend impact problem demonstrably requires it.
- No duplicated full history/evidence across Axiom, SncSinCore and SkillState; use digests/references and plane-specific projections.
- No VLM opinion promoted directly into a skill.
- No RepoArk/WebGate runtime dependency before activation gates.
- No renderer becomes default without benchmark/fidelity proof.
- No integration is accepted without before/after eval evidence.

---

# 24. Definition of Done — one UI polish run

A run is not complete merely because a screenshot matches baseline, FastRender passes or a VLM says “looks good”. As applicable:

- no blocking runtime/layout/accessibility findings;
- intended interactions pass;
- responsive target set passes;
- candidate does not regress protected axes;
- visual findings above threshold resolved/accepted;
- no unexplained visual regression;
- evidence retains renderer/fidelity/provenance digest;
- milestone/release perturbations pass;
- FastRender-only PASS is inside calibrated evidence class or escalated;
- workflow/history/evidence references are reconstructable;
- long-term memory admission is separate from completion.

---

# 25. Definition of Done — ultra-fast loop

- reproducible benchmark harness committed;
- p50/p95/p99 recorded by tier;
- renderer/browser processes reused;
- normal source edit does not navigate;
- native UiUxMaster impact engine bounds validation scope;
- impact false-negative mutation suite passes;
- WGGo enabled only for calibrated classes;
- direct RGBA path avoids PNG round trips;
- resident Chromium provides warm Blink truth;
- Playwright TruthPath calibrates clean/cross-browser cases;
- each evidence result reports latency/fidelity/provenance;
- ecosystem integrations do not degrade L0–L2 p95 beyond explicit budget;
- `go test ./...`, `go test -race ./...` and `go vet ./...` green.

---

# 26. Immediate next execution slice

Execute in this order:

1. **E0:** upgrade UiUxMaster to Go 1.26+ and add race CI; record baseline before dependency integration.
2. Add ADR for runtime tiers + ecosystem ownership boundaries, explicitly documenting native ownership of the impact engine.
3. Build benchmark harness before locking WGGo/chromedp/Rod choices.
4. Define native `internal/impact` contracts, node/edge model and frontend-specific fixtures.
5. Implement minimal native graph kernel: forward/reverse indexes, SCC, dirty propagation, deterministic snapshots.
6. Add JS/TS/template import, CSS module and CSS custom-property analyzers.
7. Add route/story/component ownership mapping and conservative fallback policy.
8. Add `internal/fidelity` capability/risk scanner.
9. Add renderer-neutral `internal/runtime/fastrender` interface.
10. Benchmark WGGo on static/grid/dashboard/SPA/SVG/100–10k-node fixtures.
11. Implement direct RGBA crop/diff if WGGo qualifies.
12. Implement resident `chrome-headless-shell` + direct-CDP benchmark path.
13. Add HMR/render epoch and remove hot-loop reload/networkidle.
14. Bind native impact nodes to runtime semantic refs and route validation by `ImpactSet`.
15. Add deterministic overflow/clip/overlap checks.
16. Build impact false-negative mutation tests and benchmark 1k/10k/100k graphs.
17. **Optional only after step 16:** inspect isolated AutoTraceLab SCC/DAG/incremental primitives and compare against the native baseline; do not adopt by default.
18. Only after the data plane is measured stable, integrate **Axiom** `DesignPolishRun` control plane.
19. Define **SncSinCore** design-memory ontology/admission and start embedded `epmemory`.
20. Add **SkillState** bounded reasoning projection + SncSinCore MemoryPort.
21. Build local semantic critic and memory-assisted critique eval.
22. Build controlled skill-evolution replay/shadow gates.
23. Add **DeepSearch** as optional research sidecar with SncSinCore admission.
24. Build FastRender/FastBrowser/TruthPath parity corpus and full adversarial evals.
25. Run bare-vs-integrated end-to-end benchmark; keep each integration only if it improves the intended metric without unacceptable latency/complexity regression.
