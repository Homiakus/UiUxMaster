# UiUxMaster FMEA execution overlay

Status: **ACTIVE**  
Authority: `MASTER_PLAN.md` remains the single product execution roadmap. This file is a **risk-gating overlay**, not a competing roadmap.  
Risk source of truth: `docs/FMEA_RISK_REGISTER.md`  
Governance ADR: `docs/adr/0002-fmea-risk-governance.md`  
Operational tracker: GitHub issue #15

## Purpose

Every architecture task is now evaluated not only by implementation status, but by the engineering risk it creates, mitigates, monitors, accepts, or leaves unchanged.

The planning model is:

```text
MASTER_PLAN task
  -> affected FMEA IDs
  -> mitigation issue(s)
  -> independent closure gate
  -> residual S/O/D + RPN
  -> only then architectural risk may close
```

`internal/fidelity.RiskLevel` remains runtime evidence-routing data. `FMEA-###` remains engineering/planning risk data. They must never share one field or enum.

## Planning status model

Do not use a single ambiguous DONE flag for architecture maturity. Use these four states in reviews and future plan reconciliation:

1. `IMPLEMENTED` — code/type/function exists and focused tests pass.
2. `INTEGRATED` — capability participates in the intended canonical product path.
3. `OPERATIONALLY_PROVEN` — real runtime / fault / parity / held-out evidence proves the claim.
4. `RELEASE_GATED` — required FMEA closure gates and release checks are satisfied.

A task may be IMPLEMENTED while its associated architecture risk remains OPEN.

## Global release barrier

The following risks are verification-integrity blockers. No production-grade/final-gate claim is allowed while any applicable blocker remains OPEN unless it has an explicit dated risk acceptance.

| Order | Risk | Issue | Initial RPN | Gate |
|---:|---|---:|---:|---|
| 1 | FMEA-002 — Axiom canonical change scope loss | #4 | 420 | direct-pipeline/Axiom scope and route equivalence |
| 2 | FMEA-004 — optimistic TruthPath capabilities | #6 | 360 | real worker/browser readiness + real CI capture |
| 3 | FMEA-001 — silent L3 -> L2 downgrade | #3 | 320 | fail-closed minimum-tier attestation |
| 4 | FMEA-009 — repair-loop reward hacking | #11 | 315 | independent held-out/final verification |
| 5 | FMEA-003 — render freshness not revision-bound | #5 | 280 | expected/observed revision attestation |
| 6 | FMEA-008 — stale fidelity calibration | #10 | 252 | environment/version-keyed calibration invalidation |

These six form the **Verification Integrity Barrier**.

## Second risk tranche

After the Verification Integrity Barrier, execute the following in risk/value order:

| Order | Risk | Issue | Initial RPN | Primary outcome |
|---:|---|---:|---:|---|
| 7 | FMEA-005 | #7 | 240 | trustworthy independent impact/invalidation telemetry |
| 8 | FMEA-007 | #9 | 189 | idempotent durable repair and memory side effects |
| 9 | FMEA-006 | #8 | 189 | code-backed plan/documentation maturity reconciliation |
| 10 | FMEA-012 | #14 | 175 | environment-bound visual baselines |
| 11 | FMEA-010 | #12 | 160 | adversarial cross-project memory isolation/poisoning proof |
| 12 | FMEA-011 | #13 | 128 | enforced review/status/risk gates on `main` |

## Dependency graph

```text
FMEA-002  Axiom request equivalence
   |\
   | +----------------------+
   v                        v
FMEA-001                 FMEA-003
minimum tier             revision freshness
   |                        |
   +-----------+------------+
               v
            FMEA-004
      operational TruthPath
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
     +---------+----------+----------+
     v                    v          v
 FMEA-005              FMEA-007   FMEA-012
 telemetry             idempotency baseline identity
     |                    |          |
     +-------------+------+----------+
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

The diagram is a default execution ordering, not a claim that every item is strictly technically dependent on the previous one. Parallel work is allowed when its closure evidence remains independent.

## Required task metadata

Every new or changed architectural task touching a Critical/High risk must carry:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S=<n> O=<n> D=<n> RPN=<n>
Risk gate: <exact test/eval/fault-injection/review evidence>
Residual target: S=<n> O=<n> D=<n> RPN=<n>
Evidence: <test name / benchmark artifact / PR / issue / runtime proof>
```

If `Risk action: none`, the task must explain why the architecture delta cannot materially affect an existing failure mode.

## Definition of Done for a mitigation task

A mitigation issue is not DONE when code is written. It is DONE only when all of the following are true:

- implementation is merged;
- named closure tests/evals pass;
- failure injection or held-out verification exists where specified;
- `docs/FMEA_RISK_REGISTER.md` is re-scored;
- residual target is met or residual risk is explicitly accepted;
- linked tracker item in #15 is updated;
- no new failure mode was discovered without a new `FMEA-###` ID.

## Milestone metrics

Every architecture milestone review must report at least:

```text
open_critical_risks
open_high_risks
sum_open_rpn
sum_residual_rpn
false_pass_incidents
scope_equivalence_failures
stale_revision_rejections
truthpath_unavailable_events
calibration_invalidations
heldout_repair_regression_escape_rate
cross_project_memory_leakage_rate
baseline_incompatibility_rejections
idempotency_replay_duplicates
```

Target invariants:

- `false_pass_incidents = 0` for known fail-open paths;
- `cross_project_memory_leakage_rate = 0`;
- `idempotency_replay_duplicates = 0`;
- every final-gate PASS attests the actual evidence tier and environment identity;
- every production release records the applicable FMEA snapshot.

## Current execution queue

### Barrier A — verification integrity

- [ ] #4 / FMEA-002 — preserve canonical changed files/tokens/nodes through Axiom; remove planner divergence; stable run identity.
- [ ] #6 / FMEA-004 — runtime capability discovery; real worker/browser smoke; provenance versions.
- [ ] #3 / FMEA-001 — prohibit silent TruthPath downgrade; typed insufficient-evidence outcome; minimum-tier assertion.
- [ ] #11 / FMEA-009 — independent final verification and held-out repair eval.
- [ ] #5 / FMEA-003 — bind epoch/readiness to source/build/change digest.
- [ ] #10 / FMEA-008 — key and invalidate calibration by exact runtime environment.

### Barrier B — operational correctness

- [ ] #7 / FMEA-005 — split impact/invalidation stage measurement and accounting tests.
- [ ] #9 / FMEA-007 — idempotency keys/CAS + crash-boundary fault tests.
- [ ] #14 / FMEA-012 — canonical baseline environment identity and incompatible-baseline rejection.
- [ ] #12 / FMEA-010 — multi-project isolation and poisoning adversarial suite.

### Barrier C — planning and delivery trust

- [ ] #8 / FMEA-006 — reconcile `MASTER_PLAN.md`, README, docs, old issues and real code state using the four maturity states.
- [ ] #13 / FMEA-011 — enforce required CI/review/risk gates on `main` where repository administration permits.

## Planning reconciliation rule

`MASTER_PLAN.md` currently contains historical status text that can conflict with later task records. Until #8 is completed, the following precedence applies when deciding whether a capability is safe to depend on:

```text
release-gate evidence
> operational runtime/fault/parity evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```

Do not downgrade a known residual risk merely because a historical task says `DONE`.

## Change review trigger

Re-run an FMEA delta whenever a PR changes any of:

- evidence routing or fallback;
- legal PASS rules;
- impact/invalidation semantics;
- render readiness/freshness;
- runtime/browser/model capabilities or versions;
- verifier semantics;
- autonomous repair or completion policy;
- Axiom retries/durable side effects;
- SncSinCore namespace/admission/retrieval;
- SkillState evolution/promotion;
- baseline identity/update;
- privacy/provenance;
- CI/release policy.

This overlay is complete only when the risk register, mitigation issues, tracker #15 and release evidence agree.