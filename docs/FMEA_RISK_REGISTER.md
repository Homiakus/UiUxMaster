# UiUxMaster Architecture FMEA Risk Register

- Status: **ACTIVE**
- Initial review: 2026-09-04
- Last re-score: 2026-09-05
- Scope: architecture, verification integrity, orchestration, memory/evolution, baselines, delivery controls
- Planning authority: `MASTER_PLAN.md`
- Engineering-risk authority: this register
- Machine mirror: `planning/fmea-risk.json`
- Execution overlay: `planning/FMEA_EXECUTION_PLAN.md`
- Governance ADR: `docs/adr/0002-fmea-risk-governance.md`

## 1. Risk-domain separation

`internal/fidelity.RiskLevel` is runtime routing data: how likely an approximate renderer is to diverge from browser truth and which evidence tier is required.

`FMEA-###` is engineering/planning data: how the architecture can fail, the effect, occurrence, detectability, required mitigation and independent closure evidence.

These domains must never share one field or enum.

## 2. Scoring and status

Use integer scores 1–10:

- **Severity (S)**: 1 negligible; 10 can invalidate system trust, leak protected data, create destructive behavior or consequential false PASS.
- **Occurrence (O)**: 1 exceptional/strongly prevented; 10 expected/frequent.
- **Detection difficulty (D)**: 1 almost certainly detected before escape; 10 likely to escape controls.
- **RPN** = `S × O × D`.

Priority defaults:

| Priority | Trigger |
|---|---|
| Critical | `RPN >= 250`, or Severity 9–10 with credible false-PASS/security/destructive path |
| High | `RPN 120–249` |
| Medium | `RPN 60–119` |
| Low | `RPN < 60` |

Status vocabulary: `OPEN`, `MITIGATING`, `ACCEPTED`, `CLOSED`.

Closure requires merged implementation, the named independent evidence, residual re-score, planning/machine-state synchronization and a reopen trigger. Code existence alone never closes risk.

## 3. Risk summary

| ID | Failure mode | Initial S/O/D | Initial RPN | Current S/O/D | Current RPN | Priority | Status | Mitigation |
|---|---|---:|---:|---:|---:|---|---|---|
| FMEA-001 | Required TruthPath silently downgrades to L2 | 10/4/8 | 320 | 10/4/8 | 320 | Critical | OPEN | #3 |
| FMEA-002 | Axiom loses canonical change/ImpactSet scope | 10/6/7 | 420 | **10/1/2** | **20** | Low residual | **CLOSED** | #4 / PR #17 |
| FMEA-003 | Render epoch is not bound to requested source/build revision | 10/4/7 | 280 | 10/4/7 | 280 | Critical | OPEN | #5 |
| FMEA-004 | TruthPath advertises capabilities without proven runtime readiness | 9/5/8 | 360 | **9/1/2** | **18** | Low residual | **CLOSED** | #6 / PR #18 |
| FMEA-005 | Impact/invalidation telemetry are not independently measured | 6/10/4 | 240 | 6/10/4 | 240 | High | OPEN | #7 |
| FMEA-006 | Planning/documentation state contradicts implemented state | 7/9/3 | 189 | 7/9/3 | 189 | High | OPEN | #8 |
| FMEA-007 | Durable retries can duplicate repair/memory side effects | 9/3/7 | 189 | 9/3/7 | 189 | High | OPEN | #9 |
| FMEA-008 | Fidelity calibration remains trusted after environment/version drift | 9/4/7 | 252 | 9/4/7 | 252 | Critical | OPEN | #10 |
| FMEA-009 | Repair loop optimizes against the same signals that approve completion | 9/5/7 | 315 | 9/5/7 | 315 | Critical | OPEN | #11 |
| FMEA-010 | Memory/evolution leaks scope or promotes poisoned evidence | 10/2/8 | 160 | 10/2/8 | 160 | High | OPEN | #12 |
| FMEA-011 | High-risk changes can land on unprotected `main` without required gates | 8/4/4 | 128 | 8/4/4 | 128 | High | OPEN | #13 |
| FMEA-012 | Visual baseline is compared under incompatible render environment | 7/5/5 | 175 | 7/5/5 | 175 | High | OPEN | #14 |

Current milestone metrics after FMEA-004 closure:

```text
open_critical_risks = 4
open_high_risks = 6
sum_open_rpn = 2248
sum_closed_residual_rpn = 38
```

## 4. Detailed risks

### FMEA-001 — Silent TruthPath downgrade

**Failure mode:** policy selects L3 TruthPath, but absent/unavailable L3 collection can execute on L2 FastBrowser.  
**Effect:** clean-state/final-gate/cross-browser/calibration requirements may be judged on weaker evidence, creating false PASS.  
**Owner:** `internal/runtime/dispatcher`, `internal/engine`.  
**Mitigation:** #3.  
**Closure gate:** typed unavailable/insufficient-evidence semantics; no silent L3→L2 substitution; actual packet tier is checked against policy-selected minimum tier; final/clean-state/cross-browser gates cannot PASS from weaker evidence; only explicitly legal escalation remains.  
**Residual target:** `S=10 O=1 D=2 RPN=20`.

### FMEA-002 — Axiom canonical-pipeline scope loss — CLOSED

**Failure mode:** Axiom entered `engine.Pipeline` through a lossy `Change` projection and could independently project an advisory evidence plan back into canonical `EvidenceNeed`.  
**Effect:** `ImpactSet`/`ValidationScope` could detach from the actual edit.  
**Owner:** `control/axiom/controlplane`, `control/axiom/uiuxadapter`, `internal/engine`.  
**Mitigation:** #4, PR #17.  

Implemented controls:

- durable Axiom `Change` carries stable run/project/source identity, files, tokens, nodes, routes, viewport/theme, whole-site override, base target and complete validation need;
- Axiom advisory `EvidencePlan` can no longer narrow canonical `engine.EvidenceNeed` or choose a weaker route;
- direct/Axiom scope and route equivalence is executable regression evidence.

Closure evidence:

- PR #17 merged as `468abe87ac76d54b3e887951e30b71e300c450d7`;
- `TestPipelineAdapterPreservesCanonicalScopeAndRoute`;
- `TestPipelineAdapterDoesNotAllowAxiomPlanToNarrowCanonicalRequest`;
- `ci` #191 PASS;
- `axiom-control` #33 PASS including real `TestAxiomFastCDPEndToEndIntegration`.

Re-score: Severity remains 10. O `6→1` because canonical scope is lossless and duplicate route narrowing is removed. D `7→2` because equivalence/anti-narrowing tests plus real browser integration detect regression.  
**Residual:** `10/1/2 RPN=20`.  
**Reopen:** canonical-field loss, hard-coded run identity, independent Axiom tier selection or equivalence-test failure.

### FMEA-003 — Render freshness not revision-bound

**Failure mode:** browser epoch advances without being tied to requested source/build revision.  
**Effect:** wrong/stale content can be captured and pass checks.  
**Owner:** `internal/runtime/fastcdp`, evidence provenance.  
**Mitigation:** #5.  
**Closure gate:** expected/observed revision digest in readiness and packet provenance; stale/wrong revision cannot release PASS evidence; mismatch follows defined recollect/reset/escalation semantics.  
**Residual target:** `10/1/2 RPN=20`.

### FMEA-004 — TruthPath capability optimism — CLOSED

**Failure mode:** Playwright adapter advertised full browser/scenario/ARIA/font capability independently of whether a real worker/runtime/browser was runnable.  
**Effect:** architecture could treat L3 as production-ready while execution was absent or mock-only.  
**Owner:** `internal/runtime/playwright`, CI.  
**Mitigation:** #6, PR #18.  

Implemented controls:

- `Capabilities()` is fail-closed until `Probe()` succeeds;
- checked-in worker protocol is versioned `1.0.0` and Playwright is pinned exactly to `1.62.1`;
- probe validates worker entrypoint, exact protocol version, exact Playwright version and each browser engine independently;
- an engine is advertised only if its bundled executable exists, launches successfully and returns a non-empty browser version;
- absent worker/browser is unavailable rather than an L3 capability;
- capture/scenario responses carry worker, Playwright and browser versions;
- evidence is rejected if runtime identity differs from the identity attested by probe;
- canonical `evidence.Packet.Renderer.Version/FidelityID` contains runtime identity needed by later FMEA-008 calibration invalidation;
- mock tests are unit evidence only; a dedicated `truthpath` workflow provides real Chromium proof.

Closure evidence:

- PR #18 merged as `7433e1bd39c431ab0ac69181e2caf0fde9dd1921`;
- `truthpath` workflow #2 PASS: worker syntax, exact Playwright install, bundled Chromium install, unit tests and real `TestTruthPathRealChromium` probe/capture/scenario;
- `ci` #197 PASS: module lock, full tests, race, vet, real FastCDP Chromium integration and benchmarks;
- `TestTruthPathCapabilitiesFailClosedUntilProbe`;
- `TestTruthPathProbeAdvertisesOnlyLaunchableVersionedBrowsers`;
- `TestTruthPathProbeRejectsProtocolAndRuntimeVersionDrift`;
- `TestTruthPathCaptureRejectsIdentityChangeAfterProbe`;
- `TestTruthPathMissingWorkerIsUnavailable`.

Re-score: Severity remains 9 because a future regression can still invalidate a TruthPath claim. O `5→1` because capability is now derived from a real versioned launch probe rather than static declaration. D `8→2` because missing runtime, version drift, browser launchability, identity drift and real capture are all executable gates.  
**Residual:** `9/1/2 RPN=18` — target met.  
**Reopen:** pre-probe capability advertisement, non-versioned worker/runtime, unlaunched browser advertisement, missing-worker fallback, provenance identity loss, or real TruthPath CI failure.  
**Boundary:** lifecycle invalidation of stored calibration after environment drift remains FMEA-008; FMEA-004 supplies its canonical runtime identity.

### FMEA-005 — False telemetry split

**Failure mode:** impact and invalidation appear as separate metrics while sharing one combined timing.  
**Effect:** optimization/regression decisions use misleading stage data.  
**Owner:** `internal/engine`, telemetry.  
**Mitigation:** #7.  
**Closure gate:** independent timers/counters and accounting tests/benchmarks prove values originate from distinct stages.  
**Residual target:** `6/1/2 RPN=12`.

### FMEA-006 — Planning/documentation state drift

**Failure mode:** historical status prose contradicts actual implementation/integration/operational evidence.  
**Effect:** engineers depend on overstated capabilities or wrong ownership/naming.  
**Owner:** `MASTER_PLAN.md`, README, architecture docs, issues.  
**Mitigation:** #8.  
**Closure gate:** reconcile claims to `IMPLEMENTED -> INTEGRATED -> OPERATIONALLY_PROVEN -> RELEASE_GATED`; eliminate stale TruthPath/FastPath state and add consistency checks where practical.  
**Residual target:** `7/2/2 RPN=28`.

### FMEA-007 — Duplicate durable side effects

**Failure mode:** durable retry/restart repeats source repair or memory admission after effect success but before completion persistence.  
**Effect:** duplicate/destructive edits or duplicated/contradictory memory facts.  
**Owner:** Axiom workflow, repair writer, memory admission.  
**Mitigation:** #9.  
**Closure gate:** idempotency key/CAS + fault injection after effect-before-completion + replay proving one observable outcome.  
**Residual target:** `9/1/2 RPN=18`.

### FMEA-008 — Calibration drift

**Failure mode:** previously legal calibration remains trusted after renderer/browser/worker/environment changes.  
**Effect:** approximate evidence is accepted against obsolete parity assumptions.  
**Owner:** fidelity/calibration state, runtime identity.  
**Mitigation:** #10.  
**Closure gate:** calibration key includes exact runtime/environment identity; incompatible/expired calibration invalidates automatically; CI covers upgrade/mismatch behavior. FMEA-004 now provides worker/Playwright/browser identity in canonical evidence provenance.  
**Residual target:** `9/1/2 RPN=18`.

### FMEA-009 — Repair verifier overfit / reward hacking

**Failure mode:** autonomous repair optimizes against the same visible signals later used for completion.  
**Effect:** apparent score improves while regressions/generalization failures escape.  
**Owner:** repair/comparison/final verification/eval.  
**Mitigation:** #11.  
**Closure gate:** independent final verifier, held-out/perturbed scenarios, L3 for applicable high-risk/final gates, held-out regression escape metric.  
**Residual target:** `9/2/3 RPN=54`.

### FMEA-010 — Memory scope leakage / epistemic poisoning

**Failure mode:** project-private evidence or poisoned facts leak across namespaces or are promoted into reusable canonical memory.  
**Effect:** privacy breach and persistent wrong guidance.  
**Owner:** SncSinCore adapter, admission/provenance/evolution.  
**Mitigation:** #12.  
**Closure gate:** adversarial multi-project isolation, provenance/conflict/retraction proof, poisoning corpus, promotion replay/shadow/non-regression/rollback.  
**Residual target:** `10/1/3 RPN=30`.

### FMEA-011 — Ungated main

**Failure mode:** high-risk changes can land on `main` without required review/status/risk gates.  
**Effect:** known Critical paths can regress despite tests/governance.  
**Owner:** repository delivery policy.  
**Mitigation:** #13.  
**Closure gate:** enforce required CI/review checks where permissions permit; direct-push exceptions are documented/audited.  
**Residual target:** `8/1/2 RPN=16`.

### FMEA-012 — Baseline environment mismatch

**Failure mode:** visual baseline is compared under incompatible browser/engine/version/viewport/DPR/theme/font/fixture environment.  
**Effect:** false regression or false PASS hidden by tolerance widening.  
**Owner:** visual baseline identity, comparison engine.  
**Mitigation:** #14.  
**Closure gate:** baseline key includes browser/engine version, viewport, DPR, theme, font digest, renderer/worker version, fixture revision and locale/timezone where relevant; incompatible comparison is rejected.  
**Residual target:** `7/1/2 RPN=14`.

## 5. Execution order

Completed:

1. `FMEA-002` — CLOSED, residual RPN 20.
2. `FMEA-004` — CLOSED, residual RPN 18.

Current open order:

1. `FMEA-001` — **CURRENT**
2. `FMEA-009`
3. `FMEA-003`
4. `FMEA-008`
5. `FMEA-005`
6. `FMEA-007`
7. `FMEA-012`
8. `FMEA-010`
9. `FMEA-006`
10. `FMEA-011`

`FMEA-001`, `FMEA-009`, `FMEA-003`, and `FMEA-008` remain the open **Verification Integrity Barrier**.

## 6. Planning/change-control contract

Every Critical/High architecture task carries:

```text
Risks: FMEA-### [, FMEA-###]
Risk action: mitigate | monitor | accept | none
Initial risk: S/O/D/RPN
Risk gate: exact independent test/eval/fault/review evidence
Residual target: S/O/D/RPN
Evidence: concrete test/artifact/PR/runtime proof
```

Perform an FMEA delta review when changing routing/fallback/legal PASS, impact/invalidation, render readiness/freshness, browser/renderer/model capability/version, verifier semantics, autonomous repair, Axiom retries/side effects, memory boundaries, privacy/provenance, baseline identity or CI/release gates.

Closed risks are regression guards: breaking a closure invariant re-opens the same ID.
