package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	// ErrInsufficientEvidenceTier means the runtime returned evidence weaker than
	// the route selected by policy. Such a packet must never reach the verifier.
	ErrInsufficientEvidenceTier = errors.New("engine: collected evidence below required tier")
	// ErrUnknownEvidenceTier means either the planned collector route or the
	// packet's runtime tier cannot be mapped to the canonical evidence ladder.
	ErrUnknownEvidenceTier = errors.New("engine: unknown evidence tier")
)

// EvidenceStrength is the monotonic collector-side evidence ladder. L4 semantic
// judgement is a post-collection stage, so a TierSemantic plan requires at least
// L2 browser evidence at the collection boundary.
type EvidenceStrength uint8

const (
	StrengthStatic EvidenceStrength = iota
	StrengthFastRender
	StrengthFastBrowser
	StrengthTruthPath
	StrengthSemantic
)

// MinimumCollectionStrength returns the weakest packet that is legal for a
// policy-selected route. Higher-strength evidence is legal upward escalation.
func MinimumCollectionStrength(tier EvidenceTier) (EvidenceStrength, error) {
	switch tier {
	case TierStatic:
		return StrengthStatic, nil
	case TierFastRender:
		return StrengthFastRender, nil
	case TierFastBrowser:
		return StrengthFastBrowser, nil
	case TierTruthPath:
		return StrengthTruthPath, nil
	case TierSemantic:
		return StrengthFastBrowser, nil
	default:
		return 0, fmt.Errorf("%w: planned route %q", ErrUnknownEvidenceTier, tier)
	}
}

// PacketEvidenceStrength normalizes runtime provenance. Runtime adapters use
// short physical tiers (L2/L3), while routing uses descriptive tier names such
// as L2_fastbrowser/L3_truthpath; both map to the same monotonic strength.
func PacketEvidenceStrength(tier string) (EvidenceStrength, error) {
	normalized := strings.ToLower(strings.TrimSpace(tier))
	switch normalized {
	case "l0", "l0_static", "static":
		return StrengthStatic, nil
	case "l1", "l1_fastrender", "fastrender":
		return StrengthFastRender, nil
	case "l2", "l2_fastbrowser", "fastbrowser":
		return StrengthFastBrowser, nil
	case "l3", "l3_truthpath", "truthpath":
		return StrengthTruthPath, nil
	case "l4", "l4_semantic", "semantic":
		return StrengthSemantic, nil
	default:
		return 0, fmt.Errorf("%w: packet renderer tier %q", ErrUnknownEvidenceTier, tier)
	}
}

// ValidateCollectedEvidence prevents a weaker runtime packet from being used
// as proof for a stronger policy route. It is intentionally protocol-neutral
// and is called both by the canonical Pipeline and by the standard dispatcher.
func ValidateCollectedEvidence(plan ValidationPlan, packet evidence.Packet) error {
	minimum, err := MinimumCollectionStrength(plan.Route.Tier)
	if err != nil {
		return err
	}
	actual, err := PacketEvidenceStrength(packet.Renderer.Tier)
	if err != nil {
		return err
	}
	if actual < minimum {
		return fmt.Errorf(
			"%w: route=%s minimum=%d actual=%d renderer=%q",
			ErrInsufficientEvidenceTier,
			plan.Route.Tier,
			minimum,
			actual,
			packet.Renderer.Name,
		)
	}
	return nil
}
