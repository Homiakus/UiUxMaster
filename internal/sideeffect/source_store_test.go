package sideeffect

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestFMEA007SourceCASReplayAfterRestartIsExactlyOnce(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "source-state.json")
	initial := SourceState{HTML: "<main>old</main>", CSS: "main{color:black}"}
	desired := SourceState{HTML: "<main>new</main>", CSS: "main{color:green}"}
	store, err := NewFileSourceStore(path, initial)
	if err != nil { t.Fatal(err) }
	op := Operation{RunID: "run-7", Activity: "repair_commit", Iteration: 1, Kind: "source_repair", PayloadDigest: desired.Digest(), RetryClass: RetryIdempotent}
	first, err := store.CompareAndSwap(ctx, op, initial.Digest(), desired)
	if err != nil { t.Fatal(err) }
	if !first.Applied || first.Reused || first.LogicalID == "" { t.Fatalf("first receipt=%#v", first) }

	restarted, err := NewFileSourceStore(path, SourceState{})
	if err != nil { t.Fatal(err) }
	second, err := restarted.CompareAndSwap(ctx, op, initial.Digest(), desired)
	if err != nil { t.Fatal(err) }
	if second.Applied || !second.Reused || second.OperationID != first.OperationID || second.LogicalID != first.LogicalID || second.ResultDigest != first.ResultDigest {
		t.Fatalf("replay receipt=%#v first=%#v", second, first)
	}
	if got := restarted.Current(ctx); got != desired { t.Fatalf("state=%#v want=%#v", got, desired) }
}

func TestFMEA007SourceCASRejectsConcurrentRevision(t *testing.T) {
	ctx := context.Background()
	initial := SourceState{HTML: "old"}
	store := NewMemorySourceStore(initial)
	other := SourceState{HTML: "other"}
	opOther := Operation{RunID: "other", Activity: "repair_commit", Kind: "source_repair", PayloadDigest: other.Digest(), RetryClass: RetryIdempotent}
	if _, err := store.CompareAndSwap(ctx, opOther, initial.Digest(), other); err != nil { t.Fatal(err) }
	desired := SourceState{HTML: "mine"}
	op := Operation{RunID: "mine", Activity: "repair_commit", Kind: "source_repair", PayloadDigest: desired.Digest(), RetryClass: RetryIdempotent}
	_, err := store.CompareAndSwap(ctx, op, initial.Digest(), desired)
	if !errors.Is(err, ErrCASConflict) { t.Fatalf("err=%v want ErrCASConflict", err) }
}

func TestFMEA007LogicalOperationRejectsPayloadMutation(t *testing.T) {
	store := NewMemorySourceStore(SourceState{HTML: "a"})
	ctx := context.Background()
	desired := SourceState{HTML: "b"}
	op := Operation{RunID: "run", Activity: "repair", Iteration: 1, Kind: "source_repair", PayloadDigest: desired.Digest(), RetryClass: RetryIdempotent}
	if _, err := store.CompareAndSwap(ctx, op, SourceState{HTML: "a"}.Digest(), desired); err != nil { t.Fatal(err) }
	mutated := SourceState{HTML: "c"}
	mutatedOp := op
	mutatedOp.PayloadDigest = mutated.Digest()
	_, err := store.CompareAndSwap(ctx, mutatedOp, SourceState{HTML: "a"}.Digest(), mutated)
	if !errors.Is(err, ErrOperationConflict) { t.Fatalf("err=%v want ErrOperationConflict", err) }
}
