# Research Foundations

UiUxMaster converts recent GUI-agent, frontend-generation and evaluation research into explicit engineering principles. Research informs architecture; it is not treated as an unquestionable product specification.

## 1. Iterative render-guided refinement

### UI2CodeN

Reference: https://arxiv.org/abs/2511.08195

Useful ideas:

- frontend generation is an interactive visual optimization problem;
- additional render/critique iterations can improve fidelity;
- relative preference is often more useful than a poorly calibrated absolute visual score.

UiUxMaster consequence:

- mandatory render → critique → repair loops for serious polish;
- pairwise/candidate comparison contracts;
- no single “beauty score” completion gate.

### Vision-Guided Iterative Refinement for Frontend Code Generation

Reference: https://arxiv.org/abs/2604.05839

Useful idea:

- separating generation from a vision-guided critic can improve frontend output over multiple refinement rounds.

UiUxMaster consequence:

- critic is an independent verifier signal;
- engine records critique history and re-renders after repair.

---

## 2. Hierarchy before pixels

### DesignCoder

Reference: https://arxiv.org/abs/2506.13663

Useful idea:

- hierarchy-aware decomposition improves structural and visual reconstruction.

UiUxMaster consequence:

```text
page → section → component → element
```

is a first-class model used for evidence localization and repair ownership.

### DCGen

Reference: https://arxiv.org/abs/2406.16386

Useful idea:

- smaller visual regions can reduce omission and placement errors compared with treating a dense UI as one undifferentiated image.

UiUxMaster consequence:

- whole-page screenshots are navigation context;
- semantic critique should prefer the smallest relevant crop.

---

## 3. Progressive visual attention

### ScreenSpot-Pro

Reference: https://arxiv.org/abs/2504.07981

Useful idea:

- professional/dense interfaces remain difficult for GUI grounding models; narrowing the search space matters.

### RegionFocus

Reference: https://arxiv.org/abs/2505.00684

Useful idea:

- test-time visual scaling through targeted regions/zoom can materially improve grounding.

UiUxMaster consequence:

```text
whole page
→ suspicious section
→ component crop
→ element + geometry
→ stronger model only if uncertainty remains
```

The system should spend vision compute adaptively rather than processing every full-resolution screenshot with the strongest VLM.

---

## 4. Multi-sensor evidence

### WebLINX

Reference: https://arxiv.org/abs/2402.05930

Useful idea:

- large raw HTML/DOM representations need retrieval/pruning; multimodal context benefits from selecting relevant structure.

UiUxMaster consequence:

- do not dump the entire DOM into an agent prompt;
- retrieve semantic subtrees/elements related to the region/task;
- combine screenshot, ARIA, geometry and computed styles.

### WebCompat / XCompat

Reference: https://arxiv.org/abs/2608.12518

Useful idea:

- cross-environment web failures benefit from combining visual and structural information.

UiUxMaster consequence:

- browser/device compatibility is a separate release-level verifier;
- screenshot evidence alone is insufficient.

---

## 5. Grounded aesthetic evaluation

### AesEval-Bench

Reference: https://arxiv.org/abs/2603.01083

Useful ideas:

- aesthetic problems should be decomposed across dimensions;
- useful evaluators should detect, classify and localize defects.

UiUxMaster consequence:

A finding contains:

```text
axis
severity
confidence
region
element refs
evidence
repair direction
```

### TASTE

Reference: https://arxiv.org/abs/2605.20731

Useful idea:

- professional design judgement is multidimensional and current multimodal judges do not reliably match expert preference with one scalar score.

UiUxMaster consequence:

- preserve independent axes such as typography, hierarchy, color, spatial rhythm and identity;
- use human/reference calibration datasets for high-quality evals.

### WebDevJudge

Reference: https://arxiv.org/abs/2510.18560

Useful idea:

- automated model judges can diverge from expert web-development assessment.

UiUxMaster consequence:

- VLM PASS is never sufficient evidence of completion.

---

## 6. History-aware critique and learning from failures

### HiViG

Reference: https://arxiv.org/abs/2606.11078

Useful idea:

- GUI criticism improves when it is grounded in compact action/history context rather than only the latest screenshot.

UiUxMaster consequence:

Each refinement pass receives:

- goal;
- previous state;
- applied patch summary;
- resolved/unresolved findings;
- current evidence.

### UI-Voyager

Reference: https://arxiv.org/abs/2603.24533

Useful idea:

- failed trajectories contain useful supervision; critical fork points between success/failure should not be discarded.

UiUxMaster consequence:

- store failed design trajectories for eval/lesson extraction;
- detect repeated oscillation during repair;
- promote only generalized lessons that survive replay.

---

## 7. Step-level verification

### STEVE

Reference: https://arxiv.org/abs/2503.12532

Useful idea:

- dense stepwise verification can be more informative than only a final trajectory reward.

UiUxMaster consequence:

- significant typography/composition/color/interaction patches get targeted re-render/checkpoints;
- failures can be attributed to a smaller patch set.

---

## 8. Interaction over screenshots

### PlayCoder

Reference: https://arxiv.org/abs/2604.19742

Useful idea:

- compile/test success can miss GUI state and interaction errors that task-oriented playthroughs reveal.

UiUxMaster consequence:

Visual verification has two families:

```text
static render verification
interactive scenario verification
```

A polished screenshot cannot prove a modal, form, menu or responsive transition is correct.

---

## 9. Anti-reward-hacking verification

### Building to the Test

Reference: https://arxiv.org/abs/2606.28430

Useful idea:

- coding agents can optimize to visible tests without implementing the user-intended solution.

### SpecBench

Reference: https://arxiv.org/abs/2605.21384

Useful idea:

- visible verifier saturation does not imply held-out compositional/specification correctness.

### Verification Horizon

Reference: https://arxiv.org/abs/2606.26300

Useful idea:

- a verifier is a proxy for intent and must evolve with the generator/system.

UiUxMaster consequence:

- use hidden/perturbed widths, content lengths, themes and failure modes;
- keep specification/interaction/accessibility/visual signals independent;
- periodically mutate evaluation cases so the agent cannot merely memorize a visible suite.

---

## 10. Architectural synthesis

The research above is summarized by twelve project invariants:

1. Render over code assumptions.
2. Interaction over static screenshot claims.
3. Relative preference over one absolute aesthetic score.
4. Hierarchy before pixel repair.
5. Coarse-to-fine visual attention.
6. Multimodal evidence over a single sensor.
7. Localization before repair.
8. History-aware critique.
9. Step verification over final-only verification.
10. Independent verifiers over a single AI judge.
11. Hidden/adversarial evaluation over visible-test optimization.
12. Failure-derived learning with promotion gates rather than uncontrolled self-modification.

These are architectural constraints, not merely prompt instructions.
