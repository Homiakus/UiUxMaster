package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

func TestFMEA001TruthPathUnavailableDoesNotDowngradeToL2(t *testing.T) {
	l2 := &mockL2Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}}
	d := New(Config{L2Collector: l2})
	plan := engine.ValidationPlan{
		Route: engine.RouteDecision{Tier: engine.TierTruthPath},
		EvidencePlan: evidenceplan.Plan{BrowserTruth: true},
	}

	packet, err := d.Collect(context.Background(), engine.ValidationRequest{RunID: "fmea001-no-l3"}, plan)
	if !errors.Is(err, ErrCollectorUnavailable) {
		t.Fatalf("error = %v, want ErrCollectorUnavailable", err)
	}
	if l2.called {
		t.Fatal("L2 collector was called for a required TruthPath route")
	}
	if packet.Renderer.Tier != "" {
		t.Fatalf("unavailable TruthPath returned usable packet: %#v", packet.Renderer)
	}
}

func TestFMEA001TruthPathRejectsWeakerPacketFromConfiguredL3(t *testing.T) {
	l2 := &mockL2Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}}
	l3 := &mockL3Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "misconfigured-truthpath"}}}
	d := New(Config{L2Collector: l2, L3Collector: l3})
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierTruthPath}}

	packet, err := d.Collect(context.Background(), engine.ValidationRequest{RunID: "fmea001-weak-l3"}, plan)
	if !errors.Is(err, engine.ErrInsufficientEvidenceTier) {
		t.Fatalf("error = %v, want ErrInsufficientEvidenceTier", err)
	}
	if !l3.called {
		t.Fatal("configured L3 collector was not called")
	}
	if l2.called {
		t.Fatal("dispatcher attempted L2 fallback after insufficient L3 evidence")
	}
	if packet.Renderer.Tier != "" {
		t.Fatalf("insufficient evidence escaped dispatcher: %#v", packet.Renderer)
	}
}

func TestFMEA001TruthPathAcceptsAttestedL3Packet(t *testing.T) {
	l3 := &mockL3Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L3", Name: "playwright-chromium"}}}
	d := New(Config{L3Collector: l3})
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierTruthPath}}

	packet, err := d.Collect(context.Background(), engine.ValidationRequest{RunID: "fmea001-valid-l3"}, plan)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if packet.Renderer.Tier != "L3" {
		t.Fatalf("tier = %q, want L3", packet.Renderer.Tier)
	}
}

func TestFMEA001UnknownRouteDoesNotDefaultToL2(t *testing.T) {
	l2 := &mockL2Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}}
	d := New(Config{L2Collector: l2})
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.EvidenceTier("L9_unknown")}}

	_, err := d.Collect(context.Background(), engine.ValidationRequest{RunID: "fmea001-unknown-route"}, plan)
	if !errors.Is(err, ErrInvalidRoute) {
		t.Fatalf("error = %v, want ErrInvalidRoute", err)
	}
	if l2.called {
		t.Fatal("unknown route silently used L2")
	}
}

func TestFMEA001UpwardL1ToL2EscalationRemainsLegal(t *testing.T) {
	l2 := &mockL2Collector{packet: evidence.Packet{Renderer: evidence.RendererRef{Tier: "L2", Name: "fastcdp"}}}
	d := New(Config{L2Collector: l2, EscalateL1ToL2OnUnsupported: true})
	plan := engine.ValidationPlan{Route: engine.RouteDecision{Tier: engine.TierFastRender}}

	packet, err := d.Collect(context.Background(), engine.ValidationRequest{RunID: "fmea001-upward"}, plan)
	if err != nil {
		t.Fatalf("upward escalation: %v", err)
	}
	if !l2.called || packet.Renderer.Tier != "L2" {
		t.Fatalf("upward escalation did not produce L2 evidence: called=%v packet=%#v", l2.called, packet.Renderer)
	}
}
