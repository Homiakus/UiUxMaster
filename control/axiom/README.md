# UiUxMaster Axiom control plane

This nested module keeps durable orchestration dependencies out of the root FastPath module.

## Boundary

The control plane owns run-level concerns:

- execution identity and plan digest;
- budgets;
- cancellation;
- ordered history;
- durable state via the file-backed constructor;
- coarse-grained orchestration of `PlanEvidence -> CollectVerify -> Decide`.

The execution plane remains responsible for hot operations:

- impact analysis;
- WGGo rendering;
- resident Chrome / FastCDP;
- DOM/AX/font/runtime evidence collection;
- deterministic verification;
- ROI screenshots and future local VLM inspection.

Axiom must not wrap individual CDP commands. One Axiom activity may contain many in-process browser calls.

## Dependency isolation

`control/axiom` is a separate Go module and pins:

```text
github.com/Homiakus/axiom v0.0.0-20260902054936-44cea54c5cea
```

The root `go.mod` intentionally does not depend on Axiom, Pebble or Prometheus.

## Current P0 workflow

```text
change
  -> PlanEvidence
  -> CollectVerify
  -> Decide
  -> terminal Run projection
```

Heavy evidence is not stored in Axiom state. The workflow stores only compact plans, counters, summaries and decisions. Content-addressed artifact references are the intended persistence mechanism for screenshots/DOM dumps/VLM artifacts in the next slice.

## API

`controlplane.Runner` exposes:

- `NewMemory`
- `NewFile`
- `Start`
- `Run`
- `StartAndRun`
- `Cancel`
- `Load`
- `History`
- `PlanDigest`

The public projection uses UiUxMaster-local types rather than exposing `adgo.Execution`, which keeps a future `Runtime -> Engine/Host` migration behind this module boundary.
