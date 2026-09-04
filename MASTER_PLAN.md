# MASTER PLAN — UiUxMaster

This is the **single living execution plan** for UiUxMaster. Do not create a competing roadmap.

`MASTER_PLAN.md` is authoritative for naming, ownership, runtime tiers, phase status and execution order. Subordinate documents must be synchronized to it before they are used to justify architectural changes.

Detailed subordinate specifications:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — ownership and evidence architecture;
- [`docs/ULTRA_FAST_VISUAL_LOOP.md`](docs/ULTRA_FAST_VISUAL_LOOP.md) — low-latency rendering/verification architecture;
- [`docs/ECOSYSTEM_INTEGRATION_PROGRAM.md`](docs/ECOSYSTEM_INTEGRATION_PROGRAM.md) — Axiom/SncSinCore/SkillState/DeepSearch integration contracts and optional ecosystem reuse gates;
- [`docs/RESEARCH_FOUNDATIONS.md`](docs/RESEARCH_FOUNDATIONS.md) — research → invariant traceability;
- [`docs/adr/0001-runtime-and-ownership.md`](docs/adr/0001-runtime-and-ownership.md) — accepted runtime-tier and ownership decision.

---

# 0. Current implementation snapshot

The repository has moved beyond foundation/prototyping. The current state is a **functional alpha of the execution/control substrate**.

Implemented and exercised on `main`:

```text
Go 1.26.4
+ root test/race/vet CI
+ Chromium integration CI
+ native frontend impact graph/resolver
+ fidelity risk/capability model
+ renderer-neutral FastRender contract
+ WGGo RGBA renderer/ROI crop
+ resident raw-CDP Chromium runtime
+ warm bounded page pool
+ explicit render epoch
+ DOMSnapshot/AX/fonts/runtime diagnostics
+ ROI screenshot capture
+ canonical evidence.Packet projection
+ deterministic verifier suite
+ in-memory RGBA visual diff primitive
+ benchmark harnesses with p50/p95/p99
+ isolated Axiom control module
+ Axiom → FastCDP → verifier → engine vertical slice
+ durable runner restart/reopen test
+ thin MCP baseline
```

The main architectural gap is **integration**, not absence of primitives:

```text
source change
→ ImpactSet
→ validation scope/invalidation
→ fidelity/risk routing
→ WGGo or FastCDP
→ deterministic verification
→ canonical evidence
→ Axiom/host/MCP
```

The pieces on both sides of this pipeline exist, but this full path is not yet the single canonical product execution path.

The largest capability gaps after that are:

1. Playwright TruthPath and FastPath/TruthPath calibration;
2. Design Intelligence / semantic critic;
3. relative candidate comparison and repair loop;
4. SncSinCore memory;
5. SkillState bounded reasoning/evolution;
6. DeepSearch research adapter;
7. full MCP product surface and adversarial evals.

## Documentation debt that must not be ignored

- `docs/ARCHITECTURE.md` currently uses an older conflicting L0–L4 evidence nomenclature. The tier names in this master plan are authoritative and the subordinate document must be updated.
- `docs/ULTRA_FAST_VISUAL_LOOP.md` still describes `chromedp/cdproto` as an initially preferred implementation even though the current FastBrowser implementation is a custom raw-CDP transport. Driver choice remains provisional until comparative benchmarks are complete.
- `README.md` still describes the project as an early foundation before browser adapters; that is no longer accurate.
- open issues/checklists for FastPath and TruthPath must be synchronized with actual completion state rather than treated as the source of truth.

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

The normal edit/polish loop must target **sub-second latency**. Common warm deterministic checks should target the **tens-of-milliseconds range** whenever changed scope and fidelity requirements permit it.

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
28. **Implemented primitive != integrated capability.** A checklist is complete only when the capability participates in the intended product path and has the required tests/measurements.
29. **Master-plan terminology is canonical.** Subordinate documents may add detail but may not redefine L0–L4, ownership or phase semantics.

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
                    Invalidation / Scope Policy
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
          in-process         raw direct CDP      clean/cross-browser
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

The canonical hot path is:

```text
change
→ native impact engine
→ invalidation/scope
→ fidelity router
→ WGGo or resident CDP
→ deterministic verifier
→ evidence.Packet
```

Axiom durable workflows, SncSinCore retrieval, SkillState evolution, DeepSearch and full Playwright are invoked only when their capability is required.

---

# 4. Runtime evidence tiers — canonical naming

These names are authoritative throughout the project.

## L0 — Static/source preflight

Use for:

- changed-file/module/token detection;
- source dependency graph updates;
- unsupported-feature detection;
- component ownership mapping;
- obvious static violations;
- validation-scope calculation.

Target: microseconds to low milliseconds.

## L1 — WGGo FastRender

Current implementation: `internal/runtime/wggo` over `go-webengine/engine`.

Current calibrated capability envelope is intentionally narrower than the target:

- direct RGBA rendering: implemented;
- in-memory crop/ROI: implemented;
- renderer/fidelity metadata: implemented;
- stable geometry/style inspection: **not yet implemented** through the current public WGGo adapter;
- scenarios: not implemented;
- browser accuracy: false by definition.

Use L1 only for evidence it can actually prove. Unsupported geometry/styles/scenario requirements must escalate rather than fabricate evidence.

## L2 — FastBrowser

Current implementation: `internal/runtime/fastcdp` using a custom raw-CDP websocket transport and resident Chromium-family process.

Implemented:

- process launched once;
- bounded warm context/page pool;
- explicit render epoch/readiness bridge;
- `DOMSnapshot.captureSnapshot` with selected computed styles;
- accessibility tree capture;
- font-state capture;
- runtime/network/console diagnostics;
- ROI screenshot capture;
- canonical packet projection;
- per-operation runtime latency;
- page discard/replacement;
- recovery classification/policy.

Still required:

- comparative raw-CDP vs `chromedp/cdproto` vs Rod vs warm Playwright benchmark before the driver is considered permanently selected;
- complete stale-state health model;
- recovery executor that actually applies the component → page → context → browser policy across the runtime;
- integration with ImpactSet/invalidation so evidence collection is scoped by the canonical change pipeline.

## L3 — TruthPath

Playwright against real Chromium/Chrome and selected Firefox/WebKit environments.

Use for:

- clean-state verification;
- isolation-sensitive scenarios;
- protected screenshot baselines;
- complex interactions;
- cross-browser milestone/release verification;
- L1/L2 fidelity calibration.

**Current state: not implemented.** This is a major next capability after the canonical FastPath pipeline is wired.

## L4 — Semantic visual critique

Local-first VLM/hierarchical critic for unresolved semantic questions:

- hierarchy;
- typography relationships;
- composition/crop;
- color balance;
- generic-template/card-soup smell;
- art direction;
- subtle visual polish.

**Current state: architecture/rubric only; no production critic provider yet.**

---

# 5. Core packages and ownership

## Implemented packages

```text
internal/design              canonical rubric baseline
internal/evidence            normalized evidence contracts
internal/evidenceplan        current evidence-shape planner
internal/engine              evaluation + tier routing decisions
internal/mcpserver           thin MCP adapter
internal/impact              native frontend dependency/impact model/resolver
internal/fidelity            capability/risk model
internal/runtime/fastrender  renderer-neutral render contract
internal/runtime/wggo        WGGo L1 adapter
internal/runtime/fastcdp     resident Chromium raw-CDP L2 runtime
internal/verifier            deterministic verification
internal/visualdiff          in-memory RGBA comparison primitive

control/axiom                isolated nested Go module for Axiom control plane
control/axiom/controlplane   run model, budgets, history, memory/file stores
control/axiom/uiuxadapter    UiUx execution adapter + FastCDP collector
```

`control/axiom` is intentionally a **separate module**, not `internal/control`. This isolates Axiom/Pebble/Prometheus-related dependency weight from the root FastPath module. This is the current accepted implementation boundary unless future measurements justify a change.

## Planned packages/capabilities

```text
internal/invalidation        ImpactSet → validation-scope policy
internal/runtime/playwright  TruthPath adapter
internal/critic              relative/hierarchical critique domain
internal/vlm                 model-neutral local VLM providers
internal/knowledge           SncSinCore memory/admission/retrieval adapter
internal/skillruntime        SkillState bounded projection/evolution adapter
internal/research            DeepSearch optional research adapter
internal/memory              design-memory domain policy
internal/eval                adversarial/replay/benchmark corpus
```

Domain packages depend on local interfaces. External library types do not leak across ownership boundaries.

`internal/impact` is a **native UiUxMaster subsystem**. It must not expose AutoTraceLab types or assume AutoTraceLab process/block semantics.

---

# 6. Canonical execution and renderer contracts

Renderer contract:

```text
Render(ctx, RenderRequest) → RenderEvidence
Inspect(ctx, InspectRequest) → StructuralEvidence
CaptureRegion(ctx, RegionRequest) → PixelBuffer / ArtifactRef
RunScenario(ctx, ScenarioRequest) → ScenarioEvidence
Capabilities() → RendererCapabilities
```

Canonical validation path to implement now:

```text
ValidationRequest
  ├─ change set / changed files
  ├─ intent
  ├─ target project/page/story
  ├─ viewport/theme/scenario
  ├─ final-gate flag
  └─ budget
        ↓
ImpactResolver
        ↓
ImpactSet
        ↓
InvalidationPolicy
        ↓
ValidationScope
        ↓
Fidelity assessment + evidence need
        ↓
RouteDecision
        ↓
L0 / L1 / L2 / L3 collector
        ↓
evidence.Packet
        ↓
Deterministic verifier
        ↓
engine.Report / next action
```

No MCP/Axiom-specific type belongs inside this domain path.

Evidence records must retain, as applicable:

- hierarchy/semantic refs;
- geometry;
- selected styles;
- runtime issues;
- raw RGBA locally or artifact ref/digest;
- renderer/version/tier;
- fidelity/confidence;
- latency breakdown;
- provenance/digest;
- impact/scope correlation.

---

# 7. Fidelity/capability routing

## LOW risk

Examples: static/lightly dynamic HTML, ordinary block/flex/grid/table, basic typography/positioning, supported SVG/images.

Preferred route: L1 only for capability classes proven by the current adapter and parity corpus; otherwise L2.

## MEDIUM risk

Examples: React/Vue runtime DOM, complex nested layout, custom fonts, SVG-heavy components, moderate transforms/animations.

Preferred route: L1 speculative where useful + L2 confirmation.

## HIGH risk

Examples: canvas/WebGL, unsupported CSS filters/masks/paint features, shadow DOM/custom elements with uncertain support, browser-API-dependent layout, interaction/animation as the result.

Preferred route: L2/L3; L1 may only assist localization.

The feature scanner/risk model must keep expanding toward unsupported CSS/functions, canvas/WebGL, SVG features, shadow DOM/custom elements, pseudo elements/selectors, browser APIs, custom fonts, transforms/filters/masks, dynamic measurements and hydration/runtime dependencies.

---

# 8. UiUxMaster Incremental Impact Engine (P0)

The impact engine answers:

> Given this source/design/runtime change, what is the smallest conservative set of UI states and rendered regions that must be revalidated?

Current implementation already provides:

- frontend-specific node/edge kinds;
- race-safe forward and reverse adjacency;
- deterministic snapshots/order;
- Tarjan SCC;
- stable source/module/style/component/token IDs;
- JS/TS import scanning;
- CSS import/token indexing;
- unresolved dynamic import → explicit uncertainty;
- component instance/page/region primitives;
- runtime semantic-ref binding primitives;
- conservative broad flag for unknown/uncertain nodes;
- resolver benchmarks and synthetic project benchmark harness.

Still required for exit:

- explicit `internal/invalidation` scope policy;
- cheap incremental/delta graph update rather than only rebuild/query primitives;
- route/story registry ingestion rather than only builder-level primitives;
- automatic L1/L2 runtime semantic-ref feedback into the graph;
- critical-route policy;
- false-negative mutation/adversarial suite;
- 1k/10k/100k recorded gates and allocations;
- full/incremental recompute parity;
- `ImpactSet` wired into the canonical router/runtime path.

Performance targets remain hypotheses until recorded:

- 1k-node local impact p95 <1 ms;
- 10k-node p95 <5 ms;
- 100k-node bounded local change p95 <20 ms;
- no full traversal/revalidation on a known leaf edit;
- impact false-negative rate must be explicitly measured.

## AutoTraceLab optional reuse gate

AutoTraceLab is not a dependency or owner of this subsystem. Isolated domain-neutral graph primitives may be compared only after the native baseline and mutation suite are stable. Adopt nothing unless correctness, latency, allocation, maintenance and provenance gates clearly win.

---

# 9. Warm render lifecycle

Target lifecycle:

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

Current FastCDP already provides the resident process, warm bounded page pool and epoch bridge. The missing step is connecting source-change/ImpactSet semantics to warm target ownership and collection scope.

Do not use arbitrary sleeps or generic `networkidle` as the normal HMR readiness condition.

---

# 10. In-memory visual path

Preferred L1 path:

```text
render → RGBA → crop/subimage → diff/statistics → optional VLM encoding of selected crop
```

Implemented foundations:

- WGGo returns `image.RGBA`;
- WGGo ROI crop stays in memory;
- `internal/visualdiff.CompareRGBA` compares RGBA directly and returns changed bounds/statistics.

Still required:

- wire visualdiff into the canonical validation pipeline;
- region clustering beyond one bounding rectangle;
- DOM/semantic intersection;
- protected baseline storage and L3 calibration.

For L2, ROI screenshot is already implemented and remains the default visual capture strategy. Full-page captures are milestone/debug artifacts rather than hot-path defaults.

---

# 11. Latency telemetry

Current `evidence.Packet` exposes L2 runtime timing for epoch wait, snapshot, pixels, accessibility, fonts, diagnostics, total and retries.

The **end-to-end** canonical run must expand this to at least:

```text
impact_ms
invalidation_ms
fidelity_scan_ms
route_ms
fast_render_ms
hmr_wait_ms
browser_snapshot_ms
roi_capture_ms
pixel_diff_ms
verify_ms
memory_query_ms
vlm_ms
synthesis_ms
total_ms
```

Also record:

- cold/warm state;
- evidence tier;
- validation scope size;
- browser/page reuse vs reset;
- provenance/fidelity ID.

Never claim system speedup without p50/p95/p99 distributions and a defined scenario.

---

# 12. Benchmark program

## Already implemented

- `cmd/uiuxbench` for impact/project indexing/WGGo with p50/p95/p99;
- `cmd/uiuxcdpbench` for resident raw-CDP operations with p50/p95/p99;
- CI smoke execution of both;
- real Chromium FastCDP integration CI.

## Still required before locking runtime choices

### Drivers

- raw CDP current implementation;
- `chromedp/cdproto` comparison;
- Rod comparison;
- Playwright attached to existing browser comparison;
- Playwright component/context-reuse comparison when applicable.

### Engines / paths

- WGGo/go-webengine;
- `chrome-headless-shell` where available;
- modern Chromium Headless;
- branded Chrome where useful;
- Playwright Chromium/Firefox/WebKit TruthPath;
- Lightpanda structural-only experiment only if it demonstrates value, never pixel truth.

### Required artifact policy

CI must not only print benchmark JSON to `/tmp`. Introduce a reproducible benchmark artifact/history mechanism with scenario/version/browser/hardware metadata before performance gates are treated as stable evidence.

### UI fixtures

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

### Metrics

p50/p95/p99, CPU, RSS, allocations, protocol bytes/round trips, image encode cost, fidelity vs TruthPath, false PASS/FAIL and impact false negatives.

---

# 13. Initial latency targets — hypotheses, not promises

- L0/impact common local scope: <5 ms.
- L1 simple component validation: <30 ms where feasible.
- L1 pixel/diff operations: 1–10 ms targets.
- L2 structural ROI snapshot: 2–20 ms target.
- L2 small ROI screenshot: 5–30 ms target.
- L2 common warm deterministic validation: <50 ms where feasible, p95 <100 ms aspiration.
- L3 TruthPath prioritizes clean/cross-browser correctness, not artificial low latency.

Orders-of-magnitude speedup comes from eliminating cold launch/navigation/full-page capture/VLM from most iterations, not from one renderer marketing claim.

---

# 14. Axiom control-plane integration

Axiom is used for **selected multi-step/explainable workflows**, never around each CDP/render primitive.

## Current implementation

Axiom integration has started and is real:

```text
control/axiom/                    separate Go module
  controlplane/                   Axiom runtime wrapper + Run projection
  uiuxadapter/                    execution adapter
  uiuxadapter/FastCDPCollector    resident L2 collector
```

Current P0 workflow:

```text
PlanEvidence
→ CollectVerify
→ Decide
```

Already implemented:

- Go 1.26 compatibility;
- pinned Axiom dependency;
- root FastPath dependency isolation check;
- separate Axiom CI with test/race/vet;
- explicit compact budgets/usage;
- cancellation;
- ordered history projection;
- in-memory store;
- file-backed durable store;
- restart/reopen recovery test;
- real Axiom → FastCDP → deterministic verifier → engine integration test.

## Important current limitation

The Axiom adapter currently reaches FastCDP through `evidenceplan` rather than consuming the complete canonical ImpactSet → invalidation → fidelity/router pipeline. Fixing this is higher priority than adding more workflow states.

## Workflow evolution

The current P0 `adgo.Definition` workflow is accepted as a vertical slice. New richer workflows should prefer the Axiom declarative `model` frontend when it improves correctness/readability and does not force a needless rewrite of stable low-level code.

Future workflow candidates:

- `DesignPolishRun`;
- `CandidateComparisonRun`;
- `TruthPathCalibrationRun`;
- `CrossBrowserReleaseRun`;
- `DesignEvalRun`;
- `SkillPromotionRun`.

`DesignPolishRun` eventually adds state/events for impact resolution, findings, repairs, candidate comparison, fidelity escalation and independent verification.

## Durability policy

Durability exists earlier than the original plan expected. Do not remove a working isolated capability merely to match the old sequence. Instead:

- keep durable state compact;
- keep heavy evidence outside workflow state by digest/ref;
- measure crash-resume/recovery value and overhead;
- add an ADR/eval if durability becomes a production default rather than an optional constructor.

Exit: multi-step polish/eval runs are reproducible/explainable, use the canonical execution pipeline, and introduce no measurable L0–L2 hot-path regression.

---

# 15. SncSinCore epistemic design memory

**Current state: not implemented.**

Use SncSinCore for admitted long-term evidence-backed knowledge, not operational state or raw conversation history.

Start with embedded `epmemory`; activate segmented `memoryv2` only after corpus/memory/latency thresholds justify it.

Ontology targets:

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

Admission:

```text
runtime observation
→ evidence.Packet
→ candidate memory atoms
→ provenance/scope/time validation
→ SncSinCore admission
```

Every admitted fact retains run ID, evidence digest, renderer/fidelity, environment, critic/rule version, scope and outcome.

Namespace firewall:

```text
knowledge/global-design
knowledge/project/<id>
evidence/project/<id>
research/global
skillmeta/<skill-id>
```

Project-private content never leaks into global memory. Conflicts are preserved rather than averaged away.

---

# 16. SkillState bounded state + controlled evolution

**Current state: not implemented.**

SkillState owns the bounded typed working projection seen by the reasoning model. It is not canonical truth and not a scheduler.

Target state:

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

Model output mutates state only through typed patches with expected revision/digest. Stale/rejected patches mutate nothing.

Evolution pipeline:

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

---

# 17. DeepSearch research plane

**Current state: not implemented; optional.**

DeepSearch remains a sidecar/provider. UiUxMaster must work fully for local rendering/verification without Python or DeepSearch installed.

Use only for research/acquisition needs such as current standards, unfamiliar patterns, product/domain research, benchmark/source verification or explicit evidence requests.

Research output never mutates active design rules directly:

```text
DeepSearch bundle
→ provenance/source validation
→ claim extraction
→ memory candidates
→ SncSinCore gates
→ validated knowledge
```

---

# 18. Optional ecosystem reuse and operational gates

## AutoTraceLab

No runtime dependency and no ownership of impact semantics. Compare isolated graph techniques only if the native implementation exposes a measured gap.

## IRIS patterns

Potential compatibility schema only around:

```text
Claim / Evidence / Artifact / Provenance / Confidence / Scope
```

Do not import IRIS Studio application semantics.

## RepoArk

No runtime dependency. Consider later for benchmark/release artifact archival when reproducibility/history becomes an operational need.

## WebGate

No local-runtime dependency. Consider only for authenticated remote browser/device workers after remote execution is a committed feature.

---

# 19. MCP tool surface

## Current surface

Implemented:

- `uiux_get_rubric`;
- `uiux_evaluate_evidence`.

The MCP adapter remains correctly thin, but the runtime/impact pipeline is **not yet exposed as a complete MCP product**.

## Next runtime surface

Implement only after the canonical domain path exists:

- `uiux_plan_validation`;
- `uiux_capture`;
- `uiux_inspect_layout`;
- `uiux_inspect_accessibility`;
- `uiux_run_scenario` when a capable runtime path exists.

Caller requests evidence intent/fidelity, not WGGo/FastCDP/Playwright vendor selection.

## Later surface

Visual comparison:

- `uiux_compare_baseline`;
- `uiux_compare_candidates`;
- `uiux_localize_visual_change`.

Semantic critique:

- `uiux_critique_page`;
- `uiux_critique_region`;
- `uiux_rank_candidates`.

Synthesis:

- `uiux_recommend_repairs`;
- `uiux_verify_completion`.

Memory/evolution/research:

- `uiux_record_lesson`;
- `uiux_replay_lesson`;
- `uiux_run_design_eval`;
- optional explicit research operation.

Large screenshots/diffs/traces/evidence/memory/benchmark artifacts are resources/references, not huge inline payloads.

---

# 20. Execution phases

## Phase 0 — Foundation & invariants

**Status: NEARLY COMPLETE**

- [x] Go module initialized.
- [x] MCP Go SDK integrated.
- [x] canonical rubric/evidence packet/engine baseline.
- [x] first MCP tools over stdio.
- [x] unit tests.
- [x] Go upgraded to 1.26.4.
- [x] `go test ./...` CI.
- [x] `go test -race ./...` CI.
- [x] `go vet ./...` CI.
- [x] accepted runtime/ecosystem ownership ADR.
- [x] root-vs-Axiom dependency isolation CI.
- [ ] staticcheck if pinned/reproducible and useful.
- [ ] MCP schema contract tests.
- [ ] dependency/license inventory baseline.
- [ ] synchronize README/subordinate docs with this plan.

Exit: core tests/vet/race green, schemas contract-tested, documentation terminology aligned, vendor types remain behind adapters.

## Phase 1 — Design Intelligence Core

**Status: EARLY / PARTIAL**

- [x] canonical design-axis rubric baseline.
- [ ] convert premium editorial/motion/responsive rules into versioned structured rules.
- [ ] stable rule IDs/categories.
- [ ] domain types for `Finding`, `Evidence`, `RepairHypothesis`, `CritiquePass`, `CandidateComparison`.
- [ ] page→section→component→element hierarchy model.
- [ ] hard constraints vs preferences.
- [ ] product profiles.
- [ ] original prompt/research material under traceable knowledge sources.

## Phase 2 — Ultra-Fast Runtime

**Status: SUBSTANTIAL FOUNDATION IMPLEMENTED; CANONICAL PIPELINE NOT YET COMPLETE**

### 2A Benchmark + fidelity scanner

- [x] benchmark harness exists.
- [x] p50/p95/p99 reporting exists.
- [x] renderer capability model exists.
- [x] LOW/MEDIUM/HIGH fidelity risk model exists.
- [x] WGGo baseline measurement exists.
- [x] raw FastCDP warm measurements exist.
- [ ] comparative raw-CDP/chromedp/Rod/warm-Playwright benchmark.
- [ ] CPU/RSS/protocol/allocation coverage across representative scenarios.
- [ ] persistent benchmark/fidelity artifacts with environment metadata.
- [ ] FastPath/TruthPath fidelity corpus.

### 2B UiUxMaster Incremental Impact Engine

- [x] native `ImpactSet`/resolver contracts.
- [x] frontend-specific node/edge model.
- [x] deterministic forward/reverse graph.
- [x] SCC implementation.
- [x] source/import/CSS-token analyzers.
- [x] unresolved dynamic dependency → conservative uncertainty.
- [x] component/page/region builder primitives.
- [x] runtime semantic-ref binding primitive.
- [x] basic benchmarks/project fixture harness.
- [ ] explicit `internal/invalidation` package.
- [ ] cheap delta/incremental update path and full-vs-incremental parity.
- [ ] route/story registry ingestion.
- [ ] automatic runtime semantic-ref feedback from L1/L2.
- [ ] critical-route/widening policy.
- [ ] false-negative mutation suite.
- [ ] recorded 1k/10k/100k performance gates.
- [ ] feed `ImpactSet` into the canonical validation router/runtime.

### 2C WGGo FastRender

- [x] renderer-neutral interface.
- [x] WGGo adapter.
- [x] direct RGBA render.
- [x] direct ROI crop.
- [x] capability/fidelity metadata.
- [x] router escalates geometry when L1 cannot prove it.
- [ ] stable WGGo geometry/style inspection API or explicit permanent pixel-only envelope.
- [ ] visualdiff wired to L1 pipeline.
- [ ] parity fixtures against L2/L3.
- [ ] evidence classes explicitly calibrated for allowed L1 PASS.

### 2D Resident Chromium FastBrowser

- [x] resident browser process.
- [x] raw-CDP transport.
- [x] bounded warm page pool.
- [x] isolated browser context ownership.
- [x] explicit render epoch bridge/gate.
- [x] DOMSnapshot + selected styles.
- [x] accessibility tree capture.
- [x] font-state capture.
- [x] runtime/network/console diagnostics.
- [x] ROI screenshot capture.
- [x] canonical evidence packet projection.
- [x] per-stage L2 latency metrics.
- [x] page discard/replacement path.
- [x] recovery classification/policy model.
- [x] real Chromium integration CI.
- [ ] comparative driver selection evidence.
- [ ] full runtime recovery executor using component→page→context→browser ladder.
- [ ] comprehensive stale-state health checks.
- [ ] ImpactSet/scope-driven warm-target selection and evidence collection.

### 2E Playwright TruthPath

**Status: NOT STARTED**

- [ ] clean-state screenshots/ARIA/errors/fonts.
- [ ] worker/adapter with vendor types contained outside domain core.
- [ ] complex scenarios.
- [ ] browser matrix.
- [ ] deterministic baseline controls.
- [ ] L1/L2 calibration corpus.
- [ ] protected baseline storage/reference policy.

Phase 2 exit: normal local edit does not navigate/relaunch, ImpactSet bounds scope, the router chooses the cheapest sufficient tier, L1 PASS is calibrated, L2 is resident Blink truth, and L3 independently calibrates clean/cross-browser cases.

## Phase 3 — Deterministic verifiers

**Status: IN PROGRESS**

Implemented:

- [x] horizontal viewport overflow.
- [x] ancestor clipping.
- [x] interactive clipping severity.
- [x] interactive target overlap.
- [x] hidden interactive controls.
- [x] pointer-events disabled controls.
- [x] target-size policy with inline-link exception.
- [x] style invariants.
- [x] missing/ignored AX node checks.
- [x] missing actionable role.
- [x] accessible-name checks.
- [x] font-set loading/font-face error checks.
- [x] runtime/network/console failures enter canonical evidence.

Still required:

- [ ] generalized offscreen/invalid geometry.
- [ ] fixed/sticky obstruction.
- [ ] computable color contrast.
- [ ] focus trap/focus obstruction.
- [ ] duplicate IDs.
- [ ] stronger hidden/zero-size control classification.
- [ ] truncation anomalies.
- [ ] responsive failure rules.
- [ ] mutation/adversarial recall and false-positive metrics.

Run every rule on the cheapest tier capable of proving it.

## Phase 4 — Visual regression/localization

**Status: EARLY FOUNDATION**

- [x] in-memory RGBA comparison primitive.
- [x] changed-pixel count/ratio/bounds/max-delta statistics.
- [x] browser ROI capture available from L2.
- [ ] baseline abstraction/versioning.
- [ ] region clustering.
- [ ] changed-pixel density per region.
- [ ] DOM box intersection.
- [ ] semantic visual-change findings.
- [ ] Playwright protected clean/cross-browser baselines.

## Phase 5 — Axiom control plane

**Status: P0 VERTICAL SLICE IMPLEMENTED; FULL POLISH WORKFLOW PENDING**

- [x] Go 1.26 compatibility.
- [x] isolated nested Axiom module.
- [x] pinned Axiom version.
- [x] Axiom-specific CI with race/vet.
- [x] P0 `PlanEvidence → CollectVerify → Decide` workflow.
- [x] explicit compact budgets/usage.
- [x] cancellation.
- [x] history/explain projection.
- [x] in-memory execution.
- [x] file-backed durable execution.
- [x] restart/reopen recovery test.
- [x] real FastCDP collector integration.
- [ ] Axiom adapter consumes canonical Impact→Invalidation→Fidelity→Runtime pipeline.
- [ ] `DesignPolishRun` state/events/claims.
- [ ] model-frontend workflow for new rich flows where justified.
- [ ] typed critic/repair/memory/TruthPath activities.
- [ ] explicit iteration/VLM/TruthPath/candidate/repair budgets.
- [ ] retry/idempotency/fault-injection proof for external side effects.
- [ ] durability product-value/overhead measurement before making it a mandatory default.

## Phase 6 — SncSinCore memory

**Status: NOT STARTED**

- [ ] ontology/namespaces.
- [ ] evidence→candidate admission mapper.
- [ ] embedded `epmemory` start.
- [ ] bounded ContextPack retrieval.
- [ ] provenance/conflict/retraction/scope tests.
- [ ] memory on/off held-out eval.
- [ ] `memoryv2` only after scale threshold.

## Phase 7 — SkillState bounded reasoning

**Status: NOT STARTED**

- [ ] typed UiUx skill state/patch schema.
- [ ] project Axiom/domain state into bounded Σ.
- [ ] externalize large artifacts by digest.
- [ ] SncSinCore MemoryPort.
- [ ] CAS/stale/oscillation gates.
- [ ] remove implicit long-history replay.
- [ ] token/task-success benchmarks.

## Phase 8 — Progressive local visual critic

**Status: NOT STARTED BEYOND RUBRIC/EVIDENCE CONTRACTS**

- [ ] page→section→component crops.
- [ ] model-neutral local provider.
- [ ] edge model first.
- [ ] stronger model only on uncertainty.
- [ ] structured grounded output.
- [ ] relevant SncSinCore memory only when useful.
- [ ] renderer/fidelity provenance retained.

## Phase 9 — Relative design search

**Status: NOT STARTED**

- [ ] baseline abstraction.
- [ ] candidate A/B when justified.
- [ ] per-axis pairwise comparison.
- [ ] hard correctness/accessibility constraints.
- [ ] select/merge.
- [ ] re-render and independently verify.

Absolute aggregate score is never the sole completion gate.

## Phase 10 — Interaction playthrough

**Status: NOT STARTED AS A COMPLETE SCENARIO SYSTEM**

Scenarios eventually cover navigation, menus, dialogs, forms, loading/error, hover/focus/touch, resize, theme and keyboard-only flows. FastBrowser handles warm/local flows; TruthPath proves clean/cross-browser flows.

## Phase 11 — Cross-browser/perturbation

**Status: NOT STARTED**

Milestone: representative responsive matrix + selected TruthPath.

Release: browser × viewport × theme × perturbations.

Perturbations include arbitrary widths, long RU/DE text, missing/slow media/font, 200% zoom, reduced motion, extreme accent, empty/large data, slow/error network.

## Phase 12 — Controlled evolution with SkillState

**Status: NOT STARTED**

- [ ] candidate heuristics only from admitted evidence.
- [ ] immutable candidate skill versions.
- [ ] replay corpus.
- [ ] shadow/current-vs-candidate eval.
- [ ] design improvement + non-regression + latency/token gates.
- [ ] authorized promotion.
- [ ] rollback validation.

## Phase 13 — DeepSearch research plane

**Status: NOT STARTED / OPTIONAL**

- [ ] optional sidecar/feature flag.
- [ ] bounded `Researcher` port.
- [ ] source/provenance admission.
- [ ] DeepSearch→SncSinCore path.
- [ ] cache/staleness policy.
- [ ] optional periodic/manual Axiom research workflow.
- [ ] no dependency of hot loop on research availability.

## Phase 14 — Adversarial evals

**Status: NOT STARTED AS A SYSTEMATIC HARNESS**

Inject controlled defects including CTA shift, contrast reduction, hierarchy collapse, spacing flattening, clipping/crop damage, focus loss, overflow, tiny targets, dark-theme mismatch, card soup, renderer fidelity traps and impact-engine traps.

Measure detection recall, false positives, localization, severity, repair success, regression, impact false-negative rate, FastRender false PASS/FAIL, parity, latency/cost.

## Phase 15 — MCP productization

**Status: EARLY BASELINE**

- [x] official Go MCP SDK integrated.
- [x] stdio server baseline.
- [x] thin protocol adapter boundary.
- [x] `uiux_get_rubric`.
- [x] `uiux_evaluate_evidence`.
- [ ] stable JSON schema contract tests.
- [ ] `uiux_plan_validation`.
- [ ] `uiux_capture` routed through canonical runtime.
- [ ] layout/accessibility inspection tools.
- [ ] scenario tool.
- [ ] bounded deterministic outputs.
- [ ] artifacts as resources.
- [ ] cacheable catalogs.
- [ ] OpenTelemetry.
- [ ] stateless HTTP later when useful.
- [ ] tasks only for genuine long operations with client support.

## Phase 16 — Safety/privacy/legal/provenance

**Status: POLICY FOUNDATION ONLY**

- [x] local-first architectural rule documented.
- [x] large binary evidence kept out of canonical inline packet by design.
- [ ] purpose limitation/data minimization implementation.
- [ ] redact secrets/tokens/PII before optional external model.
- [ ] explicit retention policy.
- [ ] audit external calls.
- [ ] authorized-target guardrails.
- [ ] dependency/model/renderer license/provenance inventory.
- [ ] reference-design provenance policy enforcement.

## Phase 17 — Optional ecosystem/operational expansion

**Status: DEFERRED BY DESIGN**

Only after activation gates:

- isolated AutoTraceLab graph primitive comparison if the native engine reveals a real gap;
- RepoArk for benchmark/release archival/mirroring;
- WebGate for authenticated resilient remote browser/device workers.

None is required for the first production UiUxMaster.

---

# 21. Ecosystem implementation sequence E0→E7

Later layers still depend on earlier evidence/contracts. Status below reflects current code rather than the original starting point.

## E0 — Compatibility baseline — MOSTLY COMPLETE

- [x] Go 1.26+.
- [x] root race CI.
- [x] Axiom race CI.
- [x] runtime/ecosystem ownership ADR.
- [x] Axiom version pinned in isolated module.
- [x] AutoTraceLab explicitly non-required.
- [ ] dependency/license inventory.
- [ ] persistent benchmark baseline/artifacts.
- [ ] synchronize subordinate docs/readme.

## E1 — Native UiUxMaster Impact Engine — IN PROGRESS

- [x] ImpactSet/resolver/node/edge contracts.
- [x] native graph kernel.
- [x] source/import/CSS-token analyzers.
- [x] reverse adjacency and SCC.
- [x] conservative unresolved-dependency fallback.
- [x] runtime binding primitive.
- [ ] dedicated invalidation policy.
- [ ] route/story ingestion.
- [ ] automatic runtime-ref feedback.
- [ ] canonical router integration.
- [ ] 1k/10k/100k recorded gates.
- [ ] false-negative mutation suite.

## E2 — Axiom control plane — P0 IMPLEMENTED / RICH FLOW PENDING

- [x] coarse-grained typed activities.
- [x] renderer/data primitives remain direct.
- [x] budgets/cancellation/history.
- [x] in-memory execution.
- [x] optional durable execution + recovery test.
- [x] FastCDP end-to-end adapter.
- [ ] canonical pipeline integration.
- [ ] DesignPolishRun.
- [ ] richer retry/idempotency/fault proofs.
- [ ] explainability/eval requirements for repair/candidate workflows.

## E3 — SncSinCore memory — NOT STARTED

1. Ontology/namespaces.
2. Admission mapper.
3. Embedded `epmemory`.
4. Minimal requirement-driven ContextPacks.
5. Feed semantic critic/repair planning only where useful.
6. Conflict/scope/retraction tests.
7. Memory on/off eval.
8. `memoryv2` only after measured threshold.

## E4 — SkillState bounded state — NOT STARTED

1. Typed state/patch.
2. Projection from Axiom/domain state.
3. Large artifacts by reference.
4. SncSinCore MemoryPort.
5. CAS/oscillation protections.
6. Replace long-history replay.
7. Token/task-success benchmark.

## E5 — Controlled evolution — NOT STARTED

1. Candidate heuristics from repeated admitted evidence.
2. Immutable candidate skill versions.
3. Replay corpus.
4. A/B/shadow evals.
5. Non-regression + latency/token gates.
6. Authorized promotion.
7. Validated rollback.

## E6 — DeepSearch adapter — NOT STARTED / OPTIONAL

1. Optional feature/sidecar.
2. Bounded research port.
3. Provenance/source admission.
4. SncSinCore ingestion only after gates.
5. Cache/staleness.
6. Periodic/manual research workflow only if useful.

## E7 — Ecosystem hardening — FUTURE

1. Full race suite across all added modules.
2. Fault injection across activities/memory/render/research.
3. Prove no L0–L2 p95 regression.
4. Dependency-upgrade compatibility suite.
5. Privacy/scope isolation.
6. License/provenance report.
7. End-to-end bare-vs-integrated held-out eval.
8. Remove integrations that no longer justify complexity.

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
- false PASS/FAIL;
- reset frequency by reset level;
- stale-page detection/recovery success.

## Canonical pipeline

- % validation runs using ImpactSet-derived scope;
- % runs routed without hard-coded renderer choice;
- whole-site fallback frequency and reason;
- time from source change to actionable evidence;
- scope false-negative rate.

## Axiom

- replay/recovery success;
- duplicate side effects = 0;
- explain/history completeness;
- no measurable hot-path regression;
- canonical-pipeline usage rather than FastCDP-specific bypass.

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

- UI polish preference/success vs baseline;
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
- No renderer becomes the permanent default without comparative benchmark/fidelity proof.
- No integration is accepted without before/after eval evidence.
- No checklist item is marked complete merely because a type/function exists; it must satisfy its product-path exit condition.
- No duplicate planner/router architectures: converge `evidenceplan`, impact/invalidation and `engine.RouteValidation` into one coherent policy path rather than growing parallel decision systems.

---

# 24. Definition of Done — one UI polish run

A run is not complete merely because a screenshot matches baseline, FastRender passes or a VLM says “looks good”. As applicable:

- source change resolved to an explicit conservative validation scope;
- selected evidence tier is explainable by capabilities/fidelity/risk;
- no blocking runtime/layout/accessibility findings;
- intended interactions pass;
- responsive target set passes;
- candidate does not regress protected axes;
- visual findings above threshold resolved/accepted;
- no unexplained visual regression;
- evidence retains renderer/fidelity/provenance digest;
- milestone/release perturbations pass;
- FastRender-only PASS is inside a calibrated evidence class or escalated;
- workflow/history/evidence references are reconstructable;
- long-term memory admission is separate from completion.

---

# 25. Definition of Done — ultra-fast loop

- reproducible benchmark harness committed;
- benchmark results retained as artifacts/history with environment metadata;
- p50/p95/p99 recorded by tier;
- renderer/browser processes reused;
- normal source edit does not navigate/relaunch;
- native Impact Engine + invalidation policy bound validation scope;
- impact false-negative mutation suite passes;
- L1 enabled only for calibrated classes;
- direct RGBA path avoids PNG round trips;
- resident Chromium provides warm Blink truth;
- recovery ladder is actually executable, not only modeled;
- Playwright TruthPath calibrates clean/cross-browser cases;
- each evidence result reports end-to-end latency/fidelity/provenance/scope;
- Axiom/MCP consume the same canonical pipeline rather than vendor-specific bypasses;
- ecosystem integrations do not degrade L0–L2 p95 beyond explicit budget;
- `go test ./...`, `go test -race ./...` and `go vet ./...` remain green.

---

# 26. Immediate next execution slice

Execute in this order. Do not start SncSinCore/SkillState/DeepSearch before the canonical validation pipeline and TruthPath foundations are stable unless a blocking dependency proves otherwise.

## P0 — Converge the existing execution substrate

1. **Synchronize documentation terminology**: update `docs/ARCHITECTURE.md`, `docs/ULTRA_FAST_VISUAL_LOOP.md`, `README.md` and open FastPath issue status so this master plan is the unambiguous source of truth for L0–L4 and current implementation state.
2. Add `internal/invalidation` with explicit `ImpactSet → ValidationScope` policy, including local/shared/global token, SCC, unknown/dynamic, critical-route and user-forced widening rules.
3. Define one protocol-independent `ValidationRequest`/`ValidationScope` orchestration boundary. Avoid creating a second router or planner.
4. Wire changed files/project index → `ImpactResolver.ApplyChanges` → `ValidationScope`.
5. Converge `internal/evidenceplan` and `engine.RouteValidation` so evidence shape, fidelity and renderer tier are decided by one coherent policy path rather than parallel planners.
6. Add a runtime dispatcher/collector boundary that executes L0/L1/L2 according to `RouteDecision` without exposing WGGo/FastCDP vendor choice to callers.
7. Wire WGGo RGBA + `internal/visualdiff` into the L1 path. Until geometry/styles become available, keep L1 geometry/style PASS prohibited and escalate to L2.
8. Wire FastCDP collection to ImpactSet-derived regions/pages and preserve explicit epoch/diagnostic watermarks.
9. Add an end-to-end integration fixture proving:

   ```text
   changed source/CSS token
   → ImpactSet
   → bounded ValidationScope
   → fidelity route
   → WGGo or FastCDP
   → evidence.Packet
   → verifier
   → engine decision
   ```

10. Expand telemetry from L2-only timing to end-to-end impact/invalidation/fidelity/route/render/verify totals.

## P0 — Finish FastPath engineering gates

11. Add impact false-negative mutation/adversarial tests for dynamic import, cycles, shared tokens, CSS cascade, route alias, stale runtime refs and runtime-only component instances.
12. Record 1k/10k/100k impact benchmarks and allocation gates.
13. Benchmark current raw CDP against `chromedp/cdproto`, Rod and warm Playwright on the same fixtures. Keep raw CDP only if it remains the best overall fit for latency, allocations, complexity and capability.
14. Turn the existing recovery policy into an executable recovery controller for component → page → context → browser resets; add fault-injection tests.
15. Add broader stale-state health checks for unexpected navigation/origin, broken epoch bridge, invalid context/session, stale service-worker/cache where observable, and bounded resource growth.
16. Persist benchmark results as CI artifacts/history instead of `/tmp`-only output.

## P1 — Build TruthPath and calibration

17. Implement `internal/runtime/playwright` behind a narrow vendor-neutral worker/adapter contract.
18. Implement clean-state capture: screenshot, ARIA, errors, failed requests, fonts and semantic/layout evidence mapped into the same `evidence.Packet`.
19. Implement scenario actions and deterministic baseline controls.
20. Add Chromium parity fixtures first, then selected Firefox/WebKit coverage.
21. Build the L1/L2/L3 calibration corpus and define which evidence classes may legally PASS on L1/L2 without L3.
22. Add protected baseline/artifact references and semantic visual-diff localization.

## P1 — Expose the canonical pipeline to control/MCP

23. Refactor the Axiom adapter so `CollectVerify` invokes the canonical validation pipeline rather than a FastCDP-specific planning bypass.
24. Add `uiux_plan_validation` and `uiux_capture` MCP tools over the same domain path; then add layout/accessibility inspection tools.
25. Add MCP schema contract tests and bounded artifact/resource handling.

## P2 — Close the design loop

26. Complete Phase 1 Design Intelligence domain types/rules/profiles.
27. Implement progressive local semantic critic with structured grounded findings.
28. Add relative baseline/candidate comparison with hard correctness/accessibility constraints.
29. Evolve Axiom from the P0 linear validation flow to `DesignPolishRun` / candidate comparison / TruthPath calibration workflows.
30. Add host repair application and independent re-verification.

## P3 — Memory, bounded reasoning and research

31. Define SncSinCore design-memory ontology/admission and start embedded `epmemory`.
32. Add SkillState bounded reasoning projection + SncSinCore MemoryPort.
33. Build replay/shadow/non-regression gates for controlled skill evolution.
34. Add DeepSearch only as an optional research sidecar with provenance admission through SncSinCore.
35. Run bare-vs-integrated end-to-end held-out benchmarks and remove any integration that does not improve its intended metric enough to justify latency/complexity/dependency cost.

---

# 27. Current milestone definition

The **current milestone** is not “add more subsystems”. It is:

> **Make the already-implemented Impact, Fidelity, WGGo, FastCDP, Verifier, Engine and Axiom/MCP boundaries converge into one measured canonical validation pipeline, then add TruthPath as its independent correctness/calibration oracle.**

Milestone exit evidence:

- one changed-file integration test demonstrates bounded scope → routing → evidence → verification;
- no parallel planner/router ambiguity remains;
- normal warm local validation uses no browser relaunch/navigation;
- ImpactSet participates in every local validation unless explicitly bypassed with a documented reason;
- L1 cannot silently prove unsupported geometry/styles;
- L2 recovery is executable and fault-tested;
- driver selection has comparative measurements;
- benchmark artifacts are retained;
- TruthPath can independently reproduce/calibrate at least one representative L2 fixture;
- Axiom and MCP invoke the same protocol-independent pipeline;
- root and Axiom CI stay green under test/race/vet.
