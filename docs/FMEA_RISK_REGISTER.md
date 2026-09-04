# UiUxMaster Architecture FMEA Risk Register

- Status: **ACTIVE**
- Initial review: 2026-09-04
- Scope: architecture, verification integrity, orchestration, memory/evolution, baselines, delivery controls
- Planning authority: `MASTER_PLAN.md` remains the single execution-order authority. This register is the subordinate risk ledger used by plan tasks, issues, architecture review and release gates.
- Decision record: `docs/adr/0002-fmea-risk-governance.md`

## 1. Do not confuse two different risk domains

UiUxMaster already has `internal/fidelity.RiskLevel`. That risk answers:

> How likely is an approximate renderer to diverge from browser truth, and which evidence tier is required?

This FMEA register answers:

> How can the architecture, verifier, control plane, memory, repair loop or delivery process fail, what is the effect, how likely is it, how hard is it to detect, and what evidence is required to reduce the risk?

`fidelity.RiskLevel` remains runtime routing data. `FMEA-###` IDs are engineering/planning data. They MUST NOT be overloaded into the same field or enum.

---

## 2. Scoring model

Use 1-10 integer scores:

- **Severity (S)**: 1 = negligible, 10 = can invalidate system trust, create destructive behavior, leak protected data, or produce a consequential false PASS.
- **Occurrence (O)**: 1 = exceptional/strongly prevented, 10 = expected/frequent under normal use.
- **Detection difficulty (D)**: 1 = almost certainly detected before escape, 10 = very likely to escape existing controls.
- **RPN** = `S × O × D`.

Priority policy:

| Priority | Default trigger |
|---|---|
| Critical | `RPN >= 250`, or Severity 9-10 with a credible false-PASS/security/destructive-integrity path |
| High | `RPN 120-249` |
| Medium | `RPN 60-119` |
| Low | `RPN < 60` |

RPN is a prioritization aid, not an acceptance rule. A low-occurrence catastrophic mode can still be Critical.

Status vocabulary:

- `OPEN` — failure mode is active and mitigation is not complete;
- `MITIGATING` — implementation is in progress;
- `ACCEPTED` — residual risk was explicitly accepted with rationale and review date;
- `CLOSED` — closure gate passed and residual score/evidence were recorded.

---

## 3. Planning integration contract

Every new or materially changed `MASTER_PLAN.md` execution task that touches a Critical/High risk should include:

```text
Risks: FMEA-001, FMEA-003
Risk action: mitigate | monitor | accept | none
Risk gate: <test/eval/fault-injection/review evidence required>
Residual target: S=<n> O=<n> D=<n> RPN=<n>
```

Rules:

1. Every Critical/High risk has at least one executable GitHub mitigation issue/task.
2. A mitigation task cannot be marked DONE while its FMEA closure gate is unproven.
3. A risk cannot be marked CLOSED because code was merely added; closure requires the named independent evidence.
4. Architecture-changing PRs list affected `FMEA-###` IDs or explicitly state `none` with rationale.
5. New failure modes discovered during implementation receive new stable IDs; do not silently repurpose an old risk.
6. Re-scoring must record why `O` or `D` changed. Severity usually remains unchanged unless the failure effect itself changes.
7. Accepted Critical risk requires explicit maintainer approval and a review/expiry date.
8. Release/final-gate work must not cross an unresolved Critical risk that can invalidate the evidence used by that release gate.

### Architecture-delta triggers

Perform an FMEA delta review whenever a PR changes:

- L0-L4 routing, fallback or legal-PASS semantics;
- impact/invalidation scope semantics;
- warm-state readiness/freshness;
- browser/renderer/model capability claims or versions;
- deterministic verifier semantics;
- autonomous repair/comparison/completion gates;
- Axiom retries/durability/external side effects;
- memory namespaces, admission, evolution or promotion;
- privacy/provenance boundaries;
- protected baseline identity/update policy;
- CI/release gates.

---

## 4. Active risk summary

| ID | Failure mode | S | O | D | RPN | Priority | Status | Mitigation |
|---|---|---:|---:|---:|---:|---|---|---|
| FMEA-001 | Required TruthPath silently downgrades to L2 | 10 | 4 | 8 | 320 | Critical | OPEN | #3 |
| FMEA-002 | Axiom loses canonical change/ImpactSet scope | 10 | 6 | 7 | 420 | Critical | OPEN | #4 |
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10 | 4 | 7 | 280 | Critical | OPEN | #5 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9 | 5 | 8 | 360 | Critical | OPEN | #6 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6 | 10 | 4 | 240 | High | OPEN | #7 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7 | 9 | 3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9 | 3 | 7 | 189 | High | OPEN | #9 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9 | 4 | 7 | 252 | Critical | OPEN | #10 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9 | 5 | 7 | 315 | Critical | OPEN | #11 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10 | 2 | 8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8 | 4 | 4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under an incompatible render environment | 7 | 5 | 5 | 175 | High | OPEN | #14 |

Initial `O` and `D` values are engineering estimates based on the current architecture/code controls, not field incident statistics. Re-score them when production/held-out telemetry becomes available.

---

## 5. Detailed FMEA

### FMEA-001 — Silent TruthPath downgrade

**Failure mode**  
Policy selects L3 TruthPath, but an absent L3 collector causes dispatcher execution on L2 FastBrowser.

**Effect**  
A clean-state, final-gate, cross-browser or calibration requirement can be evaluated using weaker evidence. The worst result is a false PASS with no obvious indication that the required oracle was unavailable.

**Observed mechanism**  
`internal/runtime/dispatcher.Dispatcher.Collect` currently routes `TierTruthPath` to L2 when `d.l3 == nil`.

**Current controls**  
Tier routing exists; architecture documentation distinguishes collector failure from application failure. There is no minimum-actual-tier attestation at the fallback point.

**Initial score**: `S=10 O=4 D=8 RPN=320` — **Critical**  
**Owner boundary**: `internal/runtime/dispatcher`, `internal/engine`  
**Mitigation**: issue #3

**Closure gate**

- L3 unavailable fails closed with typed evidence-insufficiency/collector-unavailable semantics;
- actual packet tier is checked against the policy-selected minimum tier;
- final/clean-state/cross-browser gates cannot PASS from an L2 substitution;
- regression tests preserve only explicitly legal lower-tier escalation behavior.

**Residual target**: `S=10 O=1 D=2 RPN=20`.

---

### FMEA-002 — Axiom canonical-pipeline scope loss

**Failure mode**  
Axiom invokes `engine.Pipeline`, but its compact `Change` projection does not carry changed files/tokens/nodes and related canonical request semantics.

**Effect**  
The workflow appears canonical while `ImpactSet`/`ValidationScope` can be detached from the actual edit. Affected UI can be omitted from verification.

**Observed mechanism**

- `control/axiom/controlplane.Change` lacks canonical change-set fields;
- `control/axiom/uiuxadapter.toValidationRequest` maps intent/final-gate/evidence needs/region but not changed files/tokens/nodes;
- `PlanEvidence` still builds an independent `evidenceplan` before collection.

**Current controls**  
A `Pipeline` adapter exists and direct engine tests exist, but equivalence of Axiom input → canonical scope is not guaranteed by the current schema.

**Initial score**: `S=10 O=6 D=7 RPN=420` — **Critical**  
**Owner boundary**: `control/axiom/controlplane`, `control/axiom/uiuxadapter`, `internal/engine`  
**Mitigation**: issue #4

**Closure gate**

- lossless canonical request/change projection or stable request reference;
- Axiom/direct-pipeline equivalence fixture for changed file and changed token;
- non-empty expected ImpactSet-derived scope reaches runtime collector;
- duplicate planner cannot independently select a different route;
- run identity is caller-stable, not hard-coded across executions.

**Residual target**: `S=10 O=1 D=2 RPN=20`.

---

### FMEA-003 — Render freshness not revision-bound

**Failure mode**  
A monotonic browser epoch advances, but the signal is not cryptographically/logically tied to the source/build revision being validated.

**Effect**  
Wrong/stale content can be captured after an early or incorrectly-associated HMR/app signal and may pass deterministic checks.

**Observed mechanism**  
`EpochGate` stores only a numeric epoch. `window.__UIUX_SIGNAL_RENDER__(epoch)` is an application-level contract. Health checks verify bridge/session health, not correspondence to the requested source change.

**Current controls**  
Monotonic epoch gate, explicit helper, stale page health checks and reset ladder.

**Initial score**: `S=10 O=4 D=7 RPN=280` — **Critical**  
**Owner boundary**: `internal/runtime/fastcdp`, canonical evidence provenance  
**Mitigation**: issue #5

**Closure gate**

- readiness token includes expected/observed source or build revision digest;
- packet provenance records the observed revision;
- wrong-revision and stale-revision fault tests cannot release PASS evidence;
- missing/mismatched revision follows defined recollect/reset/escalation semantics.

**Residual target**: `S=10 O=1 D=2 RPN=20`.

---

### FMEA-004 — TruthPath capability optimism

**Failure mode**  
The adapter reports full browser/scenario/ARIA/font capability even when a real Playwright worker/runtime is not known to be runnable.

**Effect**  
Policy and planning can treat TruthPath as production-ready while execution is absent, externally dependent, or demonstrated only with mocks.

**Observed mechanism**  
`WorkerCmd` defaults to `node`, `WorkerScript` is optional/empty by default, and `Capabilities()` is unconditional. No bundled worker appears in the current package tree.

**Current controls**  
Narrow Go adapter contracts and mock/unit tests; parity test scaffolding exists.

**Initial score**: `S=9 O=5 D=8 RPN=360` — **Critical**  
**Owner boundary**: `internal/runtime/playwright`, runtime bootstrap/CI  
**Mitigation**: issue #6

**Closure gate**

- worker/browser readiness probe drives advertised capability;
- missing worker cannot be routed as operational L3;
- real worker + Chromium CI smoke produces canonical evidence;
- evidence/calibration includes worker and browser versions.

**Residual target**: `S=9 O=1 D=2 RPN=18`.

---

### FMEA-005 — False stage telemetry

**Failure mode**  
Impact and invalidation are shown as separate latency metrics while currently sharing the same combined timer.

**Effect**  
Performance diagnosis and architecture optimization can be based on incorrect subsystem attribution and double-looking metrics.

**Observed mechanism**  
`internal/engine/pipeline.go` assigns `elapsedImpact` to both `ImpactMS` and `InvalidationMS`.

**Initial score**: `S=6 O=10 D=4 RPN=240` — **High**  
**Owner boundary**: `internal/engine`, `internal/invalidation`  
**Mitigation**: issue #7

**Closure gate**  
Independent stage timing with injection tests proving impact-only and invalidation-only delay attribution; total accounting invariant tested.

**Residual target**: `S=6 O=1 D=2 RPN=12`.

---

### FMEA-006 — Planning/documentation state drift

**Failure mode**  
Authoritative high-level status text, detailed DONE tasks, current milestone text and open GitHub issues disagree about what is implemented/integrated/proven.

**Effect**  
Agents/maintainers can repeat completed work, skip unfinished work, or trust capabilities whose task checkbox exceeded their actual DoD.

**Current controls**  
`MASTER_PLAN.md` explicitly declares itself authoritative and requires subordinate synchronization, but this contract is not mechanically enforced.

**Initial score**: `S=7 O=9 D=3 RPN=189` — **High**  
**Owner boundary**: planning/docs/release governance  
**Mitigation**: issue #8

**Closure gate**

- reconcile plan/readme/docs/issues against code and real tests;
- distinguish `implemented`, `integrated`, `operationally proven`, `release-gated`;
- every architectural DONE item has evidence matching its DoD;
- current milestone contains only residual/open work.

**Residual target**: `S=7 O=2 D=2 RPN=28`.

---

### FMEA-007 — Non-idempotent durable side effects

**Failure mode**  
A durable activity is retried/replayed after a failure boundary and repeats an external side effect (repair patch, memory admission, future artifact write).

**Effect**  
Duplicate/double-applied patches, duplicated facts, divergent history and non-reproducible crash recovery.

**Current controls**  
Axiom durable history/store and cancellation exist. Memory has transactional semantics. An end-to-end exactly-once/idempotency proof around all external effects is not established.

**Initial score**: `S=9 O=3 D=7 RPN=189` — **High**  
**Owner boundary**: `control/axiom`, `internal/repair`, `internal/memory`  
**Mitigation**: issue #9

**Closure gate**  
Stable idempotency keys + expected-revision/CAS writes + fault injection after side effect/before completion demonstrates no duplicate semantic effect.

**Residual target**: `S=9 O=1 D=2 RPN=18`.

---

### FMEA-008 — Calibration drift

**Failure mode**  
A legal PASS classification remains valid after renderer/browser/worker/environment behavior changes.

**Effect**  
Systematically stale fidelity assumptions can make an approximate tier trusted beyond its actual envelope.

**Current controls**  
Calibration matrix and parity corpus concepts exist. Risk remains if validity is not keyed to exact runtime/environment identity and freshness.

**Initial score**: `S=9 O=4 D=7 RPN=252` — **Critical**  
**Owner boundary**: `internal/fidelity`, runtime adapters, benchmark/parity artifacts  
**Mitigation**: issue #10

**Closure gate**  
Calibration key includes renderer/browser/worker/environment versions; mismatch/expiry invalidates legal PASS and forces recalibration/escalation.

**Residual target**: `S=9 O=1 D=2 RPN=18`.

---

### FMEA-009 — Repair-loop reward hacking / verifier overfit

**Failure mode**  
The repair proposer optimizes against the same visible verifier/rubric signals that later approve its own completion.

**Effect**  
Measured score improves while hidden interaction/responsive/product constraints regress.

**Current controls**  
Protected axes, relative comparison, adversarial harness and architectural language about independent verification exist. Independence needs to be enforced by execution policy for high-risk/final work.

**Initial score**: `S=9 O=5 D=7 RPN=315` — **Critical**  
**Owner boundary**: `internal/repair`, `internal/critic`, `internal/eval`, TruthPath/final-gate policy  
**Mitigation**: issue #11

**Closure gate**

- candidate generator cannot mark itself complete;
- high-risk/final repair uses independent evidence path;
- hidden/held-out perturbations reject score-gaming patches;
- regression escape rate and held-out repair success are measured.

**Residual target**: `S=9 O=2 D=3 RPN=54`.

---

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode**  
Project-private or incorrect evidence crosses namespace/promotion boundaries and influences another project or global design knowledge.

**Effect**  
Confidentiality breach and systematic incorrect behavior amplified by memory and controlled evolution.

**Current controls**  
Namespace firewall, provenance/conflict/retraction tests, admission policy, replay/shadow/non-regression evolution and rollback.

**Initial score**: `S=10 O=2 D=8 RPN=160` — **High**  
**Owner boundary**: `internal/memory`, `internal/skillstate`, `internal/evolution`  
**Mitigation**: issue #12

**Closure gate**  
Adversarial multi-project isolation/poisoning suite demonstrates zero cross-project leakage and blocks poisoned promotion; retraction propagates correctly.

**Residual target**: `S=10 O=1 D=3 RPN=30`.

---

### FMEA-011 — Ungated main branch

**Failure mode**  
A high-risk verifier/routing/control change lands on `main` without required CI/review/risk-delta evidence.

**Effect**  
One change can silently degrade the verification system before downstream work notices.

**Current controls**  
Root and Axiom CI workflows exist, but the current `main` branch protection reports disabled/no required status checks.

**Initial score**: `S=8 O=4 D=4 RPN=128` — **High**  
**Owner boundary**: repository governance  
**Mitigation**: issue #13

**Closure gate**  
Protected/ruleset `main`, required CI, PR-based high-risk changes, FMEA delta checklist, auditable emergency bypass.

**Residual target**: `S=8 O=1 D=2 RPN=16`.

---

### FMEA-012 — Baseline environment mismatch

**Failure mode**  
Visual regression compares a baseline captured under a different browser/font/DPR/theme/fixture environment.

**Effect**  
Flaky false regressions, missed real regressions, or pressure to broaden tolerance until meaningful defects escape.

**Current controls**  
Versioned baseline abstraction exists/planned; evidence includes parts of renderer/environment provenance. A complete compatibility key must guard comparison.

**Initial score**: `S=7 O=5 D=5 RPN=175` — **High**  
**Owner boundary**: `internal/visualdiff`, TruthPath baseline management  
**Mitigation**: issue #14

**Closure gate**  
Canonical baseline environment key; incompatible key rejects comparison; update operation records old/new digest + reason; deterministic masks are scoped rather than global tolerance inflation.

**Residual target**: `S=7 O=1 D=2 RPN=14`.

---

## 6. Risk burn-down order

Use dependency and failure-effect order, not only descending RPN:

1. **FMEA-001** — fail closed on required L3; otherwise every later final-gate result can be semantically invalid.
2. **FMEA-002** — make Axiom carry the real canonical change scope; otherwise orchestration can validate the wrong surface.
3. **FMEA-004** — make TruthPath operational readiness truthful.
4. **FMEA-003** — bind warm evidence to the requested revision.
5. **FMEA-008** — make calibration expire on runtime/environment drift.
6. **FMEA-009** — enforce independent final verification of autonomous repair.
7. **FMEA-005** — repair telemetry before further performance optimization.
8. **FMEA-007** — prove durable side-effect idempotency before expanding autonomous write workflows.
9. **FMEA-010** — adversarially prove memory/evolution isolation before broad/global reuse.
10. **FMEA-012** — environment-key protected baselines before scaling visual regression corpus.
11. **FMEA-006** — reconcile plan/docs/issues continuously while the above work lands.
12. **FMEA-011** — enforce repository rules so the risk gates themselves cannot be bypassed.

Items 1-6 form the **verification-integrity barrier**. Production-grade claims should not cross that barrier while their corresponding Critical risks are OPEN.

---

## 7. Risk metrics

Track at milestone/release review:

```text
open_critical_risks
open_high_risks
oldest_open_high_risk_days
initial_total_rpn
residual_total_rpn
critical_risk_closure_rate
high_risk_tasks_with_verification_evidence_pct
reopened_risks
false_pass_incidents
collector_downgrade_attempts
scope_equivalence_failures
stale_revision_evidence_rejections
calibration_invalidations
repair_heldout_regression_escape_rate
cross_project_memory_leakage_rate
```

Do not optimize the aggregate RPN by manipulating scoring. Risk scores are explanatory; closure evidence is the actual control.

---

## 8. Closure record template

Append/update the relevant risk entry when closing or accepting:

```text
Status: CLOSED | ACCEPTED
Closed/accepted at: YYYY-MM-DD
Implemented by: <PR/commit/task>
Verification evidence: <tests/eval/artifacts>
Initial: S=<n> O=<n> D=<n> RPN=<n>
Residual: S=<n> O=<n> D=<n> RPN=<n>
Why O changed: ...
Why D changed: ...
Remaining assumptions: ...
Next review: YYYY-MM-DD | not required
Accepted by: <required only for ACCEPTED>
```

A closed risk may be reopened under the same ID when the same failure mode returns (for example after a renderer upgrade). A materially different failure mechanism receives a new ID.
