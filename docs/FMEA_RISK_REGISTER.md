# UiUxMaster Architecture FMEA Risk Register

Status: **ACTIVE**  
Initial review: 2026-09-04  
Last re-score: 2026-09-05  
Planning authority: `MASTER_PLAN.md`  
Engineering-risk authority: this file  
Machine mirror: `planning/fmea-risk.json`  
Execution overlay: `planning/FMEA_EXECUTION_PLAN.md`  
Tracker: issue #15

## Governance

`internal/fidelity.RiskLevel` is runtime evidence-routing risk. `FMEA-###` is architecture/delivery risk. They are separate domains.

RPN = Severity × Occurrence × Detection difficulty. A risk closes only after merged mitigation, named independent evidence, residual re-score, synchronized planning state and explicit reopen triggers. Closed risks remain permanent regression guards.

## Summary

| ID | Failure mode | Initial RPN | Residual/current RPN | Status | Closure/current issue |
|---|---|---:|---:|---|---|
| FMEA-001 | required TruthPath silently downgrades | 320 | **20** | CLOSED | #3 / PR #19 |
| FMEA-002 | Axiom loses canonical scope | 420 | **20** | CLOSED | #4 / PR #17 |
| FMEA-003 | render freshness not source-revision bound | 280 | **20** | CLOSED | #5 / PR #21 |
| FMEA-004 | optimistic TruthPath capabilities | 360 | **18** | CLOSED | #6 / PR #18 |
| FMEA-005 | false impact/invalidation telemetry split | 240 | **12** | CLOSED | #7 / PR #23 |
| FMEA-006 | planning/documentation drift | 189 | 189 | OPEN | #8 |
| FMEA-007 | duplicate durable external effects | 189 | **18** | CLOSED | #9 / PR #24 |
| FMEA-008 | stale fidelity calibration after environment drift | 252 | **18** | CLOSED | #10 / PR #22 |
| FMEA-009 | repair reward hacking / verifier reuse | 315 | **54** | CLOSED | #11 / PR #20 |
| FMEA-010 | memory scope leakage / epistemic poisoning | 160 | **160** | **CURRENT** | #12 |
| FMEA-011 | ungated main | 128 | 128 | OPEN | #13 |
| FMEA-012 | incompatible visual baseline environment | 175 | **14** | CLOSED | #14 / PR #25 |

```text
open_critical_risks = 0
open_high_risks = 3
sum_open_rpn = 477
sum_closed_residual_rpn = 194
```

Verification Integrity Barrier: **PASSED**.  
Operational-correctness current risk: **FMEA-010**.

## Closed risks

### FMEA-001 — fail-closed minimum evidence tier

Residual `10/1/2 RPN=20`. PR #19, merge `e85cc1977493f481e4c76321bd829d297782325f`; `ci` #204, `axiom-control` #37, `truthpath` #9. Required tiers cannot silently downgrade and actual packet tier is attested before verification.

Reopen if a required tier can substitute weaker evidence, unknown routes gain a usable fallback, or attestation moves after verifier/evaluation.

### FMEA-002 — canonical Axiom scope

Residual `10/1/2 RPN=20`. PR #17, merge `468abe87ac76d54b3e887951e30b71e300c450d7`; `ci` #191, `axiom-control` #33. Canonical run/project/source/files/tokens/nodes/routes/viewports/themes/base/need scope survives Axiom projection; advisory plans cannot narrow it.

Reopen on canonical field loss, hard-coded run identity, independent Axiom tier selection or direct/Axiom scope-equivalence regression.

### FMEA-003 — revision-bound render freshness

Residual `10/1/2 RPN=20`. PR #21, merge `384158d3da78edb9fd7fb5b864106d736ad9ab5b`; `ci` #217, `axiom-control` #39, `truthpath` #22. `RenderToken{epoch, revision}` and packet provenance reject stale/wrong/revisionless readiness before verifier/PASS.

Reopen if epoch-only state can satisfy a revision-bound request or mismatch can reach verifier/PASS.

### FMEA-004 — runtime-attested TruthPath

Residual `9/1/2 RPN=18`. PR #18, merge `7433e1bd39c431ab0ac69181e2caf0fde9dd1921`; `ci` #197, `truthpath` #2. Worker, exact Playwright runtime and launchable browser versions must be probed before capability advertisement.

Reopen if unprobed/unversioned runtime can advertise L3 or runtime identity disappears from evidence.

### FMEA-005 — independent impact/invalidation telemetry

Residual `6/1/2 RPN=12`. PR #23, merge `b574f86870c5e604998347f941fd961e3a8fc657`; `ci` #230, `axiom-control` #47, `truthpath` #35. Impact and invalidation have independent measured stages, counters and benchmark artifact `scope_stages.json`.

Reopen if one timing is copied into both fields, stages double-count total, or independent benchmark distributions disappear.

### FMEA-007 — exactly-once observable durable effects

Residual `9/1/2 RPN=18`. PR #24, merge `e9e1139f93ad47c785841e99bed69a7d427b872e`; `ci` #237, `axiom-control` #50, `truthpath` #42.

Controls: attempt-independent logical operation IDs, payload-drift rejection, source expected-revision CAS, atomic source-state+receipt persistence, `EpMemoryStore.CommitOnce`, graph edge/conflict dedupe, stable Axiom external-effect key, bounded retry/timeout and crash-after-effect replay proof.

Reopen if a retried effect lacks stable semantic identity, source update bypasses CAS, state/receipt can persist separately, or replay can multiply memory graph state.

### FMEA-008 — exact runtime calibration authority

Residual `9/1/2 RPN=18`. PR #22, merge `d90b2c4fc99facaed4575b9cbe4f4c47bb5af41e`; `ci` #224, `axiom-control` #45, `truthpath` #29. Approximate-tier PASS requires current exact approximate↔TruthPath calibration identity, class-specific evidence and non-expired quality/coverage records.

Reopen if an L1/L2 PASS bypasses calibration authority or stale/weak/mismatched records remain legal.

### FMEA-009 — independent autonomous-repair completion

Residual `9/2/3 RPN=54`. PR #20, merge `717e090b11e294dbdeb1032c1ed5bb99d4e4d017`; `ci` #212, `truthpath` #17. Optimization cannot self-approve; independent final gate + private held-out probes authorize completion and rejected outcomes cannot enter success memory.

Reopen if proposal and completion reuse the same oracle/signals, held-out cases become proposal-visible, or rejected candidates enter success memory.

### FMEA-012 — canonical protected-baseline environment — CLOSED

Initial `7/5/5 RPN=175` -> residual **`7/1/2 RPN=14`**. Issue #14, PR #25, merge `64053497b52cffd83de37ef3396aee2cdef4354a`; final head `39a8c35d82fb46a324a10b9b369a8b9867996044`; `ci` #247, `axiom-control` #56, `truthpath` #52.

Implemented controls:

- versioned `RenderEnvironmentIdentity` covers renderer/worker/browser/engine versions, actual platform, viewport, DPR, theme, font-set digest, locale, timezone and fixture revision;
- incomplete material dimensions are fail-closed, never wildcards;
- protected baseline/candidate canonical keys must match **before** masks, tolerance or pixel diff;
- old direct `BaselineRGBA -> CompareRGBA` dispatcher path is removed;
- candidate normalized environment is persisted in `evidence.Packet.Environment` with environment-key/comparison-digest attestation;
- stored baseline digest is verified against actual baseline pixels before comparison;
- baseline `Put` is create-only; update requires reviewed expected-version+digest CAS and records old/new digest/environment/version/rationale/reviewer;
- dynamic masks must be contained within a current visible semantic owner and image bounds;
- comparator instability and baseline churn are partitioned by exact environment key and churn is derived from audit history.

Closure tests include `TestFMEA012EnvironmentKeyRejectsMaterialMismatch`, `TestFMEA012ComparatorRejectsIncompatibleBeforeTolerance`, `TestFMEA012DispatcherRejectsIncompatibleBaselineBeforeTolerance`, `TestFMEA012DispatcherRejectsBaselineWithoutEnvironment`, `TestFMEA012ExactKeyComparisonIsDeterministic`, `TestFMEA012ReviewedBaselineUpdateRecordsOldNewIdentity`, `TestFMEA012BaselineChurnMetricsArePartitionedByEnvironmentKey`, `TestFMEA012ComparatorRejectsBaselinePixelDigestMismatch`, `TestFMEA012DynamicMaskCannotEscapeOwner`, and `TestFMEA012OwnedMaskOnlyExcludesOwnedPixels`.

Reopen if protected pixels are compared before exact environment compatibility, material identity becomes wildcarded, tolerance compensates for environment drift, baseline update bypasses reviewed CAS/provenance, digest metadata can diverge from pixels, masks escape semantic owners, or packet environment provenance disappears.

## Open risks

### FMEA-010 — Memory scope leakage / epistemic poisoning — CURRENT

Initial/current `10/2/8 RPN=160`. Issue #12.

Failure mode: project-private evidence can leak into another project/global context, or poisoned/unproven facts can be promoted into reusable canonical memory and survive conflict/retraction/rollback.

Current known boundary: read paths use namespace `CanAccess`, but read filtering alone does not prove write/admission/promotion isolation. `AdmissionRequest` still accepts a target namespace, so FMEA-010 must make source scope/provenance an admission invariant rather than trusting the caller-selected destination.

Closure gate:

- adversarial project-A/project-B/global write+read isolation;
- private evidence cannot be admitted/promoted to global without an explicit promotion policy and independent evidence requirements;
- provenance source scope and target namespace are consistent and immutable/auditable;
- poisoning corpus exercises contradictory, low-confidence, stale and forged-scope inputs;
- conflicts preserve truth rather than silently replacing it;
- retract/supersede/promotion rollback removes promoted visibility without deleting source evidence;
- replay/shadow/non-regression proof for promotion decisions;
- durable/adapted stores must preserve the FMEA-007 idempotency contract.

Residual target: `10/1/3 RPN=30`.

### FMEA-006 — Planning/documentation drift

Current `7/9/3 RPN=189`. Issue #8. Reconcile claims to `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`, remove stale ownership/status prose and add consistency checks. Residual target `7/2/2 RPN=28`.

### FMEA-011 — Ungated main

Current `8/4/4 RPN=128`. Issue #13. Enforce required CI/review/risk gates where repository permissions permit and audit direct-push exceptions. Residual target `8/1/2 RPN=16`.

## Execution order

1. **FMEA-010 — CURRENT**
2. FMEA-006
3. FMEA-011

## Change-control contract

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent test/eval/fault/review evidence
Residual target: S/O/D/RPN
Evidence: concrete test/artifact/CI/runtime/held-out proof
```

Perform an FMEA delta whenever changing routing/fallback/legal PASS, impact/invalidation, render readiness/freshness, runtime identity/calibration, verifier semantics, autonomous repair, Axiom retries/side effects, memory boundaries, baseline identity, privacy/provenance or CI/release policy.
