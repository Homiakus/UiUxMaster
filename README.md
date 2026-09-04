# UiUxMaster

[![Go Version](https://img.shields.io/badge/Go-1.26.4-00ADD8?logo=go)](https://go.dev/)
[![MCP Spec](https://img.shields.io/badge/MCP%20Spec-2026--07--28-blue)](https://modelcontextprotocol.io/)
[![MCP Server](https://img.shields.io/badge/MCP%20Server-v0.3.0-green)](cmd/uiuxmaster-mcp)
[![Pure Go](https://img.shields.io/badge/Runtime-Pure%20Go%20(Zero%20CGO)-brightgreen)](#multi-tier-visual-runtime)
[![CI Tests](https://img.shields.io/badge/Tests-100%25%20Pass%20(Race%20Clean)-success)](#running-tests)
[![Evaluation](https://img.shields.io/badge/Adversarial%20Recall-100%25-blueviolet)](#adversarial-evaluations--defect-injection)
[![License](https://img.shields.io/badge/License-Apache--2.0%20%2F%20MIT-lightgrey)](#license)

> **Evidence-driven UI/UX engineering system and Model Context Protocol (MCP) server for AI coding agents.**

UiUxMaster is a high-performance, resident visual and interaction verification engine. It enables autonomous AI coding agents to **design, render, inspect, critique, compare, repair, and independently verify** web interfaces using deterministic browser evidence, sub-35ms warm visual loops, and epistemic design defect memory.

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
Incremental Impact Resolution (<1ms / 357ns leaf)
        │
        ▼
Fidelity & Risk Routing (L0 / L1 / L2 / L3)
        │
        ▼
Warm Browser Render & Evidence Capture (<35ms)
(DOMSnapshot + Accessibility Tree + Fonts + ROI RGBA Crop)
        │
        ▼
Deterministic Verifiers (Layout, Contrast, Target Size, Overflows, Focus)
        │
        ▼
Multi-Viewport & Semantic Design Critique (Mobile, Tablet, Desktop)
        │
        ▼
Relative Candidate Comparison (10 Canonical Axes with Hard Constraints)
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

## Model Context Protocol (MCP) Server

UiUxMaster provides a comprehensive Model Context Protocol (MCP) server adhering to specification `2026-07-28`. It exposes 10 domain tools for AI agents:

### Available Tools

| Tool | Purpose | Primary Inputs | Output |
| :--- | :--- | :--- | :--- |
| `uiux_get_rubric` | Returns canonical design principles and 10 quality axes | None | Rubric principles and axes |
| `uiux_evaluate_evidence` | Evaluates raw browser evidence packet and recommends cheapest next step | `Packet` | Grounded `Report` |
| `uiux_plan_validation` | Resolves source changes to minimal scope and plans execution tier | `ChangedFiles`, `ChangedTokens`, `Intent` | `ValidationScope`, `Assessment`, `Route` |
| `uiux_capture` | Runs full canonical pipeline from source edit to verified evidence packet | `HTML`, `CSS`, `URL`, `Region` | `Packet`, `Report`, `Telemetry` |
| `uiux_inspect_layout` | Deterministic verification of overflow, clipping, overlap, target sizing | `Packet`, `Policy` | `Issues`, `Passed` |
| `uiux_inspect_accessibility` | ARIA tree inspection, missing names, focus sequence anomalies | `Packet` | `Issues`, `Nodes`, `Passed` |
| `uiux_critique_page` | Progressive semantic critique producing grounded findings & repair hypotheses | `Packet`, `Profile`, `Level` | `Findings`, `Hypotheses`, `Passed` |
| `uiux_compare_candidates` | Pairwise A/B comparison across 10 axes with hard constraint gates | `BaselinePacket`, `CandidatePacket` | `Comparison`, `Winner`, `Passed` |
| `uiux_recommend_repairs` | Autonomous closed-loop repair: generates concrete HTML/CSS patches | `HTML`, `CSS`, `Profile` | `RepairLoopResult`, `Patches` |
| `uiux_run_scenario` | Validates and inspects interactive user playthrough scenarios | `Scenario` | `Valid`, `ActionCount`, `Summary` |

---

## MCP Server Configuration

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "uiuxmaster": {
      "command": "d:/Programms/UiUxMaster/build/uiuxmaster-mcp.exe",
      "args": []
    }
  }
}
```

### Cursor (`.cursor/mcp.json` or Global Settings)

```json
{
  "mcpServers": {
    "uiuxmaster": {
      "command": "go",
      "args": ["run", "./cmd/uiuxmaster-mcp"],
      "cwd": "d:/Programms/UiUxMaster"
    }
  }
}
```

### Google Antigravity / Gemini (`mcp_config.json`)

```json
{
  "mcpServers": {
    "uiuxmaster": {
      "command": "go",
      "args": ["run", "d:/Programms/UiUxMaster/cmd/uiuxmaster-mcp"]
    }
  }
}
```

---

## Multi-Tier Visual Runtime

UiUxMaster uses fidelity-based capability routing to select the cheapest sufficient tier capable of proving a property:

| Tier | Engine | Latency | Capabilities | Role |
| :--- | :--- | :--- | :--- | :--- |
| **L0** | Native Go AST/Token Scanner | **< 1 ms** (357ns leaf) | File dependencies, CSS custom properties, import graph | Static preflight & scope bounding |
| **L1** | In-Process WGGo (`internal/runtime/wggo`) | **1 – 10 ms** | Pure Go in-process RGBA rendering, in-memory ROI crop | Instant speculative visual diffing |
| **L2** | Resident FastBrowser (`internal/runtime/fastcdp`) | **< 35 ms** | DOMSnapshot, ARIA tree, computed styles, fonts, ROI capture | Warm Blink ground truth |
| **L3** | Playwright TruthPath (`internal/runtime/playwright`) | Clean Session | Clean browser launch, Firefox/WebKit parity, interaction | Cross-browser reference & calibration |

---

## 10 Canonical Rubric Axes

Every visual and interaction evaluation compares candidates across 10 canonical axes with strict hard constraints:

1. **Visual Hierarchy & Typographic Rhythm**: Single clear `<h1>`, proportional modular scales, consistent vertical rhythm.
2. **Spatial Composition & Negative Space**: Intentional white space, macro/micro margins, consistent grid alignment.
3. **Color Balance & Contrast Fidelity**: WCAG 2.2 AA (4.5:1 / 3:1) compliance, APCA perceptual contrast, semantic accent usage.
4. **Design System Token Consistency**: CSS custom property adherence, unified corner radiuses, semantic spacing tokens.
5. **Accessibility & Responsive Art Direction**: Touch target sizes (>= 44x44px), zero horizontal overflow across 375px/768px/1440px.
6. **Information Density & Scannability**: F/Z-pattern layouts, visual anchors, legible data presentation without clutter.
7. **Motion & Interaction Feedback**: Pointer states, visible focus outlines, smooth transitions respecting `prefers-reduced-motion`.
8. **Layout Stability & Zero Overflow**: Zero unexpected cumulative layout shifts (CLS), container-aware boundaries.
9. **Semantic HTML & Landmark Structure**: Semantic `<header>`, `<main>`, `<nav>`, `<footer>`, valid heading nesting.
10. **Rendering Performance & Budget Compliance**: Sub-50ms render latency, minimal DOM depth, low allocation footprint.

---

## Adversarial Evaluations & Defect Injection

UiUxMaster includes a dedicated adversarial evaluation harness (`internal/eval`) that measures verifier accuracy by injecting synthetic defects across 10 categories:

```text
=== RUN   TestAdversarialEvalSuite_RecallAndLocalization
  Adversarial Eval Results:
    Total Injected:  10
    Total Detected:  10
    Total Localized: 10
    Recall:          100.00%
    Precision:       100.00%
    Localization:    100.00%
  [layout.viewport_overflow]     Recall: 100.0%
  [interaction.target_too_small] Recall: 100.0%
  [dom.duplicate_id]             Recall: 100.0%
  [typography.text_truncation]   Recall: 100.0%
  [accessibility.missing_name]   Recall: 100.0%
  [layout.fixed_obstruction]     Recall: 100.0%
  [design.heading_hierarchy]     Recall: 100.0%
  [interaction.overlap]          Recall: 100.0%
  [accessibility.focus_sequence] Recall: 100.0%
  [interaction.pointer_disabled] Recall: 100.0%
--- PASS: TestAdversarialEvalSuite_RecallAndLocalization (0.00s)
```

---

## Repository Layout

```text
UiUxMaster/
├── cmd/
│   ├── uiuxmaster-mcp/          # Model Context Protocol (MCP) server entry point
│   ├── uiuxcdpbench/            # Chromium CDP benchmark harness
│   └── uiuxbench/               # End-to-end telemetry and pipeline benchmarks
├── internal/
│   ├── critic/                  # Multi-viewport audit and semantic critique
│   ├── design/                  # Canonical rubric, pairwise comparison, and scoring
│   ├── engine/                  # Orchestration pipeline and stage telemetry
│   ├── eval/                    # Adversarial defect injection and recall evaluation harness
│   ├── evidence/                # Normalized evidence contracts and packet types
│   ├── evidenceplan/            # Evidence shape planner
│   ├── evolution/               # Controlled skill versioning and replay evaluation
│   ├── fidelity/                # Risk classification and runtime capability router
│   ├── impact/                  # Project indexing, AST parsing, and dependency resolver
│   ├── invalidation/            # Scope calculation and invalidation policies
│   ├── mcpserver/               # Model Context Protocol adapter (10 tools)
│   ├── memory/                  # SncSinCore ontology, epistemic store, and admission
│   ├── repair/                  # Autonomous patch generation and live re-verification
│   ├── research/                # DeepSearch research adapter and provenance admission
│   ├── runtime/
│   │   ├── dispatcher/          # Fidelity-based runtime execution dispatcher
│   │   ├── fastcdp/             # Resident raw-CDP Chromium client and warm page pool
│   │   ├── fastrender/          # Renderer-neutral FastRender contracts
│   │   ├── playwright/          # Playwright TruthPath reference & scenario adapter
│   │   └── wggo/                # Pure Go WGGo RGBA rendering adapter
│   ├── skillstate/              # Bounded state machine and MemoryPort bridge
│   ├── verifier/                # Deterministic layout, contrast, focus, and a11y verifiers
│   └── visualdiff/              # In-memory RGBA pixel diffing and bounding box union
├── control/
│   └── axiom/                   # Isolated Axiom durable workflow module
├── docs/                        # Subordinate specifications and architecture ADRs
└── MASTER_PLAN.md               # Authoritative living engineering roadmap
```

---

## Getting Started

### Prerequisites
- **Go**: `1.26.4` or later
- **Chromium / Google Chrome**: Installed locally for L2 resident CDP execution (defaults to system Chrome path)

### Running Tests
Execute the complete unit, integration, and evaluation test suite under the Go race detector:

```bash
go test -race ./...
```

To run tests in the isolated Axiom control module:

```bash
cd control/axiom && go test -race ./...
```

### Running the MCP Server
Launch the standard Model Context Protocol server over stdio:

```bash
go run ./cmd/uiuxmaster-mcp
```

Or build the standalone binary:

```bash
go build -o ./build/uiuxmaster-mcp.exe ./cmd/uiuxmaster-mcp
```

### Running Benchmarks
Evaluate warm CDP capture, DOM extraction, and rendering latency:

```bash
go test -bench=. -benchmem ./internal/runtime/fastcdp/...
```

---

## Engineering Status

UiUxMaster has completed **Phase P0 through Phase P5** of [`MASTER_PLAN.md`](MASTER_PLAN.md):
- [x] **P0**: Core contracts, evidence model, and WGGo L1 renderer.
- [x] **P1**: Resident Chromium L2 runtime, raw CDP transport, and warm pool.
- [x] **P2**: Deterministic verifiers, pairwise comparator, and autonomous repair loop.
- [x] **P3**: SncSinCore epistemic memory, SkillState bounded reasoning, and evolution gating.
- [x] **P4**: Real-world project verification, live multi-viewport audit, and memory-backed repair.
- [x] **P5**: Extended deterministic verifiers, interactive playthrough scenarios, full MCP product surface (10 tools), and adversarial evaluation harness (100% recall).

Refer to [`MASTER_PLAN.md`](MASTER_PLAN.md) for the living roadmap and [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for evidence invariants.

---

## License

This project is licensed under the Apache License 2.0 and MIT License.
