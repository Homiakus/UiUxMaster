package engine_test

import (
	"errors"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

func TestEvidenceTierGuardRejectsL2ForTruthPath(t *testing.T) {
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierTruthPath}}
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}
	if err := engine.ValidateCollectedEvidence(plan, packet); !errors.Is(err, engine.ErrInsufficientEvidenceTier) {
		t.Fatalf("error = %v", err)
	}
}

func TestEvidenceTierGuardAcceptsL3ForTruthPath(t *testing.T) {
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierTruthPath}}
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L3", Name: "playwright-chromium"}}
	if err := engine.ValidateCollectedEvidence(plan, packet); err != nil {
		t.Fatalf("error = %v", err)
	}
}

func TestEvidenceTierGuardAllowsUpwardL1ToL2(t *testing.T) {
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierFastRender}}
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}
	if err := engine.ValidateCollectedEvidence(plan, packet); err != nil {
		t.Fatalf("upward escalation rejected: %v", err)
	}
}

func TestEvidenceTierGuardSemanticNeedsAtLeastBrowserCollection(t *testing.T) {
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierSemantic}}
	if err := engine.ValidateCollectedEvidence(plan, evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2"}}); err != nil {
		t.Fatalf("L2 semantic input rejected: %v", err)
	}
	if err := engine.ValidateCollectedEvidence(plan, evidence.Packet{Renderer: evidence.RendererRef{Tier: "L1"}}); !errors.Is(err, engine.ErrInsufficientEvidenceTier) {
		t.Fatalf("L1 semantic input error = %v", err)
	}
}

func TestEvidenceTierGuardRejectsUnknownPacketTier(t *testing.T) {
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierTruthPath}}
	packet := evidence.Packet{Renderer: evidence.RendererRef{Tier: "unknown-tier"}}
	if err := engine.ValidateCollectedEvidence(plan, packet); !errors.Is(err, engine.ErrUnknownEvidenceTier) {
		t.Fatalf("error = %v", err)
	}
}
