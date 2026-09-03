# UiUxMaster Ecosystem Integration Program

This document is subordinate to `MASTER_PLAN.md`. It defines how UiUxMaster should integrate selected Homiakus libraries without turning the hot UI-validation path into a distributed framework stack.

## 0. Goal

Use existing libraries only where they provide a proven architectural capability:

- **AutoTraceLab** → impact/dependency graph and incremental invalidation ideas/kernel;
- **Axiom** → durable control plane for long-running design/verification workflows;
- **SncSinCore** → evidence-backed long-term design memory and bounded retrieval;
- **SkillState** → bounded typed working state and controlled evolution of design skills;
- **DeepSearch** → optional external research/acquisition plane;
- **IRIS patterns** → evidence/claim/artifact/provenance compatibility, not a runtime dependency;
- **RepoArk/WebGate** → later operational integrations only after activation gates are met.

The central rule is:

```text
Fast data plane != control plane != memory plane != research plane.
```

No library may own more than one of these planes unless an explicit ADR proves the need.

---

# 1. Dependency and ownership rules

## 1.1 Hot-path rule

The 20–100 ms target loop may depend only on in-process, bounded-cost components:

```text
change → impact graph → fidelity router → WGGo/CDP → deterministic verifier → evidence packet
```

Do **not** call Axiom durable activities, SncSinCore retrieval, SkillState evolution, DeepSearch, external MCP, or cloud services on every geometry/pixel check.

## 1.2 Canonical ownership

- UiUxMaster `internal/evidence` owns the runtime evidence wire/domain contract.
- AutoTrace-derived graph owns impact relationships, not design truth.
- Axiom owns workflow execution/history for selected long-running operations, not renderer state.
- SncSinCore owns admitted long-term epistemic memory, not current operational state.
- SkillState owns bounded LLM working projection, not durable workflow history or canonical facts.
- DeepSearch owns acquisition/research, not canonical design rules.
- MCP is a protocol adapter only.

## 1.3 No vendor leakage

Domain packages must depend on local ports/interfaces. External libraries remain behind adapters:

```text
internal/impact      ← AutoTrace adapter
internal/control     ← Axiom adapter
internal/knowledge   ← SncSinCore adapter
internal/skillstate  ← SkillState adapter
internal/research    ← DeepSearch adapter
```

## 1.4 Version pinning

All pre-v1 dependencies must be pinned to an explicit tag/commit and upgraded through a compatibility test matrix.

Axiom currently requires Go 1.26+, so UiUxMaster must upgrade and validate its toolchain before direct integration.

---

# 2. Integration A — AutoTraceLab impact graph (P0)

## Objective

Convert source changes into the smallest conservative validation scope.

```text
changed file/token
→ module
→ component
→ component instance
→ semantic DOM ref
→ render region
→ viewport/theme/scenario
```

## Important boundary

Do not import the AutoTraceLab React application. Its useful concepts are dependency DAGs, deterministic graph algorithms, incremental SceneEngine-style dirty propagation, geometry/indexing patterns and stable ordering.

Before UiUxMaster depends on it, extract or expose a small domain-neutral Go graph package with a stable API.

## Proposed upstream extraction

Candidate package in AutoTraceLab:

```text
pkg/impactgraph/
    graph.go
    builder.go
    reverse_index.go
    scc.go
    dirty.go
    query.go
    snapshot.go
```

Required characteristics:

- deterministic node/edge ordering;
- forward and reverse adjacency;
- SCC detection and condensation DAG for cycles;
- bounded dirty propagation;
- immutable/read-optimized snapshot plus cheap delta updates;
- stable IDs;
- serialization/version field;
- no React/browser dependency;
- benchmarks with 100/1k/10k/100k nodes.

## UiUxMaster adapter

```text
internal/impact/
    port.go
    model.go
    autotrace_adapter.go
    ownership.go
    runtime_refs.go
```

Suggested port:

```go
type Resolver interface {
    ApplyChanges(ctx context.Context, changes ChangeSet) (ImpactSet, error)
    ResolveComponent(ctx context.Context, componentID string) (ImpactSet, error)
    ResolveToken(ctx context.Context, tokenID string) (ImpactSet, error)
}
```

## Graph node kinds

At minimum:

- SourceFile
- Module
- StyleSheet
- DesignToken
- Component
- ComponentVariant
- ComponentInstance
- Route/Page
- SemanticElementRef
- RenderRegion
- ViewportProfile
- ThemeProfile
- Scenario

## Edge kinds

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
2. import graph from JS/TS/Go templates where applicable;
3. CSS module/style imports;
4. CSS custom-property definitions and references;
5. explicit component/test/design IDs;
6. route/story registry;
7. runtime DOM annotations discovered by FastBrowser.

Do not require a perfect whole-program frontend compiler before useful invalidation works.

## Invalidation policy

- local component style → affected instances only;
- shared component → all known instances on representative pages;
- local design token → consumers;
- global typography/reset/theme token → broad representative-page invalidation;
- unknown ownership → conservative page/site expansion;
- cycle/SCC → invalidate the SCC as one unit.

## Performance gates

Targets to validate:

- 1k-node local change impact p95 < 1 ms;
- 10k-node local change p95 < 5 ms;
- 100k-node bounded local change p95 < 20 ms;
- zero full-graph traversal on a known local leaf change;
- memory budget documented.

## Tests

- deterministic output independent of map iteration;
- cycle/SCC fixtures;
- token fanout;
- component fanout;
- stale runtime-ref invalidation;
- false-negative mutation suite;
- conservative fallback tests;
- incremental vs full recomputation parity.

## Exit gate

A local CSS/component edit no longer causes whole-page/site validation unless graph uncertainty requires it.

---

# 3. Integration B — Axiom control plane (P0, after hot data-plane contracts)

## Objective

Use Axiom for **longer-lived, explainable workflows**, not for each renderer operation.

Axiom's declarative `model` frontend is preferred for new Go code; use typed activities for external effects.

## Toolchain prerequisite

1. Upgrade UiUxMaster from Go 1.25 to Go 1.26+.
2. Update CI matrix.
3. Run `go test ./...`, `go test -race ./...`, `go vet ./...`.
4. Pin an Axiom tag/commit.
5. Add a compatibility smoke test before enabling workflows.

## Workflow candidates

Use Axiom for:

- `DesignPolishRun`;
- `CandidateComparisonRun`;
- `TruthPathCalibrationRun`;
- `CrossBrowserReleaseRun`;
- `DesignEvalRun`;
- `SkillPromotionRun`.

Do not use Axiom for:

- `getBoundingClientRect`;
- one `DOMSnapshot`;
- one WGGo render;
- one ROI screenshot;
- one pixel diff.

## DesignPolishRun state

Suggested state:

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

Suggested events:

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

Examples:

- cannot complete while blocking findings remain;
- cannot declare TruthPath verified without corresponding evidence digest;
- iteration <= configured budget;
- candidate cannot win if a protected accessibility/correctness axis regresses;
- external action IDs are idempotent;
- a collector failure cannot transition to PASS.

## Activities

Use typed activities for:

- request FastRender/FastBrowser evidence;
- request TruthPath scenario;
- run semantic critic;
- apply/record a repair through host adapter;
- persist evidence bundle;
- query SncSinCore;
- invoke SkillState projection/evolution candidate evaluation.

Activities expose **one attempt**; Axiom owns retry/timeout/idempotency policy where applicable.

## Budget policy

Represent explicit run budgets:

```text
max_iterations
max_total_latency
max_vlm_calls
max_truthpath_runs
max_candidates
max_repair_attempts
```

Routing should use remaining budget + uncertainty to choose escalation.

## Storage strategy

- in-memory mode for local ephemeral polish run initially;
- Pebble/durable store only when crash-resume provides measured value;
- avoid persistence in the tens-of-ms hot loop;
- evidence blobs remain in evidence/artifact store, referenced by digest/URI.

## Tests

- transition table tests;
- invariant/claim mutation tests;
- cancellation;
- retry/idempotency;
- crash/replay reconstruction if durable mode enabled;
- collector failure != pass;
- budget exhaustion;
- exact explain/history output for representative runs.

## Exit gate

A multi-step polish/eval run is reproducible, explainable and resumable where configured without increasing ordinary L0–L2 hot-path latency.

---

# 4. Integration C — SncSinCore epistemic design memory (P0/P1)

## Objective

Turn accumulated UI/UX runs into evidence-backed reusable knowledge rather than raw logs or vector chunks.

Use SncSinCore first through its in-memory `epmemory` mode. Move to `memoryv2` only after corpus scale requires it.

## Memory ontology

Node types should include:

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

## Admission pipeline

Do not ingest every run as truth.

```text
runtime observation
→ evidence.Packet
→ candidate memory atoms
→ admission validation
→ provenance/scope/time gates
→ SncSinCore corpus
```

Required provenance:

- UiUxMaster run ID;
- evidence digest;
- renderer/fidelity source;
- browser/environment where relevant;
- rule/critic version;
- accepted/rejected outcome;
- originating project/profile scope.

## Retrieval contract

UiUxMaster should expose a narrow local port:

```go
type Memory interface {
    Query(ctx context.Context, q Query, budget Budget) (MemoryView, error)
    Propose(ctx context.Context, candidates []Candidate) error
}
```

Use explicit requirements/targets so retrieval answers questions such as:

```text
What validated repairs exist for weak hero typography hierarchy
in dark SaaS landing pages, and what counterexamples exist?
```

rather than generic similarity search.

## Context budget

Critic input should receive a minimal sufficient `ContextPack`, for example:

- <= 5 validated similar cases;
- <= 3 counterexamples/refutations;
- applicable invariants;
- unresolved evidence conflicts;
- provenance references.

Never dump the whole design corpus into the model.

## Scope/security

Namespaces:

```text
knowledge/global-design
knowledge/project/<id>
evidence/project/<id>
research/global
skillmeta/<skill-id>
```

Apply scope authorization before retrieval. Project-private screenshots/text must not leak into global design memory.

## Conflict preservation

If evidence disagrees, preserve competing evidence and confidence. Do not average disagreement into one universal rule.

## Storage scale

Stage 1: `epmemory` embedded snapshot.

Stage 2 activation gate for `memoryv2`:

- corpus exceeds agreed node/latency threshold;
- memory pressure becomes measurable;
- incremental ingestion requires segmented persistence.

## Tests

- deterministic artifact/query replay;
- scope leakage tests;
- temporal/provenance gates;
- conflict retention;
- query budget enforcement;
- prompt-injection-resistant context packing;
- retraction invalidates derived knowledge;
- retrieval improves eval outcome vs no-memory baseline.

## Exit gate

Historical runs measurably improve repair/critique success on held-out UI fixtures without leaking scope, exploding context, or promoting unsupported claims.

---

# 5. Integration D — SkillState bounded working state and controlled skill evolution (P1)

## Objective

Prevent long polish trajectories from growing context linearly and create an evidence-gated path for improving design skills.

SkillState is a projection, **not source of truth**. It must not become a second scheduler; Axiom/host owns workflow execution and retries.

## Working-state projection

Define a UiUxMaster skill state such as:

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

Keep large screenshots, DOM, history, traces and full findings outside state by digest/URI.

## Typed patch model

Model output may propose only typed incremental changes to working state. Enforce expected revision + digest CAS.

Rejected patch mutates nothing.

## MemoryPort

Connect SkillState to SncSinCore through a narrow memory port:

```text
bounded working state Σ
+ latest observation O
+ verified MemoryView on demand
```

No implicit conversation-history replay.

## Oscillation/repeated-failure memory

Track compact hypothesis/result IDs:

```text
repair A → broke B
repair B → restored A
```

If a loop is detected, set an escalation flag requiring a higher-level design reconsideration rather than another local patch.

## Controlled evolution pipeline

```text
Observation
→ CandidateHeuristic
→ CandidateSkillVersion
→ ReplayEval
→ ShadowEval
→ NonRegressionGate
→ AuthorizedPromotion
→ Rollback-capable ActiveVersion
```

Never self-modify security/privacy/authorization gates.

## Candidate evidence requirements

Before promotion require:

- multiple unrelated UI fixtures;
- positive effect on target rubric axes;
- no unacceptable accessibility/correctness regression;
- false-positive rate within threshold;
- latency/token impact measured;
- counterexamples included;
- provenance retained;
- rollback package validated.

## Tests

- state token/atom budget;
- revision/digest CAS;
- stale patch rejection;
- crash-safe reconstruction from host state;
- memory namespace firewall;
- replay evaluator determinism;
- shadow mode cannot mutate active skill;
- promotion rollback;
- mutation tests for all evolution gates.

## Exit gate

Long polish runs maintain bounded model context, and new design heuristics can only become active after reproducible replay/shadow evidence.

---

# 6. Integration E — DeepSearch research plane (P2, optional install)

## Objective

Continuously refresh design/UX/standards knowledge without putting Python/browser research in the local render hot path.

DeepSearch already provides adaptive cost-tier acquisition and MCP/REST interfaces. Treat it as an optional sidecar/provider.

## Trigger conditions

Invoke only for explicit or policy-triggered research needs:

- new product/domain profile;
- current WCAG/browser/design-system requirement;
- unfamiliar design pattern;
- periodic standards/research refresh;
- benchmark/source verification;
- user asks for evidence/references.

Do not invoke on every UI edit.

## Adapter

```text
internal/research/
    port.go
    deepsearch_client.go
    admission.go
```

Provider can use local REST or stdio sidecar; default UiUxMaster installation must remain usable without Python/DeepSearch.

Suggested contract:

```go
type Researcher interface {
    Research(ctx context.Context, req ResearchRequest) (ResearchBundle, error)
    InspectSource(ctx context.Context, url string) (SourceAssessment, error)
}
```

## Admission path

```text
DeepSearch research bundle
→ source/provenance validation
→ claim extraction
→ candidate knowledge
→ SncSinCore admission gates
→ validated design knowledge
```

DeepSearch output is evidence input, not automatically a design rule.

## Caching/budgets

- content-addressable cache by query + source version where possible;
- explicit latency/page/token budget;
- reuse recent research artifact before crawling again;
- store admitted knowledge in SncSinCore, not duplicate research blobs in prompt state.

## Tests

- sidecar unavailable → UiUxMaster still works;
- timeout/cancellation;
- duplicate source dedup;
- stale source handling;
- malicious/prompt-injected web content remains untrusted data;
- source citation/provenance survives admission;
- research update cannot mutate active skill directly.

## Exit gate

Research can refresh knowledge asynchronously/explicitly with provenance, while local edit/render validation remains independent of DeepSearch availability.

---

# 7. IRIS-compatible evidence/provenance patterns (P2, no direct dependency initially)

Adopt/align contracts for:

```text
Claim
Evidence
Artifact
Provenance
Confidence
Scope
```

Do not import the full IRIS Studio application into UiUxMaster.

Prefer a small compatibility package or schema mapping if cross-project exchange becomes necessary.

Every significant design conclusion should be traceable:

```text
claim: CTA overlaps navigation
→ geometry evidence
→ screenshot crop
→ renderer identity/fidelity
→ verifier version
→ run
```

Activation gate for a shared package: at least two projects need the same stable contract and schema divergence is already causing measurable maintenance cost.

---

# 8. RepoArk and WebGate activation gates (P3)

## RepoArk

Do not add to runtime. Consider only for:

- benchmark artifact archival;
- release backup/mirroring;
- reproducibility snapshots.

Activation gate: UiUxMaster has release artifacts/benchmark history whose loss/reproducibility is an operational problem.

## WebGate

Do not add to local-first runtime now. Consider when UiUxMaster gains remote browser/device workers.

Potential future topology:

```text
UiUxMaster control plane
→ authenticated remote worker transport
→ Windows Chrome / Linux Chromium / macOS Safari / Android workers
```

Activation gate:

- remote workers are a committed product requirement;
- local runner is insufficient;
- transport resilience is benchmarked as a real need.

The remote transport must preserve authorization, mTLS/authentication, bounded retries, idempotency and evidence provenance.

---

# 9. Integrated target architecture

```text
                         MCP Host / Coding Agent
                                  │
                                  ▼
                          UiUxMaster MCP
                                  │
                                  ▼
                     ┌──── Axiom Control Plane ────┐
                     │                              │
                     ▼                              ▼
            Validation Router                 SkillState Σ
                     │                              │
          ┌──────────┼───────────┐                  │
          ▼          ▼           ▼                  │
       L1 WGGo    L2 FastCDP   L3 TruthPath         │
          │          │           │                  │
          └──────────┼───────────┘                  │
                     ▼                              │
              Evidence Packet                      │
                     │                              │
       ┌─────────────┼──────────────┐               │
       ▼             ▼              ▼               │
 AutoTrace impact  deterministic  local VLM        │
 graph/scope        verifiers      critic           │
       │             │              │               │
       └─────────────┴──────┬───────┘               │
                            ▼                       │
                       repair decision ─────────────┘
                            │
                            ▼
                         Axiom
                            │
                     evidence/admission
                            ▼
                       SncSinCore
                            ▲
                            │
                      DeepSearch (optional)
                            │
                      research evidence
```

Important: AutoTrace impact resolution belongs before expensive evidence collection. SncSinCore retrieval belongs before semantic critique only when historical knowledge is useful. Neither is required for a deterministic local geometry check.

---

# 10. Ordered implementation sequence

## E0 — Compatibility baseline

1. Pin current UiUxMaster benchmark/CI baseline.
2. Upgrade Go toolchain to 1.26+ to unlock Axiom integration.
3. Add `go test -race ./...` CI job.
4. Add dependency/license inventory.
5. Record exact versions/commits for Axiom, SncSinCore, SkillState and any extracted AutoTrace module.
6. Add ADR: ecosystem ownership boundaries.

## E1 — AutoTrace impact graph

1. Define UiUxMaster `ImpactGraph` port and fixtures first.
2. Extract/stabilize a domain-neutral AutoTrace Go graph package or implement a compatible adapter around an existing stable kernel.
3. Build source/import/CSS-token analyzers.
4. Build reverse indexes and SCC/dirty propagation.
5. Bind runtime semantic refs from CDP/WGGo evidence.
6. Feed `ImpactSet` into validation router.
7. Benchmark local vs broad changes.
8. Add false-negative mutation suite.

## E2 — Axiom control plane

1. Define `DesignPolishRun` state/events/claims.
2. Implement typed activities around existing UiUxMaster ports.
3. Keep data-plane operations direct/in-process.
4. Add budgets/cancellation/idempotency.
5. Run representative workflows in-memory.
6. Add durable Pebble only after a crash-resume use case is proven.
7. Expose workflow explain/history through internal diagnostics, not huge MCP output.

## E3 — SncSinCore memory

1. Define design-memory ontology and namespaces.
2. Add evidence→candidate admission mapper.
3. Start embedded `epmemory` corpus.
4. Query minimal ContextPacks under explicit budgets.
5. Feed memory only to semantic critic/repair planning where useful.
6. Add conflict/scope/retraction tests.
7. Compare eval performance with memory on/off.
8. Activate `memoryv2` only at measured scale threshold.

## E4 — SkillState bounded state

1. Define typed `UiUxSkillState` and patch schema.
2. Project Axiom/domain state into bounded Σ.
3. Store large artifacts by digest/reference only.
4. Connect SncSinCore `MemoryPort`.
5. Add stale/CAS/oscillation protections.
6. Replace implicit long conversation replay in polish agent context.
7. Benchmark token reduction and task success.

## E5 — Controlled evolution

1. Collect candidate heuristics from repeated evidence-backed patterns.
2. Keep candidate skill versions immutable.
3. Build replay corpus from adversarial and real fixtures.
4. Run current-vs-candidate A/B/shadow evals.
5. Gate on design improvement + non-regression + latency/token cost.
6. Require authorized promotion.
7. Validate rollback before activation.

## E6 — DeepSearch research adapter

1. Keep optional feature flag/sidecar.
2. Implement bounded `Researcher` port.
3. Add source/provenance admission.
4. Feed accepted claims into SncSinCore, not directly into active skill.
5. Add cache/staleness policy.
6. Add periodic/manual research refresh workflow through Axiom only if useful.

## E7 — Ecosystem hardening

1. Full race tests.
2. Fault injection across Axiom activities, memory, renderer and research adapters.
3. Benchmarks proving no regression in L0–L2 latency.
4. Dependency upgrade compatibility suite.
5. Privacy/scope isolation tests.
6. License/provenance report.
7. End-to-end eval comparing bare UiUxMaster vs integrated system.

---

# 11. Success metrics

The integrations are justified only if they improve measurable outcomes.

## Impact graph

- percent of edits validated without whole-page/site scan;
- invalidation p95;
- false-negative scope rate;
- average affected region count.

## Axiom

- workflow recovery/replay success;
- duplicate side-effect rate = 0;
- explainability/history completeness;
- no measurable hot-path latency regression.

## SncSinCore

- repair success uplift on held-out fixtures;
- context tokens saved vs naive history/retrieval;
- scope leakage = 0;
- conflict-preservation tests pass.

## SkillState

- bounded prompt state across 50+ iteration trajectories;
- stale patch rejection rate/behavior correct;
- oscillation detection precision;
- candidate promotion regression rate.

## DeepSearch

- research freshness/coverage;
- admitted-source provenance completeness;
- no dependency of ordinary local validation on research service availability.

## Whole system

- UI polish success rate;
- regression escape rate;
- p50/p95/p99 validation latency by tier;
- VLM calls per successful repair;
- TruthPath escalations per edit;
- tokens/cost per solved design defect;
- user/independent critic preference vs baseline.

---

# 12. Non-goals / anti-overengineering guards

- Do not create a second orchestration engine inside SkillState.
- Do not run Axiom dispatch around every renderer/CDP primitive.
- Do not turn SncSinCore into operational state storage.
- Do not make DeepSearch mandatory for local operation.
- Do not import the AutoTraceLab UI/React application into UiUxMaster.
- Do not duplicate the same evidence/history in Axiom, SncSinCore and SkillState; each stores a different projection/reference.
- Do not promote VLM opinions directly into canonical skills.
- Do not introduce RepoArk/WebGate runtime dependencies before their activation gates.
- Do not accept integration complexity without before/after benchmark and eval evidence.
