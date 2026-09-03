# UiUxMaster Architecture

## 1. Purpose

UiUxMaster is not a screenshot scorer. It is a UI/UX engineering control system for agents.

The system must answer four different questions without collapsing them into one unreliable signal:

1. **Did the application render and behave correctly?**
2. **Did the implementation preserve accessibility and responsive behavior?**
3. **Did the visual result regress relative to a protected baseline?**
4. **Is the result actually a stronger design according to the intended product/art direction?**

These questions require different evidence and different verifiers.

---

## 2. Architectural boundaries

### 2.1 Domain core

The domain core must not import Playwright, Ollama, a cloud VLM SDK, or MCP.

It owns:

- design axes and rules;
- evidence types;
- findings;
- critique passes;
- comparison decisions;
- validation policy;
- design-memory promotion policy.

### 2.2 Runtime adapters

Adapters convert external observations into the canonical `evidence.Packet`.

Examples:

- Playwright worker;
- pixel-diff analyzer;
- local VLM;
- accessibility checker;
- future browser/device farm.

### 2.3 MCP adapter

MCP owns only protocol concerns:

- tool/resource schemas;
- result size/serialization;
- transport;
- caching hints when relevant;
- task/resource exposure.

A domain function must be callable without MCP.

---

## 3. Evidence hierarchy

Use the cheapest trustworthy evidence first.

```text
L0 runtime
   page errors, failed requests, missing fonts, readiness

L1 structure
   ARIA snapshot, semantic roles, DOM geometry, computed styles

L2 deterministic visual
   screenshot baseline, pixel/region diff, clipping/overlap

L3 semantic visual
   local VLM critique of targeted crop + structural context

L4 escalated semantic visual
   stronger model / multi-candidate comparison only when uncertainty warrants it
```

The engine should stop escalating when the current evidence is sufficient to make a high-confidence decision.

---

## 4. Canonical evidence packet

The packet is the interoperability boundary between runtime collection and reasoning.

It must carry:

- run/scenario identity;
- browser/viewport/theme environment;
- semantic element references;
- bounding boxes and selected styles;
- deterministic runtime issues;
- suspicious visual regions;
- grounded visual findings;
- artifact references (screenshots, diffs, traces);
- optional ARIA snapshot.

Large binary artifacts do not belong inline in MCP tool results. MCP should return resource/artifact references.

---

## 5. Stable element references

CSS selectors alone are too brittle for agent feedback.

Within a validation run, create semantic references using the strongest available identity:

1. explicit test/design ID;
2. accessibility role + accessible name;
3. stable application ID;
4. structural path as fallback;
5. CSS selector only as a last-mile locator.

A finding should be understandable as:

```text
hero/actions → button "Publish" → overlaps primary navigation by 18px
```

rather than:

```text
#app > div:nth-child(3) > div:nth-child(2)
```

---

## 6. Hierarchical visual model

Every rendered page can be represented as:

```text
Page
└─ Section
   └─ Component
      └─ Element
```

The runtime worker should infer hierarchy from explicit annotations where available and otherwise use semantic/layout grouping.

Why this matters:

- visual criticism at full-page resolution loses local detail;
- pixel regions alone lack semantic meaning;
- repair suggestions should target the smallest meaningful ownership boundary.

---

## 7. Deterministic verifiers

Deterministic checks should own problems that do not require aesthetic judgement.

Examples:

- horizontal overflow;
- clipped text/control;
- hidden/offscreen interactive element;
- overlap;
- invalid geometry;
- missing accessible name;
- focus obstruction;
- failed network request;
- runtime exception;
- broken font load;
- screenshot mismatch beyond configured tolerance.

These findings should not be sent to a VLM merely to rediscover them in prose.

---

## 8. Semantic visual critic

The semantic critic exists for questions such as:

- hierarchy feels flat;
- typography relationships are weak;
- card chrome overwhelms content;
- image crop harms composition;
- accent use is excessive;
- section rhythm is repetitive;
- design is technically correct but generic.

### Required critic input

- user/design brief;
- relevant rubric axes;
- current crop/screenshot;
- hierarchy path;
- nearby element refs;
- normalized geometry/style evidence;
- previous critique + applied change;
- optional baseline/candidate image.

### Required output

Structured finding:

```json
{
  "axis": "typography",
  "severity": "medium",
  "confidence": 0.86,
  "region_id": "hero-copy",
  "element_ids": ["hero-h1", "hero-lede"],
  "evidence": ["heading and lede have near-identical visual weight"],
  "suggestion": "increase type-scale separation before changing decorative styling"
}
```

The model is a critic, not an oracle. Its result remains one verifier signal.

---

## 9. Relative design judgement

Absolute aesthetic scoring is poorly calibrated and encourages false precision.

Prefer pairwise or small-set comparison:

```text
baseline vs candidate A
candidate A vs candidate B
```

Compare by independent axes, then apply hard constraints:

- candidate violating accessibility cannot win overall due to aesthetics;
- candidate breaking interaction cannot win due to visual novelty;
- protected brand/product constraints remain explicit.

A combined score may exist for ranking convenience but must preserve per-axis evidence.

---

## 10. Visual regression vs design critique

These are intentionally separate.

### Regression verifier

Asks:

> Did the render unexpectedly change relative to a protected baseline?

### Design critic

Asks:

> Is the candidate a stronger design for the actual intent?

A perfect match to a poor baseline is not a successful redesign.

---

## 11. History-aware refinement

Each critique pass should receive a compact history:

```text
goal
previous state
applied patch summary
previous findings
resolved finding IDs
unresolved finding IDs
current evidence
```

The engine should explicitly detect oscillation:

```text
fix A → breaks B → restore A → breaks A again
```

and force a higher-level reconsideration rather than continuing local patching.

---

## 12. Browser worker contract

The initial Playwright worker should expose a small RPC/JSON-lines API independent from MCP.

### `capture`

Input:

```json
{
  "url": "http://127.0.0.1:3000",
  "viewport": {"width": 1440, "height": 900},
  "colorScheme": "dark",
  "fullPage": true
}
```

Output conceptually maps to `evidence.Packet` and artifact paths.

### `scenario`

Runs ordered actions:

- navigate;
- click;
- fill;
- press;
- hover;
- resize;
- wait-for-ready;
- capture checkpoint.

### `inspect`

Returns targeted semantic/geometry/style evidence for selected refs/roles/regions.

The Go adapter owns worker lifecycle, timeouts and error conversion.

---

## 13. Pixel diff architecture

Use two layers:

### Playwright assertion layer

For ordinary stable visual-regression assertions.

### Region-analysis layer

For agent reasoning:

1. create diff mask;
2. cluster connected/nearby changed pixels;
3. discard tiny noise according to policy;
4. create region boxes;
5. intersect regions with semantic DOM boxes;
6. emit ranked suspicious regions.

This turns pixels into actionable evidence.

---

## 14. Local VLM provider interface

The core should depend on an interface similar to:

```text
Critique(ctx, CritiqueRequest) → CritiqueResult
Compare(ctx, ComparisonRequest) → ComparisonResult
```

Provider configuration supplies:

- endpoint;
- model;
- max image dimensions;
- timeout;
- concurrency;
- privacy mode;
- capability profile.

Do not encode `MiniCPM`, `Qwen`, `Ollama`, or a cloud vendor into domain types.

---

## 15. Design memory

Memory stores evidence-backed generalized knowledge, not conversations.

Suggested record:

```text
ID
status: observation | candidate | validated | invariant
rule
scope
positive examples
counterexamples
source runs
validation score
created/updated
```

Promotion must require replay across unrelated fixtures.

---

## 16. Adversarial evaluation

The project needs its own benchmark because a UI critic can overfit to visible fixtures.

Mutation operators should introduce controlled defects such as:

- spacing flattening;
- type hierarchy collapse;
- color contrast reduction;
- overflow;
- clipping;
- misalignment;
- broken focus state;
- bad crop;
- excessive radius/card chrome;
- dark-theme mismatch;
- touch target shrinkage.

Ground truth is known because the mutation is generated by the harness.

Metrics:

- detection recall;
- false positives;
- localization IoU/element accuracy;
- severity calibration;
- repair success;
- regression rate;
- cost and latency.

---

## 17. MCP 2026 direction

UiUxMaster should target the current stateless MCP direction.

Architectural consequences:

- avoid coupling runtime state to an MCP session;
- identify runs explicitly in parameters/resources;
- make tool listings stable and cache-friendly;
- use resources/artifacts for large evidence;
- use OpenTelemetry for structured remote observability;
- treat deprecated roots/sampling/logging features as compatibility-only, not new architectural dependencies;
- adopt Tasks only when long-running browser/eval jobs need it and host support is sufficient.

---

## 18. Security and privacy

Default policy:

```text
local browser
→ local structural checks
→ local image diff
→ local VLM
```

Optional external model use must be an explicit provider choice.

Before external transmission:

- crop to the minimum relevant region;
- redact secrets/token-like values;
- minimize DOM text;
- record provider/model and purpose;
- obey retention policy.

UiUxMaster must not provide a mechanism for bypassing target authorization. The operator remains responsible for having permission to test the target application.

---

## 19. Failure philosophy

The system must distinguish:

- **application failure** — page is broken;
- **collector failure** — Playwright/VLM adapter failed;
- **evidence insufficiency** — cannot decide yet;
- **critic disagreement** — independent signals conflict;
- **accepted deviation** — user explicitly accepts a finding.

Never convert “collector failed” into “design passed”.

---

## 20. Evolution rule

When a new capability is proposed, ask:

1. Does it add a new evidence source, verifier, policy or transport?
2. Which package owns it?
3. Can it remain independent of MCP?
4. Can its output be represented in the canonical evidence model?
5. How will it be evaluated adversarially?
6. What is the cheapest earlier signal that could avoid invoking it?

If these answers are unclear, do not add the abstraction yet.
