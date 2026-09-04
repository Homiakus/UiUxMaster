# UiUxMaster

> **Evidence-driven UI/UX engineering system for AI coding agents.**

UiUxMaster is a high-performance, resident visual and interaction verification engine. It enables autonomous AI coding agents to **design, render, inspect, critique, compare, repair, and independently verify** web interfaces using deterministic browser evidence, sub-50ms warm visual loops, and epistemic design defect memory.

---

## Core Thesis

```text
CODE IS A HYPOTHESIS
RENDER IS EVIDENCE
INTERACTION IS THE RESULT
```

A frontend change cannot be deemed complete simply because generated CSS looks plausible or an isolated unit test succeeds. UiUxMaster closes the engineering loop against ground truth:

```text
Design Intent / Task
        │
        ▼
   Source Edit
        │
        ▼
Incremental Impact Resolution (<1ms)
        │
        ▼
Fidelity & Risk Routing (L0 / L1 / L2 / L3)
        │
        ▼
Warm Browser Render & Evidence Capture (<35ms)
(DOMSnapshot + Accessibility Tree + Fonts + ROI RGBA Crop)
        │
        ▼
Deterministic Verifiers (Layout, Contrast, Touch Targets, Overflows)
        │
        ▼
Multi-Viewport & Semantic Design Critique
        │
        ▼
Relative Candidate Comparison (10 Canonical Axes)
        │
        ▼
Autonomous Host Repair & Live Re-verification
        │
        ▼
SncSinCore Epistemic Memory Admission (Admitted Pattern Atoms)
        │
        ↺
```

---

## Architecture & Core Subsystems

### 1. Native Incremental Impact Engine (`internal/impact`)
- Crawls project directories (Vite, Next.js, Vanilla HTML/CSS), builds route trees, parses CSS custom properties/design tokens, and indexes component dependencies.
- Translates source changes into an exact `ImpactSet`, ensuring only invalidated routes and visual regions are re-evaluated without costly full-site reloads.

### 2. Multi-Tier Visual Runtime
- **L0 Static Preflight**: Microsecond AST, token, and style dependency checks.
- **L1 In-Process FastRender (`internal/runtime/wggo`)**: Pure Go in-process RGBA rendering and ROI cropping via `go-webengine/engine`.
- **L2 Resident FastBrowser (`internal/runtime/fastcdp`)**: Resident raw-CDP Chromium runtime featuring bounded warm page pools, sub-50ms DOMSnapshot/AX extraction, and GPU-accelerated ROI viewport capture.
- **L3 TruthPath (`internal/runtime/playwright`)**: Clean-session, cross-browser reference verification and parity calibration oracle.

### 3. Deterministic Verifier Suite (`internal/verifier`)
- Independent verification gates for:
  - **WCAG 2.2 AA Contrast**: Accurate APCA / relative luminance algorithms.
  - **Touch Target Sizing**: Identifies interactive targets below minimum 44×44px thresholds.
  - **Layout & Overflow**: Detects horizontal scroll leaks, negative margins, and clipping defects.
  - **Runtime Integrity**: Monitors console exceptions, unresolved layout shifts, and missing font metrics.

### 4. Multi-Viewport Live Audit (`internal/critic`)
- Executes synchronized audits across standard device viewports:
  - **Mobile**: 375 × 667
  - **Tablet**: 768 × 1024
  - **Desktop**: 1440 × 900
- Produces grounded `Finding` records with exact element identifiers, bounding geometries, and ROI evidence digests.

### 5. Relative Candidate Comparison (`internal/design`)
- Evaluates candidate layouts pairwise against baseline implementations across 10 canonical rubric axes:
  1. Visual Hierarchy & Typographic Rhythm
  2. Spatial Composition & Negative Space
  3. Color Balance & Contrast Fidelity
  4. Design System Token Consistency
  5. Accessibility & Responsive Art Direction
  6. Information Density & Scannability
  7. Motion & Interaction Feedback
  8. Layout Stability & Zero Overflow
  9. Semantic HTML & Landmark Structure
  10. Rendering Performance & Budget Compliance
- Enforces strict hard constraints so no visual polish can mask an accessibility or layout regression.

### 6. Autonomous Host Repair Loop (`internal/repair`)
- Proposes targeted HTML and CSS patches based on grounded defect evidence.
- Applies modifications to live workspace files, executes warm browser re-renders, and proves 100% defect remediation without protected-axis regression.

### 7. Epistemic Memory & Skill Evolution (`internal/memory`, `internal/skillstate`, `internal/evolution`)
- **SncSinCore Epistemic Memory**: In-process graph store (`EpMemoryStore`) supporting atomic transactional commits, conflict preservation, and bounded `ContextPack` retrieval without context window bloat.
- **Namespace Firewalls**: Strict separation between `knowledge/global-design`, `knowledge/project/<id>`, `evidence/project/<id>`, `research/global`, and `skillmeta/<skill-id>`.
- **SkillState Projection**: Bounded typed states, CAS patch validation, semantic oscillation detection, and non-regression replay/shadow evaluation gates before skill promotion.

### 8. MCP Server & Control Plane (`cmd/uiuxmaster-mcp`, `control/axiom`)
- Protocol-independent core domain wrapped by a clean Model Context Protocol (MCP) server adhering to specification `2026-07-28`.
- Integrated with Axiom durable control workflows for long-running design polish runs.

---

## Repository Layout

```text
UiUxMaster/
├── cmd/
│   ├── uiuxmaster-mcp/          # MCP server entry point
│   ├── uiuxcdpbench/            # Chromium CDP benchmark harness
│   └── uiuxbench/               # End-to-end telemetry and pipeline benchmarks
├── internal/
│   ├── critic/                  # Multi-viewport audit and semantic critique
│   ├── design/                  # Canonical rubric, pairwise comparison, and scoring
│   ├── engine/                  # Orchestration pipeline and stage telemetry
│   ├── evidence/                # Normalized evidence contracts and packet types
│   ├── evidenceplan/            # Evidence shape planner
│   ├── evolution/               # Controlled skill versioning and replay evaluation
│   ├── fidelity/                # Risk classification and runtime capability router
│   ├── impact/                  # Project indexing, AST parsing, and dependency resolver
│   ├── invalidation/            # Scope calculation and invalidation policies
│   ├── mcpserver/               # Model Context Protocol adapter
│   ├── memory/                  # SncSinCore ontology, epistemic store, and admission
│   ├── repair/                  # Autonomous patch generation and live re-verification
│   ├── research/                # DeepSearch research adapter and provenance admission
│   ├── runtime/
│   │   ├── dispatcher/          # Fidelity-based runtime execution dispatcher
│   │   ├── fastcdp/             # Resident raw-CDP Chromium client and warm page pool
│   │   ├── fastrender/          # Renderer-neutral FastRender contracts
│   │   ├── playwright/          # Playwright TruthPath reference adapter
│   │   └── wggo/                # Pure Go WGGo RGBA rendering adapter
│   ├── skillstate/              # Bounded state machine and MemoryPort bridge
│   ├── verifier/                # Deterministic layout, contrast, and a11y verifiers
│   └── visualdiff/              # In-memory RGBA pixel diffing and bounding box union
├── control/
│   └── axiom/                   # Isolated Axiom durable workflow module
├── docs/                        # Subordinate specifications and architecture ADRs
└── MASTER_PLAN.md               # Authoritative living engineering roadmap
```

---

## Getting Started

### Prerequisites
- **Go**: 1.26.4 or later
- **Chromium / Google Chrome**: Installed locally for L2 resident CDP execution (defaults to system Chrome path)

### Running Tests
Execute the complete unit and integration test suite with the race detector:

```bash
go test -race ./...
```

To run tests in the isolated Axiom control module:

```bash
cd control/axiom && go test -race ./...
```

### Running Benchmarks
Evaluate warm CDP capture, DOM extraction, and rendering latency:

```bash
go test -bench=. -benchmem ./internal/runtime/fastcdp/...
```

### Running the MCP Server
Launch the standard Model Context Protocol server over stdio:

```bash
go run ./cmd/uiuxmaster-mcp
```

---

## Engineering Status

UiUxMaster has completed **Phase P0 through Phase P4** of [`MASTER_PLAN.md`](MASTER_PLAN.md):
- [x] **P0**: Core contracts, evidence model, and WGGo L1 renderer.
- [x] **P1**: Resident Chromium L2 runtime, raw CDP transport, and warm pool.
- [x] **P2**: Deterministic verifiers, pairwise comparator, and autonomous repair loop.
- [x] **P3**: SncSinCore epistemic memory, SkillState bounded reasoning, and evolution gating.
- [x] **P4**: Real-world project verification, live multi-viewport audit, and memory-backed repair.

Refer to [`MASTER_PLAN.md`](MASTER_PLAN.md) for current milestones and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for detailed evidence invariants.
