package memory

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

func TestAdmissionMapper_ValidateProvenance(t *testing.T) {
	mapper := NewAdmissionMapper(nil)

	validProv := ProvenanceRecord{
		RunID:          "run_123",
		EvidenceDigest: "sha256:abc456",
		Renderer:       "fastcdp",
		Tier:           fidelity.TierL2,
		Environment:    "chromium-128",
		Timestamp:      time.Now(),
		Outcome:        "CONFIRMED",
	}

	if err := mapper.ValidateProvenance(validProv); err != nil {
		t.Fatalf("expected valid provenance, got: %v", err)
	}

	// Missing RunID
	invalidProv := validProv
	invalidProv.RunID = ""
	if err := mapper.ValidateProvenance(invalidProv); err == nil {
		t.Fatalf("expected error for missing RunID")
	}

	// Missing EvidenceDigest
	invalidProv = validProv
	invalidProv.EvidenceDigest = ""
	if err := mapper.ValidateProvenance(invalidProv); err == nil {
		t.Fatalf("expected error for missing EvidenceDigest")
	}

	// Stale timestamp
	invalidProv = validProv
	invalidProv.Timestamp = time.Now().Add(-48 * time.Hour)
	if err := mapper.ValidateProvenance(invalidProv); err == nil {
		t.Fatalf("expected error for stale timestamp")
	}

	// Future timestamp
	invalidProv = validProv
	invalidProv.Timestamp = time.Now().Add(1 * time.Hour)
	if err := mapper.ValidateProvenance(invalidProv); err == nil {
		t.Fatalf("expected error for future timestamp")
	}
}

func TestAdmissionMapper_AdmitPacketAndCritique(t *testing.T) {
	mapper := NewAdmissionMapper(nil)
	ctx := context.Background()

	prov := ProvenanceRecord{
		RunID:          "run_456",
		EvidenceDigest: "sha256:digest789",
		Renderer:       "fastcdp",
		Tier:           fidelity.TierL2,
		Environment:    "blink-warm",
		Timestamp:      time.Now(),
		Outcome:        "CONFIRMED",
	}

	ns, err := NewProjectKnowledgeNamespace("proj_test")
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	req := AdmissionRequest{
		TargetNamespace: ns,
		Provenance:      prov,
		Confidence:      0.9,
		Tags:            []string{"ci", "test"},
	}

	// 1. Admit Packet
	packet := &evidence.Packet{RunID: "run_456"}
	packetBundle, err := mapper.AdmitPacket(ctx, packet, req)
	if err != nil {
		t.Fatalf("failed to admit packet: %v", err)
	}
	if len(packetBundle.Atoms) < 2 {
		t.Fatalf("expected at least 2 atoms from packet (env + artifact), got %d", len(packetBundle.Atoms))
	}
	if len(packetBundle.Edges) < 1 {
		t.Fatalf("expected at least 1 edge, got %d", len(packetBundle.Edges))
	}

	// 2. Admit Critique Pass
	pass := &design.CritiquePass{
		ID:            "pass_001",
		Level:         design.LevelPage,
		GroundedScore: 8.5,
		Duration:      150 * time.Millisecond,
		Findings: []design.Finding{
			{
				ID:             "f_contrast",
				Axis:           "accessibility",
				Category:       "contrast",
				Title:          "Low text contrast",
				Description:    "Text contrast ratio is 3.2:1 (required 4.5:1)",
				Severity:       evidence.SeverityCritical,
				Confidence:     0.95,
				HardConstraint: true,
			},
			{
				ID:          "f_low_conf",
				Axis:        "aesthetics",
				Category:    "spacing",
				Description: "Dubious spacing issue",
				Confidence:  0.2, // Below MinConfidence 0.5 -> should be filtered
			},
		},
		Hypotheses: []design.RepairHypothesis{
			{
				ID:              "h_contrast_fix",
				FindingIDs:      []string{"f_contrast"},
				Strategy:        "css_token_update",
				ProposedChanges: "--color-text: #111111;",
				ExpectedOutcome: "Contrast ratio improves to 7.1:1",
				Confidence:      0.9,
			},
		},
	}

	critiqueBundle, err := mapper.AdmitCritiquePass(ctx, pass, req)
	if err != nil {
		t.Fatalf("failed to admit critique pass: %v", err)
	}

	// Should contain eval atom, 1 high-conf finding atom, 1 repair pattern atom
	if len(critiqueBundle.Atoms) != 3 {
		t.Fatalf("expected 3 admitted atoms, got %d", len(critiqueBundle.Atoms))
	}
	// Edges should connect finding -> eval, and pattern -> finding
	if len(critiqueBundle.Edges) != 2 {
		t.Fatalf("expected 2 admitted edges, got %d", len(critiqueBundle.Edges))
	}

	// 3. Admit Repair Outcome
	successHypothesis := &pass.Hypotheses[0]
	succBundle, err := mapper.AdmitRepairOutcome(ctx, successHypothesis, true, req)
	if err != nil {
		t.Fatalf("failed to admit successful repair: %v", err)
	}
	if len(succBundle.Atoms) != 1 {
		t.Fatalf("expected 1 repair pattern atom for success, got %d", len(succBundle.Atoms))
	}

	failBundle, err := mapper.AdmitRepairOutcome(ctx, successHypothesis, false, req)
	if err != nil {
		t.Fatalf("failed to admit failed repair: %v", err)
	}
	// Failure should generate counterexample atom + refutes edge
	if len(failBundle.Atoms) != 2 {
		t.Fatalf("expected 2 atoms (pattern + counterexample) for failure, got %d", len(failBundle.Atoms))
	}
	if len(failBundle.Edges) != 1 || failBundle.Edges[0].Relation != RelRefutes {
		t.Fatalf("expected 1 refutes edge for repair failure")
	}
}
