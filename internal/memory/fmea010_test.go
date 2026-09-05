package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

func fmea010ProjectScopes(t *testing.T, project string) (Namespace, Namespace) {
	t.Helper()
	evidenceNS, err := NewProjectEvidenceNamespace(project)
	if err != nil { t.Fatal(err) }
	knowledgeNS, err := NewProjectKnowledgeNamespace(project)
	if err != nil { t.Fatal(err) }
	return evidenceNS, knowledgeNS
}

func fmea010Prov(source Namespace, project, run, digest string, confidence float64) ProvenanceRecord {
	return ProvenanceRecord{
		RunID: run,
		EvidenceDigest: digest,
		Renderer: "playwright",
		ProjectScope: project,
		SourceNamespace: source.String(),
		Timestamp: time.Now(),
		Outcome: "CONFIRMED",
	}
}

func fmea010Finding(id string, ns Namespace, prov ProvenanceRecord, confidence float64) MemoryAtom {
	return MemoryAtom{
		ID: id,
		Kind: NodeDesignFinding,
		Namespace: ns,
		Provenance: prov,
		Confidence: confidence,
		Data: DesignFindingAtom{FindingID: id, Axis: "spacing", Category: "layout", Title: "General layout observation", Description: "Observed across deterministic fixture"},
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestFMEA010DenyUnscopedAndCrossProjectRetrieval(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	sourceA, projectA := fmea010ProjectScopes(t, "project-a")
	_, projectB := fmea010ProjectScopes(t, "project-b")
	provA := fmea010Prov(sourceA, "project-a", "run-a", "sha256:evidence-a", 1)
	atomA := fmea010Finding("finding:a", projectA, provA, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{atomA}}); err != nil { t.Fatal(err) }

	if _, err := store.Query(ctx, QueryRequest{}); !errors.Is(err, ErrScopeRequired) { t.Fatalf("unscoped Query err=%v want ErrScopeRequired", err) }
	if _, err := store.RetrieveContextPack(ctx, ContextPackRequest{}); !errors.Is(err, ErrScopeRequired) { t.Fatalf("unscoped ContextPack err=%v want ErrScopeRequired", err) }

	gotA, err := store.Query(ctx, QueryRequest{Namespace: projectA})
	if err != nil || gotA.Total != 1 { t.Fatalf("project A query=%#v err=%v", gotA, err) }
	gotB, err := store.Query(ctx, QueryRequest{Namespace: projectB})
	if err != nil || gotB.Total != 0 { t.Fatalf("project B leaked A: %#v err=%v", gotB, err) }
	global := NewGlobalDesignNamespace()
	gotGlobal, err := store.Query(ctx, QueryRequest{Namespace: global})
	if err != nil || gotGlobal.Total != 0 { t.Fatalf("global leaked project A: %#v err=%v", gotGlobal, err) }

	if _, err := store.GetAtom(ctx, projectB, "finding:a"); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("project B GetAtom err=%v", err) }
	if _, err := store.GetAtom(ctx, global, "finding:a"); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("global GetAtom err=%v", err) }
	if _, err := store.GetEdges(ctx, global, "finding:a"); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("global GetEdges err=%v", err) }
}

func TestFMEA010OrdinaryAdmissionCannotBroadenScopeOrForgeLineage(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	sourceA, projectA := fmea010ProjectScopes(t, "project-a")
	_, projectB := fmea010ProjectScopes(t, "project-b")
	provA := fmea010Prov(sourceA, "project-a", "run-a", "sha256:evidence-a", 1)

	globalAtom := fmea010Finding("leak:global", NewGlobalDesignNamespace(), provA, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{globalAtom}}); !errors.Is(err, ErrAdmissionRoute) { t.Fatalf("project->global err=%v want ErrAdmissionRoute", err) }
	crossAtom := fmea010Finding("leak:b", projectB, provA, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{crossAtom}}); !errors.Is(err, ErrAdmissionRoute) { t.Fatalf("project A->B err=%v want ErrAdmissionRoute", err) }

	forged := provA
	forged.SourceNamespace = projectB.String()
	forgedAtom := fmea010Finding("forged", projectA, forged, 1)
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: sourceA, Atoms: []MemoryAtom{forgedAtom}}); !errors.Is(err, ErrAdmissionRoute) { t.Fatalf("forged provenance err=%v want ErrAdmissionRoute", err) }

	global := NewGlobalDesignNamespace()
	globalProv := ProvenanceRecord{RunID: "global-run", EvidenceDigest: "sha256:global", Renderer: "curated", ProjectScope: "global", SourceNamespace: global.String(), Timestamp: time.Now(), Outcome: "CONFIRMED"}
	globalRule := MemoryAtom{ID: "global:rule", Kind: NodeDesignRule, Namespace: global, Provenance: globalProv, Confidence: 1, Data: DesignRuleAtom{RuleID: "global", Axis: "a11y", Category: "global", Title: "Global rule", Description: "Curated global rule", Weight: 1}}
	if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: global, Atoms: []MemoryAtom{globalRule}}); err != nil { t.Fatal(err) }
	if err := store.Retract(ctx, projectA, "global:rule", "malicious project mutation", provA); !errors.Is(err, ErrUnauthorizedAccess) { t.Fatalf("project mutated global err=%v", err) }
	if _, err := store.GetAtom(ctx, global, "global:rule"); err != nil { t.Fatalf("global rule disappeared after denied mutation: %v", err) }
}

func seedFMEA010PromotionSources(t *testing.T, store *EpMemoryStore, project string) (Namespace, Namespace, []string, []string) {
	t.Helper()
	source, knowledge := fmea010ProjectScopes(t, project)
	p1 := fmea010Prov(source, project, "run-1", "sha256:independent-one", 0.95)
	p2 := fmea010Prov(source, project, "run-2", "sha256:independent-two", 0.96)
	a1 := fmea010Finding("source:one", knowledge, p1, 0.95)
	a2 := fmea010Finding("source:two", knowledge, p2, 0.96)
	if err := store.Commit(context.Background(), AdmissionBundle{SourceNamespace: source, Atoms: []MemoryAtom{a1, a2}}); err != nil { t.Fatal(err) }
	return source, knowledge, []string{a1.ID, a2.ID}, []string{p1.EvidenceDigest, p2.EvidenceDigest}
}

func fmea010PromotionRequest(source Namespace, sourceIDs, digests []string, candidateID string, confidence float64, description string) PromotionRequest {
	global := NewGlobalDesignNamespace()
	return PromotionRequest{
		SourceNamespace: source,
		TargetNamespace: global,
		SourceAtomIDs: sourceIDs,
		IndependentEvidenceDigests: digests,
		Candidate: MemoryAtom{
			ID: candidateID,
			Kind: NodeDesignRule,
			Namespace: global,
			Confidence: confidence,
			Data: DesignRuleAtom{RuleID: candidateID, Axis: "layout", Category: "spacing", Title: "General spacing rule", Description: description, Weight: 0.9, Version: "promoted.v1"},
			Tags: []string{"generalized", "promoted"},
		},
		VerifierID: "independent-promotion-verifier-v1",
		Rationale: "two independent project-local observations support a sanitized generalized rule",
	}
}

func TestFMEA010PromotionReplayAndRetractionPropagation(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	source, project, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-a")
	req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:promoted-spacing", 0.95, "Use consistent spacing intervals for repeated controls")
	op := sideeffect.Operation{RunID: "promotion-run", Activity: "promote_rule", Iteration: 1, Kind: "memory_promotion", PayloadDigest: PromotionPayloadDigest(req), RetryClass: sideeffect.RetryIdempotent}

	rec1, receipt1, err := store.Promote(ctx, op, req)
	if err != nil { t.Fatalf("Promote: %v", err) }
	rec2, receipt2, err := store.Promote(ctx, op, req)
	if err != nil { t.Fatalf("Promote replay: %v", err) }
	if !receipt1.Applied || receipt1.Reused || receipt2.Applied || !receipt2.Reused || rec1.PromotionID != rec2.PromotionID { t.Fatalf("promotion receipts=%#v %#v records=%#v %#v", receipt1, receipt2, rec1, rec2) }
	if rec1.DecisionDigest == "" || rec1.VerifierID == "" || len(rec1.SourceAtomIDs) != 2 { t.Fatalf("incomplete promotion provenance: %#v", rec1) }

	global := NewGlobalDesignNamespace()
	promoted, err := store.GetAtom(ctx, global, req.Candidate.ID)
	if err != nil { t.Fatalf("promoted global rule not readable: %v", err) }
	if promoted.Provenance.SourceNamespace != source.String() || promoted.Provenance.DecisionDigest != rec1.DecisionDigest || len(promoted.Provenance.SourceAtomIDs) != 2 { t.Fatalf("candidate provenance=%#v", promoted.Provenance) }
	if _, err := store.GetAtom(ctx, project, sourceIDs[0]); err != nil { t.Fatalf("project source missing: %v", err) }

	prov := fmea010Prov(source, "project-a", "retract-run", "sha256:retract", 1)
	if err := store.Retract(ctx, project, sourceIDs[0], "source invalidated", prov); err != nil { t.Fatalf("retract source: %v", err) }
	if _, err := store.GetAtom(ctx, global, req.Candidate.ID); !errors.Is(err, ErrAtomNotFound) { t.Fatalf("promoted claim stayed active after source retraction: %v", err) }
	updated, ok := store.PromotionRecord(rec1.PromotionID)
	if !ok || updated.Status != PromotionRevoked { t.Fatalf("promotion record after source retract=%#v ok=%v", updated, ok) }
	if stored := store.atoms[sourceIDs[0]]; stored == nil || stored.Status != StatusRetracted { t.Fatalf("private source history was deleted instead of retracted: %#v", stored) }
}

func TestFMEA010PoisonedPromotionCandidatesFailClosed(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		mutate func(*PromotionRequest, *EpMemoryStore, Namespace, Namespace)
	}{
		{name: "low-confidence", mutate: func(r *PromotionRequest, _ *EpMemoryStore, _, _ Namespace) { r.Candidate.Confidence = 0.4 }},
		{name: "private-marker", mutate: func(r *PromotionRequest, _ *EpMemoryStore, _, _ Namespace) { rule := r.Candidate.Data.(DesignRuleAtom); rule.Description = "secret credential from source project"; r.Candidate.Data = rule }},
		{name: "project-identifier", mutate: func(r *PromotionRequest, _ *EpMemoryStore, _, _ Namespace) { rule := r.Candidate.Data.(DesignRuleAtom); rule.Description = "project-poison internal spacing"; r.Candidate.Data = rule }},
		{name: "conflicted-source", mutate: func(r *PromotionRequest, store *EpMemoryStore, source, knowledge Namespace) {
			prov := fmea010Prov(source, "project-poison", "conflict", "sha256:conflict", 1)
			ce := MemoryAtom{ID: "ce:source", Kind: NodeCounterexample, Namespace: knowledge, Provenance: prov, Confidence: 1, Data: CounterexampleAtom{TargetEntityID: r.SourceAtomIDs[0], Reason: "contradiction", RefutingDigest: prov.EvidenceDigest}}
			if err := store.Commit(ctx, AdmissionBundle{SourceNamespace: source, Atoms: []MemoryAtom{ce}, Edges: []MemoryEdge{{FromID: ce.ID, ToID: r.SourceAtomIDs[0], Relation: RelRefutes, Weight: 1, Provenance: prov, CreatedAt: time.Now()}}}); err != nil { t.Fatal(err) }
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewEpMemoryStore()
			source, knowledge, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-poison")
			req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:poison-test", 0.95, "General rule supported by independent fixtures")
			tc.mutate(&req, store, source, knowledge)
			op := sideeffect.Operation{RunID: "poison-run", Activity: "promote", Iteration: 1, Kind: "memory_promotion", PayloadDigest: PromotionPayloadDigest(req), RetryClass: sideeffect.RetryIdempotent}
			if _, _, err := store.Promote(ctx, op, req); err == nil { t.Fatalf("poisoned promotion unexpectedly succeeded") }
			if result, err := store.Query(ctx, QueryRequest{Namespace: NewGlobalDesignNamespace()}); err != nil || result.Total != 0 { t.Fatalf("poisoned rule escaped into global: %#v err=%v", result, err) }
		})
	}
}

func TestFMEA010PromotionRollbackPreservesPrivateSources(t *testing.T) {
	ctx := context.Background()
	store := NewEpMemoryStore()
	source, project, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-rollback")
	req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:rollback", 0.95, "Generalized rollback test rule")
	promoteOp := sideeffect.Operation{RunID: "rollback-run", Activity: "promote", Kind: "memory_promotion", PayloadDigest: PromotionPayloadDigest(req), RetryClass: sideeffect.RetryIdempotent}
	rec, _, err := store.Promote(ctx, promoteOp, req)
	if err != nil { t.Fatal(err) }

	rb := PromotionRollbackRequest{PromotionID: rec.PromotionID, ReviewerID: "independent-reviewer", Reason: "shadow non-regression later failed"}
	rbOp := sideeffect.Operation{RunID: "rollback-run", Activity: "rollback", Kind: "memory_promotion_rollback", PayloadDigest: PromotionRollbackPayloadDigest(rb), RetryClass: sideeffect.RetryIdempotent}
	rbRec, first, err := store.RollbackPromotion(ctx, rbOp, rb)
	if err != nil { t.Fatal(err) }
	_, second, err := store.RollbackPromotion(ctx, rbOp, rb)
	if err != nil { t.Fatal(err) }
	if rbRec.Status != PromotionRolledBack || !first.Applied || first.Reused || second.Applied || !second.Reused { t.Fatalf("rollback state=%#v receipts=%#v %#v", rbRec, first, second) }
	if _, err := store.GetAtom(ctx, NewGlobalDesignNamespace(), req.Candidate.ID); !errors.Is(err, ErrAtomNotFound) { t.Fatalf("rolled back global candidate still active: %v", err) }
	for _, id := range sourceIDs {
		if _, err := store.GetAtom(ctx, project, id); err != nil { t.Fatalf("private source %s lost on rollback: %v", id, err) }
	}
}
