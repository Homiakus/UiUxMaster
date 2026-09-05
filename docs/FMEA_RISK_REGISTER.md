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
- Operational tracker: issue #15

## 1. Risk-domain separation

`internal/fidelity.RiskLevel` is runtime routing data: how likely approximate evidence is to diverge from browser truth and which execution tier is required.

`FMEA-###` is engineering/planning risk: how the architecture can fail, the effect, occurrence, detectability, mitigation, closure evidence, residual score and reopen trigger.

These domains must never share one field or enum.

## 2. Scoring and closure policy

- **Severity (S)**: 1 negligible; 10 can invalidate trust, leak protected data, create destructive behavior or consequential false PASS.
- **Occurrence (O)**: 1 exceptional/strongly prevented; 10 expected/frequent.
- **Detection difficulty (D)**: 1 almost certainly caught before escape; 10 likely to escape controls.
- **RPN** = `S × O × D`.

Priority: Critical `>=250` or credible Severity 9–10 false-PASS/security/destructive path; High `120–249`; Medium `60–119`; Low `<60`.

Status: `OPEN`, `MITIGATING`, `ACCEPTED`, `CLOSED`.

A risk closes only after merged implementation, named independent closure evidence, residual re-score, synchronized register/machine-state/tracker and an explicit reopen trigger. Code existence alone never closes risk. Closed risks remain permanent regression guards.

## 3. Risk summary

| ID | Failure mode | Initial S/O/D | Initial RPN | Current S/O/D | Current RPN | Priority | Status | Mitigation |
|---|---|---:|---:|---:|---:|---|---|---|
| FMEA-001 | Required TruthPath silently downgrades to L2 | 10/4/8 | 320 | **10/1/2** | **20** | Low residual | **CLOSED** | #3 / PR #19 |
| FMEA-002 | Axiom loses canonical change/ImpactSet scope | 10/6/7 | 420 | **10/1/2** | **20** | Low residual | **CLOSED** | #4 / PR #17 |
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10/4/7 | 280 | **10/1/2** | **20** | Low residual | **CLOSED** | #5 / PR #21 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9/5/8 | 360 | **9/1/2** | **18** | Low residual | **CLOSED** | #6 / PR #18 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6/10/4 | 240 | **6/1/2** | **12** | Low residual | **CLOSED** | #7 / PR #23 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7/9/3 | 189 | 7/9/3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9/3/7 | 189 | **9/1/2** | **18** | Low residual | **CLOSED** | #9 / PR #24 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9/4/7 | 252 | **9/1/2** | **18** | Low residual | **CLOSED** | #10 / PR #22 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9/5/7 | 315 | **9/2/3** | **54** | Low residual | **CLOSED** | #11 / PR #20 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10/2/8 | 160 | 10/2/8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8/4/4 | 128 | 8/4/4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under an incompatible render environment | 7/5/5 | 175 | 7/5/5 | 175 | High | **CURRENT** | #14 |

Current milestone metrics:

```text
open_critical_risks = 0
open_high_risks = 4
sum_open_rpn = 652
sum_closed_residual_rpn = 180
```

The **Verification Integrity Barrier is PASSED**. The active operational-correctness risk is **FMEA-012**.

## 4. Detailed risks

### FMEA-001 — Silent TruthPath downgrade — CLOSED

**Failure mode:** a policy-selected L3 requirement could silently execute on L2.  
**Controls:** no implicit L3→L2 fallback; unknown routes fail closed; selected collector availability is typed; dispatcher and canonical Pipeline attest actual tier before verifier/evaluation; upward escalation remains legal.  
**Evidence:** PR #19 merge `e85cc1977493f481e4c76321bd829d297782325f`; `TestFMEA001TruthPathUnavailableDoesNotDowngradeToL2`; weaker-packet, unknown-route, pipeline-guard and real Chromium dispatcher tests; `ci` #204, `axiom-control` #37, `truthpath` #9.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** any required route can substitute weaker evidence, unknown routes gain a usable fallback, attestation moves after verifier/evaluation, or real L3 dispatcher CI fails.

### FMEA-002 — Axiom canonical-pipeline scope loss — CLOSED

**Failure mode:** Axiom could enter canonical validation through a lossy scope projection or independently narrow evidence need.  
**Controls/evidence:** lossless run/project/source/files/tokens/nodes/routes/viewports/themes/base/need payload; no advisory-plan narrowing; direct/Axiom scope+route equivalence and anti-narrowing tests; PR #17 merge `468abe87ac76d54b3e887951e30b71e300c450d7`; `ci` #191, `axiom-control` #33.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** canonical field loss, hard-coded run identity, independent Axiom tier selection, or equivalence-test failure.

### FMEA-003 — Render freshness not revision-bound — CLOSED

**Failure mode:** numeric render epoch could advance while browser state represented another source/build revision.  
**Controls:** `RenderToken{Epoch, Revision}`; revision-bound waiters require exact revision; wrong/newer and revisionless signals fail closed; packet provenance records expected/observed revision and epoch; canonical Pipeline validates before verifier; mismatch resets warm page lineage.  
**Evidence:** PR #21 merge `384158d3da78edb9fd7fb5b864106d736ad9ab5b`; stale/wrong/matching/legacy/provenance tests; real Axiom/Chromium integration; `ci` #217, `axiom-control` #39, `truthpath` #22.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** epoch-only state satisfies a revision-bound request, provenance loses revision identity, mismatch reaches verifier/PASS, or real HMR recovery CI fails.

### FMEA-004 — TruthPath capability optimism — CLOSED

**Failure mode:** Playwright adapter could advertise capability without a proven worker/runtime/browser.  
**Controls/evidence:** fail-closed pre-probe capabilities; worker protocol `1.0.0`; exact Playwright `1.62.1`; browser advertised only after executable discovery + real launch + version; missing-worker and runtime-identity drift rejection; runtime identity reaches packet provenance; PR #18 merge `7433e1bd39c431ab0ac69181e2caf0fde9dd1921`; `ci` #197, `truthpath` #2.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** pre-probe claims, unversioned runtime, unlaunched browser advertisement, missing-worker fallback, provenance identity loss or real TruthPath CI failure.

### FMEA-005 — False telemetry split — CLOSED

**Failure mode:** impact and invalidation appeared separate while sharing one timing.  
**Controls:** canonical `ResolveImpact` and `InvalidateImpact`; non-overlapping timers; separate impact/scope counters; `MeasuredStageMS + UnattributedMS` accounting; packet/pipeline latency equality; CI `scope_stages.json`.  
**Evidence:** PR #23 merge `b574f86870c5e604998347f941fd961e3a8fc657`; impact-only and invalidation-only 35 ms fault tests; accounting test; benchmark artifact `9965578584`, SHA256 `9fb59af3c563330d901f97b3bc4b922a0fed8a9602041305232b8f5df5e2ebb0`; 200-iteration means impact `2.979 µs`, invalidation `1.213 µs`; `ci` #230, `axiom-control` #47, `truthpath` #35.  
**Residual:** `6/1/2 RPN=12`.  
**Reopen:** shared timer returns, injected latency leaks across stage metrics, named stages double-count total, packet/pipeline telemetry diverge, or independent benchmark distributions disappear.

### FMEA-006 — Planning/documentation state drift

**Failure mode:** historical status prose contradicts implementation/integration/operational evidence.  
**Effect:** engineers depend on overstated capabilities or wrong ownership/naming.  
**Owner:** `MASTER_PLAN.md`, README, architecture docs, issues.  
**Mitigation:** #8.  
**Closure gate:** reconcile claims to `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`; eliminate stale TruthPath/FastPath state; add consistency checks where practical.  
**Residual target:** `7/2/2 RPN=28`.

### FMEA-007 — Duplicate durable side effects — CLOSED

**Failure mode:** durable retry/restart could repeat source repair or memory admission after target success but before workflow completion persistence.  
**Effect:** destructive duplicate edits, graph multiplicity, or workflow/external-state divergence.  
**Owner:** Axiom external-effect boundary, repair source target, memory admission.  
**Mitigation:** #9, PR #24.

Implemented controls:

- semantic operation identity is stable across retry attempts; attempt number is excluded;
- payload digest is bound to the logical operation; payload drift in the same logical slot fails closed;
- source effect uses exact expected-revision CAS;
- file-backed source target atomically persists source state and effect receipt using fsync + rename;
- restart/reopen replay returns the original receipt instead of mutating source again;
- `EpMemoryStore.CommitOnce` binds admission to the stable logical operation;
- semantic memory edges and conflicts are deduplicated even for legacy `Commit` callers;
- repair memory mapper/commit errors are no longer silently swallowed;
- Axiom `iterate_repair` is an explicit `ExternalEffect` with stable `{execution}:{node}` idempotency key, bounded retry and timeout;
- durable Axiom activity identity is exposed to adapters without leaking Axiom types outside the control-plane boundary;
- `AfterEffect` fault seam models crash/failure after target acceptance and before activity completion.

Closure evidence:

- PR #24 merged as `e9e1139f93ad47c785841e99bed69a7d427b872e`;
- `TestFMEA007SourceCASReplayAfterRestartIsExactlyOnce`;
- `TestFMEA007SourceCASRejectsConcurrentRevision`;
- `TestFMEA007LogicalOperationRejectsPayloadMutation`;
- `TestFMEA007CommitOnceReusesReceiptWithoutDuplicatingGraph`;
- `TestFMEA007LegacyCommitStillDeduplicatesEdgesAndConflicts`;
- `TestFMEA007LogicalMemoryOperationRejectsPayloadDrift`;
- `TestFMEA007AxiomRetryReusesTargetEffectsAndStableActivityKey`: source + memory effects succeed, a transient failure is injected afterward, Axiom retries, both target receipts are reused and both attempts expose the same Axiom idempotency key;
- completed file-backed Axiom run reloads without invoking the external effect again;
- `ci` #237, `axiom-control` #50 and `truthpath` #42 PASS;
- reliability contract: `docs/reliability/FMEA007_DURABLE_SIDE_EFFECTS.md`.

Severity remains 9 because a future target adapter bypassing the contract could still cause destructive duplication. O `3→1` because both current externally visible repair effects are target-idempotent and source mutation additionally uses CAS. D `7→2` because crash-window replay, reopen, payload-drift, CAS, graph-dedupe, stable Axiom-key and real regression gates are executable CI evidence.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** a retried external effect lacks stable logical identity, source update bypasses expected-revision CAS, state and receipt can persist separately, memory replay can multiply semantic graph state, Axiom retries use different semantic keys, or crash/reopen fault tests fail.

Boundary: `EpMemoryStore` remains an in-process epistemic store. Cross-project persistence/isolation/poisoning is owned by FMEA-010; future durable memory adapters must preserve the FMEA-007 operation/receipt contract.

### FMEA-008 — Calibration drift — CLOSED

**Failure mode:** previously legal approximate-tier calibration could remain trusted after environment/version drift.  
**Controls:** exact current approximate↔TruthPath environment key; durable corpus/artifact/coverage/quality/expiry records; class-specific calibration; canonical `PassAuthority`; missing/expired/weak/mismatched calibration becomes insufficiency/upward escalation; real FastCDP/Playwright runtime identity proof.  
**Evidence:** PR #22 merge `d90b2c4fc99facaed4575b9cbe4f4c47bb5af41e`; exact-key/version-drift/expiry/coverage/persistence tests; `TestTruthPathCalibrationRealChromium`; `ci` #224, `axiom-control` #45, `truthpath` #29.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** L1/L2 PASS bypasses `PassAuthority`, material identity is omitted, stale/weak records remain legal, one evidence class substitutes for another, or real runtime-drift proof fails.

### FMEA-009 — Repair verifier overfit / reward hacking — CLOSED

**Failure mode:** repair optimization could use the same signals that authorize completion.  
**Controls:** optimization cannot set final PASS; separate independent `FinalGate`; private held-out probes; protected-axis and held-out vetoes; rejected candidates cannot enter success memory; oscillation escalates.  
**Evidence:** PR #20 merge `717e090b11e294dbdeb1032c1ed5bb99d4e4d017`; self-approval, oracle-reuse, held-out, poisoning, oscillation and baseline-integrity tests; real `TestRepairFinalGateRealChromium`; `ci` #212, `truthpath` #17.  
**Residual:** `9/2/3 RPN=54`.  
**Reopen:** optimization regains completion authority, final verifier reuses proposal oracle, held-out inputs become proposal-visible, rejected candidates enter success memory, oscillation continues silently, or real L3 final-gate CI fails.

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode:** project-private evidence or poisoned facts leak across namespaces or are promoted into reusable canonical memory.  
**Effect:** privacy breach and persistent wrong guidance.  
**Owner:** SncSinCore adapter, admission/provenance/evolution.  
**Mitigation:** #12.  
**Closure gate:** adversarial multi-project isolation, provenance/conflict/retraction proof, poisoning corpus, promotion replay/shadow/non-regression/rollback.  
**Residual target:** `10/1/3 RPN=30`.

### FMEA-011 — Ungated main

**Failure mode:** high-risk changes can land on `main` without required review/status/risk gates.  
**Effect:** Critical/high-risk paths can regress despite available tests/governance.  
**Owner:** repository delivery policy.  
**Mitigation:** #13.  
**Closure gate:** enforce required CI/review checks where permissions permit; direct-push exceptions documented/audited; risk-gating status visible to release workflow.  
**Residual target:** `8/1/2 RPN=16`.

### FMEA-012 — Baseline environment mismatch — CURRENT

**Failure mode:** protected visual baseline is compared under incompatible browser/engine/version/viewport/DPR/theme/font/fixture/runtime environment.  
**Effect:** false regressions, suppressed real regressions, flaky CI and pressure to widen global tolerances until meaningful defects escape.  
**Owner:** baseline identity/store, visual comparison engine, capture provenance.  
**Mitigation:** #14.  
**Closure gate:** canonical environment key includes browser/engine/runtime versions, viewport, DPR, theme, font-set digest, renderer/worker version, fixture revision and locale/timezone where material; incompatible baseline/candidate keys are rejected before diff/tolerance evaluation; baseline update is an explicit provenance-bearing operation recording old/new digest and rationale; deterministic masks are scoped to declared semantic ownership and cannot hide undeclared regions; exact-key comparison is deterministic across replay.  
**Residual target:** `7/1/2 RPN=14`.

## 5. Execution order

Completed:

1. `FMEA-002` — residual RPN 20.
2. `FMEA-004` — residual RPN 18.
3. `FMEA-001` — residual RPN 20.
4. `FMEA-009` — residual RPN 54.
5. `FMEA-003` — residual RPN 20.
6. `FMEA-008` — residual RPN 18.
7. `FMEA-005` — residual RPN 12.
8. `FMEA-007` — residual RPN 18.

Current open order:

1. `FMEA-012` — **CURRENT**
2. `FMEA-010`
3. `FMEA-006`
4. `FMEA-011`

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

Perform an FMEA delta review when changing routing/fallback/legal PASS, impact/invalidation, readiness/freshness, runtime/browser/model capability/version, calibration authority, verifier semantics, autonomous repair/completion, Axiom retry/side effects, memory boundaries, privacy/provenance, baseline identity or CI/release gates.
