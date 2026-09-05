package repair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/sideeffect"
)

type Patch struct {
	ID          string `json:"id"`
	FindingID   string `json:"finding_id"`
	Description string `json:"description"`
	TargetType  string `json:"target_type"`
	OldContent  string `json:"old_content,omitempty"`
	NewContent  string `json:"new_content"`
}

type RepairLoopRequest struct {
	RunID         string                `json:"run_id"`
	HTML          string                `json:"html"`
	CSS           string                `json:"css"`
	Profile       design.ProductProfile `json:"profile"`
	ProtectedAxes []string              `json:"protected_axes,omitempty"`
	MaxIterations int                   `json:"max_iterations,omitempty"`
	ProjectID     string                `json:"project_id,omitempty"`
	RiskClass     RepairRiskClass       `json:"risk_class,omitempty"`
}

type RepairMetrics struct {
	Iterations           int     `json:"iterations"`
	CandidateEvaluations int     `json:"candidate_evaluations"`
	OscillationCount     int     `json:"oscillation_count"`
	HeldOutCases         int     `json:"held_out_cases"`
	HeldOutFailures      int     `json:"held_out_failures"`
	RegressionEscapes    int     `json:"regression_escapes"`
	HeldOutEscapeRate    float64 `json:"held_out_escape_rate"`
}

type RepairLoopResult struct {
	RunID              string                     `json:"run_id"`
	InitialFindings    int                        `json:"initial_findings"`
	FinalFindings      int                        `json:"final_findings"`
	FixedFindings      []string                   `json:"fixed_findings"`
	PatchesApplied     []Patch                    `json:"patches_applied"`
	RepairedHTML       string                     `json:"repaired_html"`
	RepairedCSS        string                     `json:"repaired_css"`
	CandidateImproved  bool                       `json:"candidate_improved"`
	Passed             bool                       `json:"passed"`
	Comparison         design.CandidateComparison `json:"comparison"`
	FinalGate          FinalGateResult            `json:"final_gate"`
	Metrics            RepairMetrics              `json:"metrics"`
	TerminationReason  string                     `json:"termination_reason,omitempty"`
	EscalationRequired bool                       `json:"escalation_required"`
	SourceCommitted    bool                       `json:"source_committed"`
	MemoryAdmitted     bool                       `json:"memory_admitted"`
	AdmittedAtoms      int                        `json:"admitted_atoms"`
	SideEffects        []sideeffect.Receipt       `json:"side_effects,omitempty"`
	Summary            string                     `json:"summary"`
}

type HostRepairEngine struct {
	Pipeline    *engine.Pipeline
	Critic      *critic.LocalSemanticCritic
	Comparator  design.Comparator
	FinalGate   FinalGate
	Store       *memory.EpMemoryStore
	Admission   *memory.AdmissionMapper
	Source      *sideeffect.SourceStore
	// AfterEffect is an operational/fault-injection hook invoked only after a
	// target has durably accepted the effect. Returning an error models process
	// loss before the enclosing workflow/activity records completion.
	AfterEffect func(kind string, receipt sideeffect.Receipt) error
}

func New(pipeline *engine.Pipeline) *HostRepairEngine {
	if pipeline == nil { pipeline = &engine.Pipeline{} }
	return &HostRepairEngine{Pipeline: pipeline, Critic: critic.New(), Comparator: design.NewComparator()}
}

func NewWithFinalGate(pipeline *engine.Pipeline, finalGate FinalGate) *HostRepairEngine {
	e := New(pipeline); e.FinalGate = finalGate; return e
}

func NewWithMemory(pipeline *engine.Pipeline, store *memory.EpMemoryStore) *HostRepairEngine {
	e := New(pipeline); e.Store = store; e.Admission = memory.NewAdmissionMapper(nil); return e
}

func NewWithMemoryAndFinalGate(pipeline *engine.Pipeline, finalGate FinalGate, store *memory.EpMemoryStore) *HostRepairEngine {
	e := NewWithFinalGate(pipeline, finalGate); e.Store = store; e.Admission = memory.NewAdmissionMapper(nil); return e
}

func (e *HostRepairEngine) RunRepairLoop(ctx context.Context, req RepairLoopRequest) (RepairLoopResult, error) {
	if err := ctx.Err(); err != nil { return RepairLoopResult{}, err }
	if e == nil || e.Pipeline == nil || e.Critic == nil || e.Comparator == nil {
		return RepairLoopResult{}, fmt.Errorf("repair: optimization pipeline, critic and comparator are required")
	}
	if req.RunID == "" { req.RunID = "repair-run-1" }
	if req.MaxIterations <= 0 { req.MaxIterations = 3 }
	if len(req.ProtectedAxes) == 0 { req.ProtectedAxes = []string{"accessibility", "responsive", "typography", "interaction"} }
	if req.RiskClass == "" { req.RiskClass = RepairRiskHigh }

	currentHTML, currentCSS := req.HTML, req.CSS
	metrics := RepairMetrics{}
	terminationReason := "max_iterations"
	escalationRequired := false

	baseRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID: fmt.Sprintf("%s-baseline", req.RunID), ProjectID: req.ProjectID,
		Need: engine.EvidenceNeed{Geometry: true, Styles: true}, HTML: []byte(currentHTML), CSS: []byte(currentCSS),
	})
	if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: execute baseline: %w", err) }
	baselineCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{RunID: "baseline", Profile: req.Profile, Packet: baseRes.Packet, ProtectedAxes: req.ProtectedAxes})
	if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: critique baseline: %w", err) }
	workingCritique := baselineCritique
	initialFindingCount := len(baselineCritique.Findings)
	appliedPatches := make([]Patch, 0)
	fixedFindingIDs := make([]string, 0)
	seenStates := map[string]int{repairStateDigest(currentHTML, currentCSS): 0}
	lastFindingState := findingStateDigest(workingCritique.Findings)

	for iter := 1; iter <= req.MaxIterations; iter++ {
		if len(workingCritique.Findings) == 0 && workingCritique.HardViolations == 0 { terminationReason = "optimization_clean"; break }
		patches := generatePatches(workingCritique.Findings, currentHTML, currentCSS)
		if len(patches) == 0 { terminationReason = "no_local_patch"; escalationRequired = true; break }
		for _, p := range patches {
			switch p.TargetType { case "html": currentHTML = p.NewContent; case "css": currentCSS = p.NewContent }
			appliedPatches = append(appliedPatches, p)
			fixedFindingIDs = append(fixedFindingIDs, p.FindingID)
		}
		metrics.Iterations = iter
		state := repairStateDigest(currentHTML, currentCSS)
		if _, repeated := seenStates[state]; repeated {
			metrics.OscillationCount++; terminationReason = "source_state_oscillation"; escalationRequired = true; break
		}
		seenStates[state] = iter
		nextRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
			RunID: fmt.Sprintf("%s-iter-%d", req.RunID, iter), ProjectID: req.ProjectID,
			Need: engine.EvidenceNeed{Geometry: true, Styles: true}, HTML: []byte(currentHTML), CSS: []byte(currentCSS),
		})
		if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: re-verify iteration %d: %w", iter, err) }
		metrics.CandidateEvaluations++
		nextCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{RunID: fmt.Sprintf("iter-%d", iter), Profile: req.Profile, Packet: nextRes.Packet, ProtectedAxes: req.ProtectedAxes})
		if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: critique iteration %d: %w", iter, err) }
		nextFindingState := findingStateDigest(nextCritique.Findings)
		if len(nextCritique.Findings) > 0 && nextFindingState == lastFindingState {
			metrics.OscillationCount++; workingCritique = nextCritique; terminationReason = "repeated_finding_state"; escalationRequired = true; break
		}
		workingCritique = nextCritique
		lastFindingState = nextFindingState
		if len(workingCritique.Findings) == 0 && workingCritique.HardViolations == 0 { terminationReason = "optimization_clean"; break }
	}

	candRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID: fmt.Sprintf("%s-candidate-optimization", req.RunID), ProjectID: req.ProjectID,
		Need: engine.EvidenceNeed{Geometry: true, Styles: true}, HTML: []byte(currentHTML), CSS: []byte(currentCSS),
	})
	if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: execute optimization candidate: %w", err) }
	candCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{RunID: "candidate-optimization", Profile: req.Profile, Packet: candRes.Packet, ProtectedAxes: req.ProtectedAxes})
	if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: critique optimization candidate: %w", err) }
	comparison, err := e.Comparator.Compare(ctx, design.ComparisonRequest{
		RunID: fmt.Sprintf("cmp:%s", req.RunID), BaselineID: "baseline", CandidateID: "candidate_repaired",
		BaselinePacket: baseRes.Packet, CandidatePacket: candRes.Packet, BaselineCritique: &baselineCritique,
		CandidateCritique: &candCritique, ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: compare optimization candidate: %w", err) }
	candidateImproved := comparison.PassedConstraints && comparison.PreferredCandidate == "candidate_repaired"

	finalGate := FinalGateResult{VerifierID: "unconfigured", Independent: false, Passed: false, ReasonCodes: []string{"independent_final_gate_unconfigured"}}
	if escalationRequired {
		finalGate.ReasonCodes = []string{"optimization_escalation_required"}
	} else if !candidateImproved {
		finalGate.ReasonCodes = []string{"optimization_candidate_not_preferred"}
	} else if e.FinalGate != nil {
		finalGate = FinalGateResult{VerifierID: e.FinalGate.VerifierID()}
		if !e.FinalGate.IndependentFrom(e.Pipeline) {
			finalGate.ReasonCodes = []string{"final_gate_not_independent"}; escalationRequired = true
		} else {
			finalGate, err = e.FinalGate.Verify(ctx, FinalVerificationRequest{
				RunID: req.RunID, ProjectID: req.ProjectID, BaselineHTML: req.HTML, BaselineCSS: req.CSS,
				CandidateHTML: currentHTML, CandidateCSS: currentCSS, Profile: req.Profile,
				ProtectedAxes: append([]string(nil), req.ProtectedAxes...), RiskClass: req.RiskClass,
			})
			if err != nil { return RepairLoopResult{}, fmt.Errorf("repair: independent final gate: %w", err) }
			if !finalGate.Independent { finalGate.Passed = false; finalGate.ReasonCodes = append(finalGate.ReasonCodes, "final_gate_did_not_attest_independence") }
		}
	}

	passed := candidateImproved && finalGate.Independent && finalGate.Passed && !escalationRequired
	if !passed { escalationRequired = true }
	metrics.HeldOutCases = finalGate.HeldOut.Total
	metrics.HeldOutFailures = finalGate.HeldOut.Failed
	metrics.RegressionEscapes = finalGate.HeldOut.RegressionEscapes
	metrics.HeldOutEscapeRate = finalGate.HeldOut.EscapeRate

	sideEffects := make([]sideeffect.Receipt, 0)
	sourceCommitted := false
	if passed && e.Source != nil && len(appliedPatches) > 0 {
		desired := sideeffect.SourceState{HTML: currentHTML, CSS: currentCSS}
		op := sideeffect.Operation{
			RunID: req.RunID, Activity: "repair_source_commit", Iteration: metrics.Iterations,
			Kind: "source_repair", PayloadDigest: desired.Digest(), RetryClass: sideeffect.RetryIdempotent,
		}
		receipt, commitErr := e.Source.CompareAndSwap(ctx, op, sideeffect.SourceState{HTML: req.HTML, CSS: req.CSS}.Digest(), desired)
		if commitErr != nil { return RepairLoopResult{}, fmt.Errorf("repair: commit source effect: %w", commitErr) }
		sourceCommitted = true
		sideEffects = append(sideEffects, receipt)
		if e.AfterEffect != nil {
			if hookErr := e.AfterEffect("source_repair", receipt); hookErr != nil { return RepairLoopResult{}, hookErr }
		}
	}

	memoryAdmitted := false
	admittedAtomCount := 0
	if passed && e.Store != nil && e.Admission != nil && len(appliedPatches) > 0 {
		var ns memory.Namespace
		if req.ProjectID != "" { ns, _ = memory.NewProjectKnowledgeNamespace(req.ProjectID) } else { ns = memory.NewGlobalDesignNamespace() }
		prov := memory.ProvenanceRecord{
			RunID: req.RunID, EvidenceDigest: fmt.Sprintf("final-gate:%s:%s", finalGate.VerifierID, req.RunID),
			Renderer: finalGate.EvidenceTier, Timestamp: time.Now(),
		}
		for i, p := range appliedPatches {
			hypothesis := &design.RepairHypothesis{
				ID: p.ID, FindingIDs: []string{p.FindingID}, Strategy: p.Description, ProposedChanges: p.NewContent,
				ExpectedOutcome: "Independent final gate + held-out suite passed without protected-axis regression", Confidence: 1.0,
			}
			bundle, admitErr := e.Admission.AdmitRepairOutcome(ctx, hypothesis, true, memory.AdmissionRequest{
				TargetNamespace: ns, Provenance: prov, Confidence: 1.0,
				Tags: []string{"repair_lesson", "independent_final_gate", p.TargetType},
			})
			if admitErr != nil { return RepairLoopResult{}, fmt.Errorf("repair: map memory admission for patch %s: %w", p.ID, admitErr) }
			if bundle == nil { return RepairLoopResult{}, fmt.Errorf("repair: memory admission for patch %s returned nil bundle", p.ID) }
			payload := sideeffect.DigestBytes([]byte(strings.Join([]string{req.ProjectID, p.ID, p.FindingID, p.TargetType, p.NewContent, finalGate.VerifierID}, "\x00")))
			op := sideeffect.Operation{
				RunID: req.RunID, Activity: "repair_memory_admission", Iteration: i + 1,
				Kind: "memory_admission", PayloadDigest: payload, RetryClass: sideeffect.RetryIdempotent,
			}
			receipt, commitErr := e.Store.CommitOnce(ctx, op, *bundle)
			if commitErr != nil { return RepairLoopResult{}, fmt.Errorf("repair: commit memory admission for patch %s: %w", p.ID, commitErr) }
			memoryAdmitted = true
			admittedAtomCount += len(bundle.Atoms)
			sideEffects = append(sideEffects, receipt)
			if e.AfterEffect != nil {
				if hookErr := e.AfterEffect("memory_admission", receipt); hookErr != nil { return RepairLoopResult{}, hookErr }
			}
		}
	}

	summary := fmt.Sprintf(
		"Autonomous repair: %d initial -> %d optimization findings; %d patches; candidate_improved=%v; independent_pass=%v; heldout_escape_rate=%.3f; escalation=%v",
		initialFindingCount, len(candCritique.Findings), len(appliedPatches), candidateImproved, passed, metrics.HeldOutEscapeRate, escalationRequired,
	)

	return RepairLoopResult{
		RunID: req.RunID, InitialFindings: initialFindingCount, FinalFindings: len(candCritique.Findings),
		FixedFindings: fixedFindingIDs, PatchesApplied: appliedPatches, RepairedHTML: currentHTML, RepairedCSS: currentCSS,
		CandidateImproved: candidateImproved, Passed: passed, Comparison: comparison, FinalGate: finalGate, Metrics: metrics,
		TerminationReason: terminationReason, EscalationRequired: escalationRequired, SourceCommitted: sourceCommitted,
		MemoryAdmitted: memoryAdmitted, AdmittedAtoms: admittedAtomCount, SideEffects: sideEffects, Summary: summary,
	}, nil
}

func repairStateDigest(html, css string) string {
	sum := sha256.Sum256([]byte(html + "\x00" + css))
	return hex.EncodeToString(sum[:])
}

func findingStateDigest(findings []design.Finding) string {
	keys := make([]string, 0, len(findings))
	for _, finding := range findings {
		keys = append(keys, strings.Join([]string{finding.RuleID, finding.Axis, finding.Category, finding.Title, string(finding.Severity), fmt.Sprintf("hard=%t", finding.HardConstraint)}, "|"))
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return hex.EncodeToString(sum[:])
}

func generatePatches(findings []design.Finding, currentHTML, currentCSS string) []Patch {
	patches := make([]Patch, 0)
	for _, f := range findings {
		switch {
		case strings.Contains(f.ID, "heading_missing"):
			if !strings.Contains(strings.ToLower(currentHTML), "<h1") {
				newHTML := currentHTML
				if strings.Contains(newHTML, "<body>") { newHTML = strings.Replace(newHTML, "<body>", "<body>\n  <h1>Main Page Title</h1>", 1) } else { newHTML = "<h1>Main Page Title</h1>\n" + newHTML }
				patches = append(patches, Patch{ID: fmt.Sprintf("patch:h1:%s", f.ID), FindingID: f.ID, Description: "Insert missing top-level <h1> heading", TargetType: "html", OldContent: currentHTML, NewContent: newHTML})
			}
		case strings.Contains(f.ID, "overflow"):
			newCSS := currentCSS + "\n* { box-sizing: border-box; }\nhtml, body { max-width: 100%; }\nbody { width: auto !important; }\n"
			patches = append(patches, Patch{ID: fmt.Sprintf("patch:overflow:%s", f.ID), FindingID: f.ID, Description: "Remove forced body width and constrain layout without hiding overflow", TargetType: "css", OldContent: currentCSS, NewContent: newCSS})
		case strings.Contains(f.ID, "a11y_name"):
			reButton := regexp.MustCompile(`(?i)<button([^>]*)>`)
			if reButton.MatchString(currentHTML) && !strings.Contains(currentHTML, "aria-label") {
				newHTML := reButton.ReplaceAllString(currentHTML, `<button aria-label="Action Button"$1>`)
				patches = append(patches, Patch{ID: fmt.Sprintf("patch:a11y:%s", f.ID), FindingID: f.ID, Description: "Add aria-label to unlabelled actionable element", TargetType: "html", OldContent: currentHTML, NewContent: newHTML})
			}
		}
	}
	return patches
}
