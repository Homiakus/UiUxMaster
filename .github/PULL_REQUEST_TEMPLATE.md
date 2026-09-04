## Summary

<!-- What changes and why? -->

## Verification

<!-- Commands, tests, evals, benchmark/fault-injection artifacts. -->

```text
<verification evidence>
```

## Architecture / FMEA delta

- Architecture boundary changed: [ ] no [ ] yes
- Evidence routing / legal-PASS semantics changed: [ ] no [ ] yes
- Impact / invalidation semantics changed: [ ] no [ ] yes
- Warm-state readiness / evidence freshness changed: [ ] no [ ] yes
- Durable retry / external side effect changed: [ ] no [ ] yes
- Memory / evolution / privacy boundary changed: [ ] no [ ] yes
- Baseline / calibration / runtime capability changed: [ ] no [ ] yes

**Affected FMEA IDs:** `none` <!-- or FMEA-001, FMEA-003, ... -->

**If `none`, rationale:**

<!-- Explain why this change cannot introduce/change a failure mode in docs/FMEA_RISK_REGISTER.md. -->

**Risk action:** `none` <!-- mitigate | monitor | accept | none -->

**Initial → residual score for risks being mitigated:**

```text
FMEA-___: S=_ O=_ D=_ RPN=_  ->  S=_ O=_ D=_ RPN=_
```

**Closure evidence:**

<!-- Name the exact independent test/eval/fault injection required by the FMEA closure gate. -->

## Evidence integrity checklist

- [ ] A required evidence tier cannot silently downgrade to a weaker tier and still PASS.
- [ ] Actual evidence provenance/tier is auditable against policy requirements.
- [ ] Changed source scope reaches the canonical ImpactSet/ValidationScope path where applicable.
- [ ] New/retried external side effects have idempotency/CAS semantics where applicable.
- [ ] Capability/calibration claims are tied to the runtime/environment version they describe.
- [ ] Autonomous repair/selection is independently re-verified for protected/high-risk outcomes.
- [ ] No project-private memory/evidence crosses a broader namespace without an explicit admission gate.
- [ ] Documentation/plan status is updated when implementation maturity changes.

## Planning traceability

<!-- For Critical/High risks, link the mitigation issue/task and quote the risk gate. -->

- Related mitigation issue(s):
- `MASTER_PLAN.md` task(s), if applicable:
- FMEA register updated: [ ] not needed [ ] yes

## Residual assumptions

<!-- What can still fail after this PR? State assumptions rather than hiding them in comments. -->
