# FMEA-010 — memory isolation, poisoning resistance and promotion authority

Status: implementation/closure candidate. Risk authority remains `docs/FMEA_RISK_REGISTER.md` until merged CI evidence is recorded.

## Safety objective

Epistemic memory must never widen information visibility merely because a caller selected a broader namespace, omitted a scope, reused an atom ID, or created a graph edge. Project-private evidence can become reusable global knowledge only through one explicit promotion transaction that records sanitized output, independent evidence, verifier identity and rollback lineage.

## Canonical namespace rules

Namespaces are durable JSON strings and are never inferred from serialized private struct fields.

- `knowledge/global-design` — reusable global design knowledge.
- `research/global` — globally readable research evidence; it is not interchangeable with global design knowledge.
- `knowledge/project/<project>` — project-private admitted knowledge.
- `evidence/project/<project>` — project-private evidence/source scope.
- `skillmeta/<skill>` — skill-local metadata.

Read authorization and mutation authorization are intentionally different:

- project scope may read its own private partitions plus global namespaces;
- global scope cannot read project-private partitions;
- project A cannot read project B;
- missing scope is not administrator scope and fails closed;
- read access to a global namespace never grants project authority to mutate it;
- ordinary write may move evidence→knowledge only within the same project;
- `research/global` cannot ordinary-write `knowledge/global-design`;
- project→global and project A→B ordinary writes are forbidden.

## Store-level authorization

Mapper validation is not a trust boundary. `EpMemoryStore.Commit` and `CommitOnce` independently verify:

1. non-empty source namespace for every non-empty bundle;
2. each atom target is an authorized ordinary route;
3. atom and edge provenance source namespace matches the bundle source;
4. project/global provenance matches the source scope;
5. confidence and edge weights are bounded;
6. duplicate IDs inside one bundle are rejected;
7. an existing atom ID cannot be rewritten into another namespace;
8. retracted/superseded atoms cannot be silently resurrected by ordinary admission;
9. every graph endpoint must resolve to an active atom;
10. outgoing edges must be asserted from a namespace the source may mutate;
11. an edge cannot point into another private project;
12. conflict/refutation edges require mutation authority on their target.

These checks prevent a direct-store caller from bypassing mapper policy.

## Retrieval and mutation rules

`Query`, `RetrieveContextPack`, `GetAtom` and `GetEdges` all require an explicit valid scope. `GetEdges` hides edges if either endpoint is missing, inactive or inaccessible. Context-pack conflicts are emitted only when both conflicting atoms are active and visible to the request scope.

`Retract` and `Supersede` require explicit mutation scope. Project scope cannot retract or supersede global knowledge merely because it may read that knowledge.

## Repair admission

Autonomous repair memory is project-private. A successful repair with no explicit `ProjectID` may commit its source repair but does not enter long-term memory. It must never fall back to global memory. With a `ProjectID`, repair admission is bound to `evidence/project/<id> -> knowledge/project/<id>` and retains FMEA-007 exactly-once semantics.

## Research ingestion

Research ingestion is ordinary admission into its selected research scope. Default ingestion is `research/global -> research/global`. Research claims do not become `knowledge/global-design` merely because they are public; global design reuse requires an explicit promotion/generalization decision.

## Explicit project → global promotion

`EpMemoryStore.Promote` is the sole project-private → `knowledge/global-design` path.

A legal promotion requires:

- project-private source namespace;
- exact `knowledge/global-design` target;
- a new global `DesignRule` candidate;
- candidate confidence >= 0.90;
- independent verifier ID and rationale;
- at least two active source atoms from the exact same source project;
- source confidence >= 0.80;
- no active conflict on any source;
- valid source/project lineage;
- fresh source provenance (`PromotionMaxSourceAge`);
- at least two distinct evidence digests;
- at least two distinct source run IDs;
- caller-supplied evidence digest set exactly equal to the source atoms' evidence set;
- generalized candidate free of project identifier and known private/credential markers;
- deterministic payload digest bound to source IDs, evidence set, candidate semantics, verifier and rationale.

The promoted atom records:

- source namespace;
- source atom IDs;
- decision digest;
- verifier ID;
- global project scope;
- promotion run identity.

The `PromotionRecord` additionally records distinct source run IDs and evidence digests.

## Promotion side-effect semantics

Promotion and rollback use the same semantic-operation/receipt discipline as FMEA-007. Same logical operation + same payload reuses the prior receipt. Payload drift conflicts rather than creating a second effect.

Manual promotion rollback retracts the promoted global atom but preserves private sources. Retraction or supersession of any source atom conservatively revokes every active promotion depending on that source. Private source history is retained as retracted/superseded history rather than deleted.

## Evolution boundary

Reusable skill evolution is also fail-closed:

- a heuristic requires at least two globally safe admitted `DesignFinding` atoms;
- findings must be from at least two distinct evidence digests and run identities;
- project-private findings cannot feed reusable skill evolution directly;
- candidate promotion requires an explicit `PromotionAuthorization` from replay + shadow/non-regression evaluation;
- an empty replay corpus is a failed gate, never a PASS;
- authorization binds the exact corpus digest and is rechecked at promotion;
- candidate heuristic lineage (finding IDs, evidence digests, run IDs) must be present.

## Adversarial closure suite

Named regression tests include:

- `TestFMEA010DenyUnscopedAndCrossProjectRetrieval`
- `TestFMEA010OrdinaryAdmissionCannotBroadenScopeOrForgeLineage`
- `TestFMEA010AtomIDCollisionCannotRewriteAnotherNamespace`
- `TestFMEA010GraphEdgesCannotCrossPrivateProjectsOrMutateGlobalConflicts`
- `TestFMEA010InactiveEdgeEndpointNeverLeaks`
- `TestFMEA010PromotionReplayAndRetractionPropagation`
- `TestFMEA010PoisonedPromotionCandidatesFailClosed`
- `TestFMEA010PromotionRejectsFakeExtraEvidenceDigest`
- `TestFMEA010PromotionRejectsSameRunTwice`
- `TestFMEA010PromotionRejectsStaleSourceProvenance`
- `TestFMEA010PromotionRecordCarriesDistinctRunLineage`
- `TestFMEA010PromotionRollbackPreservesPrivateSources`
- `TestFMEA010RepairWithoutProjectNeverGlobalizesMemory`
- `TestStoreMemoryPortRejectsGlobalEscape`
- `TestEvolutionHeuristicRejectsSinglePrivateAndNonIndependentEvidence`
- `TestEvolutionPromotionRejectsEmptyCorpusAndCorpusDrift`

FMEA-007 replay tests remain mandatory regression evidence because promotion, rollback and project repair memory are externally observable epistemic effects.

## Reopen triggers

Reopen FMEA-010 if any of the following becomes possible:

- missing scope means all-project/admin visibility;
- direct store commit can widen source visibility;
- atom ID collision can move or overwrite another namespace;
- project A can link into project B or mutate global conflict state;
- project-private content can enter global knowledge through ordinary admission;
- promotion can claim independence using duplicate/fabricated evidence or one run;
- stale/conflicted/low-confidence/private-marked sources can promote;
- promoted provenance loses source IDs/scope/verifier/decision identity;
- source retraction/supersession leaves a dependent promoted claim active;
- rollback deletes private evidence instead of revoking only global visibility;
- evolution accepts a single/private finding, empty replay corpus, or stale corpus authorization;
- FMEA-007 idempotency/replay regression tests fail.
