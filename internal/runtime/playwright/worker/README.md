# TruthPath Playwright worker

This directory is the versioned Node execution boundary for Tier L3 TruthPath evidence.

## Runtime contract

- Worker protocol: `1.0.0`
- Playwright: **exactly `1.62.1`**
- Browser capability is runtime-attested: an engine is advertised only after its bundled executable exists, launches successfully, and reports a non-empty version.
- `probe`, `capture`, and `scenario` responses carry worker, Playwright, and browser runtime identity.
- The Go adapter rejects protocol/runtime version mismatch and rejects evidence if browser identity changes after the readiness probe.

A missing worker, missing Playwright package, missing browser executable, launch failure, or identity mismatch is an unavailable TruthPath condition. It must never be interpreted as an L3 PASS.

## Local real-runtime check

```bash
cd internal/runtime/playwright/worker
npm install --ignore-scripts --no-audit --no-fund
npx playwright install --with-deps chromium
cd ../../../..
UIUX_TRUTHPATH_INTEGRATION=1 go test ./internal/runtime/playwright -run '^TestTruthPathRealChromium$' -count=1 -v
```

The GitHub `truthpath` workflow runs the same real Chromium gate independently from ordinary mock/unit tests.
