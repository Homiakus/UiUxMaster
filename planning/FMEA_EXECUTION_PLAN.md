# UiUxMaster FMEA execution overlay

Status: **ACTIVE**  
Authority: `MASTER_PLAN.md` remains the single product execution roadmap. This file is a risk-gating overlay, not a competing roadmap.  
Risk source of truth: `docs/FMEA_RISK_REGISTER.md`  
Machine state: `planning/fmea-risk.json`  
Governance ADR: `docs/adr/0002-fmea-risk-governance.md`  
Operational tracker: GitHub issue #15

## Planning model

```text
MASTER_PLAN task
  -> affected FMEA IDs
  -> mitigation issue/PR
  -> independent closure gate
  -> residual S/O/D + RPN
  -> only then risk may close
```

Architecture maturity is reviewed as:

`IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`

`internal/fidelity.RiskLevel` remains runtime evidence-routing data. `FMEA-###` remains engineering/planning risk data.

## Closed verification-integrity risks

### FMEA-002 — CLOSED

Initial `10/6/7 RPN=420` -> residual `10/1/2 RPN=20`.  
Issue #4, PR #17, merge `468abe87ac76d54b3e887951e30b71e300c450d7`.

Closure: lossless canonical Axiom scope, anti-narrowing route ownership, direct/Axiom equivalence, `ci` #191 and `axiom-control` #33 including real Chrome.

### FMEA-004 — CLOSED

Initial `9/5/8 RPN=360` -> residual `9/1/2 RPN=18`.  
Issue #6, PR #18, merge `7433e1bd39c431ab0ac69181e2caf0fde9dd1921`.

Closure: fail-closed pre-probe capability state, versioned worker protocol, exact Playwright pin, real launch-validated browser capability, missing-worker rejection, runtime identity provenance/drift detection, `truthpath` #2 and `ci` #197.

### FMEA-001 — CLOSED

Initial `10/4/8 RPN=320` -> residual `10/1/2 RPN=20`.  
Issue #3, PR #19, merge `e85cc1977493f481e4c76321bd829d297782325f`.

Closure:

- required L3 has no implicit L2 fallback;
- unknown routes fail closed;
- dispatcher and canonical Pipeline both attest actual packet strength against the policy-selected minimum before verifier/evaluation;
- L1→L2 upward escalation remains legal;
- real `TestTruthPathDispatcherRealChromium` composes runtime-attested Playwright L3 with the guard;
- `ci` #204, `axiom-control` #37 and `truthpath` #9 PASS.

### FMEA-009 — CLOSED

Initial `9/5/7 RPN=315` -> residual `9/2/3 RPN=54`.  
Issue #11, PR #20, merge `717e090b11e294dbdeb1032c1ed5bb99d4e4d017`.

Closure:

- candidate optimization produces advisory `CandidateImproved`; only a separate `FinalGate` can authorize `Passed=true`;
- a second Pipeline wrapper is not considered independent when it reuses the same collector instance;
- final verification runs with `FinalGate=true` and therefore clean-state L3 TruthPath under the FMEA-001 tier contract;
- held-out probes stay private and return aggregate cases/failures/regression-escape metrics only;
- a visible score improvement can be rejected by a hidden product requirement;
- protected-axis regressions, hard violations, held-out escape and failed independent comparison veto completion;
- repeated source/finding state escalates rather than continuing local score optimization;
- SncSinCore success admission is gated on independent final PASS;
- the first real-browser final gate rejected a mock-approved overflow patch, which was then corrected to remove the forced-width cause rather than hide the symptom;
- `ci` #212 and `truthpath` #17 PASS, including real `TestRepairFinalGateRealChromium`.

Reopen FMEA-009 if optimization regains completion authority, final verification can reuse the optimization oracle, held-out inputs become proposer-visible, rejected candidates can enter success memory, oscillation is silently continued, or real L3 final-gate CI fails.

## Verification Integrity Barrier — remaining open risks

| Order | Risk | Issue | RPN | Closure gate |
|---:|---|---:|---:|---|
| 1 | **FMEA-003** — render freshness not revision-bound | #5 | 280 | expected/observed source/build revision attestation |
| 2 | FMEA-008 — stale fidelity calibration | #10 | 252 | environment/version-keyed calibration invalidation |

No production-grade/final-gate claim is allowed while an applicable blocker remains OPEN unless there is explicit dated risk acceptance.

## Operational correctness tranche

| Risk | Issue | RPN | Primary outcome |
|---|---:|---:|---|
| FMEA-005 | #7 | 240 | trustworthy independent impact/invalidation telemetry |
| FMEA-007 | #9 | 189 | idempotent durable repair and memory side effects |
| FMEA-012 | #14 | 175 | environment-bound visual baselines |
| FMEA-010 | #12 | 160 | adversarial cross-project memory isolation/poisoning proof |
| FMEA-006 | #8 | 189 | code-backed planning/documentation maturity reconciliation |
| FMEA-011 | #13 | 128 | enforced review/status/risk gates on `main` |

## Dependency graph

```text
FMEA-002 CLOSED   FMEA-004 CLOSED
       \             /
        +-----------+
              |
              v
         FMEA-001 CLOSED
              |
       +------+------+
       v             v
   FMEA-003       FMEA-009 CLOSED
 revision bind    independent repair acceptance
       |             |
       v             |
   FMEA-008 <---------+
 calibration validity
       |
       v
 VERIFICATION INTEGRITY BARRIER PASSED
       |
 +-----+--------+---------+
 v              v         v
FMEA-005     FMEA-007   FMEA-012
telemetry    idempotency baseline identity
 +--------------+---------+
                v
             FMEA-010
         memory isolation
                |
                v
             FMEA-006
      planning reconciliation
                |
                v
             FMEA-011
        enforced delivery gates
```

The graph is a default risk/value execution order; independent work may proceed in parallel when closure evidence remains independent.

## Current queue

### Barrier A — verification integrity

- [x] #4 / FMEA-002 — canonical Axiom scope. **CLOSED: residual RPN 20.**
- [x] #6 / FMEA-004 — runtime-attested TruthPath. **CLOSED: residual RPN 18.**
- [x] #3 / FMEA-001 — fail-closed minimum evidence tier. **CLOSED: residual RPN 20.**
- [x] #11 / FMEA-009 — independent final verification and held-out repair eval. **CLOSED: residual RPN 54; PR #20; CI #212; truthpath #17.**
- [ ] #5 / FMEA-003 — bind epoch/readiness to source/build/change digest. **CURRENT.**
- [ ] #10 / FMEA-008 — key/invalidate calibration by exact runtime environment.

### Barrier B — operational correctness

- [ ] #7 / FMEA-005 — split impact/invalidation stage measurement and accounting tests.
- [ ] #9 / FMEA-007 — idempotency keys/CAS + crash-boundary fault tests.
- [ ] #14 / FMEA-012 — canonical baseline environment identity and incompatible-baseline rejection.
- [ ] #12 / FMEA-010 — multi-project isolation and poisoning adversarial suite.

### Barrier C — planning and delivery trust

- [ ] #8 / FMEA-006 — reconcile `MASTER_PLAN.md`, README, docs and issues using four maturity states.
- [ ] #13 / FMEA-011 — enforce CI/review/risk gates on `main` where repository administration permits.

## Required task metadata / DoD

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent evidence
Residual target: S/O/D/RPN
Evidence: test / CI / runtime / fault injection / held-out eval
```

A mitigation is DONE only after implementation is merged, named independent evidence passes, register + machine state + tracker agree, and residual target is met or explicitly accepted.

## Current milestone metrics

```text
open_critical_risks = 2
open_high_risks = 6
sum_open_rpn = 1613
sum_closed_residual_rpn = 112
```

Also track false-PASS incidents, collector downgrade attempts, scope-equivalence failures, stale-revision rejections, TruthPath unavailable events, calibration invalidations, held-out repair regression escape rate, cross-project memory leakage, incompatible-baseline rejections and duplicate durable side effects.

## Planning reconciliation precedence

Until FMEA-006 closes:

```text
release-gate evidence
> operational runtime/fault/parity/held-out evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```

Re-run an FMEA delta whenever a PR changes evidence routing/fallback/legal PASS, impact/invalidation, render readiness/freshness, runtime/browser/model capability/version, verifier semantics, autonomous repair/completion, Axiom retries/side effects, memory boundaries, baseline identity, privacy/provenance, or CI/release policy.