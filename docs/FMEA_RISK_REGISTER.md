# UiUxMaster Architecture FMEA Risk Register

- Status: **ACTIVE**
- Initial review: 2026-09-04
- Last re-score: 2026-09-04
- Scope: architecture, verification integrity, orchestration, memory/evolution, baselines, delivery controls
- Planning authority: `MASTER_PLAN.md` remains the single execution-order authority. This register is the authoritative engineering-risk ledger used by plan tasks, issues, architecture review and release gates.
- Machine-readable mirror: `planning/fmea-risk.json`
- Execution overlay: `planning/FMEA_EXECUTION_PLAN.md`
- Decision record: `docs/adr/0002-fmea-risk-governance.md`

## 1. Risk-domain separation

UiUxMaster has two different risk domains and they must never share one field or enum.

`internal/fidelity.RiskLevel` answers: how likely is an approximate renderer to diverge from browser truth, and which evidence tier is required?

`FMEA-###` answers: how can the architecture, verifier, control plane, memory, repair loop or delivery process fail; what is the effect; how likely is it; how hard is it to detect; and what evidence is required to reduce the risk?

## 2. Scoring and status

Use integer scores 1–10:

- **Severity (S)**: 1 negligible; 10 can invalidate system trust, create destructive behavior, leak protected data, or produce consequential false PASS.
- **Occurrence (O)**: 1 exceptional/strongly prevented; 10 expected/frequent.
- **Detection difficulty (D)**: 1 almost certainly detected before escape; 10 likely to escape controls.
- **RPN** = `S × O × D`.

Priority:

| Priority | Default trigger |
|---|---|
| Critical | `RPN >= 250`, or Severity 9–10 with credible false-PASS/security/destructive-integrity path |
| High | `RPN 120–249` |
| Medium | `RPN 60–119` |
| Low | `RPN < 60` |

RPN is a prioritization aid, not an acceptance rule.

Status vocabulary:

- `OPEN` — active failure mode; mitigation incomplete;
- `MITIGATING` — implementation/verification in progress;
- `ACCEPTED` — residual risk explicitly accepted with rationale and review date;
- `CLOSED` — closure gate passed, implementation merged, residual score and evidence recorded.

## 3. Planning integration contract

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S=<n> O=<n> D=<n> RPN=<n>
Risk gate: <exact test/eval/fault-injection/review evidence>
Residual target: S=<n> O=<n> D=<n> RPN=<n>
Evidence: <test/artifact/PR/runtime proof>
```

Rules:

1. Every Critical/High risk has an executable mitigation issue.
2. Code existence does not close risk.
3. Closure requires the named independent evidence and residual re-score.
4. Architecture-changing PRs list affected FMEA IDs or `none` with rationale.
5. New failure modes receive new stable IDs.
6. Re-scoring records why O or D changed. Severity normally stays fixed unless the failure effect changes.
7. Accepted Critical risk requires explicit maintainer approval plus review/expiry date.
8. Release/final-gate work cannot cross an unresolved applicable Critical risk that invalidates its evidence.
9. Closed risks are regression guards: breaking their closure invariant re-opens the same ID.

### Architecture-delta triggers

Perform an FMEA delta review whenever a PR changes routing/fallback/legal PASS semantics, impact/invalidation, warm-state readiness/freshness, browser/renderer/model capability claims or versions, verifier semantics, repair/completion gates, Axiom retry/durability/external side effects, memory namespaces/admission/evolution, privacy/provenance, protected-baseline identity/update policy, or CI/release gates.

## 4. Risk summary

| ID | Failure mode | Initial S/O/D | Initial RPN | Current S/O/D | Current RPN | Priority | Status | Mitigation |
|---|---|---:|---:|---:|---:|---|---|---|
| FMEA-001 | Required TruthPath silently downgrades to L2 | 10/4/8 | 320 | 10/4/8 | 320 | Critical | OPEN | #3 |
| FMEA-002 | Axiom loses canonical change/ImpactSet scope | 10/6/7 | 420 | **10/1/2** | **20** | Low residual | **CLOSED** | #4 / PR #17 |
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10/4/7 | 280 | 10/4/7 | 280 | Critical | OPEN | #5 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9/5/8 | 360 | 9/5/8 | 360 | Critical | OPEN | #6 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6/10/4 | 240 | 6/10/4 | 240 | High | OPEN | #7 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7/9/3 | 189 | 7/9/3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9/3/7 | 189 | 9/3/7 | 189 | High | OPEN | #9 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9/4/7 | 252 | 9/4/7 | 252 | Critical | OPEN | #10 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9/5/7 | 315 | 9/5/7 | 315 | Critical | OPEN | #11 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10/2/8 | 160 | 10/2/8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8/4/4 | 128 | 8/4/4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under incompatible render environment | 7/5/5 | 175 | 7/5/5 | 175 | High | OPEN | #14 |

Current engineering estimates are not field statistics. Re-score with production/held-out telemetry when available.

## 5. Detailed risks

### FMEA-001 — Silent TruthPath downgrade

**Failure mode:** policy selects L3 TruthPath, but absent L3 collection can execute on L2 FastBrowser.  
**Effect:** clean-state/final-gate/cross-browser/calibration requirements may be judged on weaker evidence, creating false PASS.  
**Owner:** `internal/runtime/dispatcher`, `internal/engine`.  
**Mitigation:** #3.  
**Closure gate:** L3 unavailable returns typed insufficient-evidence/collector-unavailable result; actual packet tier is attested against minimum selected tier; final/clean-state/cross-browser gates cannot PASS from L2 substitution; only explicitly legal escalation/fallback remains.  
**Residual target:** `S=10 O=1 D=2 RPN=20`.

### FMEA-002 — Axiom canonical-pipeline scope loss — CLOSED

**Failure mode:** Axiom entered `engine.Pipeline` through a lossy `Change` projection and could independently project an advisory evidence plan back into canonical `EvidenceNeed`.  
**Effect:** `ImpactSet`/`ValidationScope` could detach from the actual edit and affected UI could be omitted from verification.  
**Owner:** `control/axiom/controlplane`, `control/axiom/uiuxadapter`, `internal/engine`.  
**Mitigation:** #4, PR #17.  

**Implemented controls:**

- durable Axiom `Change` now carries stable `RunID`, `ProjectID`, `SourceDigest`, changed files/tokens/nodes, target routes, viewports, themes, whole-site override, base target and a complete protocol-neutral `ValidationNeed`;
- `engine.ValidationRequest` retains project/source identity;
- Pipeline mode receives the canonical durable change directly;
- Axiom advisory `EvidencePlan` is no longer projected back into `engine.EvidenceNeed`, so the control plane cannot narrow canonical scope or choose a weaker route;
- legacy flags can only widen advisory requirements for old durable runs.

**Closure evidence:**

- PR #17 merged to `main` as `468abe87ac76d54b3e887951e30b71e300c450d7`;
- `TestPipelineAdapterPreservesCanonicalScopeAndRoute` proves direct/Axiom scope and route equivalence and verifies changed-file/token-derived scope reaches the collector;
- `TestPipelineAdapterDoesNotAllowAxiomPlanToNarrowCanonicalRequest` proves an empty/stale Axiom plan cannot narrow canonical evidence need/tier;
- `ci` run #191 passed test, race, vet, real FastCDP Chromium integration and benchmark stages;
- `axiom-control` run #33 passed module lock, Axiom isolation, tests, race, vet, real Chrome discovery and `TestAxiomFastCDPEndToEndIntegration`.

**Re-score rationale:** Severity remains 10 because a future regression can still omit required UI. Occurrence falls `6 -> 1` because the canonical payload is now lossless for durable scope inputs and duplicate route narrowing is removed. Detection difficulty falls `7 -> 2` because equivalence and anti-narrowing regressions are executable CI tests and a real-browser path is exercised.  
**Residual:** `S=10 O=1 D=2 RPN=20` — target met.  
**Status:** `CLOSED`. Re-open on canonical-field loss, hard-coded run identity, independent Axiom tier selection, or equivalence-test failure.

### FMEA-003 — Render freshness not revision-bound

**Failure mode:** browser epoch advances without being logically/cryptographically tied to requested source/build revision.  
**Effect:** wrong/stale content can be captured and pass deterministic checks.  
**Owner:** `internal/runtime/fastcdp`, evidence provenance.  
**Mitigation:** #5.  
**Closure gate:** expected/observed revision digest in readiness token and packet provenance; stale/wrong-revision fault tests cannot release PASS evidence; mismatch uses defined recollect/reset/escalation behavior.  
**Residual target:** `S=10 O=1 D=2 RPN=20`.

### FMEA-004 — TruthPath capability optimism

**Failure mode:** Playwright adapter advertises full browser/scenario/ARIA/font capability even when worker/runtime/browser readiness is not proven.  
**Effect:** policy can select L3 while production execution is absent or only mock-proven.  
**Owner:** `internal/runtime/playwright`, dispatcher/router integration, CI.  
**Mitigation:** #6.  
**Closure gate:** explicit readiness probe validates worker identity/version, Playwright runtime and installed engines; advertised browsers equal validated engines; worker is bundled/versioned or explicitly validated; missing worker is unavailable rather than L3-capable; at least Chromium has non-mock CI capture; worker/browser versions reach provenance/calibration identity.  
**Residual target:** `S=9 O=1 D=2 RPN=18`.

### FMEA-005 — False telemetry split

**Failure mode:** impact and invalidation are exposed as separate metrics while sharing one combined timing.  
**Effect:** optimization/regression decisions are based on misleading stage data.  
**Owner:** `internal/engine`, telemetry.  
**Mitigation:** #7.  
**Closure gate:** independent timers/counters for impact resolution and invalidation; accounting tests and benchmark output prove values originate from distinct stages.  
**Residual target:** `S=6 O=1 D=2 RPN=12`.

### FMEA-006 — Planning/documentation state drift

**Failure mode:** historical `DONE`/phase prose contradicts actual code/integration/operational evidence.  
**Effect:** engineers depend on capabilities whose maturity is overstated or misunderstand canonical ownership/naming.  
**Owner:** `MASTER_PLAN.md`, README, architecture docs, issues.  
**Mitigation:** #8.  
**Closure gate:** reconcile all current claims to `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`; remove contradictory TruthPath/FastPath state and stale issue wording; automated documentation consistency checks where practical.  
**Residual target:** `S=7 O=2 D=2 RPN=28`.

### FMEA-007 — Duplicate durable side effects

**Failure mode:** durable retry/restart can repeat source repair or memory-admission side effects after effect success but before activity completion is persisted.  
**Effect:** duplicate/destructive edits or duplicated/contradictory memory facts.  
**Owner:** Axiom workflow boundary, repair writer, memory admission.  
**Mitigation:** #9.  
**Closure gate:** idempotency key/CAS for external effects; fault injection after effect-before-completion; replay proves exactly-once observable outcome.  
**Residual target:** `S=9 O=1 D=2 RPN=18`.

### FMEA-008 — Calibration drift

**Failure mode:** a previously legal L1/L2 calibration remains trusted after renderer/browser/worker/environment changes.  
**Effect:** approximate evidence can be accepted against obsolete parity assumptions.  
**Owner:** fidelity/calibration store, runtime identity.  
**Mitigation:** #10.  
**Closure gate:** calibration key includes exact relevant runtime versions/environment; incompatible/expired calibration invalidates automatically; CI covers upgrade/mismatch behavior.  
**Residual target:** `S=9 O=1 D=2 RPN=18`.

### FMEA-009 — Repair verifier overfit / reward hacking

**Failure mode:** autonomous repair optimizes against the same visible verifier/rubric/candidate signals later used to approve completion.  
**Effect:** apparent score improves while regressions/generalization failures escape.  
**Owner:** repair/comparison/final verification/eval.  
**Mitigation:** #11.  
**Closure gate:** independent final verifier path, held-out/perturbed scenarios, L3 for applicable high-risk/final gates, held-out regression-escape metric.  
**Residual target:** `S=9 O=2 D=3 RPN=54`.

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode:** project-private evidence or poisoned facts leak across namespaces or are promoted into reusable canonical memory.  
**Effect:** privacy breach and persistent wrong guidance across projects.  
**Owner:** SncSinCore adapter, admission/provenance/evolution.  
**Mitigation:** #12.  
**Closure gate:** adversarial multi-project isolation, provenance/conflict/retraction proof, poisoning corpus, promotion replay/shadow/non-regression/rollback.  
**Residual target:** `S=10 O=1 D=3 RPN=30`.

### FMEA-011 — Ungated main

**Failure mode:** high-risk architecture changes can land on unprotected `main` without required reviews/status checks/risk gates.  
**Effect:** known Critical paths can regress despite having tests/governance.  
**Owner:** repository delivery policy.  
**Mitigation:** #13.  
**Closure gate:** enforce required CI/review checks where repository permissions allow; risk-aware PR template/gate is non-optional for applicable changes; direct-push exception is documented/audited.  
**Residual target:** `S=8 O=1 D=2 RPN=16`.

### FMEA-012 — Baseline environment mismatch

**Failure mode:** visual baseline is compared under a different browser/engine/version/viewport/DPR/theme/font/fixture environment.  
**Effect:** false regression or false PASS hidden by tolerance widening.  
**Owner:** visual baseline identity, comparison engine.  
**Mitigation:** #14.  
**Closure gate:** baseline key includes browser/engine version, viewport, DPR, theme, font digest, renderer/worker version, fixture revision and locale/timezone where relevant; incompatible comparison is rejected rather than tolerance-widened.  
**Residual target:** `S=7 O=1 D=2 RPN=14`.

## 6. Execution order

Completed:

1. `FMEA-002` — CLOSED, residual RPN 20.

Current open order:

1. `FMEA-004`
2. `FMEA-001`
3. `FMEA-009`
4. `FMEA-003`
5. `FMEA-008`
6. `FMEA-005`
7. `FMEA-007`
8. `FMEA-012`
9. `FMEA-010`
10. `FMEA-006`
11. `FMEA-011`

`FMEA-004`, `FMEA-001`, `FMEA-009`, `FMEA-003`, and `FMEA-008` remain the open **Verification Integrity Barrier**. Production/final-gate claims must not cross an applicable open blocker without explicit dated risk acceptance.

## 7. Milestone metrics

Track:

- `open_critical_risks`;
- `open_high_risks`;
- `sum_open_rpn` and `sum_residual_rpn`;
- closure/reopen rate and oldest open High/Critical age;
- `false_pass_incidents`;
- `scope_equivalence_failures`;
- `collector_downgrade_attempts`;
- `stale_revision_evidence_rejections`;
- `truthpath_unavailable_events`;
- `calibration_invalidations`;
- `heldout_repair_regression_escape_rate`;
- `cross_project_memory_leakage_rate`;
- `baseline_incompatibility_rejections`;
- `idempotency_replay_duplicates`.

## 8. Closure record template

```text
Risk: FMEA-###
Status: CLOSED | ACCEPTED
Implementation: <PR/commit>
Initial: S/O/D/RPN
Residual: S/O/D/RPN
Why O changed: ...
Why D changed: ...
Closure evidence:
- test/eval/fault injection
- CI/runtime artifact
- independent verifier where required
Residual assumptions: ...
Reopen trigger: ...
Reviewed: YYYY-MM-DD
```
