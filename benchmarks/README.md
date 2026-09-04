# UiUxMaster Benchmark History & Artifacts

This directory tracks reproducible benchmark baselines across UiUxMaster runtime, impact, and browser execution layers.

## Tracked Baselines

1. **Impact Scaling (1k/10k/100k)**:
   - Measured via `go test -bench=BenchmarkResolver -benchmem ./internal/impact`
   - Strict allocation gates enforced by `TestImpactAllocationGates`.
   - Leaf queries: O(1) ~350ns, 4 allocs/op across all scales.

2. **FastBrowser CDP Benchmarks**:
   - Measured via `go run ./cmd/uiuxcdpbench`
   - Resident session evaluation, DOM snapshotting, ROI clip screenshots, and diagnostic barrier timing.

3. **Driver Comparison Evaluation**:
   - Generated via `go run ./cmd/uiuxcdpbench -comparative`
   - Full comparative evaluation documented in `docs/DRIVER_COMPARISON_REPORT.md`.

## CI Persistence

In CI (`.github/workflows/ci.yml`), benchmark runs write all output to `./build/benchmarks/` and are published via `actions/upload-artifact@v4` with a 30-day retention window, replacing ephemeral `/tmp` execution.
