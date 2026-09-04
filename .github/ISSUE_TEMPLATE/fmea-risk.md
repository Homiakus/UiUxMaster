---
name: Architecture FMEA risk
description: Record a new architecture/design failure mode with scoring, mitigation and closure evidence.
title: "[FMEA-___] "
labels: []
assignees: []
---

## Failure mode

<!-- One failure mode only. Describe what fails, not the proposed solution. -->

## Effect

<!-- Worst credible system/user/evidence effect. Explicitly state false-PASS, integrity, privacy or destructive consequences where applicable. -->

## Causes / mechanism

<!-- Code path, architectural condition, retry/fallback/environment mechanism. -->

## Existing controls

<!-- What currently reduces occurrence or improves detection? -->

## Initial FMEA score

- Severity `S`: _ / 10
- Occurrence `O`: _ / 10
- Detection difficulty `D`: _ / 10
- RPN `S × O × D`: _
- Priority: `Critical | High | Medium | Low`

**Scoring rationale:**

## Owner boundary

<!-- Package/subsystem/role, not necessarily a named person. -->

## Required mitigation

1. 

## Verification / closure gate

<!-- Independent tests/evals/fault injection/artifacts required before CLOSED. -->

- [ ] 

## Residual target

- Severity `S`: _ / 10
- Occurrence `O`: _ / 10
- Detection difficulty `D`: _ / 10
- Residual RPN: _

## Planning traceability

- `MASTER_PLAN.md` task(s):
- Related PR(s):
- Register entry added/updated: [ ]

## Closure record

<!-- Complete only when CLOSED/ACCEPTED. -->

- Status: `OPEN | MITIGATING | ACCEPTED | CLOSED`
- Verification evidence:
- Why O changed:
- Why D changed:
- Remaining assumptions:
- Next review:
- Accepted by (ACCEPTED only):
