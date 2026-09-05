# UiUxMaster FMEA execution overlay

Status: **ACTIVE**  
Product-roadmap authority: `MASTER_PLAN.md`  
Risk authority: `docs/FMEA_RISK_REGISTER.md`  
Machine state: `planning/fmea-risk.json`  
Tracker: issue #15

Architecture maturity: `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`.

## Closure sequence

```text
MASTER_PLAN task
 -> affected FMEA IDs
 -> mitigation issue / PR
 -> independent closure evidence
 -> residual S/O/D + RPN
 -> register + machine mirror + tracker sync
 -> CLOSED
```

## Completed barriers

### Verification integrity — PASSED

| Risk | Initial | Residual | Evidence |
|---|---:|---:|---|
| FMEA-002 | 420 | 20 | PR #17 / ci #191 / axiom #33 |
| FMEA-004 | 360 | 18 | PR #18 / ci #197 / truthpath #2 |
| FMEA-001 | 320 | 20 | PR #19 / ci #204 / axiom #37 / truthpath #9 |
| FMEA-009 | 315 | 54 | PR #20 / ci #212 / truthpath #17 |
| FMEA-003 | 280 | 20 | PR #21 / ci #217 / axiom #39 / truthpath #22 |
| FMEA-008 | 252 | 18 | PR #22 / ci #224 / axiom #45 / truthpath #29 |

### Operational correctness closures

| Risk | Initial | Residual | Evidence |
|---|---:|---:|---|
| FMEA-005 — independent impact/invalidation telemetry | 240 | 12 | PR #23 / ci #230 / axiom #47 / truthpath #35 |
| FMEA-007 — exactly-once durable effects | 189 | 18 | PR #24 / ci #237 / axiom #50 / truthpath #42 |
| FMEA-012 — protected-baseline environment identity | 175 | **14** | PR #25 / ci #247 / axiom #56 / truthpath #52 |

### FMEA-012 closure

- canonical versioned render-environment identity includes renderer/worker/browser/engine versions, platform, viewport, DPR, theme, font-set digest, locale, timezone and fixture revision;
- missing material dimensions are fail-closed;
- exact baseline/candidate environment key equality is required before mask/tolerance/pixel comparison;
- dispatcher no longer has a direct protected `BaselineRGBA -> CompareRGBA` path;
- candidate environment is persisted in canonical packet provenance;
- stored baseline digest is verified against actual pixels;
- baseline replacement is reviewed version+digest CAS with old/new digest/environment/version/rationale/reviewer history;
- masks cannot leave visible semantic owner bounds;
- comparator instability and baseline churn are partitioned by environment key;
- final head `39a8c35d82fb46a324a10b9b369a8b9867996044`, merge `64053497b52cffd83de37ef3396aee2cdef4354a`.

Reopen if environment compatibility moves after diff/tolerance, material dimensions become wildcards, baseline update bypasses reviewed CAS, digest/pixels diverge, mask ownership can be escaped or packet environment provenance is lost.

## CURRENT — FMEA-010 memory isolation / poisoning

Issue #12. Initial/current `S=10 O=2 D=8 RPN=160`. Residual target `10/1/3 RPN=30`.

### Architecture problem

Current read/query/context-pack paths use `CanAccess`, but read filtering does not make admission/promotion safe. `AdmissionRequest` still accepts a caller-selected target namespace. Therefore source provenance/scope must become a write-side invariant rather than trusting destination selection.

### Required implementation order

1. **Canonical scope lineage**
   - bind every admitted atom/bundle to source scope/project in provenance;
   - reject source-scope / target-namespace mismatch;
   - never infer global eligibility merely because target is global.

2. **Explicit promotion gate**
   - project-private -> global is a distinct operation, not ordinary `Commit`;
   - require policy, independent evidence, minimum confidence/coverage and no unresolved conflict/counterexample;
   - preserve source atom; global promotion is a traceable derived assertion.

3. **Isolation and poisoning adversarial suite**
   - project A data must be invisible/unadmittable to project B except approved global knowledge;
   - forged project scope, stale provenance, contradictory facts, low-confidence and malicious tags/payloads fail or stay quarantined;
   - conflicting evidence preserves both truth claims + conflict state instead of overwrite.

4. **Retraction/supersede/rollback**
   - retracting or invalidating promoted evidence revokes global visibility while preserving source/private history;
   - rollback is idempotent and retains FMEA-007 durable effect semantics.

5. **Shadow/replay/non-regression evidence**
   - promotion candidate can be replayed deterministically;
   - shadow evaluation proves no protected/global regression before activation;
   - tests cover promotion, rejection, retraction and rollback across multiple projects.

### FMEA-010 closure gates

- adversarial project-A/project-B/global read **and write** isolation;
- no private-to-global promotion through ordinary admission;
- source provenance cannot be forged to cross namespace boundaries;
- poisoning corpus and conflict-preservation tests;
- promotion has explicit evidence/provenance and deterministic decision digest;
- retraction/rollback removes promoted visibility but not source history;
- existing FMEA-007 idempotency and FMEA-009 independent-completion guards remain green;
- root test/race/vet plus triggered Axiom/TruthPath workflows pass.

## Remaining planning/delivery risks

After FMEA-010:

1. FMEA-006 — planning/documentation reconciliation, RPN 189 -> target 28.
2. FMEA-011 — `main` delivery enforcement, RPN 128 -> target 16.

## Current queue

- [x] FMEA-002 — residual 20.
- [x] FMEA-004 — residual 18.
- [x] FMEA-001 — residual 20.
- [x] FMEA-009 — residual 54.
- [x] FMEA-003 — residual 20.
- [x] FMEA-008 — residual 18.
- [x] FMEA-005 — residual 12.
- [x] FMEA-007 — residual 18.
- [x] FMEA-012 — residual 14; PR #25.
- [ ] **FMEA-010 — CURRENT.**
- [ ] FMEA-006.
- [ ] FMEA-011.

## Metrics

```text
open_critical_risks = 0
open_high_risks = 3
sum_open_rpn = 477
sum_closed_residual_rpn = 194
```

## Task metadata / DoD

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent evidence
Residual target: S/O/D/RPN
Evidence: test / CI / runtime / fault injection / held-out eval
```

Until FMEA-006 closes, evidence precedence is:

```text
release-gate evidence
> operational runtime/fault/parity/held-out evidence
> integrated canonical-path tests
> focused implementation tests
> checklist/status prose
```
