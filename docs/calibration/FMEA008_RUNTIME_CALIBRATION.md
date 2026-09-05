# FMEA-008 — runtime calibration validity contract

Status: implementation/closure candidate.

## Invariant

A clean L1/L2 result is diagnostic evidence, not authority to issue PASS, unless the exact approximate runtime and exact TruthPath oracle pair has a current parity-corpus calibration record for every evidence class being claimed.

`CalibrationMatrix` answers which tiers may *in principle* prove an evidence class. `CalibrationAuthority` independently answers whether the exact current runtime pair is still calibrated. These decisions are intentionally separate.

## Exact calibration key

The environment key covers both approximate and TruthPath identities, including renderer name/version, fidelity ID, browser family/version, worker/runtime version, platform and applicable viewport/device-scale/theme/font-profile dimensions. A change to either side creates a different key and invalidates the stored record.

A parity record also carries corpus digest, artifact reference, sample count/pass count, creation timestamp and optional expiry. Policy adds minimum sample coverage, minimum pass rate and maximum age.

## Legal PASS flow

1. Canonical validation collects evidence at the cheapest useful tier.
2. Tier and source-revision provenance are attested before verifier execution.
3. Deterministic verification may still use uncalibrated L1/L2 evidence diagnostically.
4. When `RequireLegalPass`/`FinalGate` is active, `PassAuthority` evaluates evidence classes and exact runtime calibration.
5. Missing, expired, weak-corpus or environment-mismatched calibration becomes explicit missing evidence and recommends upward escalation/recalibration; it can never become PASS.
6. L3 TruthPath remains authoritative without approximate-tier parity calibration.
7. Axiom canonical pipeline requests legal-pass authority. The legacy direct collector path is diagnostic only and cannot issue `DecisionPass`.

## Persistence

`CalibrationRegistry.SaveFile` writes a versioned JSON snapshot atomically. Every persisted record includes the canonical environment key, corpus digest and artifact reference. `LoadCalibrationRegistry` validates every restored record before it can be used by `CalibrationAuthority`.

## Closure tests

- same exact validated key retains legal PASS;
- renderer/browser/worker/runtime/platform/viewport mutation invalidates the old record;
- missing, expired, insufficient-coverage and insufficient-quality calibration fail closed;
- persisted snapshot round-trip preserves exact key/corpus/artifact identity;
- canonical Pipeline converts invalid calibration into evidence insufficiency rather than PASS;
- real Chromium proof derives the approximate FastCDP and TruthPath identities from running runtimes and demonstrates immediate invalidation after TruthPath identity drift.
