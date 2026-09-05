package fastcdp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestFMEA003StaleRevisionCannotReleaseEvidence(t *testing.T) {
	gate := NewEpochGate()
	if !gate.AdvanceToken(RenderToken{Epoch: 10, Revision: "rev-old"}) {
		t.Fatal("expected initial token to advance")
	}
	_, err := gate.WaitAfterRevision(context.Background(), 9, "rev-new")
	if !errors.Is(err, ErrRevisionMismatch) {
		t.Fatalf("err = %v, want ErrRevisionMismatch", err)
	}
	var mismatch *RevisionMismatchError
	if !errors.As(err, &mismatch) || mismatch.Observed.Epoch != 10 || mismatch.Observed.Revision != "rev-old" {
		t.Fatalf("mismatch = %#v", mismatch)
	}
}

func TestFMEA003NewerEpochWithWrongRevisionFailsClosed(t *testing.T) {
	gate := NewEpochGate()
	gate.AdvanceToken(RenderToken{Epoch: 20, Revision: "rev-base"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := gate.WaitAfterRevision(ctx, 20, "rev-requested")
		result <- err
	}()

	if !gate.AdvanceToken(RenderToken{Epoch: 21, Revision: "rev-other"}) {
		t.Fatal("expected newer token to advance")
	}
	if err := <-result; !errors.Is(err, ErrRevisionMismatch) {
		t.Fatalf("err = %v, want ErrRevisionMismatch", err)
	}
}

func TestFMEA003MatchingRevisionReleasesWaiter(t *testing.T) {
	gate := NewEpochGate()
	gate.AdvanceToken(RenderToken{Epoch: 30, Revision: "rev-base"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result := make(chan struct {
		token RenderToken
		err   error
	}, 1)
	go func() {
		token, err := gate.WaitAfterRevision(ctx, 30, "rev-31")
		result <- struct {
			token RenderToken
			err   error
		}{token: token, err: err}
	}()

	gate.AdvanceToken(RenderToken{Epoch: 31, Revision: "rev-31"})
	got := <-result
	if got.err != nil {
		t.Fatalf("WaitAfterRevision: %v", got.err)
	}
	if got.token != (RenderToken{Epoch: 31, Revision: "rev-31"}) {
		t.Fatalf("token = %#v", got.token)
	}
}

func TestFMEA003LegacyRevisionlessSignalCannotSatisfyBoundWaiter(t *testing.T) {
	gate := NewEpochGate()
	gate.Advance(40)
	_, err := gate.WaitAfterRevision(context.Background(), 39, "rev-required")
	if !errors.Is(err, ErrRevisionMismatch) {
		t.Fatalf("err = %v, want ErrRevisionMismatch", err)
	}
}

func TestFMEA003PacketCarriesExpectedAndObservedRevision(t *testing.T) {
	packet := ToPacket(CollectedEvidence{Epoch: 50, Revision: "rev-50"}, PacketOptions{
		ExpectedRevision: "rev-50",
		Viewport: evidence.Viewport{Width: 320, Height: 200},
	})
	if packet.Freshness == nil {
		t.Fatal("expected freshness provenance")
	}
	if packet.Freshness.Epoch != 50 || packet.Freshness.ExpectedRevision != "rev-50" || packet.Freshness.ObservedRevision != "rev-50" {
		t.Fatalf("freshness = %#v", packet.Freshness)
	}
	if !packet.Freshness.Matches() {
		t.Fatal("matching freshness must attest")
	}
}

func TestFMEA003BridgePayloadSupportsRevisionAndLegacySignals(t *testing.T) {
	token, err := parseRenderSignalPayload(`{"epoch":61,"revision":"build-a"}`)
	if err != nil || token != (RenderToken{Epoch: 61, Revision: "build-a"}) {
		t.Fatalf("revision payload: token=%#v err=%v", token, err)
	}
	legacy, err := parseRenderSignalPayload("62")
	if err != nil || legacy != (RenderToken{Epoch: 62}) {
		t.Fatalf("legacy payload: token=%#v err=%v", legacy, err)
	}
}
