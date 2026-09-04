# ADR-0002 — Engineering FMEA risk governance

- Status: Accepted
- Date: 2026-09-04

## Context

UiUxMaster already has a runtime `fidelity.RiskLevel`. That value answers one narrow execution question: how likely an approximate renderer is to diverge from browser truth and therefore which evidence tier is required.

The project also needs a different risk model for architecture and delivery. Examples include silent evidence-tier downgrade, stale evidence being accepted, a control-plane adapter bypassing the canonical impact path, duplicated side effects after durable retries, documentation drift, and calibration becoming stale after a renderer/browser upgrade.

Using one `Risk` field for both concepts would mix execution routing with engineering governance and would make both less trustworthy.

## Decision

### 1. Runtime fidelity risk and engineering FMEA risk are different domains

Runtime fidelity risk remains owned by `internal/fidelity` and is an input to evidence routing.

Engineering/design risk is tracked in `docs/FMEA_RISK_REGISTER.md` and is an input to planning, review, release gates, and mitigation work.

The FMEA register must not be imported into the L0-L2 hot path and must not change renderer routing directly.

### 2. FMEA uses explicit failure modes and residual risk

Each engineering risk has a stable `FMEA-###` identifier and records:

- failure mode;
- effect;
- causes/mechanism;
- existing controls;
- Severity (`S`), Occurrence (`O`) and Detection difficulty (`D`) on 1-10 scales;
- `RPN = S × O × D`;
- priority and status;
- accountable ownership boundary;
- mitigation work item(s);
- verification/closure evidence;
- residual `S/O/D/RPN` target.

`D=10` means the failure is very difficult to detect before it escapes.

RPN is not the only gate. A failure that can create a false PASS, confidentiality breach, destructive side effect, or loss of evidence integrity may be Critical because of severity even when occurrence is estimated as low.

### 3. High-risk planning work must be traceable

Every Critical or High FMEA risk must have at least one executable mitigation issue/task. Architecture-changing pull requests must state either the affected FMEA IDs or `none` with a rationale.

A mitigation task is not complete merely because code exists. It must contain the verification evidence defined by the corresponding risk entry.

### 4. Risk closure is evidence based

A risk may be marked `CLOSED` only when:

1. the planned mitigation is implemented;
2. the specified test/eval/fault-injection gate passes;
3. residual `S/O/D/RPN` is recorded;
4. evidence is linked in the register;
5. no open Critical failure mode remains hidden behind a fallback.

Risk acceptance is explicit, time-bounded, and records the accepting authority and review date. `ACCEPTED` is not equivalent to `CLOSED`.

### 5. Fail-closed evidence semantics have priority

If a requested evidence tier is required for correctness and the collector is unavailable, UiUxMaster must report evidence insufficiency/collector failure rather than silently substitute a weaker tier and allow a PASS.

The actual evidence packet tier must remain auditable against the minimum tier selected by policy.

### 6. FMEA is reviewed on architecture deltas

An FMEA delta review is required when a change introduces or materially changes any of the following:

- evidence tiers or fallback behavior;
- impact/invalidation semantics;
- durable retries or external side effects;
- browser/renderer/model versions or capability claims;
- memory namespaces/admission/promotion rules;
- security/privacy boundaries;
- autonomous repair or independent verification;
- release/CI gates.

## Consequences

### Positive

- project risks cannot be confused with renderer-fidelity routing;
- severe false-PASS paths receive explicit owners and closure tests;
- residual risk becomes visible instead of disappearing when a task is checked off;
- architectural decisions, issues and pull requests share stable risk identifiers;
- the hot validation path remains free from process-governance state.

### Costs

- maintainers must update FMEA entries when architecture changes;
- some work cannot be declared complete until independent verification exists;
- a small amount of review/CI process is added.

These costs are accepted because UiUxMaster is itself a verification system: an untracked failure of the verifier can invalidate every downstream result.

## Rejected alternatives

### Reuse `fidelity.RiskLevel` for FMEA

Rejected because renderer divergence risk and engineering failure risk have different causes, lifecycles, scoring and owners.

### Track risks only as prose in `MASTER_PLAN.md`

Rejected because risk lifecycle, residual score and closure evidence need stable identifiers and a dedicated register. `MASTER_PLAN.md` remains the execution-order authority; the FMEA register is a subordinate engineering-risk ledger.

### RPN-only prioritization

Rejected because low-occurrence but catastrophic false-PASS, privacy or destructive-side-effect modes can be under-ranked by RPN alone.

## Verification

This ADR is operationalized through:

- `docs/FMEA_RISK_REGISTER.md`;
- GitHub mitigation issues carrying `FMEA-###` IDs;
- the pull-request risk checklist;
- tests/evals/fault injection named in each Critical/High risk closure gate.
