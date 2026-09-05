# UiUxMaster Architecture FMEA Risk Register

- Status: **ACTIVE**
- Initial review: 2026-09-04
- Last re-score: 2026-09-05
- Scope: architecture, verification integrity, orchestration, memory/evolution, baselines, delivery controls
- Planning authority: `MASTER_PLAN.md`
- Engineering-risk authority: this register
- Machine mirror: `planning/fmea-risk.json`
- Execution overlay: `planning/FMEA_EXECUTION_PLAN.md`
- Governance ADR: `docs/adr/0002-fmea-risk-governance.md`

## 1. Risk-domain separation

`internal/fidelity.RiskLevel` is runtime routing data: how likely approximate evidence is to diverge from browser truth and which execution tier is required.

`FMEA-###` is engineering/planning risk: how architecture can fail, the effect, occurrence, detectability, mitigation, closure evidence, residual score and reopen trigger.

These domains must never share one field or enum.

## 2. Scoring and closure policy

- **Severity (S)**: 1 negligible; 10 can invalidate trust, leak protected data, create destructive behavior or consequential false PASS.
- **Occurrence (O)**: 1 exceptional/strongly prevented; 10 expected/frequent.
- **Detection difficulty (D)**: 1 almost certainly caught before escape; 10 likely to escape controls.
- **RPN** = `S × O × D`.

Default priority:

| Priority | Trigger |
|---|---|
| Critical | `RPN >= 250`, or Severity 9–10 with credible false-PASS/security/destructive path |
| High | `RPN 120–249` |
| Medium | `RPN 60–119` |
| Low | `RPN < 60` |

Status: `OPEN`, `MITIGATING`, `ACCEPTED`, `CLOSED`.

A risk closes only after merged implementation, its named independent closure evidence, residual re-score, synchronized register/machine-state/tracker, and an explicit reopen trigger. Code existence alone never closes risk. Closed risks remain regression guards.

## 3. Risk summary

| ID | Failure mode | Initial S/O/D | Initial RPN | Current S/O/D | Current RPN | Priority | Status | Mitigation |
|---|---|---:|---:|---:|---:|---|---|---|
| FMEA-001 | Required TruthPath silently downgrades to L2 | 10/4/8 | 320 | **10/1/2** | **20** | Low residual | **CLOSED** | #3 / PR #19 |
| FMEA-002 | Axiom loses canonical change/ImpactSet scope | 10/6/7 | 420 | **10/1/2** | **20** | Low residual | **CLOSED** | #4 / PR #17 |
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10/4/7 | 280 | 10/4/7 | 280 | Critical | OPEN | #5 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9/5/8 | 360 | **9/1/2** | **18** | Low residual | **CLOSED** | #6 / PR #18 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6/10/4 | 240 | 6/10/4 | 240 | High | OPEN | #7 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7/9/3 | 189 | 7/9/3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9/3/7 | 189 | 9/3/7 | 189 | High | OPEN | #9 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9/4/7 | 252 | 9/4/7 | 252 | Critical | OPEN | #10 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9/5/7 | 315 | **9/2/3** | **54** | Low residual | **CLOSED** | #11 / PR #20 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10/2/8 | 160 | 10/2/8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8/4/4 | 128 | 8/4/4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under an incompatible render environment | 7/5/5 | 175 | 7/5/5 | 175 | High | OPEN | #14 |

Current milestone metrics:

```text
open_critical_risks = 2
open_high_risks = 6
sum_open_rpn = 1613
sum_closed_residual_rpn = 112
```

## 4. Detailed risks

### FMEA-001 — Silent TruthPath downgrade — CLOSED

**Failure mode:** a policy-selected L3 TruthPath requirement could silently execute on L2 FastBrowser.  
**Effect:** clean-state/final/cross-browser requirements could be reported from weaker evidence and escape as false PASS.  
**Owner:** `internal/runtime/dispatcher`, `internal/engine`.  
**Mitigation:** #3, PR #19.

Implemented controls:

- `TierTruthPath` has no L2 fallback;
- unknown routes fail with `ErrInvalidRoute` rather than selecting L2;
- unavailable selected collectors return typed `ErrCollectorUnavailable`;
- `internal/engine` owns a protocol-neutral monotonic evidence-strength model that normalizes physical packet tiers (`L0`…`L4`) and descriptive route tiers;
- dispatcher checks `actual evidence strength >= policy-selected minimum` before returning a packet;
- canonical `engine.Pipeline` independently repeats the same check before verifier/evaluation, so custom collectors cannot bypass it;
- L1→L2 upward escalation remains legal; downward substitution is rejected;
- L4 semantic is modeled as a post-collection judgement whose collector-side minimum is L2 browser evidence.

Closure evidence:

- PR #19 merged as `e85cc1977493f481e4c76321bd829d297782325f`;
- `TestFMEA001TruthPathUnavailableDoesNotDowngradeToL2`;
- `TestFMEA001TruthPathRejectsWeakerPacketFromConfiguredL3`;
- `TestFMEA001TruthPathAcceptsAttestedL3Packet`;
- `TestFMEA001UnknownRouteDoesNotDefaultToL2`;
- `TestFMEA001UpwardL1ToL2EscalationRemainsLegal`;
- `TestPipelineRejectsWeakCustomCollectorBeforeVerifier`;
- `TestTruthPathDispatcherRealChromium` composes the real runtime-attested Playwright/Chromium L3 path with dispatcher attestation;
- `ci` #204 PASS;
- `axiom-control` #37 PASS;
- `truthpath` #9 PASS.

Re-score: Severity remains 10 because a future downgrade regression could still invalidate a gate. O `4→1` because weaker substitution paths were removed and both dispatcher and Pipeline enforce the lower bound. D `8→2` because missing collector, weak packet, unknown route, custom collector bypass and real L3 composition are executable CI evidence.  
**Residual:** `10/1/2 RPN=20` — target met.  
**Reopen:** any required route can select weaker evidence; unknown route gains a usable fallback; attestation moves after verification/evaluation; or real L3 dispatcher integration fails.

### FMEA-002 — Axiom canonical-pipeline scope loss — CLOSED

**Failure mode:** Axiom entered `engine.Pipeline` through a lossy change projection and could independently narrow canonical evidence need.  
**Owner:** `control/axiom/controlplane`, `control/axiom/uiuxadapter`, `internal/engine`.  
**Mitigation:** #4, PR #17.

Controls/evidence: lossless durable run/project/source/files/tokens/nodes/routes/viewports/themes/base/need payload; no advisory-plan→canonical-need narrowing; direct/Axiom scope+route equivalence; anti-narrowing regression; `ci` #191 and `axiom-control` #33 including real Chrome.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** canonical field loss, hard-coded run identity, independent Axiom tier selection, or equivalence-test failure.

### FMEA-003 — Render freshness not revision-bound — CURRENT

**Failure mode:** monotonic browser epoch is not tied to requested source/build revision.  
**Effect:** stale/wrong content can be captured and pass deterministic checks.  
**Owner:** `internal/runtime/fastcdp`, evidence provenance.  
**Mitigation:** #5.  
**Closure gate:** expected/observed revision digest in readiness and packet provenance; stale/wrong revision cannot release PASS evidence; mismatch has explicit recollect/reset/escalation behavior.  
**Residual target:** `10/1/2 RPN=20`.

### FMEA-004 — TruthPath capability optimism — CLOSED

**Failure mode:** Playwright adapter advertised browser/scenario/ARIA/font capability without a proven runnable worker/runtime/browser.  
**Owner:** `internal/runtime/playwright`, CI.  
**Mitigation:** #6, PR #18.

Controls/evidence: fail-closed pre-probe capabilities; worker protocol `1.0.0`; exact Playwright `1.62.1`; browser advertised only after executable discovery + real launch + version; missing worker unavailable; runtime identity drift rejected; worker/Playwright/browser identity reaches canonical packet provenance; `truthpath` #2 real Chromium and `ci` #197.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** pre-probe capability claim, non-versioned runtime, unlaunched browser advertisement, missing-worker fallback, provenance identity loss or real TruthPath CI failure.  
**Boundary:** stored-calibration lifecycle invalidation remains FMEA-008.

### FMEA-005 — False telemetry split

**Failure mode:** impact and invalidation appear as separate metrics while sharing one combined timing.  
**Effect:** optimization/regression decisions use misleading stage data.  
**Owner:** `internal/engine`, telemetry.  
**Mitigation:** #7.  
**Closure gate:** independent timers/counters and accounting tests/benchmarks prove values originate from distinct stages.  
**Residual target:** `6/1/2 RPN=12`.

### FMEA-006 — Planning/documentation state drift

**Failure mode:** historical status prose contradicts actual implementation/integration/operational evidence.  
**Effect:** engineers depend on overstated capabilities or wrong ownership/naming.  
**Owner:** `MASTER_PLAN.md`, README, architecture docs, issues.  
**Mitigation:** #8.  
**Closure gate:** reconcile claims to `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`; eliminate stale TruthPath/FastPath state; add consistency checks where practical.  
**Residual target:** `7/2/2 RPN=28`.

### FMEA-007 — Duplicate durable side effects

**Failure mode:** durable retry/restart repeats source repair or memory admission after effect success but before completion persistence.  
**Effect:** duplicate/destructive edits or duplicated/contradictory memory facts.  
**Owner:** Axiom workflow, repair writer, memory admission.  
**Mitigation:** #9.  
**Closure gate:** idempotency key/CAS + fault injection after effect-before-completion + replay proving one observable outcome.  
**Residual target:** `9/1/2 RPN=18`.

### FMEA-008 — Calibration drift

**Failure mode:** previously legal calibration remains trusted after renderer/browser/worker/environment changes.  
**Effect:** approximate evidence is accepted against obsolete parity assumptions.  
**Owner:** fidelity/calibration state, runtime identity.  
**Mitigation:** #10.  
**Closure gate:** calibration key includes exact runtime/environment identity; incompatible/expired calibration invalidates automatically; CI covers upgrade/mismatch behavior. FMEA-004 supplies worker/Playwright/browser identity in canonical provenance.  
**Residual target:** `9/1/2 RPN=18`.

### FMEA-009 — Repair verifier overfit / reward hacking — CLOSED

**Failure mode:** autonomous repair optimized against the same visible verifier/rubric/candidate signals that could authorize completion.  
**Effect:** an apparently improved candidate could regress hidden interaction/responsive/product requirements and promote a false-success repair pattern into reusable memory.  
**Owner:** `internal/repair`, comparison/final verification/eval, SncSinCore admission boundary.  
**Mitigation:** #11, PR #20.

Implemented controls:

- optimization produces advisory `CandidateImproved`; it cannot set completion `Passed=true`;
- `FinalGate` is the sole completion authority;
- `PipelineFinalGate` requires a distinct canonical Pipeline and rejects reuse of the exact same collector instance behind a second wrapper;
- final baseline/candidate validation uses `FinalGate=true`, clean-state evidence and therefore L3 TruthPath under the FMEA-001 fail-closed tier contract;
- private held-out probes are unavailable to the proposal loop and expose aggregate cases/failures/regression-escape rate only, not probe identity/predicate details;
- held-out failure, hard violations, protected-axis regression or independent comparison rejection veto completion;
- original baseline critique remains immutable while iteration critique is separate mutable state;
- source-state or semantic-finding stagnation/oscillation terminates local optimization and requires escalation;
- SncSinCore repair-pattern success admission happens only after independent final PASS;
- the overflow repair was changed from symptom hiding to removal of the forced-width cause after the real browser gate rejected the original local-score fix.

Closure evidence:

- PR #20 merged as `717e090b11e294dbdeb1032c1ed5bb99d4e4d017`;
- `TestHostRepairEngine_OptimizationCannotSelfApprove`;
- `TestFMEA009SharedCollectorIsNotIndependent`;
- `TestFMEA009HeldOutRejectsVisibleScoreImprovement`;
- `TestFMEA009MemoryAdmissionRequiresIndependentPass`;
- `TestFMEA009HeldOutFailureBlocksMemoryPoisoning`;
- `TestFMEA009RepeatedFindingStateEscalates`;
- `TestFMEA009OptimizationComparisonUsesImmutableOriginalBaseline`;
- `TestRepairFinalGateRealChromium` proves L2 optimization followed by a separate real Playwright/Chromium L3 completion path plus private held-out acceptance;
- `ci` #212 PASS;
- `truthpath` #17 PASS.

Operational fault evidence: the first real-browser final-gate run rejected an overflow patch that mock evidence considered fixed. The local repair was corrected and the repeated L3 gate passed, proving the completion boundary is capable of catching optimization-oracle blind spots.  
Re-score: Severity remains 9 because a future independence/held-out regression can still allow consequential false completion. O `5→2` because proposal scoring no longer owns completion, hidden probes can independently veto, and failed repairs cannot be promoted as successful memory. D `7→3` because self-approval, oracle reuse, held-out escape, poisoning, oscillation, baseline integrity and real L3 completion are executable gates with explicit escape metrics.  
**Residual:** `9/2/3 RPN=54` — target met.  
**Reopen:** optimization can directly set PASS; final verifier reuses the optimization collector/oracle; held-out probes become visible proposal inputs; a held-out failure can still complete/admit memory; oscillation continues silently; or real L3 final-gate CI fails.

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode:** project-private evidence or poisoned facts leak across namespaces or are promoted into reusable canonical memory.  
**Effect:** privacy breach and persistent wrong guidance.  
**Owner:** SncSinCore adapter, admission/provenance/evolution.  
**Mitigation:** #12.  
**Closure gate:** adversarial multi-project isolation, provenance/conflict/retraction proof, poisoning corpus, promotion replay/shadow/non-regression/rollback.  
**Residual target:** `10/1/3 RPN=30`.

### FMEA-011 — Ungated main

**Failure mode:** high-risk changes can land on `main` without required review/status/risk gates.  
**Effect:** Critical paths can regress despite tests/governance.  
**Owner:** repository delivery policy.  
**Mitigation:** #13.  
**Closure gate:** enforce required CI/review checks where permissions permit; direct-push exceptions documented/audited.  
**Residual target:** `8/1/2 RPN=16`.

### FMEA-012 — Baseline environment mismatch

**Failure mode:** visual baseline is compared under incompatible browser/engine/version/viewport/DPR/theme/font/fixture environment.  
**Effect:** false regression or false PASS hidden by tolerance widening.  
**Owner:** baseline identity, comparison engine.  
**Mitigation:** #14.  
**Closure gate:** key includes browser/engine version, viewport, DPR, theme, font digest, renderer/worker version, fixture revision and locale/timezone where relevant; incompatible comparison is rejected.  
**Residual target:** `7/1/2 RPN=14`.

## 5. Execution order

Completed:

1. `FMEA-002` — CLOSED, residual RPN 20.
2. `FMEA-004` — CLOSED, residual RPN 18.
3. `FMEA-001` — CLOSED, residual RPN 20.
4. `FMEA-009` — CLOSED, residual RPN 54.

Current open order:

1. `FMEA-003` — **CURRENT**
2. `FMEA-008`
3. `FMEA-005`
4. `FMEA-007`
5. `FMEA-012`
6. `FMEA-010`
7. `FMEA-006`
8. `FMEA-011`

`FMEA-003` and `FMEA-008` remain the open **Verification Integrity Barrier**.

## 6. Change-control contract

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent test/eval/fault/review evidence
Residual target: S/O/D/RPN
Evidence: concrete test/artifact/CI/runtime/held-out proof
```

Perform an FMEA delta review when changing routing/fallback/legal PASS, impact/invalidation, readiness/freshness, runtime/browser/model capability/version, verifier semantics, autonomous repair/completion, Axiom retry/side effects, memory boundaries, privacy/provenance, baseline identity or CI/release gates.