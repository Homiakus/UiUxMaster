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
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10/4/7 | 280 | **10/1/2** | **20** | Low residual | **CLOSED** | #5 / PR #21 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9/5/8 | 360 | **9/1/2** | **18** | Low residual | **CLOSED** | #6 / PR #18 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6/10/4 | 240 | 6/10/4 | 240 | High | OPEN | #7 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7/9/3 | 189 | 7/9/3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9/3/7 | 189 | 9/3/7 | 189 | High | OPEN | #9 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9/4/7 | 252 | **9/1/2** | **18** | Low residual | **CLOSED** | #10 / PR #22 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9/5/7 | 315 | **9/2/3** | **54** | Low residual | **CLOSED** | #11 / PR #20 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10/2/8 | 160 | 10/2/8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8/4/4 | 128 | 8/4/4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under an incompatible render environment | 7/5/5 | 175 | 7/5/5 | 175 | High | OPEN | #14 |

Current milestone metrics:

```text
open_critical_risks = 0
open_high_risks = 6
sum_open_rpn = 1081
sum_closed_residual_rpn = 150
```

The **Verification Integrity Barrier is PASSED**. The active queue has moved to operational correctness, beginning with FMEA-005.

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
- dispatcher checks `actual evidence strength >= policy-selected minimum`;
- canonical `engine.Pipeline` independently repeats the guard before verifier/evaluation;
- L1→L2 upward escalation remains legal; downward substitution is rejected.

Closure evidence: PR #19 merge `e85cc1977493f481e4c76321bd829d297782325f`; `TestFMEA001TruthPathUnavailableDoesNotDowngradeToL2`; `TestFMEA001TruthPathRejectsWeakerPacketFromConfiguredL3`; `TestFMEA001UnknownRouteDoesNotDefaultToL2`; `TestPipelineRejectsWeakCustomCollectorBeforeVerifier`; real `TestTruthPathDispatcherRealChromium`; `ci` #204, `axiom-control` #37, `truthpath` #9 PASS.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** any required route can substitute weaker evidence, unknown routes gain a usable fallback, attestation moves after verifier/evaluation, or real L3 dispatcher CI fails.

### FMEA-002 — Axiom canonical-pipeline scope loss — CLOSED

**Failure mode:** Axiom entered `engine.Pipeline` through a lossy change projection and could independently narrow canonical evidence need.  
**Owner:** `control/axiom/controlplane`, `control/axiom/uiuxadapter`, `internal/engine`.  
**Mitigation:** #4, PR #17.

Controls/evidence: lossless durable run/project/source/files/tokens/nodes/routes/viewports/themes/base/need payload; no advisory-plan→canonical-need narrowing; direct/Axiom scope+route equivalence; anti-narrowing regression; `ci` #191 and `axiom-control` #33 including real Chrome.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** canonical field loss, hard-coded run identity, independent Axiom tier selection, or equivalence-test failure.

### FMEA-003 — Render freshness not revision-bound — CLOSED

**Failure mode:** numeric render epoch could advance while the browser still represented another source/build revision.  
**Effect:** stale/wrong content could be captured and pass deterministic checks.  
**Owner:** `internal/runtime/fastcdp`, dispatcher/Axiom collectors, evidence provenance, canonical Pipeline.  
**Mitigation:** #5, PR #21.

Implemented controls:

- freshness identity is `RenderToken{Epoch, Revision}`;
- revision-bound waiters require both `epoch > after` and exact requested revision;
- a newer epoch carrying the wrong revision fails with `ErrRevisionMismatch` rather than releasing evidence;
- revisionless legacy signals cannot satisfy a revision-bound waiter;
- stable evidence capture verifies that the full token does not change mid-capture;
- canonical packet provenance records expected revision, observed revision and epoch;
- canonical Pipeline validates revision attestation before deterministic verifier/evaluation;
- revision mismatch discards/resets the warm page and recovery begins from a new matching lineage;
- framework-specific HMR integrations remain adapters around the framework-neutral token contract.

Closure evidence:

- PR #21 merged as `384158d3da78edb9fd7fb5b864106d736ad9ab5b`;
- `TestFMEA003StaleRevisionCannotReleaseEvidence`;
- `TestFMEA003NewerEpochWithWrongRevisionFailsClosed`;
- `TestFMEA003MatchingRevisionReleasesWaiter`;
- `TestFMEA003LegacyRevisionlessSignalCannotSatisfyBoundWaiter`;
- `TestFMEA003PacketCarriesExpectedAndObservedRevision`;
- `TestFMEA003BridgePayloadSupportsRevisionAndLegacySignals`;
- engine attestation tests reject wrong/missing/internally inconsistent freshness provenance before verifier;
- real `TestAxiomFastCDPEndToEndIntegration` proves matching revision, wrong-revision rejection, warm-page discard/reset and recovery;
- `ci` #217, `axiom-control` #39 and `truthpath` #22 PASS.

Re-score: Severity remains 10 because a future stale-revision bypass could still invalidate trust. O `4→1` because epoch-only progress cannot satisfy a revision-bound request. D `7→2` because stale, wrong, legacy, packet-provenance and real HMR recovery paths are executable gates.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** epoch-only state can satisfy a revision-bound request, provenance loses expected/observed revision, mismatch reaches verifier/PASS, or real mismatch/recovery CI fails.

### FMEA-004 — TruthPath capability optimism — CLOSED

**Failure mode:** Playwright adapter advertised browser/scenario/ARIA/font capability without a proven runnable worker/runtime/browser.  
**Owner:** `internal/runtime/playwright`, CI.  
**Mitigation:** #6, PR #18.

Controls/evidence: fail-closed pre-probe capabilities; worker protocol `1.0.0`; exact Playwright `1.62.1`; browser advertised only after executable discovery + real launch + version; missing-worker rejection; runtime identity drift detection; identity reaches packet provenance; `truthpath` #2 and `ci` #197.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** pre-probe capability claim, non-versioned runtime, unlaunched browser advertisement, missing-worker fallback, provenance identity loss or real TruthPath CI failure.

### FMEA-005 — False telemetry split — CURRENT

**Failure mode:** impact and invalidation appear as separate metrics while sharing one combined timing.  
**Effect:** optimization/regression decisions use misleading stage data.  
**Owner:** `internal/engine`, telemetry.  
**Mitigation:** #7.  
**Closure gate:** independent timers/counters and accounting tests/benchmarks prove values originate from distinct stages; aggregate latency accounting must remain internally consistent.  
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

### FMEA-008 — Calibration drift — CLOSED

**Failure mode:** a previously legal L1/L2 parity calibration could remain trusted after renderer/browser/worker/runtime/environment changes.  
**Effect:** approximate evidence could issue systematic false PASS against obsolete parity assumptions.  
**Owner:** `internal/fidelity`, `internal/engine`, runtime identity providers, Axiom PASS boundary.  
**Mitigation:** #10, PR #22.

Implemented controls:

- `CalibrationMatrix` answers which tiers may in principle prove an evidence class;
- independent runtime `CalibrationAuthority` answers whether the exact current approximate↔TruthPath runtime pair is still calibrated;
- `CalibrationEnvironment` includes renderer name/version/fidelity identity plus browser, worker/runtime, platform and applicable profile/font/viewport/device-scale/theme dimensions;
- `CalibrationContext` deterministically keys both approximate and TruthPath environments;
- durable `CalibrationRecord` includes environment key, corpus digest, artifact reference, samples/pass count, creation and expiry;
- policy enforces minimum corpus coverage, minimum parity quality and maximum age/expiry;
- `CalibrationRegistry.SaveFile` persists versioned snapshots atomically and restore validates every record;
- canonical `PipelineResult.PassAuthority` separates “deterministic evidence is clean” from “this tier has legal authority to issue PASS”;
- each evidence class actually claimed by an L1/L2 plan requires its own matching calibration; one class cannot substitute for another;
- missing, expired, weak-corpus, low-quality or environment-mismatched calibration becomes explicit missing evidence/upward escalation, never PASS;
- L3 TruthPath remains authoritative without approximate-tier parity calibration;
- Axiom canonical pipeline always sets `RequireLegalPass`; legacy direct collector mode is diagnostic-only and cannot issue `DecisionPass`;
- Dispatcher derives current calibration context from actual approximate renderer identity and the configured runtime-attested TruthPath identity.

Closure evidence:

- PR #22 merged as `d90b2c4fc99facaed4575b9cbe4f4c47bb5af41e`;
- `TestFMEA008SameValidatedKeyRetainsLegalPass`;
- `TestFMEA008VersionMutationInvalidatesPreviouslyLegalPass` mutates approximate renderer/browser, TruthPath browser/worker/runtime, platform and viewport;
- `TestFMEA008ExpiredMissingAndWeakCorpusFailClosed` covers missing/expiry/coverage/quality;
- `TestFMEA008CalibrationSnapshotPersistsExactKeyAndArtifact` proves durable key/corpus/artifact identity;
- engine tests prove missing/drifted calibration becomes evidence insufficiency and the same exact key restores legal PASS;
- `TestTruthPathCalibrationRealChromium` launches the Playwright-attested Chromium binary as resident FastCDP, derives both exact runtime identities, validates the exact parity record, then mutates TruthPath worker identity and proves immediate invalidation;
- real `TestAxiomFastCDPEndToEndIntegration` now routes PASS/repair/visual decisions through canonical calibrated Pipeline and still proves FMEA-003 mismatch/discard/recovery;
- `ci` #224, `axiom-control` #45 and `truthpath` #29 PASS.

Operational fault evidence: CI first rejected the migrated Axiom test because visual evidence also claimed the Interactive class while only static/typography/pixel parity records existed. The test was corrected by adding a distinct Interactive calibration instead of weakening class inference. A later run exposed an asynchronous CDP binding-delivery race in the test helper; it was fixed with an EpochGate delivery barrier rather than sleeps.  
Re-score: Severity remains 9 because a future calibration-authority bypass could still create systematic false PASS. O `4→1` because L1/L2 legal PASS is impossible without an exact current record for every claimed class. D `7→2` because environment mutation, age/coverage/quality, persistence, Axiom bypass prevention and a real FastCDP↔TruthPath runtime pair are executable gates.  
**Residual:** `9/1/2 RPN=18`.  
**Reopen:** any L1/L2 PASS bypasses `PassAuthority`; material runtime/environment identity is omitted from the key; stale/expired/weak records remain legal; one evidence class can substitute for another; or real runtime-drift CI fails.

### FMEA-009 — Repair verifier overfit / reward hacking — CLOSED

**Failure mode:** autonomous repair optimized against the same visible verifier/rubric/candidate signals that could authorize completion.  
**Effect:** an apparently improved candidate could regress hidden requirements and promote a false-success repair pattern into reusable memory.  
**Owner:** `internal/repair`, final verification/eval, SncSinCore admission boundary.  
**Mitigation:** #11, PR #20.

Implemented controls: advisory `CandidateImproved`; sole completion authority is separate `FinalGate`; same collector cannot masquerade as independent; final verification uses clean-state TruthPath; private held-out probes can veto visible score improvement; protected-axis/hard/held-out regressions veto; oscillation escalates; success memory admission requires independent final PASS. The first real-browser gate rejected a mock-approved overflow fix and forced a root-cause repair.  
Closure evidence: PR #20 merge `717e090b11e294dbdeb1032c1ed5bb99d4e4d017`; self-approval/oracle-reuse/held-out/memory/oscillation/baseline-integrity tests; real `TestRepairFinalGateRealChromium`; `ci` #212 and `truthpath` #17 PASS.  
**Residual:** `9/2/3 RPN=54`.  
**Reopen:** optimization regains completion authority, final verification reuses optimization oracle, held-out inputs become proposer-visible, rejected repairs can enter success memory, oscillation continues silently, or real L3 final-gate CI fails.

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode:** project-private evidence or poisoned facts leak across namespaces or are promoted into reusable canonical memory.  
**Effect:** privacy breach and persistent wrong guidance.  
**Owner:** SncSinCore adapter, admission/provenance/evolution.  
**Mitigation:** #12.  
**Closure gate:** adversarial multi-project isolation, provenance/conflict/retraction proof, poisoning corpus, promotion replay/shadow/non-regression/rollback.  
**Residual target:** `10/1/3 RPN=30`.

### FMEA-011 — Ungated main

**Failure mode:** high-risk changes can land on `main` without required review/status/risk gates.  
**Effect:** high-risk paths can regress despite tests/governance.  
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

Verification Integrity Barrier completed:

1. `FMEA-002` — CLOSED, residual RPN 20.
2. `FMEA-004` — CLOSED, residual RPN 18.
3. `FMEA-001` — CLOSED, residual RPN 20.
4. `FMEA-009` — CLOSED, residual RPN 54.
5. `FMEA-003` — CLOSED, residual RPN 20.
6. `FMEA-008` — CLOSED, residual RPN 18.

Current open order:

1. `FMEA-005` — **CURRENT**
2. `FMEA-007`
3. `FMEA-012`
4. `FMEA-010`
5. `FMEA-006`
6. `FMEA-011`

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
