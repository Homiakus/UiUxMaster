package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

func runFMEA010Promotion(t *testing.T, store *EpMemoryStore, req PromotionRequest) error {
	t.Helper()
	op := sideeffect.Operation{RunID: "independence-run", Activity: "promote", Iteration: 1, Kind: "memory_promotion", PayloadDigest: PromotionPayloadDigest(req), RetryClass: sideeffect.RetryIdempotent}
	_, _, err := store.Promote(context.Background(), op, req)
	return err
}

func TestFMEA010PromotionRejectsFakeExtraEvidenceDigest(t *testing.T) {
	store := NewEpMemoryStore()
	source, _, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-independent")
	// Make both source atoms actually depend on the same evidence, then append a
	// fake second digest. The caller-supplied list must not manufacture independence.
	store.atoms[sourceIDs[1]].Atom.Provenance.EvidenceDigest = digests[0]
	req := fmea010PromotionRequest(source, sourceIDs, []string{digests[0], "sha256:fake-second"}, "rule:fake-evidence", 0.95, "Generalized rule")
	if err := runFMEA010Promotion(t, store, req); err == nil { t.Fatalf("promotion succeeded with fabricated independent digest") }
}

func TestFMEA010PromotionRejectsSameRunTwice(t *testing.T) {
	store := NewEpMemoryStore()
	source, _, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-same-run")
	store.atoms[sourceIDs[1]].Atom.Provenance.RunID = store.atoms[sourceIDs[0]].Atom.Provenance.RunID
	req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:same-run", 0.95, "Generalized rule")
	if err := runFMEA010Promotion(t, store, req); err == nil { t.Fatalf("promotion succeeded with two atoms from one run") }
}

func TestFMEA010PromotionRejectsStaleSourceProvenance(t *testing.T) {
	store := NewEpMemoryStore()
	source, _, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-stale")
	store.atoms[sourceIDs[0]].Atom.Provenance.Timestamp = time.Now().Add(-PromotionMaxSourceAge - time.Hour)
	req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:stale", 0.95, "Generalized rule")
	if err := runFMEA010Promotion(t, store, req); err == nil { t.Fatalf("promotion succeeded with stale source provenance") }
}

func TestFMEA010PromotionRecordCarriesDistinctRunLineage(t *testing.T) {
	store := NewEpMemoryStore()
	source, _, sourceIDs, digests := seedFMEA010PromotionSources(t, store, "project-lineage")
	req := fmea010PromotionRequest(source, sourceIDs, digests, "rule:lineage", 0.95, "Generalized rule")
	op := sideeffect.Operation{RunID: "lineage-run", Activity: "promote", Kind: "memory_promotion", PayloadDigest: PromotionPayloadDigest(req), RetryClass: sideeffect.RetryIdempotent}
	rec, _, err := store.Promote(context.Background(), op, req)
	if err != nil { t.Fatal(err) }
	if len(rec.SourceRunIDs) != 2 || rec.SourceRunIDs[0] == rec.SourceRunIDs[1] { t.Fatalf("source run lineage=%v", rec.SourceRunIDs) }
	if len(rec.IndependentEvidenceDigests) != 2 { t.Fatalf("evidence lineage=%v", rec.IndependentEvidenceDigests) }
}
