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

type AdmissionConfig struct {
	MinConfidence   float64
	MaxTimestampAge time.Duration
	AllowFutureSkew time.Duration
}

func DefaultAdmissionConfig() AdmissionConfig {
	return AdmissionConfig{MinConfidence: 0.5, MaxTimestampAge: 24 * time.Hour, AllowFutureSkew: 5 * time.Minute}
}

// AdmissionRequest describes a non-promotion write. SourceNamespace is where the
// evidence/knowledge originated; TargetNamespace is where the derived atoms will
// live. A caller may not widen visibility by selecting a broader target.
type AdmissionRequest struct {
	SourceNamespace Namespace
	TargetNamespace Namespace
	Provenance      ProvenanceRecord
	Confidence      float64
	Tags            []string
}

// AdmissionBundle is the store-level authorization envelope. SourceNamespace is
// mandatory even for direct Commit callers so mapper validation cannot be bypassed.
type AdmissionBundle struct {
	SourceNamespace Namespace    `json:"source_namespace"`
	Atoms           []MemoryAtom `json:"atoms"`
	Edges           []MemoryEdge `json:"edges"`
}

type AdmissionMapper struct{ config AdmissionConfig }

func NewAdmissionMapper(cfg *AdmissionConfig) *AdmissionMapper {
	if cfg == nil { def := DefaultAdmissionConfig(); cfg = &def }
	return &AdmissionMapper{config: *cfg}
}

func (m *AdmissionMapper) ValidateProvenance(p ProvenanceRecord) error {
	if strings.TrimSpace(p.RunID) == "" { return fmt.Errorf("%w: missing run_id", ErrInvalidProvenance) }
	if strings.TrimSpace(p.EvidenceDigest) == "" { return fmt.Errorf("%w: missing evidence_digest", ErrInvalidProvenance) }
	if strings.TrimSpace(p.Renderer) == "" { return fmt.Errorf("%w: missing renderer", ErrInvalidProvenance) }
	if p.Timestamp.IsZero() { return fmt.Errorf("%w: zero timestamp", ErrStaleTimestamp) }
	now := time.Now()
	if p.Timestamp.After(now.Add(m.config.AllowFutureSkew)) { return fmt.Errorf("%w: timestamp is in the future", ErrStaleTimestamp) }
	if m.config.MaxTimestampAge > 0 && now.Sub(p.Timestamp) > m.config.MaxTimestampAge { return fmt.Errorf("%w: timestamp exceeds max age", ErrStaleTimestamp) }
	return nil
}

// ValidateAdmissionRequest is the canonical ordinary-write authorization gate.
func (m *AdmissionMapper) ValidateAdmissionRequest(req AdmissionRequest) (AdmissionRequest, error) {
	if !req.SourceNamespace.IsValid() { return req, fmt.Errorf("%w: source namespace is required", ErrScopeRequired) }
	if !req.TargetNamespace.IsValid() { return req, fmt.Errorf("%w: target namespace is required", ErrScopeRequired) }
	if !CanAdmitOrdinary(req.SourceNamespace, req.TargetNamespace) {
		return req, fmt.Errorf("%w: %s -> %s requires explicit promotion/authorization", ErrAdmissionRoute, req.SourceNamespace, req.TargetNamespace)
	}
	if err := m.ValidateProvenance(req.Provenance); err != nil { return req, err }

	req.Provenance.SourceNamespace = req.SourceNamespace.String()
	if req.SourceNamespace.IsProjectPrivate() {
		projectID := req.SourceNamespace.ProjectID()
		if strings.TrimSpace(req.Provenance.ProjectScope) == "" {
			req.Provenance.ProjectScope = projectID
		}
		if req.Provenance.ProjectScope != projectID || req.TargetNamespace.ProjectID() != projectID {
			return req, fmt.Errorf("%w: project provenance %q cannot write %s", ErrAdmissionRoute, req.Provenance.ProjectScope, req.TargetNamespace)
		}
	} else if req.SourceNamespace.IsGlobal() {
		if scope := strings.TrimSpace(req.Provenance.ProjectScope); scope != "" && scope != "global" {
			return req, fmt.Errorf("%w: global admission carries project scope %q", ErrAdmissionRoute, scope)
		}
	}
	return req, nil
}

func (m *AdmissionMapper) AdmitPacket(ctx context.Context, packet *evidence.Packet, req AdmissionRequest) (*AdmissionBundle, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	if packet == nil { return nil, fmt.Errorf("%w: packet is nil", ErrInvalidAtomData) }
	var err error
	req, err = m.ValidateAdmissionRequest(req)
	if err != nil { return nil, err }
	ns, prov, now := req.TargetNamespace, req.Provenance, time.Now()

	envID := fmt.Sprintf("env_%s_%s", prov.Renderer, hashString(prov.Environment))
	envAtom := MemoryAtom{ID: envID, Kind: NodeRenderEnvironment, Namespace: ns, Provenance: prov, Confidence: 1,
		Data: RenderEnvironmentAtom{Renderer: prov.Renderer, BrowserFamily: prov.Renderer, ViewportW: 1280, ViewportH: 800, DeviceScale: 1, Theme: "default"},
		Tags: append(req.Tags, "environment", prov.Renderer), CreatedAt: now, UpdatedAt: now}
	artID := fmt.Sprintf("artifact_%s", prov.EvidenceDigest)
	artAtom := MemoryAtom{ID: artID, Kind: NodeEvidenceArtifact, Namespace: ns, Provenance: prov, Confidence: 1,
		Data: EvidenceArtifactAtom{Kind: "evidence_packet", Digest: prov.EvidenceDigest, SizeBytes: int64(len(packet.RunID))},
		Tags: append(req.Tags, "artifact", prov.Renderer), CreatedAt: now, UpdatedAt: now}
	edge := MemoryEdge{FromID: artID, ToID: envID, Relation: RelObservedOn, Weight: 1, Provenance: prov, CreatedAt: now}
	return &AdmissionBundle{SourceNamespace: req.SourceNamespace, Atoms: []MemoryAtom{envAtom, artAtom}, Edges: []MemoryEdge{edge}}, nil
}

func (m *AdmissionMapper) AdmitCritiquePass(ctx context.Context, pass *design.CritiquePass, req AdmissionRequest) (*AdmissionBundle, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	if pass == nil { return nil, fmt.Errorf("%w: critique pass is nil", ErrInvalidAtomData) }
	var err error
	req, err = m.ValidateAdmissionRequest(req)
	if err != nil { return nil, err }
	ns, prov, now := req.TargetNamespace, req.Provenance, time.Now()
	atoms := make([]MemoryAtom, 0, 1+len(pass.Findings)+len(pass.Hypotheses))
	edges := make([]MemoryEdge, 0)

	evalID := fmt.Sprintf("eval_%s_%s", prov.RunID, pass.ID)
	atoms = append(atoms, MemoryAtom{ID: evalID, Kind: NodeEvaluationResult, Namespace: ns, Provenance: prov, Confidence: 1,
		Data: EvaluationResultAtom{RunID: prov.RunID, Score: pass.GroundedScore, Passed: pass.HardViolations == 0, HardViolations: pass.HardViolations, DurationMS: pass.Duration.Milliseconds()},
		Tags: append(req.Tags, "critique_pass", string(pass.Level)), CreatedAt: now, UpdatedAt: now})

	for _, f := range pass.Findings {
		conf := f.Confidence; if conf == 0 { conf = req.Confidence }; if conf < m.config.MinConfidence { continue }
		findingID := fmt.Sprintf("finding_%s_%s", prov.RunID, f.ID)
		atoms = append(atoms, MemoryAtom{ID: findingID, Kind: NodeDesignFinding, Namespace: ns, Provenance: prov, Confidence: conf,
			Data: DesignFindingAtom{FindingID: f.ID, Axis: f.Axis, Category: f.Category, RuleID: f.RuleID, Title: f.Title, Description: f.Description, Severity: f.Severity, HardConstraint: f.HardConstraint, RegionID: f.RegionID, ElementIDs: f.ElementIDs, Suggestion: f.Suggestion},
			Tags: append(req.Tags, "finding", f.Axis, f.Category), CreatedAt: now, UpdatedAt: now})
		edges = append(edges, MemoryEdge{FromID: findingID, ToID: evalID, Relation: RelDerivedFrom, Weight: conf, Provenance: prov, CreatedAt: now})
		if f.RuleID != "" { edges = append(edges, MemoryEdge{FromID: findingID, ToID: fmt.Sprintf("rule_%s", f.RuleID), Relation: RelObservedOn, Weight: 1, Provenance: prov, CreatedAt: now}) }
	}
	for _, h := range pass.Hypotheses {
		if h.Confidence < m.config.MinConfidence { continue }
		patternID := fmt.Sprintf("pattern_%s_%s", prov.RunID, h.ID)
		atoms = append(atoms, MemoryAtom{ID: patternID, Kind: NodeRepairPattern, Namespace: ns, Provenance: prov, Confidence: h.Confidence,
			Data: RepairPatternAtom{PatternID: h.ID, Strategy: h.Strategy, TargetFiles: h.TargetFiles, PatchSnippet: h.ProposedChanges, ExpectedOutcome: h.ExpectedOutcome},
			Tags: append(req.Tags, "repair_pattern", h.Strategy), CreatedAt: now, UpdatedAt: now})
		for _, fID := range h.FindingIDs { edges = append(edges, MemoryEdge{FromID: patternID, ToID: fmt.Sprintf("finding_%s_%s", prov.RunID, fID), Relation: RelRepairedBy, Weight: h.Confidence, Provenance: prov, CreatedAt: now}) }
	}
	return &AdmissionBundle{SourceNamespace: req.SourceNamespace, Atoms: atoms, Edges: edges}, nil
}

func (m *AdmissionMapper) AdmitRepairOutcome(ctx context.Context, hypothesis *design.RepairHypothesis, succeeded bool, req AdmissionRequest) (*AdmissionBundle, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	if hypothesis == nil { return nil, fmt.Errorf("%w: hypothesis is nil", ErrInvalidAtomData) }
	var err error
	req, err = m.ValidateAdmissionRequest(req)
	if err != nil { return nil, err }
	ns, prov, now := req.TargetNamespace, req.Provenance, time.Now()
	patternID := fmt.Sprintf("pattern_%s", hashString(hypothesis.ProposedChanges))
	succ, fail, rate, tag := 0, 0, 0.0, "failed"
	if succeeded { succ, rate, tag = 1, 1, "success" } else { fail = 1 }
	atom := MemoryAtom{ID: patternID, Kind: NodeRepairPattern, Namespace: ns, Provenance: prov, Confidence: hypothesis.Confidence,
		Data: RepairPatternAtom{PatternID: hypothesis.ID, Strategy: hypothesis.Strategy, TargetFiles: hypothesis.TargetFiles, PatchSnippet: hypothesis.ProposedChanges, ExpectedOutcome: hypothesis.ExpectedOutcome, SuccessCount: succ, FailureCount: fail, SuccessRate: rate},
		Tags: append(req.Tags, "repair_pattern", hypothesis.Strategy, tag), CreatedAt: now, UpdatedAt: now}
	atoms := []MemoryAtom{atom}; edges := []MemoryEdge{}
	if !succeeded {
		ceID := fmt.Sprintf("ce_%s", patternID)
		atoms = append(atoms, MemoryAtom{ID: ceID, Kind: NodeCounterexample, Namespace: ns, Provenance: prov, Confidence: 1,
			Data: CounterexampleAtom{TargetEntityID: patternID, Reason: "Repair failed re-verification or caused regression", RefutingDigest: prov.EvidenceDigest, Observation: hypothesis.ExpectedOutcome},
			Tags: append(req.Tags, "counterexample", "refutation"), CreatedAt: now, UpdatedAt: now})
		edges = append(edges, MemoryEdge{FromID: ceID, ToID: patternID, Relation: RelRefutes, Weight: 1, Provenance: prov, CreatedAt: now})
	}
	return &AdmissionBundle{SourceNamespace: req.SourceNamespace, Atoms: atoms, Edges: edges}, nil
}

func hashString(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:8]) }
