package engine

import (
	"errors"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestFMEA003RevisionAttestationAcceptsMatchingFastCDPPacket(t *testing.T) {
	req := ValidationRequest{SourceDigest: "rev-good"}
	packet := evidence.Packet{
		Epoch:    7,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"},
		Freshness: &evidence.RenderFreshness{
			Epoch:            7,
			ExpectedRevision: "rev-good",
			ObservedRevision: "rev-good",
		},
	}
	if err := ValidateRevisionAttestation(req, packet); err != nil {
		t.Fatalf("ValidateRevisionAttestation: %v", err)
	}
}

func TestFMEA003RevisionAttestationRejectsWrongObservedRevision(t *testing.T) {
	req := ValidationRequest{SourceDigest: "rev-good"}
	packet := evidence.Packet{
		Epoch:    8,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"},
		Freshness: &evidence.RenderFreshness{
			Epoch:            8,
			ExpectedRevision: "rev-good",
			ObservedRevision: "rev-stale",
		},
	}
	if err := ValidateRevisionAttestation(req, packet); !errors.Is(err, ErrRevisionAttestation) {
		t.Fatalf("err = %v, want ErrRevisionAttestation", err)
	}
}

func TestFMEA003RevisionAttestationRejectsPacketEpochMismatch(t *testing.T) {
	packet := evidence.Packet{
		Epoch:    9,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"},
		Freshness: &evidence.RenderFreshness{
			Epoch:            10,
			ExpectedRevision: "rev",
			ObservedRevision: "rev",
		},
	}
	if err := ValidateRevisionAttestation(ValidationRequest{SourceDigest: "rev"}, packet); !errors.Is(err, ErrRevisionAttestation) {
		t.Fatalf("err = %v, want ErrRevisionAttestation", err)
	}
}

func TestFMEA003RevisionAttestationRejectsMissingFastCDPProvenance(t *testing.T) {
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}
	if err := ValidateRevisionAttestation(ValidationRequest{SourceDigest: "rev-required"}, packet); !errors.Is(err, ErrRevisionAttestation) {
		t.Fatalf("err = %v, want ErrRevisionAttestation", err)
	}
}

func TestFMEA003RevisionlessNonFastCDPPacketRemainsOutsideContract(t *testing.T) {
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L3", Name: "playwright-chromium"}}
	if err := ValidateRevisionAttestation(ValidationRequest{SourceDigest: "rev-used-by-other-system"}, packet); err != nil {
		t.Fatalf("unattested non-FastCDP packet should remain governed by its own collector contract: %v", err)
	}
}
