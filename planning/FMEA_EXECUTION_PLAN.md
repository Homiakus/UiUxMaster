# UiUxMaster FMEA execution overlay

Status: **ACTIVE**  
Authority: `MASTER_PLAN.md` remains the single product execution roadmap. This file is a **risk-gating overlay**, not a competing roadmap.  
Risk source of truth: `docs/FMEA_RISK_REGISTER.md`  
Machine state: `planning/fmea-risk.json`  
Governance ADR: `docs/adr/0002-fmea-risk-governance.md`  
Operational tracker: GitHub issue #15

## Purpose

Every architecture task is evaluated not only by implementation status, but by the engineering risk it creates, mitigates, monitors, accepts, or leaves unchanged.

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

Architecture maturity uses four states:

1. `IMPLEMENTED` — code/type/function exists and focused tests pass.
2. `INTEGRATED` — capability participates in the intended canonical product path.
3. `OPERATIONALLY_PROVEN` — real runtime / fault / parity / held-out evidence proves the claim.
4. `RELEASE_GATED` — required FMEA closure gates and release checks are satisfied.

A task may be IMPLEMENTED while its associated architecture risk remains OPEN.

## Closed risk evidence

### FMEA-002 — CLOSED

Initial: `S=10 O=6 D=7 RPN=420`  
Residual: `S=10 O=1 D=2 RPN=20`  
Issue: #4  
Implementation/closure PR: #17  
Merge: `468abe87ac76d54b3e887951e30b71e300c450d7`

Closure evidence:

- canonical durable Axiom `Change` carries stable run/project/source identity, files, tokens, nodes, routes, viewports, themes, whole-site override, base target and complete durable evidence need;
- Axiom Pipeline mode no longer maps its advisory `EvidencePlan` back into `engine.EvidenceNeed` and therefore cannot independently narrow the canonical request/tier;
- `TestPipelineAdapterPreservesCanonicalScopeAndRoute` proves direct-pipeline/Axiom scope and route equivalence;
- `TestPipelineAdapterDoesNotAllowAxiomPlanToNarrowCanonicalRequest` proves stale/empty Axiom planning cannot select weaker evidence;
- `ci` run #191 passed test, race, vet, real FastCDP Chromium integration and benchmarks;
- `axiom-control` run #33 passed module-lock/isolation, tests, race, vet, Chrome discovery and real `TestAxiomFastCDPEndToEndIntegration`.

Re-open FMEA-002 if any future change drops canonical request fields, reintroduces independent Axiom route selection, or breaks equivalence tests.

## Global release barrier

The remaining open verification-integrity blockers are:

| Order | Risk | Issue | Initial RPN | Gate |
|---:|---|---:|---:|---|
| 1 | FMEA-004 — optimistic TruthPath capabilities | #6 | 360 | real worker/browser readiness + real CI capture |
| 2 | FMEA-001 — silent L3 -> L2 downgrade | #3 | 320 | fail-closed minimum-tier attestation |
| 3 | FMEA-009 — repair-loop reward hacking | #11 | 315 | independent held-out/final verification |
| 4 | FMEA-003 — render freshness not revision-bound | #5 | 280 | expected/observed revision attestation |
| 5 | FMEA-008 — stale fidelity calibration | #10 | 252 | environment/version-keyed calibration invalidation |

No production-grade/final-gate claim is allowed while an applicable blocker remains OPEN unless there is explicit dated risk acceptance.

## Second risk tranche

After the Verification Integrity Barrier:

| Order | Risk | Issue | Initial RPN | Primary outcome |
|---:|---|---:|---:|---|
| 6 | FMEA-005 | #7 | 240 | trustworthy independent impact/invalidation telemetry |
| 7 | FMEA-007 | #9 | 189 | idempotent durable repair and memory side effects |
| 8 | FMEA-012 | #14 | 175 | environment-bound visual baselines |
| 9 | FMEA-010 | #12 | 160 | adversarial cross-project memory isolation/poisoning proof |
| 10 | FMEA-006 | #8 | 189 | code-backed plan/documentation maturity reconciliation |
| 11 | FMEA-011 | #13 | 128 | enforced review/status/risk gates on `main` |

## Dependency graph

```text
FMEA-002 CLOSED: canonical Axiom request equivalence
       |
       +-------------------+
       v                   v
   FMEA-004             FMEA-001
 TruthPath ready       fail-closed tier
       |                   |
       +---------+---------+
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

Parallel work is allowed when closure evidence remains independent.

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

A mitigation is DONE only when:

- implementation is merged;
- named closure tests/evals pass;
- fault/held-out verification exists where specified;
- `docs/FMEA_RISK_REGISTER.md` is re-scored;
- `planning/fmea-risk.json` matches the register;
- residual target is met or explicitly accepted;
- tracker #15 and mitigation issue are synchronized;
- no discovered failure mode is left without a stable `FMEA-###` ID.

## Milestone metrics

Every architecture milestone review reports at least:

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
- every final-gate PASS attests actual evidence tier and environment identity;
- every production release records the applicable FMEA snapshot.

## Current execution queue

### Barrier A — verification integrity

- [x] #4 / FMEA-002 — canonical scope through Axiom. **CLOSED: residual RPN 20; PR #17; CI #191; axiom-control #33.**
- [ ] #6 / FMEA-004 — runtime capability discovery; validated worker; real worker/browser smoke; provenance versions. **CURRENT.**
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

Until #8 is completed, safety/reliance precedence is:

```text
release-gate evidence
> operational runtime/fault/parity evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```

Do not downgrade a known residual risk because a historical task says `DONE`.

## Change review trigger

Re-run an FMEA delta whenever a PR changes evidence routing/fallback, legal PASS semantics, impact/invalidation, render readiness/freshness, runtime/browser/model capabilities or versions, verifier semantics, autonomous repair/completion, Axiom retries/durable side effects, SncSinCore memory boundaries, SkillState evolution, baseline identity, privacy/provenance, or CI/release policy.

This overlay is synchronized only when the risk register, machine state, mitigation issues, tracker #15 and release evidence agree.
