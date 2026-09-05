# UiUxMaster FMEA execution overlay

Status: **ACTIVE**  
Authority: `MASTER_PLAN.md` remains the product roadmap. This file is the risk-gating overlay.  
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

Architecture maturity: `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`.

## Verification Integrity Barrier — PASSED

| Risk | Initial RPN | Residual | Closure |
|---|---:|---:|---|
| FMEA-002 — canonical Axiom scope | 420 | 20 | #4 / PR #17 / ci #191 / axiom #33 |
| FMEA-004 — runtime-attested TruthPath | 360 | 18 | #6 / PR #18 / ci #197 / truthpath #2 |
| FMEA-001 — fail-closed evidence tier | 320 | 20 | #3 / PR #19 / ci #204 / axiom #37 / truthpath #9 |
| FMEA-009 — independent repair completion | 315 | 54 | #11 / PR #20 / ci #212 / truthpath #17 |
| FMEA-003 — revision-bound freshness | 280 | 20 | #5 / PR #21 / ci #217 / axiom #39 / truthpath #22 |
| FMEA-008 — exact runtime calibration | 252 | 18 | #10 / PR #22 / ci #224 / axiom #45 / truthpath #29 |

## Operational correctness closures

### FMEA-005 — CLOSED

Initial `6/10/4 RPN=240` -> residual `6/1/2 RPN=12`. Issue #7, PR #23, merge `b574f86870c5e604998347f941fd961e3a8fc657`.

Closure: canonical `ResolveImpact` and `InvalidateImpact` stages; independent timers/counters; 35 ms one-stage-only delay tests; non-overlapping total accounting; packet/Pipeline agreement; independent `scope_stages.json` benchmark artifact; `ci` #230, `axiom-control` #47, `truthpath` #35 PASS.

### FMEA-007 — CLOSED

Initial `9/3/7 RPN=189` -> residual `9/1/2 RPN=18`. Issue #9, PR #24, merge `e9e1139f93ad47c785841e99bed69a7d427b872e`.

Closure:

- semantic side-effect identity excludes attempt number and is split into logical operation identity plus payload digest;
- same logical operation + same payload returns the original receipt; same logical operation + changed payload fails closed;
- source mutation uses exact expected-revision CAS;
- file-backed source target atomically persists source state and side-effect receipt with fsync + rename, so reopen after effect-before-completion crash reuses the original receipt;
- memory admission uses `CommitOnce`; exact semantic edges/conflict records are deduplicated even through legacy `Commit`;
- repair source/memory side effects occur only after independent final PASS and return audit receipts;
- memory admission errors are no longer silently swallowed;
- Axiom `iterate_repair` is an explicit `ExternalEffect` with 2-minute timeout, bounded retry and stable `{execution}:{node}` key;
- Axiom `ActivityRequest` execution/node/attempt/idempotency identity is exposed to adapters at the control-plane boundary;
- fault test injects `FailureTransient` after source + memory targets accept effects; retry reuses both target receipts under the same Axiom idempotency key;
- completed file-backed Axiom execution reloads without replaying effects;
- `ci` #237, `axiom-control` #50 and `truthpath` #42 PASS.

Reopen FMEA-007 if attempt enters effect identity, source CAS/atomic receipt can be bypassed, logical retry payload drift can create another effect, memory graph replay multiplies semantic state, external-effect timeout/retry/idempotency declaration is removed, or crash-window replay proof fails.

Boundary: persistence/isolation/poisoning semantics of future durable epistemic stores remain FMEA-010; FMEA-007 owns duplicate/replay semantics and the contract those adapters must preserve.

## Operational correctness tranche — CURRENT

| Order | Risk | Issue | RPN | Primary closure outcome |
|---:|---|---:|---:|---|
| 1 | **FMEA-012** — baseline environment mismatch | #14 | 175 | canonical render-environment identity + incompatible-baseline rejection |
| 2 | FMEA-010 — memory leakage/poisoning | #12 | 160 | adversarial multi-project isolation + promotion/retraction/rollback proof |

## Planning and delivery trust

| Order | Risk | Issue | RPN | Primary closure outcome |
|---:|---|---:|---:|---|
| 3 | FMEA-006 — planning/documentation drift | #8 | 189 | reconcile claims to the four maturity states |
| 4 | FMEA-011 — ungated `main` | #13 | 128 | required CI/review/risk gates and audited exceptions |

## Dependency graph

```text
Verification integrity risks CLOSED
            |
            v
FMEA-005 CLOSED -> FMEA-007 CLOSED
            |            |
            +------+-----+
                   v
             FMEA-012 CURRENT
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

## Current queue

### Barrier A — verification integrity

- [x] FMEA-002 — residual 20.
- [x] FMEA-004 — residual 18.
- [x] FMEA-001 — residual 20.
- [x] FMEA-009 — residual 54.
- [x] FMEA-003 — residual 20.
- [x] FMEA-008 — residual 18.

### Barrier B — operational correctness

- [x] #7 / FMEA-005 — residual 12; PR #23.
- [x] #9 / FMEA-007 — residual 18; PR #24.
- [ ] #14 / FMEA-012 — canonical baseline environment identity and incompatible-baseline rejection. **CURRENT.**
- [ ] #12 / FMEA-010 — multi-project isolation and poisoning adversarial suite.

### Barrier C — planning and delivery trust

- [ ] #8 / FMEA-006 — reconcile plan/docs/issues using four maturity states.
- [ ] #13 / FMEA-011 — enforce CI/review/risk gates on `main` where repository administration permits.

## Required task metadata / DoD

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
open_high_risks = 4
sum_open_rpn = 652
sum_closed_residual_rpn = 180
```

Planning reconciliation precedence until FMEA-006 closes:

```text
release-gate evidence
> operational runtime/fault/parity/held-out evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```

Re-run an FMEA delta whenever a PR changes evidence routing/fallback/legal PASS, impact/invalidation, render readiness/freshness, runtime/browser capability/version, calibration authority, verifier semantics, repair/completion, Axiom retries/side effects, memory boundaries, baseline identity, privacy/provenance, or CI/release policy.
