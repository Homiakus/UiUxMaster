# Ultra-Fast Visual Loop

## Goal

UiUxMaster must make the frequent design loop feel interactive:

```text
code/CSS change → rendered evidence → decision
```

The target is **tens of milliseconds for the common warm path**, not seconds.

A literal 1000× speedup is not realistic for a full cold browser launch + network navigation + full-page stabilization + screenshot + VLM on every iteration. It becomes realistic only by removing those operations from the hot path.

The architectural rule is therefore:

> **Never launch, navigate, stabilize or recapture the whole page when the existing resident renderer and a smaller evidence scope can answer the question.**

---

## 1. Two runtime paths

### FastPath — resident Chromium renderer

Purpose: inner design/edit loop.

Preferred stack:

```text
Go core
  ↓ direct typed CDP
chromedp/cdproto
  ↓
long-lived chrome-headless-shell
  ↓
already-open page / story gallery
  ↓
HMR or targeted state update
```

Properties:

- browser process launched once;
- contexts/pages pooled and reused;
- application/dev server already warm;
- no `page.goto()` after every edit;
- no `networkidle` waits in the common path;
- no full-page screenshot by default;
- no Node/Playwright sidecar in the latency-critical path;
- targeted `DOMSnapshot.captureSnapshot` and clipped screenshot capture;
- VLM only after deterministic evidence is insufficient.

### TruthPath — high-fidelity Playwright

Purpose: milestone/release verification, cross-browser behavior, complex interaction flows.

Use:

- Playwright against real Chromium/Chrome;
- WebKit and Firefox where required;
- clean contexts for isolation-sensitive checks;
- screenshot stabilization and visual baselines;
- full interaction scenarios;
- hidden/perturbed responsive tests.

FastPath may be permissive about warm state. TruthPath must prove that the product works from a realistic clean state.

---

## 2. Why direct CDP in the hot path

For UiUxMaster the Go process already owns orchestration. A direct CDP client removes an extra Node process and Playwright protocol layer from the most frequent observation operations.

`chromedp`/`cdproto` is preferred initially because:

- native Go;
- direct Chrome DevTools Protocol;
- no external runtime dependency in the Go hot path;
- supports connecting to a long-running remote browser;
- typed access to DOM, Runtime, Page, CSS, Performance and related CDP domains.

Rod should be benchmarked as an alternative direct-CDP driver. Driver choice remains behind an adapter so benchmarks, not preference, decide.

Playwright remains valuable because its actionability, browser coverage, tracing, assertions and interaction semantics are stronger than a minimal CDP driver. It is moved out of the common visual-observation path, not removed from the product.

---

## 3. Browser choice

### `chrome-headless-shell` — FastPath default candidate

Use for frequent real Blink/Chromium layout and rendering where complete Chrome features are not required.

It retains Chromium/Blink rendering while being a lighter automation-oriented shell.

### Modern Chrome/Chromium Headless — TruthPath

Use when authenticity and full browser behavior matter more than minimum latency.

### Lightpanda — optional structural preflight only

Lightpanda is interesting for JS/DOM/network automation and claims roughly an order-of-magnitude speedup over Headless Chrome, but it intentionally has no graphical rendering engine and Playwright compatibility is still incomplete.

Therefore it **must not be used as the visual source of truth** for UiUxMaster.

Potential future role:

```text
very-cheap JS/DOM/network preflight
```

before a real renderer is needed.

---

## 4. Resident renderer lifecycle

The runtime should start a browser daemon once:

```text
uiuxmaster daemon starts
  ↓
launch headless-shell once
  ↓
create warm context(s)
  ↓
open target app / gallery once
  ↓
keep page alive
```

Each project/run gets explicit identity and lease ownership.

Do not launch a new browser per MCP request.

Do not create a new page unless isolation or a different origin/profile requires it.

---

## 5. Warm page pool

Maintain a bounded pool keyed by environment, for example:

```text
project
origin
viewport class
DPR
color scheme
locale when necessary
```

Example warm slots:

```text
390×844 dark
390×844 light
1440×900 dark
1440×900 light
```

Do not prewarm a Cartesian product of every browser × width × theme. Warm only high-frequency slots and create milestone contexts lazily.

Pool state must be observable and resettable.

---

## 6. HMR instead of navigation

For development servers that support HMR:

```text
agent edits source
  ↓
dev server invalidates changed module
  ↓
resident page receives HMR update
  ↓
UiUxMaster waits for update epoch / app-ready signal
  ↓
capture affected evidence
```

The runtime should integrate with an explicit readiness signal rather than sleep-based waiting.

Potential signals:

- Vite HMR lifecycle instrumentation;
- application-defined `window.__UIUX_READY_EPOCH__`;
- DOM mutation epoch;
- framework-specific commit hook adapter.

The common CSS/component loop must not reload the entire page.

---

## 7. Story/gallery mode for component polish

Component polish should use an addressable story gallery similar to modern Playwright component testing:

```text
component story already mounted
  ↓
update props/state or HMR source
  ↓
reuse rendering root
  ↓
inspect only story root
```

This makes component-level UI polishing substantially cheaper than driving the complete application shell.

UiUxMaster should define a small vendor-neutral gallery contract so React/Vue/Svelte/Solid/etc. can expose named states without coupling the core to a framework.

---

## 8. One-shot structural snapshot

Do not execute hundreds of independent `getBoundingClientRect()` / `getComputedStyle()` round trips.

For broad structural inspection prefer a small number of CDP calls.

`DOMSnapshot.captureSnapshot` can return:

- flattened DOM;
- layout tree;
- text boxes;
- selected computed styles.

UiUxMaster should request only a whitelist of styles required by current checks, for example:

```text
display
position
z-index
overflow
visibility
opacity
color
background-color
font-size
font-weight
line-height
white-space
text-overflow
pointer-events
```

The style whitelist should depend on the verifier being run.

---

## 9. Incremental invalidation

The runtime should know what changed.

Input signals may include:

- changed source files;
- changed CSS files/tokens;
- HMR module IDs;
- source-map/component ownership metadata;
- explicit design element annotations.

Then derive an invalidation scope:

```text
local component
section
page
multi-page/global token
```

Only invalidated scopes are re-inspected.

Example:

```text
Button.module.css changed
→ inspect Button instances + immediate containers
→ do not recapture footer, pricing table and unrelated pages
```

A global typography token change intentionally widens scope.

---

## 10. Screenshot strategy

A screenshot is not always necessary.

### L0/L1 checks

Use runtime + structural evidence only when possible.

### Targeted visual check

Capture only the affected region/component.

Direct CDP `Page.captureScreenshot` supports clipping and an `optimizeForSpeed` option.

For semantic critique use a crop sized for the local VLM rather than a full 4K page.

### Full-page capture

Reserve for:

- first baseline;
- major layout change;
- whole-page composition critique;
- milestone/release verification.

---

## 11. Screenshot encoding tiers

Choose representation by task.

### Geometry/diff triage

No screenshot if DOM/layout is decisive.

### Fast visual change detection

Use small ROI images and fast encoding. Benchmark JPEG/WebP/PNG for local use; lossless PNG remains necessary for strict protected baselines.

### Semantic critic

Resize/crop to model-effective resolution before inference.

### Golden regression

Use deterministic lossless capture and the TruthPath.

Do not transmit base64 image payloads through MCP. Store artifacts locally and return references.

---

## 12. Avoid expensive generic waits

Never make `networkidle` the universal readiness condition. Modern applications may intentionally keep connections open.

Prefer:

1. explicit app-ready/HMR epoch;
2. required selector/state;
3. fonts ready when typography matters;
4. one or two animation frames for layout/compositor settlement;
5. targeted stability observation only for the region under test.

Use bounded timeouts and classify readiness failure separately from design failure.

---

## 13. Frame-budget model

The theoretical lower bound of observing a browser-rendered change is tied to the browser render/compositor frame cadence and the work needed by the changed component.

UiUxMaster should optimize around **warm latency budgets**, not promises of impossible cold-start numbers.

Initial engineering targets to measure, not assume:

| Operation | Warm target |
|---|---:|
| Detect HMR/app-ready epoch | < 20 ms after app commit where feasible |
| Structural ROI snapshot | 2–20 ms |
| Local geometry checks | 1–10 ms |
| Small clipped screenshot | 5–30 ms |
| Pixel/region diff | 1–10 ms |
| Fast deterministic decision | < 50 ms total |
| Local visual critic | separate asynchronous/escalated budget |

These targets require benchmarking on representative hardware and apps. They are not correctness guarantees.

A practical frequent loop target is:

```text
~20–100 ms deterministic warm validation
```

with semantic VLM critique running only when necessary.

---

## 14. Latency accounting

Every validation run must expose a latency breakdown:

```text
queue
browser acquire
navigation (should normally be zero)
HMR/readiness
DOM snapshot
layout analysis
screenshot
pixel diff
VLM
MCP serialization
```

Optimization work is invalid without this breakdown.

Use p50/p95/p99 rather than one anecdotal measurement.

---

## 15. Benchmark matrix

Build one reproducible benchmark harness comparing:

### Drivers

- direct raw CDP baseline;
- chromedp/cdproto;
- Rod;
- Playwright library attached to warm browser;
- Playwright Test warm component/gallery mode.

### Browser engines

- chrome-headless-shell;
- modern Chromium Headless;
- branded Chrome Headless where useful;
- Lightpanda for structural-only experiments, never pixel truth.

### Scenarios

1. process cold start;
2. warm page acquire;
3. HMR CSS change;
4. component JS change;
5. DOM/layout snapshot;
6. ROI screenshot;
7. full viewport screenshot;
8. full-page screenshot;
9. click + resulting local state;
10. resize + layout evidence.

Report:

```text
p50 / p95 / p99
CPU
peak/resident memory
allocations in UiUxMaster hot path
bytes over control protocol
```

The fastest implementation that fails fidelity requirements does not win.

---

## 16. Fidelity ladder

The engine chooses the cheapest sufficient renderer/verifier:

```text
L0 runtime signal
 ↓ insufficient
L1 resident CDP DOM/layout
 ↓ insufficient
L2 resident real-renderer ROI screenshot/diff
 ↓ insufficient
L3 local visual critic on crop
 ↓ milestone / ambiguity
L4 Playwright TruthPath / real Chromium / cross-browser
```

This is how UiUxMaster gains orders of magnitude at the system level: **most iterations stop at L1/L2 instead of paying L4 every time.**

---

## 17. Failure and reset policy

Warm state can become corrupted or stale.

Detect and recover from:

- lost CDP connection;
- HMR client failure;
- renderer crash;
- runaway memory;
- stale service worker/cache;
- navigation away from target;
- unrecoverable application state.

Recovery ladder:

```text
reset component/story
→ reset page
→ reset context
→ restart browser
```

Always choose the smallest reset that restores confidence.

---

## 18. Architecture decision

The intended target architecture becomes:

```text
                         ┌──────── MCP / Agent ────────┐
                         │                              │
                         └─────────────┬────────────────┘
                                       ▼
                              UiUxMaster Go core
                                       │
                    ┌──────────────────┴──────────────────┐
                    ▼                                     ▼
               FASTPATH                               TRUTHPATH
          resident browser pool                   Playwright verifier
                    │                                     │
        direct CDP (chromedp/cdproto)          clean/high-fidelity contexts
                    │                                     │
          chrome-headless-shell                 Chromium / WebKit / Firefox
                    │                                     │
             warm app/gallery                     milestone/release
                    │
           HMR / targeted update
                    │
       DOMSnapshot + ROI screenshot
```

Playwright is no longer the mandatory transport for every observation. It remains an essential correctness verifier.

---

## 19. Definition of Done for ultra-fast mode

- browser process is reused across requests;
- normal source edit does not require navigation;
- deterministic ROI validation has a measured p95 budget;
- latency breakdown is returned with evidence;
- warm-path fidelity is checked against TruthPath on the benchmark corpus;
- page/context/browser reset paths are tested;
- full browser/Playwright escalation happens automatically when FastPath evidence is insufficient;
- no claim of visual correctness is made from a non-rendering browser such as Lightpanda.
