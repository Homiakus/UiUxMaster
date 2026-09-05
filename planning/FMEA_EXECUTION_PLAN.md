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

Implemented controls:

- `TierTruthPath` no longer has any implicit L2 fallback;
- unknown/unsupported dispatcher route fails with `ErrInvalidRoute` rather than using L2;
- unavailable policy-selected collectors return typed `ErrCollectorUnavailable`;
- `internal/engine` defines a protocol-neutral monotonic evidence-strength ladder and normalizes physical packet tiers (`L0`…`L4`) plus descriptive routing tiers;
- standard dispatcher attests actual packet strength against the policy-selected minimum before returning evidence;
- canonical `engine.Pipeline` independently repeats the attestation before deterministic verifier or engine evaluation, protecting callers that supply a custom collector;
- L1→L2 upward escalation remains legal; weaker substitution is rejected;
- L4 semantic remains a post-collection stage and therefore requires at least L2 browser evidence at its collector boundary.

Closure evidence:

- `TestFMEA001TruthPathUnavailableDoesNotDowngradeToL2`;
- `TestFMEA001TruthPathRejectsWeakerPacketFromConfiguredL3`;
- `TestFMEA001TruthPathAcceptsAttestedL3Packet`;
- `TestFMEA001UnknownRouteDoesNotDefaultToL2`;
- `TestFMEA001UpwardL1ToL2EscalationRemainsLegal`;
- `TestPipelineRejectsWeakCustomCollectorBeforeVerifier`;
- `TestTruthPathDispatcherRealChromium` proves a real runtime-attested Playwright/Chromium L3 packet through the standard dispatcher guard;
- `ci` #204 PASS;
- `axiom-control` #37 PASS;
- `truthpath` #9 PASS.

Reopen FMEA-001 if any required route can substitute weaker evidence, unknown routes obtain a usable fallback, tier attestation moves after verifier/evaluation, or the real L3 dispatcher test fails.

## Verification Integrity Barrier — remaining open risks

| Order | Risk | Issue | RPN | Closure gate |
|---:|---|---:|---:|---|
| 1 | **FMEA-009** — repair-loop verifier overfit / reward hacking | #11 | 315 | independent final verifier + held-out/perturbed scenarios + escape metric |
| 2 | FMEA-003 — render freshness not revision-bound | #5 | 280 | expected/observed source/build revision attestation |
| 3 | FMEA-008 — stale fidelity calibration | #10 | 252 | environment/version-keyed calibration invalidation |

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
   FMEA-003       FMEA-009 <- CURRENT
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
- [ ] #11 / FMEA-009 — independent final verification and held-out repair eval. **CURRENT.**
- [ ] #5 / FMEA-003 — bind epoch/readiness to source/build/change digest.
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
open_critical_risks = 3
open_high_risks = 6
sum_open_rpn = 1928
sum_closed_residual_rpn = 58
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
