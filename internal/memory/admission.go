package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var (
	ErrInvalidProvenance = errors.New("invalid memory provenance")
	ErrStaleTimestamp    = errors.New("timestamp is stale or invalid")
	ErrInvalidAtomData   = errors.New("invalid memory atom data")
	ErrLowConfidence     = errors.New("confidence score below admission threshold")
)

// AdmissionConfig defines gating thresholds for memory admission.
type AdmissionConfig struct {
	MinConfidence    float64
	MaxTimestampAge  time.Duration
	AllowFutureSkew  time.Duration
	DefaultNamespace Namespace
}

// DefaultAdmissionConfig returns standard admission policy settings.
func DefaultAdmissionConfig() AdmissionConfig {
	return AdmissionConfig{
		MinConfidence:   0.5,
		MaxTimestampAge: 24 * time.Hour,
		AllowFutureSkew: 5 * time.Minute,
	}
}

// AdmissionRequest contains metadata and parameters for an admission transaction.
type AdmissionRequest struct {
	TargetNamespace Namespace
	Provenance      ProvenanceRecord
	Confidence      float64
	Tags            []string
}

// AdmissionBundle contains validated atoms and edges ready for transactional commit.
type AdmissionBundle struct {
	Atoms []MemoryAtom `json:"atoms"`
	Edges []MemoryEdge `json:"edges"`
}

// AdmissionMapper validates raw evidence and projections, turning them into canonical memory atoms.
type AdmissionMapper struct {
	config AdmissionConfig
}

// NewAdmissionMapper creates a new AdmissionMapper with given or default configuration.
func NewAdmissionMapper(cfg *AdmissionConfig) *AdmissionMapper {
	if cfg == nil {
		def := DefaultAdmissionConfig()
		cfg = &def
	}
	return &AdmissionMapper{config: *cfg}
}

// ValidateProvenance ensures the provenance record satisfies canonical admission requirements.
func (m *AdmissionMapper) ValidateProvenance(p ProvenanceRecord) error {
	if strings.TrimSpace(p.RunID) == "" {
		return fmt.Errorf("%w: missing run_id", ErrInvalidProvenance)
	}
	if strings.TrimSpace(p.EvidenceDigest) == "" {
		return fmt.Errorf("%w: missing evidence_digest", ErrInvalidProvenance)
	}
	if strings.TrimSpace(p.Renderer) == "" {
		return fmt.Errorf("%w: missing renderer", ErrInvalidProvenance)
	}
	if p.Timestamp.IsZero() {
		return fmt.Errorf("%w: zero timestamp", ErrStaleTimestamp)
	}

	now := time.Now()
	if p.Timestamp.After(now.Add(m.config.AllowFutureSkew)) {
		return fmt.Errorf("%w: timestamp is in the future", ErrStaleTimestamp)
	}
	if m.config.MaxTimestampAge > 0 && now.Sub(p.Timestamp) > m.config.MaxTimestampAge {
		return fmt.Errorf("%w: timestamp exceeds max age", ErrStaleTimestamp)
	}

	return nil
}

// AdmitPacket extracts environment and artifact atoms from an evidence packet.
func (m *AdmissionMapper) AdmitPacket(ctx context.Context, packet *evidence.Packet, req AdmissionRequest) (*AdmissionBundle, error) {
	if packet == nil {
		return nil, fmt.Errorf("%w: packet is nil", ErrInvalidAtomData)
	}
	if err := m.ValidateProvenance(req.Provenance); err != nil {
		return nil, err
	}

	ns := req.TargetNamespace
	if ns.raw == "" {
		ns = NewGlobalDesignNamespace()
	}

	var atoms []MemoryAtom
	var edges []MemoryEdge
	now := time.Now()

	// 1. RenderEnvironment Atom
	envID := fmt.Sprintf("env_%s_%s", req.Provenance.Renderer, hashString(req.Provenance.Environment))
	envAtom := MemoryAtom{
		ID:         envID,
		Kind:       NodeRenderEnvironment,
		Namespace:  ns,
		Provenance: req.Provenance,
		Confidence: 1.0,
		Data: RenderEnvironmentAtom{
			Renderer:      req.Provenance.Renderer,
			BrowserFamily: req.Provenance.Renderer,
			ViewportW:     1280,
			ViewportH:     800,
			DeviceScale:   1.0,
			Theme:         "default",
		},
		Tags:      append(req.Tags, "environment", req.Provenance.Renderer),
		CreatedAt: now,
		UpdatedAt: now,
	}
	atoms = append(atoms, envAtom)

	// 2. Evidence Artifact Atom (Digest-based)
	artID := fmt.Sprintf("artifact_%s", req.Provenance.EvidenceDigest)
	artAtom := MemoryAtom{
		ID:         artID,
		Kind:       NodeEvidenceArtifact,
		Namespace:  ns,
		Provenance: req.Provenance,
		Confidence: 1.0,
		Data: EvidenceArtifactAtom{
			Kind:      "evidence_packet",
			Digest:    req.Provenance.EvidenceDigest,
			SizeBytes: int64(len(packet.RunID)),
		},
		Tags:      append(req.Tags, "artifact", req.Provenance.Renderer),
		CreatedAt: now,
		UpdatedAt: now,
	}
	atoms = append(atoms, artAtom)

	// Edge: Artifact observed on Environment
	edges = append(edges, MemoryEdge{
		FromID:     artID,
		ToID:       envID,
		Relation:   RelObservedOn,
		Weight:     1.0,
		Provenance: req.Provenance,
		CreatedAt:  now,
	})

	return &AdmissionBundle{Atoms: atoms, Edges: edges}, nil
}

// AdmitCritiquePass maps design findings and hypotheses into candidate memory atoms and edges.
func (m *AdmissionMapper) AdmitCritiquePass(ctx context.Context, pass *design.CritiquePass, req AdmissionRequest) (*AdmissionBundle, error) {
	if pass == nil {
		return nil, fmt.Errorf("%w: critique pass is nil", ErrInvalidAtomData)
	}
	if err := m.ValidateProvenance(req.Provenance); err != nil {
		return nil, err
	}

	ns := req.TargetNamespace
	if ns.raw == "" {
		ns = NewGlobalDesignNamespace()
	}

	var atoms []MemoryAtom
	var edges []MemoryEdge
	now := time.Now()

	// Evaluation result atom for the pass
	evalID := fmt.Sprintf("eval_%s_%s", req.Provenance.RunID, pass.ID)
	evalAtom := MemoryAtom{
		ID:         evalID,
		Kind:       NodeEvaluationResult,
		Namespace:  ns,
		Provenance: req.Provenance,
		Confidence: 1.0,
		Data: EvaluationResultAtom{
			RunID:          req.Provenance.RunID,
			Score:          pass.GroundedScore,
			Passed:         pass.HardViolations == 0,
			HardViolations: pass.HardViolations,
			DurationMS:     pass.Duration.Milliseconds(),
		},
		Tags:      append(req.Tags, "critique_pass", string(pass.Level)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	atoms = append(atoms, evalAtom)

	// Map findings
	for _, f := range pass.Findings {
		conf := f.Confidence
		if conf == 0 {
			conf = req.Confidence
		}
		if conf < m.config.MinConfidence {
			continue // filter out low confidence findings
		}

		findingID := fmt.Sprintf("finding_%s_%s", req.Provenance.RunID, f.ID)
		fAtom := MemoryAtom{
			ID:         findingID,
			Kind:       NodeDesignFinding,
			Namespace:  ns,
			Provenance: req.Provenance,
			Confidence: conf,
			Data: DesignFindingAtom{
				FindingID:      f.ID,
				Axis:           f.Axis,
				Category:       f.Category,
				RuleID:         f.RuleID,
				Title:          f.Title,
				Description:    f.Description,
				Severity:       f.Severity,
				HardConstraint: f.HardConstraint,
				RegionID:       f.RegionID,
				ElementIDs:     f.ElementIDs,
				Suggestion:     f.Suggestion,
			},
			Tags:      append(req.Tags, "finding", f.Axis, f.Category),
			CreatedAt: now,
			UpdatedAt: now,
		}
		atoms = append(atoms, fAtom)

		// Edge: Finding derived from Evaluation
		edges = append(edges, MemoryEdge{
			FromID:     findingID,
			ToID:       evalID,
			Relation:   RelDerivedFrom,
			Weight:     conf,
			Provenance: req.Provenance,
			CreatedAt:  now,
		})

		// If linked to rule, edge: Finding counterexample or evidence for rule
		if f.RuleID != "" {
			ruleNodeID := fmt.Sprintf("rule_%s", f.RuleID)
			edges = append(edges, MemoryEdge{
				FromID:     findingID,
				ToID:       ruleNodeID,
				Relation:   RelObservedOn,
				Weight:     1.0,
				Provenance: req.Provenance,
				CreatedAt:  now,
			})
		}
	}

	// Map hypotheses into candidate repair patterns
	for _, h := range pass.Hypotheses {
		if h.Confidence < m.config.MinConfidence {
			continue
		}
		patternID := fmt.Sprintf("pattern_%s_%s", req.Provenance.RunID, h.ID)
		pAtom := MemoryAtom{
			ID:         patternID,
			Kind:       NodeRepairPattern,
			Namespace:  ns,
			Provenance: req.Provenance,
			Confidence: h.Confidence,
			Data: RepairPatternAtom{
				PatternID:       h.ID,
				Strategy:        h.Strategy,
				TargetFiles:     h.TargetFiles,
				PatchSnippet:    h.ProposedChanges,
				ExpectedOutcome: h.ExpectedOutcome,
				SuccessCount:    0,
				FailureCount:    0,
				SuccessRate:     0.0,
			},
			Tags:      append(req.Tags, "repair_pattern", h.Strategy),
			CreatedAt: now,
			UpdatedAt: now,
		}
		atoms = append(atoms, pAtom)

		// Link pattern to findings it aims to repair
		for _, fID := range h.FindingIDs {
			targetFindingID := fmt.Sprintf("finding_%s_%s", req.Provenance.RunID, fID)
			edges = append(edges, MemoryEdge{
				FromID:     patternID,
				ToID:       targetFindingID,
				Relation:   RelRepairedBy,
				Weight:     h.Confidence,
				Provenance: req.Provenance,
				CreatedAt:  now,
			})
		}
	}

	return &AdmissionBundle{Atoms: atoms, Edges: edges}, nil
}

// AdmitRepairOutcome records the success or failure of a repair pattern.
func (m *AdmissionMapper) AdmitRepairOutcome(ctx context.Context, hypothesis *design.RepairHypothesis, succeeded bool, req AdmissionRequest) (*AdmissionBundle, error) {
	if hypothesis == nil {
		return nil, fmt.Errorf("%w: hypothesis is nil", ErrInvalidAtomData)
	}
	if err := m.ValidateProvenance(req.Provenance); err != nil {
		return nil, err
	}

	ns := req.TargetNamespace
	if ns.raw == "" {
		ns = NewGlobalDesignNamespace()
	}

	var atoms []MemoryAtom
	var edges []MemoryEdge
	now := time.Now()

	patternID := fmt.Sprintf("pattern_%s", hashString(hypothesis.ProposedChanges))
	succCount := 0
	failCount := 0
	rate := 0.0
	outcomeTag := "failed"
	if succeeded {
		succCount = 1
		rate = 1.0
		outcomeTag = "success"
	} else {
		failCount = 1
	}

	atom := MemoryAtom{
		ID:         patternID,
		Kind:       NodeRepairPattern,
		Namespace:  ns,
		Provenance: req.Provenance,
		Confidence: hypothesis.Confidence,
		Data: RepairPatternAtom{
			PatternID:       hypothesis.ID,
			Strategy:        hypothesis.Strategy,
			TargetFiles:     hypothesis.TargetFiles,
			PatchSnippet:    hypothesis.ProposedChanges,
			ExpectedOutcome: hypothesis.ExpectedOutcome,
			SuccessCount:    succCount,
			FailureCount:    failCount,
			SuccessRate:     rate,
		},
		Tags:      append(req.Tags, "repair_pattern", hypothesis.Strategy, outcomeTag),
		CreatedAt: now,
		UpdatedAt: now,
	}
	atoms = append(atoms, atom)

	if !succeeded {
		// Create counterexample atom
		ceID := fmt.Sprintf("ce_%s", patternID)
		ceAtom := MemoryAtom{
			ID:         ceID,
			Kind:       NodeCounterexample,
			Namespace:  ns,
			Provenance: req.Provenance,
			Confidence: 1.0,
			Data: CounterexampleAtom{
				TargetEntityID: patternID,
				Reason:         "Repair failed re-verification or caused regression",
				RefutingDigest: req.Provenance.EvidenceDigest,
				Observation:    hypothesis.ExpectedOutcome,
			},
			Tags:      append(req.Tags, "counterexample", "refutation"),
			CreatedAt: now,
			UpdatedAt: now,
		}
		atoms = append(atoms, ceAtom)

		edges = append(edges, MemoryEdge{
			FromID:     ceID,
			ToID:       patternID,
			Relation:   RelRefutes,
			Weight:     1.0,
			Provenance: req.Provenance,
			CreatedAt:  now,
		})
	}

	return &AdmissionBundle{Atoms: atoms, Edges: edges}, nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}
