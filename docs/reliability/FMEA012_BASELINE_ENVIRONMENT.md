# FMEA-012 — Protected visual baseline environment contract

Status: implementation/closure candidate for issue #14.  
Initial risk: `S=7 O=5 D=5 RPN=175`.  
Residual target after merged CI evidence: `S=7 O=1 D=2 RPN=14`.

## Failure being prevented

A screenshot is not an environment-independent truth artifact. Browser/renderer version, layout viewport, device scale, theme, installed/loaded fonts, locale, timezone, fixture revision and platform/runtime identity can all change pixels without a product-code regression. Comparing pixels first and then widening tolerance mixes these causes with genuine visual change and eventually destroys the meaning of a protected baseline.

UiUxMaster therefore treats visual-baseline compatibility as a **precondition**, not a diff heuristic.

## Canonical identity

`evidence.RenderEnvironmentIdentity` is versioned and includes:

- renderer name/version;
- worker/runtime version;
- browser family, engine and version;
- actual execution platform;
- viewport width/height and DPR;
- theme;
- font-set digest;
- locale;
- timezone;
- fixture/source revision.

The normalized identity is encoded deterministically and hashed as `render-env-v1:<sha256>`.

There are no wildcard semantics for protected baselines. A missing material field makes the identity incomplete and the comparison unavailable. This is deliberate fail-closed behavior.

## Mandatory comparison order

```text
baseline pixels + stored environment
candidate pixels + observed/current environment
        |
        v
validate both complete environment identities
        |
        v
require exact canonical environment-key equality
        |
        v
validate every dynamic mask against current semantic owner bounds
        |
        v
apply declared per-channel tolerance
        |
        v
compute deterministic pixel result + comparison digest
```

Tolerance is never used to compensate for environment incompatibility. `Tolerance=255` cannot turn a browser/font/DPR mismatch into a legal comparison.

The raw `CompareRGBA` function remains a low-level primitive for already-compatible in-memory images. Protected baseline callers must use `Comparator.CompareBaseline`; the runtime dispatcher no longer calls raw pixel comparison for a protected baseline.

## Candidate provenance

When an L1 protected-baseline comparison succeeds, the exact normalized candidate environment is persisted in the canonical `evidence.Packet.Environment`. The packet also records `BASELINE_ENVIRONMENT_ATTESTED` with baseline/candidate environment keys and deterministic comparison digest.

The current L1 WGGo/FastRender path derives runtime-owned dimensions from the actual renderer capabilities and executing `GOOS/GOARCH`. Non-runtime dimensions such as locale, timezone, fixture revision and declared font-set digest must be supplied explicitly; omission fails the comparison rather than becoming a wildcard.

Future L2/L3 raw-pixel baseline comparison must use the same canonical comparator and must derive browser/worker/font identity from the actual launched runtime rather than caller claims.

## Baseline lifecycle

`MemoryBaselineStore.Put` is create-only. Silent overwrite is forbidden.

Any baseline replacement uses `BaselineStore.Update` with:

- exact expected version;
- exact expected old digest;
- new complete environment identity;
- rationale;
- reviewer identity.

The store records old/new version, image digest, environment key, rationale, reviewer and timestamp. Protected baselines are updateable only through this explicit reviewed CAS path.

This separation makes intentional design changes auditable and prevents test code or automation from normalizing unexplained churn by overwriting the golden image.

## Dynamic masks

A dynamic mask is not a global ignore rectangle. Each `DynamicMask` must name a current visible `ElementRef` owner. The mask rectangle must be fully contained inside that owner's evidence bounds and the image bounds.

A mask cannot extend outside its declared owner to hide adjacent visual regressions. Missing owners, duplicate mask IDs, zero/negative rectangles and out-of-owner masks fail comparison.

## Operational metrics

Baseline store metrics expose:

- creates;
- reviewed updates;
- environment-changing updates.

Comparator metrics are separated by baseline environment key and expose:

- legal comparisons;
- incompatible comparison attempts;
- outcome flips for repeated same-baseline/environment/candidate-digest comparisons.

These metrics distinguish baseline churn/flakiness from genuine product diffs instead of hiding instability in global tolerance.

## Closure tests

Required executable evidence includes:

- `TestFMEA012EnvironmentKeyRejectsMaterialMismatch`;
- `TestFMEA012ComparatorRejectsIncompatibleBeforeTolerance`;
- `TestFMEA012ExactKeyComparisonIsDeterministic`;
- `TestFMEA012ReviewedBaselineUpdateRecordsOldNewIdentity`;
- `TestFMEA012DynamicMaskCannotEscapeOwner`;
- `TestFMEA012OwnedMaskOnlyExcludesOwnedPixels`;
- `TestFMEA012DispatcherRejectsIncompatibleBaselineBeforeTolerance`;
- `TestFMEA012DispatcherRejectsBaselineWithoutEnvironment`;
- updated dispatcher compatible-baseline integration with packet environment attestation.

FMEA-012 remains open until these tests plus root/race/vet and existing real-browser regression workflows pass on the final PR head and the merged evidence is synchronized into the FMEA register, machine mirror and tracker.

## Reopen conditions after closure

Reopen FMEA-012 if any protected-baseline path:

- compares pixels before exact environment compatibility;
- treats a missing material identity field as a wildcard;
- widens tolerance because environment keys differ;
- permits silent baseline overwrite;
- changes a baseline without old/new digest/environment provenance and review rationale;
- permits a dynamic mask outside its current semantic owner;
- loses candidate environment identity from canonical evidence provenance;
- or produces nondeterministic comparison outcomes for the same exact environment and candidate digest.
