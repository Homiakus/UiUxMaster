# UiUxMaster Ecosystem Integration Program

This document is subordinate to `MASTER_PLAN.md`. If any statement here conflicts with `MASTER_PLAN.md`, the master plan wins.

The purpose of this document is to define how UiUxMaster may integrate selected Homiakus libraries **without moving foreign product semantics into the hot UI-validation path**.

## 0. Capability map

Use existing libraries only where they provide a proven capability:

- **Axiom** → control plane for selected long-running/explainable workflows;
- **SncSinCore** → admitted evidence-backed long-term design knowledge and bounded retrieval;
- **SkillState** → bounded typed working projection and controlled skill evolution;
- **DeepSearch** → optional research/acquisition sidecar;
- **AutoTraceLab** → optional reference/source of isolated domain-neutral graph primitives only after the native impact engine is complete and benchmarked;
- **IRIS patterns** → small evidence/claim/artifact/provenance compatibility schema, not application import;
- **RepoArk/WebGate** → later operational integrations after explicit activation gates.

Core rule:

```text
UiUxMaster native data plane
!= Axiom control plane
!= SncSinCore knowledge plane
!= SkillState working-state/evolution plane
!= DeepSearch research plane
```

No external project owns UiUxMaster frontend semantics.

---

# 1. Ownership rules

## 1.1 Hot path

The normal tens-of-milliseconds path is intentionally small:

```text
change
→ UiUxMaster native impact engine
→ fidelity router
→ WGGo or resident Chromium/CDP
→ deterministic verifier
→ evidence packet
```

Do not invoke Axiom durable activities, SncSinCore retrieval, SkillState evolution, DeepSearch, external MCP services or cloud models on every geometry/pixel operation.

## 1.2 Canonical ownership

- `internal/design` owns UI/UX rules and product profiles.
- `internal/evidence` owns normalized runtime evidence contracts.
- `internal/impact` owns frontend source/component/token/runtime-region dependency semantics.
- `internal/invalidation` owns the policy that turns `ImpactSet` into validation scope.
- `internal/fidelity` owns renderer capability/fidelity routing.
- Axiom owns selected workflow execution/history only.
- SncSinCore owns admitted long-term epistemic memory only.
- SkillState owns bounded model-visible working projection only.
- DeepSearch owns external research/acquisition only.
- MCP owns transport/tool/resource concerns only.

## 1.3 External types do not leak

```text
internal/impact       ← native UiUxMaster types
internal/control      ← Axiom adapter
internal/knowledge    ← SncSinCore adapter
internal/skillruntime ← SkillState adapter
internal/research     ← DeepSearch adapter
```

Optional AutoTraceLab reuse must remain behind private graph helpers and may not define public `ImpactSet` semantics.

## 1.4 Version/toolchain policy

- Pin pre-v1 external dependencies to explicit tag/commit when integrated.
- Upgrade through compatibility tests.
- Axiom currently requires Go 1.26+, therefore UiUxMaster upgrades/requalifies before direct Axiom integration.
- Optional AutoTraceLab primitive reuse does **not** justify adding a dependency unless benchmark and maintenance gates are met.

---

# 2. Integration A — Native UiUxMaster Incremental Impact Engine (P0)

This capability is deliberately **not an AutoTraceLab integration**.

## Objective

Convert a source/design/runtime change into the smallest conservative validation scope.

```text
changed file/token
→ module/style
→ component/variant
→ component instance
→ route/story/page
→ semantic DOM ref
→ render region
→ viewport/theme/scenario
```

## Native package

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

Suggested port:

```go
type Resolver interface {
    ApplyChanges(ctx context.Context, changes ChangeSet) (ImpactSet, error)
    ResolveComponent(ctx context.Context, componentID string) (ImpactSet, error)
    ResolveToken(ctx context.Context, tokenID string) (ImpactSet, error)
}
```

## Node kinds

At minimum:

- SourceFile
- Module
- StyleSheet
- DesignToken
- Component
- ComponentVariant
- ComponentInstance
- Route
- Story
- Page
- SemanticElementRef
- RenderRegion
- ViewportProfile
- ThemeProfile
- Scenario

## Edge kinds

At minimum:

- imports
- styles
- consumes_token
- renders
- instantiates
- appears_on
- maps_to_runtime_ref
- affects_region
- participates_in_scenario

## Build sources

Start with deterministic analyzers:

1. changed file paths from agent/git patch;
2. JS/TS/template imports/re-exports;
3. CSS module/style imports;
4. CSS custom-property definitions/references;
5. explicit component/test/design IDs;
6. route/story registry;
7. runtime semantic refs discovered through L1/L2;
8. framework-specific adapters only when measured value justifies them.

## Invalidation policy

- local component style → affected instances;
- shared component → known instances/representative pages;
- local design token → consumers;
- global typography/reset/theme token → broad representative-page set;
- SCC/cycle → invalidate SCC as one unit;
- dynamic/unknown ownership → conservative expansion;
- critical routes may force wider checks regardless of locality.

## Graph implementation requirements

- deterministic ordering;
- forward/reverse adjacency;
- SCC + condensation DAG;
- bounded dirty propagation;
- cheap incremental update;
- stable IDs;
- immutable/read-optimized snapshot for queries;
- no browser/renderer/framework runtime dependency in the kernel;
- no generic graph framework beyond demonstrated need.

## Performance gates

Hypotheses to validate:

- 1k-node local impact p95 <1 ms;
- 10k-node local impact p95 <5 ms;
- 100k-node bounded local impact p95 <20 ms;
- no full traversal on a known leaf edit;
- incremental/full recompute parity;
- false-negative mutation suite passes.

## Tests

- deterministic map-order independence;
- cycles/SCC;
- import/re-export cycles;
- CSS token fanout;
- component fanout;
- route aliasing;
- stale runtime refs;
- dynamic import fallback;
- conservative fallback;
- incremental vs full recomputation;
- mutation tests aimed specifically at missed impact.

## Exit gate

Local UI edits no longer force whole-site validation by default, and impact false negatives are covered by an explicit mutation/adversarial suite.

---

# 3. AutoTraceLab optional graph-primitive reuse gate (P3/reference only)

AutoTraceLab is primarily a block/process diagram construction, tracing and simulation system. Its process DAG/scene/scheduler semantics are **not** UiUxMaster frontend dependency semantics.

Therefore:

- no `AutoTraceLab` runtime dependency in the first production UiUxMaster;
- no `autotrace_adapter.go` as the primary impact implementation;
- no import of React UI, process simulator, scheduler, scene model or block-domain types;
- no assumption that AutoTraceLab already solves source→component→DOM-region impact analysis.

Only after §2 is implemented and benchmarked may individual graph algorithms be inspected, such as:

- deterministic SCC;
- condensation DAG;
- reverse-adjacency utilities;
- incremental/dirty recomputation;
- deterministic graph ordering/snapshot helpers.

A primitive may be adopted only if:

1. it is genuinely domain-neutral;
2. it can be isolated cleanly;
3. UiUxMaster public/domain types remain unchanged;
4. correctness/mutation fixtures remain green;
5. p50/p95/p99 + allocations are equal or better;
6. maintenance complexity is lower than native code;
7. license/provenance is compatible.

Otherwise keep the native implementation.

---

# 4. Integration B — Axiom control plane (P0 after data-plane stabilization)

## Objective

Use Axiom for longer-lived, explainable workflow lifecycles, not for individual renderer operations.

## Prerequisite

1. Upgrade UiUxMaster to Go 1.26+.
2. Re-run `go test ./...`.
3. Add/run `go test -race ./...`.
4. Run `go vet ./...`.
5. Pin an Axiom version/commit.
6. Add compatibility smoke tests.

## Workflow candidates

- `DesignPolishRun`
- `CandidateComparisonRun`
- `TruthPathCalibrationRun`
- `CrossBrowserReleaseRun`
- `DesignEvalRun`
- `SkillPromotionRun`

Do not wrap:

- one WGGo render;
- one CDP snapshot;
- one ROI screenshot;
- one pixel diff;
- one geometry check.

## DesignPolishRun state

```text
run_id
goal
phase
iteration
impact_digest
evidence_digest
open_finding_ids
resolved_finding_ids
candidate_ids
budget
status
```

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

## Claims/invariants

- no completion with blocking findings;
- no TruthPath verification without matching evidence digest;
- iteration/budget limits;
- correctness/accessibility regression prevents aesthetic win;
- collector failure != PASS;
- external effects use stable idempotency IDs.

## Activities

Typed activities may request:

- FastRender/FastBrowser evidence;
- TruthPath scenario;
- semantic critic;
- host repair application;
- evidence persistence;
- SncSinCore query/admission;
- SkillState projection/evolution evaluation.

## Budgets

```text
max_iterations
max_total_latency
max_vlm_calls
max_truthpath_runs
max_candidates
max_repair_attempts
```

## Storage

Start in-memory. Add Pebble/durable execution only when crash-resume value is demonstrated. Large evidence stays outside workflow state by digest/URI.

## Exit gate

Longer UI polish/eval workflows become reproducible/explainable without measurable L0–L2 hot-path regression.

---

# 5. Integration C — SncSinCore epistemic design memory (P0/P1)

## Objective

Turn accumulated UI/UX evidence into admitted reusable knowledge rather than raw logs or unbounded vector chunks.

Start with embedded `epmemory`. Activate `memoryv2` only after measured scale pressure.

## Ontology

Node types:

- DesignFinding
- DesignRule
- RepairPattern
- Counterexample
- ComponentPattern
- ProductProfile
- EvidenceArtifact
- RenderEnvironment
- EvaluationResult
- ResearchSource

Relationships:

- evidence_for
- refutes
- generalizes
- observed_on
- caused_by
- repaired_by
- improves_axis
- regresses_axis
- applicable_to
- counterexample_to
- derived_from

## Admission

```text
runtime observation
→ evidence.Packet
→ candidate memory atoms
→ provenance/scope/time gates
→ admitted SncSinCore knowledge
```

Required provenance includes run ID, evidence digest, renderer/fidelity, environment, rule/critic version, project/profile scope and outcome.

## Retrieval

Use a narrow local port:

```go
type Memory interface {
    Query(ctx context.Context, q Query, budget Budget) (MemoryView, error)
    Propose(ctx context.Context, candidates []Candidate) error
}
```

Retrieve explicit requirements/targets rather than generic top-k similarity.

## Context budget

Critic should receive only a minimal sufficient view, e.g.:

- <=5 validated similar cases;
- <=3 counterexamples/refutations;
- applicable invariants;
- unresolved conflicts;
- provenance refs.

## Namespace firewall

```text
knowledge/global-design
knowledge/project/<id>
evidence/project/<id>
research/global
skillmeta/<skill-id>
```

Project-private evidence never leaks into global design memory.

## Tests

- deterministic query/artifact replay;
- scope leakage;
- temporal/provenance gates;
- conflict preservation;
- query budgets;
- injection-resistant context packing;
- retraction propagation;
- memory-enabled vs memory-disabled held-out eval.

## Exit gate

Memory measurably improves critique/repair success without context explosion or scope leakage.

---

# 6. Integration D — SkillState bounded working state and controlled evolution (P1)

## Objective

Keep long polish trajectories bounded and enable evidence-gated evolution of design skills.

SkillState is a projection, not canonical truth and not a scheduler.

## Working projection

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

Large artifacts/history remain outside by digest/URI.

## Typed patches/CAS

Require expected revision + digest. Rejected/stale patch mutates nothing.

## MemoryPort

```text
immutable Spec P
+ bounded Σ
+ latest observation O
+ verified SncSinCore MemoryView on demand
```

No implicit long-history replay.

## Oscillation

Track compact hypothesis/result IDs and detect repeated repair loops. Escalate to higher-level redesign instead of repeating the same local patch.

## Controlled evolution

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

Security/privacy/authorization gates are never self-modifying.

Promotion requires unrelated fixtures, target-axis uplift, correctness/accessibility non-regression, acceptable false positives, latency/token measurements, provenance, counterexamples and validated rollback.

## Exit gate

50+ step trajectories remain bounded and no candidate rule becomes active without reproducible evidence.

---

# 7. Integration E — DeepSearch research plane (P2 optional)

## Objective

Refresh UI/UX/standards knowledge without putting Python/browser research in the local render hot path.

DeepSearch remains an optional sidecar/provider.

## Trigger conditions

- new product/domain profile;
- current WCAG/browser/standard question;
- unfamiliar design pattern;
- periodic knowledge refresh;
- benchmark/source verification;
- explicit evidence request.

Do not invoke on every edit.

## Adapter

```go
type Researcher interface {
    Research(ctx context.Context, req ResearchRequest) (ResearchBundle, error)
    InspectSource(ctx context.Context, url string) (SourceAssessment, error)
}
```

REST or stdio sidecar may implement it. Base UiUxMaster must work without DeepSearch installed.

## Admission

```text
DeepSearch bundle
→ source/provenance validation
→ claim extraction
→ memory candidates
→ SncSinCore gates
→ validated knowledge
```

Research output never mutates active design rules directly.

Use cache/staleness policies, explicit budgets, cancellation and prompt-injection-safe handling.

---

# 8. IRIS compatibility patterns (P2)

Do not import IRIS Studio as an application dependency.

If multiple projects need compatibility, define a small stable schema around:

```text
Claim
Evidence
Artifact
Provenance
Confidence
Scope
```

Activation gate: demonstrated cross-project duplication/interop need.

---

# 9. RepoArk and WebGate activation gates (P3)

## RepoArk

No runtime dependency. Consider for benchmark/release artifact archival, mirroring and reproducibility only when history management becomes an operational issue.

## WebGate

No local-runtime dependency. Consider only when remote browser/device workers are a committed requirement.

Future remote workers must use authenticated transport, authorization, bounded retries, idempotency and provenance.

---

# 10. Integration order

```text
E0 Compatibility/toolchain baseline
 ↓
E1 Native UiUxMaster Impact Engine
 ↓
E2 Ultra-fast WGGo/CDP data plane stabilization
 ↓
E3 Axiom control plane
 ↓
E4 SncSinCore memory
 ↓
E5 SkillState bounded state
 ↓
E6 Controlled evolution + optional DeepSearch
 ↓
E7 Hardening / optional ecosystem reuse
```

AutoTraceLab is not a stage in the critical path. Its graph primitives may be reviewed only inside E7 or after the E1 native benchmark baseline is complete.

---

# 11. Cross-integration contract

A single polish run should conceptually flow as:

```text
agent/source patch
   ↓
UiUxMaster Impact Engine
   ↓ ImpactSet
validation router
   ↓
WGGo / resident Chromium / TruthPath
   ↓ EvidencePacket
Deterministic verifiers
   ↓
SkillState bounded projection ← optional SncSinCore MemoryView
   ↓
semantic critic / repair decision
   ↓
Axiom/host workflow when multi-step control is required
   ↓
repair + revalidate
   ↓
accepted evidence
   ↓
SncSinCore admission candidates
   ↓
SkillState replay/shadow evolution candidates
```

DeepSearch can feed source-backed knowledge into SncSinCore admission, never directly into active rules.

---

# 12. Shared tests

Every integration phase adds cross-system tests for:

- cancellation propagation;
- deadline/budget propagation;
- context shutdown/no goroutine leaks;
- bounded payloads;
- deterministic ordering;
- stale reference/digest handling;
- evidence provenance preservation;
- local/privacy namespace isolation;
- dependency-disabled fallback behavior;
- no L0–L2 latency regression beyond explicit budget.

Also run:

```text
go test ./...
go test -race ./...
go vet ./...
```

where supported by the active toolchain.

---

# 13. Success criteria

## Impact engine

- % local edits avoiding whole-page validation;
- p50/p95/p99 impact latency;
- false-negative scope rate;
- incremental/full parity;
- allocations/update.

## Axiom

- replay/recovery success;
- duplicate effects = 0;
- explain/history completeness;
- no hot-path latency regression.

## SncSinCore

- held-out repair-success uplift;
- context token reduction;
- scope leakage = 0;
- conflict/retraction correctness.

## SkillState

- bounded state across long runs;
- stale-patch rejection;
- oscillation detection;
- promotion non-regression.

## DeepSearch

- research freshness/coverage;
- provenance completeness;
- local UI validation remains available with sidecar disabled.

## Whole system

- time from source patch to actionable evidence;
- solved design defects/session;
- VLM calls/solved defect;
- TruthPath escalations/edit;
- regression escape rate;
- human/held-out preference uplift.

---

# 14. Anti-overengineering rules

1. Do not use Axiom around individual renderer primitives.
2. Do not make SkillState a scheduler.
3. Do not store operational truth in SncSinCore.
4. Do not invoke DeepSearch in the local hot loop.
5. Do not import AutoTraceLab application/process semantics into UiUxMaster.
6. Do not make AutoTraceLab a mandatory dependency of the impact engine.
7. Do not build a generic graph framework unless frontend impact requirements prove the need.
8. Do not duplicate full evidence/history across planes; use digests/references.
9. Do not allow VLM output to become a canonical rule without replay/shadow gates.
10. Do not activate RepoArk/WebGate before their product need exists.
11. Do not retain an integration that fails its before/after benchmark or eval.

---

# 15. Immediate integration execution order

1. Upgrade/requalify Go 1.26+ and race CI.
2. Freeze latency/correctness baseline.
3. Define native `internal/impact` contracts and fixtures.
4. Implement native graph kernel + frontend analyzers.
5. Build impact mutation/false-negative suite.
6. Benchmark native impact engine.
7. Stabilize WGGo/CDP data plane.
8. Integrate Axiom control plane.
9. Integrate SncSinCore admission/retrieval.
10. Integrate SkillState bounded projection.
11. Add controlled skill evolution.
12. Add DeepSearch as optional research provider.
13. Only then inspect isolated AutoTraceLab graph primitives if a measured native gap remains.
14. Run bare-vs-integrated held-out benchmark and remove any integration that does not justify itself.
