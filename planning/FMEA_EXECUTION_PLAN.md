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

Evidence:

- canonical durable Axiom change scope is lossless for run/project/source identity, files, tokens, nodes, routes, viewport/theme and validation need;
- Axiom cannot project an advisory plan back into a weaker canonical route;
- direct/Axiom scope and route equivalence tests pass;
- `ci` #191 and `axiom-control` #33 pass including real Chrome integration.

### FMEA-004 — CLOSED

Initial `9/5/8 RPN=360` -> residual `9/1/2 RPN=18`.  
Issue #6, PR #18, merge `7433e1bd39c431ab0ac69181e2caf0fde9dd1921`.

Evidence:

- a new Playwright adapter advertises zero usable L3 capability before readiness probe;
- worker protocol `1.0.0` and Playwright `1.62.1` are checked exactly;
- browser capability is emitted only after bundled executable discovery, real launch and non-empty browser version;
- missing worker is fail-closed;
- worker/Playwright/browser identity is carried in canonical `evidence.Packet.Renderer` provenance and evidence is rejected on post-probe identity drift;
- checked-in Node worker implements probe/capture/scenario and is versioned independently from the Go hot path;
- `truthpath` workflow #2 passes pinned runtime install plus real Chromium `TestTruthPathRealChromium` probe/capture/scenario;
- `ci` #197 passes module lock, full tests, race, vet, FastCDP Chromium integration and benchmarks.

Environment/version-keyed calibration invalidation is intentionally still owned by FMEA-008; FMEA-004 supplies the runtime identity FMEA-008 must key on.

## Verification Integrity Barrier — remaining open risks

| Order | Risk | Issue | RPN | Closure gate |
|---:|---|---:|---:|---|
| 1 | **FMEA-001** — silent L3 -> L2 downgrade | #3 | 320 | typed fail-closed unavailable/insufficient evidence + minimum actual tier attestation |
| 2 | FMEA-009 — repair-loop reward hacking | #11 | 315 | independent held-out/final verification |
| 3 | FMEA-003 — render freshness not revision-bound | #5 | 280 | expected/observed revision attestation |
| 4 | FMEA-008 — stale fidelity calibration | #10 | 252 | environment/version-keyed calibration invalidation |

No production-grade/final-gate claim is allowed while an applicable blocker remains OPEN unless there is explicit dated risk acceptance.

## Operational correctness tranche

| Order | Risk | Issue | RPN | Outcome |
|---:|---|---:|---:|---|
| 5 | FMEA-005 | #7 | 240 | independent impact/invalidation telemetry |
| 6 | FMEA-007 | #9 | 189 | idempotent durable repair/memory side effects |
| 7 | FMEA-012 | #14 | 175 | environment-bound visual baselines |
| 8 | FMEA-010 | #12 | 160 | adversarial cross-project memory isolation/poisoning proof |
| 9 | FMEA-006 | #8 | 189 | code-backed planning/documentation maturity reconciliation |
| 10 | FMEA-011 | #13 | 128 | enforced review/status/risk gates on `main` |

## Dependency graph

```text
FMEA-002 CLOSED   FMEA-004 CLOSED
       \             /
        +-----------+
              |
              v
          FMEA-001  <- CURRENT
       minimum-tier fail-closed
              |
              v
          FMEA-003
      revision freshness
              |
              v
          FMEA-008
      calibration validity
              |
              v
          FMEA-009
 independent repair acceptance
              |
              v
 VERIFICATION INTEGRITY BARRIER PASSED
              |
      +-------+--------+---------+
      v                v         v
   FMEA-005         FMEA-007   FMEA-012
   telemetry        idempotency baseline identity
      |                |         |
      +----------+-----+---------+
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

Parallel work is allowed when closure evidence remains independent.

## Current queue

### Barrier A — verification integrity

- [x] #4 / FMEA-002 — canonical scope through Axiom. **CLOSED: RPN 20.**
- [x] #6 / FMEA-004 — runtime-attested TruthPath. **CLOSED: RPN 18.**
- [ ] #3 / FMEA-001 — prohibit silent TruthPath downgrade; typed unavailable/insufficient-evidence semantics; minimum actual tier assertion. **CURRENT.**
- [ ] #11 / FMEA-009 — independent final verification and held-out repair eval.
- [ ] #5 / FMEA-003 — bind epoch/readiness to source/build/change digest.
- [ ] #10 / FMEA-008 — key and invalidate calibration by exact runtime environment.

### Barrier B — operational correctness

- [ ] #7 / FMEA-005 — split impact/invalidation stage measurement and accounting tests.
- [ ] #9 / FMEA-007 — idempotency keys/CAS + crash-boundary fault tests.
- [ ] #14 / FMEA-012 — canonical baseline environment identity and incompatible-baseline rejection.
- [ ] #12 / FMEA-010 — multi-project isolation and poisoning adversarial suite.

### Barrier C — planning and delivery trust

- [ ] #8 / FMEA-006 — reconcile `MASTER_PLAN.md`, README, docs and issues using four maturity states.
- [ ] #13 / FMEA-011 — enforce CI/review/risk gates on `main` where repository administration permits.

## Required task metadata

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent evidence
Residual target: S/O/D/RPN
Evidence: test / CI / runtime / fault injection / eval
```

A mitigation is DONE only when implementation is merged, named closure evidence passes, register + machine state + tracker agree, and residual target is met or explicitly accepted.

## Current milestone metrics

```text
open_critical_risks = 4
open_high_risks = 6
sum_open_rpn = 2248
sum_closed_residual_rpn = 38
```

Also track false-PASS incidents, collector downgrade attempts, scope-equivalence failures, stale-revision rejections, TruthPath unavailable events, calibration invalidations, held-out repair regression escape rate, cross-project memory leakage, incompatible-baseline rejections and duplicate durable side effects.

## Planning reconciliation rule

Until FMEA-006 closes:

```text
release-gate evidence
> operational runtime/fault/parity evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```

Re-run an FMEA delta whenever a PR changes evidence routing/fallback, legal PASS semantics, impact/invalidation, render readiness/freshness, runtime/browser/model capability/version, verifier semantics, autonomous repair, Axiom retries/side effects, memory boundaries, baseline identity, privacy/provenance, or CI/release policy.
