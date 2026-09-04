package repair

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/memory"
)

// Patch represents a concrete modification applied to source HTML or CSS.
type Patch struct {
	ID          string `json:"id"`
	FindingID   string `json:"finding_id"`
	Description string `json:"description"`
	TargetType  string `json:"target_type"` // "html", "css"
	OldContent  string `json:"old_content,omitempty"`
	NewContent  string `json:"new_content"`
}

// RepairLoopRequest defines the target and constraints for an autonomous repair pass.
type RepairLoopRequest struct {
	RunID         string                `json:"run_id"`
	HTML          string                `json:"html"`
	CSS           string                `json:"css"`
	Profile       design.ProductProfile `json:"profile"`
	ProtectedAxes []string              `json:"protected_axes,omitempty"`
	MaxIterations int                   `json:"max_iterations,omitempty"`
	ProjectID     string                `json:"project_id,omitempty"`
}

// RepairLoopResult summarizes the outcome of the autonomous repair & independent re-verification.
type RepairLoopResult struct {
	RunID           string                     `json:"run_id"`
	InitialFindings int                        `json:"initial_findings"`
	FinalFindings   int                        `json:"final_findings"`
	FixedFindings   []string                   `json:"fixed_findings"`
	PatchesApplied  []Patch                    `json:"patches_applied"`
	RepairedHTML    string                     `json:"repaired_html"`
	RepairedCSS     string                     `json:"repaired_css"`
	Passed          bool                       `json:"passed"`
	Comparison      design.CandidateComparison `json:"comparison"`
	MemoryAdmitted  bool                       `json:"memory_admitted"`
	AdmittedAtoms   int                        `json:"admitted_atoms"`
	Summary         string                     `json:"summary"`
}

// HostRepairEngine coordinates hypothesis generation, patch application, independent re-verification, and memory admission.
type HostRepairEngine struct {
	Pipeline   *engine.Pipeline
	Critic     *critic.LocalSemanticCritic
	Comparator *design.RelativeComparator
	Store      *memory.EpMemoryStore
	Admission  *memory.AdmissionMapper
}

// New creates an initialized HostRepairEngine.
func New(pipeline *engine.Pipeline) *HostRepairEngine {
	if pipeline == nil {
		pipeline = &engine.Pipeline{}
	}
	return &HostRepairEngine{
		Pipeline:   pipeline,
		Critic:     critic.New(),
		Comparator: design.NewComparator(),
	}
}

// NewWithMemory creates a HostRepairEngine equipped with SncSinCore epistemic memory store.
func NewWithMemory(pipeline *engine.Pipeline, store *memory.EpMemoryStore) *HostRepairEngine {
	e := New(pipeline)
	e.Store = store
	e.Admission = memory.NewAdmissionMapper(nil)
	return e
}

// RunRepairLoop executes the autonomous repair, independent re-verification, and memory admission loop.
func (e *HostRepairEngine) RunRepairLoop(ctx context.Context, req RepairLoopRequest) (RepairLoopResult, error) {
	if err := ctx.Err(); err != nil {
		return RepairLoopResult{}, err
	}

	if req.RunID == "" {
		req.RunID = "repair-run-1"
	}
	if req.MaxIterations <= 0 {
		req.MaxIterations = 3
	}
	if len(req.ProtectedAxes) == 0 {
		req.ProtectedAxes = []string{"accessibility", "responsive", "typography"}
	}

	currentHTML := req.HTML
	currentCSS := req.CSS

	// 1. Capture & Verify Baseline
	baseRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID: fmt.Sprintf("%s-baseline", req.RunID),
		HTML:  []byte(currentHTML),
		CSS:   []byte(currentCSS),
	})
	if err != nil {
		return RepairLoopResult{}, fmt.Errorf("repair: execute baseline: %w", err)
	}

	baseCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         "baseline",
		Profile:       req.Profile,
		Packet:        baseRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return RepairLoopResult{}, fmt.Errorf("repair: critique baseline: %w", err)
	}

	initialFindingCount := len(baseCritique.Findings)
	appliedPatches := make([]Patch, 0)
	fixedFindingIDs := make([]string, 0)

	// 2. Iterative Repair
	for iter := 1; iter <= req.MaxIterations; iter++ {
		if len(baseCritique.Findings) == 0 && baseCritique.HardViolations == 0 {
			break
		}

		patches := generatePatches(baseCritique.Findings, currentHTML, currentCSS)
		if len(patches) == 0 {
			break
		}

		for _, p := range patches {
			switch p.TargetType {
			case "html":
				currentHTML = p.NewContent
			case "css":
				currentCSS = p.NewContent
			}
			appliedPatches = append(appliedPatches, p)
			fixedFindingIDs = append(fixedFindingIDs, p.FindingID)
		}

		// Re-evaluate
		nextRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
			RunID: fmt.Sprintf("%s-iter-%d", req.RunID, iter),
			HTML:  []byte(currentHTML),
			CSS:   []byte(currentCSS),
		})
		if err != nil {
			return RepairLoopResult{}, fmt.Errorf("repair: re-verify iteration %d: %w", iter, err)
		}

		nextCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{
			RunID:         fmt.Sprintf("iter-%d", iter),
			Profile:       req.Profile,
			Packet:        nextRes.Packet,
			ProtectedAxes: req.ProtectedAxes,
		})
		if err != nil {
			return RepairLoopResult{}, fmt.Errorf("repair: critique iteration %d: %w", iter, err)
		}

		baseCritique = nextCritique
		if len(baseCritique.Findings) == 0 {
			break
		}
	}

	// 3. Independent Re-Verification & Relative Comparison against original baseline
	candRes, err := e.Pipeline.Execute(ctx, engine.ValidationRequest{
		RunID: fmt.Sprintf("%s-candidate-final", req.RunID),
		HTML:  []byte(currentHTML),
		CSS:   []byte(currentCSS),
	})
	if err != nil {
		return RepairLoopResult{}, fmt.Errorf("repair: execute final candidate: %w", err)
	}

	candCritique, err := e.Critic.Critique(ctx, critic.CritiqueRequest{
		RunID:         "candidate-final",
		Profile:       req.Profile,
		Packet:        candRes.Packet,
		ProtectedAxes: req.ProtectedAxes,
	})
	if err != nil {
		return RepairLoopResult{}, fmt.Errorf("repair: critique candidate final: %w", err)
	}

	comparison, err := e.Comparator.Compare(ctx, design.ComparisonRequest{
		RunID:             fmt.Sprintf("cmp:%s", req.RunID),
		BaselineID:        "baseline",
		CandidateID:       "candidate_repaired",
		BaselinePacket:    baseRes.Packet,
		CandidatePacket:   candRes.Packet,
		BaselineCritique:  &baseCritique,
		CandidateCritique: &candCritique,
		ProtectedAxes:     req.ProtectedAxes,
	})
	if err != nil {
		return RepairLoopResult{}, fmt.Errorf("repair: compare candidates: %w", err)
	}

	passed := comparison.PassedConstraints && comparison.PreferredCandidate == "candidate_repaired"
	summary := fmt.Sprintf("Autonomous repair finished: %d initial findings -> %d final findings (%d patches applied); pass=%v",
		initialFindingCount, len(candCritique.Findings), len(appliedPatches), passed)

	// 4. Commit Admitted Lesson to SncSinCore Epistemic Memory
	memoryAdmitted := false
	admittedAtomCount := 0
	if e.Store != nil && e.Admission != nil && len(appliedPatches) > 0 {
		var ns memory.Namespace
		if req.ProjectID != "" {
			ns, _ = memory.NewProjectKnowledgeNamespace(req.ProjectID)
		} else {
			ns = memory.NewGlobalDesignNamespace()
		}

		prov := memory.ProvenanceRecord{
			RunID:          req.RunID,
			EvidenceDigest: fmt.Sprintf("digest-%s", req.RunID),
			Renderer:       "pipeline-host-repair",
			Timestamp:      time.Now(),
		}

		for _, p := range appliedPatches {
			hypothesis := &design.RepairHypothesis{
				ID:              p.ID,
				FindingIDs:      []string{p.FindingID},
				Strategy:        p.Description,
				ProposedChanges: p.NewContent,
				ExpectedOutcome: "Eliminates defect without protected axis regression",
				Confidence:      1.0,
			}
			bundle, err := e.Admission.AdmitRepairOutcome(ctx, hypothesis, passed, memory.AdmissionRequest{
				TargetNamespace: ns,
				Provenance:      prov,
				Confidence:      1.0,
				Tags:            []string{"repair_lesson", p.TargetType},
			})
			if err == nil && bundle != nil {
				if commitErr := e.Store.Commit(ctx, *bundle); commitErr == nil {
					memoryAdmitted = true
					admittedAtomCount += len(bundle.Atoms)
				}
			}
		}
	}

	return RepairLoopResult{
		RunID:           req.RunID,
		InitialFindings: initialFindingCount,
		FinalFindings:   len(candCritique.Findings),
		FixedFindings:   fixedFindingIDs,
		PatchesApplied:  appliedPatches,
		RepairedHTML:    currentHTML,
		RepairedCSS:     currentCSS,
		Passed:          passed,
		Comparison:      comparison,
		MemoryAdmitted:  memoryAdmitted,
		AdmittedAtoms:   admittedAtomCount,
		Summary:         summary,
	}, nil
}

func generatePatches(findings []design.Finding, currentHTML, currentCSS string) []Patch {
	patches := make([]Patch, 0)

	for _, f := range findings {
		switch {
		case strings.Contains(f.ID, "heading_missing"):
			// Insert <h1> heading if missing
			if !strings.Contains(strings.ToLower(currentHTML), "<h1") {
				newHTML := currentHTML
				if strings.Contains(newHTML, "<body>") {
					newHTML = strings.Replace(newHTML, "<body>", "<body>\n  <h1>Main Page Title</h1>", 1)
				} else {
					newHTML = "<h1>Main Page Title</h1>\n" + newHTML
				}
				patches = append(patches, Patch{
					ID:          fmt.Sprintf("patch:h1:%s", f.ID),
					FindingID:   f.ID,
					Description: "Insert missing top-level <h1> heading",
					TargetType:  "html",
					OldContent:  currentHTML,
					NewContent:  newHTML,
				})
			}

		case strings.Contains(f.ID, "overflow"):
			// Apply max-width constraint to CSS
			newCSS := currentCSS + "\n* { box-sizing: border-box; }\nbody { max-width: 100vw; overflow-x: hidden; }\n"
			patches = append(patches, Patch{
				ID:          fmt.Sprintf("patch:overflow:%s", f.ID),
				FindingID:   f.ID,
				Description: "Constrain layout width to avoid horizontal scrollbar",
				TargetType:  "css",
				OldContent:  currentCSS,
				NewContent:  newCSS,
			})

		case strings.Contains(f.ID, "a11y_name"):
			// Add aria-label to button or element without name
			reButton := regexp.MustCompile(`(?i)<button([^>]*)>`)
			if reButton.MatchString(currentHTML) && !strings.Contains(currentHTML, "aria-label") {
				newHTML := reButton.ReplaceAllString(currentHTML, `<button aria-label="Action Button"$1>`)
				patches = append(patches, Patch{
					ID:          fmt.Sprintf("patch:a11y:%s", f.ID),
					FindingID:   f.ID,
					Description: "Add aria-label to unlabelled actionable element",
					TargetType:  "html",
					OldContent:  currentHTML,
					NewContent:  newHTML,
				})
			}
		}
	}

	return patches
}
