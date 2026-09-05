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

## Verification Integrity Barrier — PASSED

All six verification-integrity risks now have merged implementation, executable closure evidence and residual re-score.

| Risk | Initial RPN | Residual RPN | Closure |
|---|---:|---:|---|
| FMEA-002 — canonical Axiom scope | 420 | **20** | #4 / PR #17 / `ci` #191 / `axiom-control` #33 |
| FMEA-004 — runtime-attested TruthPath | 360 | **18** | #6 / PR #18 / `ci` #197 / `truthpath` #2 |
| FMEA-001 — fail-closed evidence tier | 320 | **20** | #3 / PR #19 / `ci` #204 / `axiom-control` #37 / `truthpath` #9 |
| FMEA-009 — independent repair completion | 315 | **54** | #11 / PR #20 / `ci` #212 / `truthpath` #17 |
| FMEA-003 — revision-bound render freshness | 280 | **20** | #5 / PR #21 / `ci` #217 / `axiom-control` #39 / `truthpath` #22 |
| FMEA-008 — runtime calibration validity | 252 | **18** | #10 / PR #22 / `ci` #224 / `axiom-control` #45 / `truthpath` #29 |

### FMEA-003 closure

- FastCDP readiness is a framework-neutral `RenderToken{epoch, revision}` rather than a numeric epoch alone;
- revision-bound waiters reject stale, wrong and revisionless signals even when the numeric epoch is newer;
- expected/observed revision identity is carried in canonical packet provenance;
- canonical Pipeline rejects missing/mismatched freshness before verifier/evaluation;
- mismatch discards/resets the warm page and recovery starts a new matching lineage;
- real Axiom/Chromium integration proves matching revision, wrong-revision rejection and recovery.

Reopen FMEA-003 if epoch-only state can satisfy a revision-bound request, packet provenance loses expected/observed revision, a mismatch can reach verifier/PASS, or real HMR mismatch/recovery CI fails.

### FMEA-008 closure

- `CalibrationMatrix` defines which tier may in principle prove an evidence class; runtime `CalibrationAuthority` separately decides whether the exact current L1/L2 ↔ TruthPath pair is calibrated;
- `CalibrationContext` keys approximate and TruthPath renderer/browser/worker/runtime/platform/profile/viewport-related identity;
- calibration records persist exact environment key, corpus digest, artifact reference, sample coverage/quality and age/expiry;
- canonical `PipelineResult.PassAuthority` separates diagnostic cleanliness from permission to issue PASS;
- every evidence class claimed by an L1/L2 result needs its own current matching calibration;
- missing, expired, weak-corpus or environment-mismatched calibration is explicit evidence insufficiency/upward escalation, never PASS;
- L3 TruthPath remains authoritative without approximate-tier calibration;
- Axiom canonical path always requires legal-pass authority; legacy collector-only mode is diagnostic and cannot issue `DecisionPass`;
- `TestTruthPathCalibrationRealChromium` derives exact FastCDP and Playwright/Chromium identities from actually running runtimes and invalidates the old record after TruthPath identity drift;
- `ci` #224, `axiom-control` #45 and `truthpath` #29 PASS.

Reopen FMEA-008 if an L1/L2 PASS path bypasses `PassAuthority`, a material runtime/environment dimension is omitted from the key, stale/expired/weak records remain legal, one evidence class can substitute for another, or the real runtime-drift proof fails.

## Operational correctness tranche — CURRENT

| Order | Risk | Issue | RPN | Primary closure outcome |
|---:|---|---:|---:|---|
| 1 | **FMEA-005** — false impact/invalidation telemetry split | #7 | 240 | independently measured stages + accounting/benchmark proof |
| 2 | FMEA-007 — duplicate durable side effects | #9 | 189 | idempotency keys/CAS + crash-boundary replay proof |
| 3 | FMEA-012 — baseline environment mismatch | #14 | 175 | canonical environment identity + incompatible-baseline rejection |
| 4 | FMEA-010 — memory leakage/poisoning | #12 | 160 | adversarial multi-project isolation + promotion/retraction/rollback proof |

## Planning and delivery trust

| Order | Risk | Issue | RPN | Primary closure outcome |
|---:|---|---:|---:|---|
| 5 | FMEA-006 — planning/documentation drift | #8 | 189 | reconcile claims to the four maturity states with executable evidence |
| 6 | FMEA-011 — ungated `main` | #13 | 128 | required CI/review/risk gates and audited exceptions |

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
 FMEA-003 CLOSED  FMEA-009 CLOSED
       |             |
       +------+------+
              v
         FMEA-008 CLOSED
              |
              v
 VERIFICATION INTEGRITY BARRIER PASSED
              |
       +------+-------+
       v              v
 FMEA-005 CURRENT   FMEA-007
 telemetry          idempotency
       |              |
       +------+-------+
              v
           FMEA-012
      baseline identity
              |
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
      delivery enforcement
```

The graph is the default risk/value execution order; independent work may proceed in parallel only when closure evidence remains independent and shared invariants are preserved.

## Current queue

### Barrier A — verification integrity

- [x] #4 / FMEA-002 — canonical Axiom scope. **CLOSED: residual RPN 20.**
- [x] #6 / FMEA-004 — runtime-attested TruthPath. **CLOSED: residual RPN 18.**
- [x] #3 / FMEA-001 — fail-closed minimum evidence tier. **CLOSED: residual RPN 20.**
- [x] #11 / FMEA-009 — independent final verification/held-out repair eval. **CLOSED: residual RPN 54.**
- [x] #5 / FMEA-003 — revision-bound render freshness. **CLOSED: residual RPN 20.**
- [x] #10 / FMEA-008 — exact runtime calibration authority. **CLOSED: residual RPN 18.**

### Barrier B — operational correctness

- [ ] #7 / FMEA-005 — split impact/invalidation stage measurement and accounting tests. **CURRENT.**
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
open_critical_risks = 0
open_high_risks = 6
sum_open_rpn = 1081
sum_closed_residual_rpn = 150
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

Re-run an FMEA delta whenever a PR changes evidence routing/fallback/legal PASS, impact/invalidation, render readiness/freshness, runtime/browser/model capability/version, calibration authority, verifier semantics, autonomous repair/completion, Axiom retries/side effects, memory boundaries, baseline identity, privacy/provenance, or CI/release policy.