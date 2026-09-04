package controlplane

import (
	"context"
	"time"
)

// ValidationNeed is the durable, protocol-neutral projection of engine.EvidenceNeed.
// It exists in the control plane so Axiom can persist semantic validation intent
// without depending on renderer/runtime implementation details.
type ValidationNeed struct {
	Geometry     bool `json:"geometry,omitempty"`
	Styles       bool `json:"styles,omitempty"`
	Pixels       bool `json:"pixels,omitempty"`
	Scenario     bool `json:"scenario,omitempty"`
	CleanState   bool `json:"clean_state,omitempty"`
	CrossBrowser bool `json:"cross_browser,omitempty"`
	Semantic     bool `json:"semantic,omitempty"`
}

// Change is the durable run-level input handed to the control plane.
// Canonical scope inputs are carried losslessly so the execution adapter can
// reconstruct the same supported engine.ValidationRequest semantics that a
// direct caller would use. Large source bodies/artifacts are intentionally
// excluded; persist them by digest/reference and resolve them in the execution
// plane when needed.
type Change struct {
	RunID          string         `json:"run_id,omitempty"`
	ProjectID      string         `json:"project_id,omitempty"`
	SourceDigest   string         `json:"source_digest,omitempty"`
	ChangedFiles   []string       `json:"changed_files,omitempty"`
	ChangedTokens  []string       `json:"changed_tokens,omitempty"`
	ChangedNodes   []string       `json:"changed_nodes,omitempty"`
	TargetRoutes   []string       `json:"target_routes,omitempty"`
	Viewports      []string       `json:"viewports,omitempty"`
	Themes         []string       `json:"themes,omitempty"`
	ForceWholeSite bool           `json:"force_whole_site,omitempty"`
	BaseURL        string         `json:"base_url,omitempty"`
	Need           ValidationNeed `json:"need,omitempty"`

	Intent             string  `json:"intent,omitempty"`
	Risk               string  `json:"risk,omitempty"`
	CustomFontsChanged bool    `json:"custom_fonts_changed,omitempty"`
	SemanticsChanged   bool    `json:"semantics_changed,omitempty"`
	InteractionChanged bool    `json:"interaction_changed,omitempty"`
	RuntimeChanged     bool    `json:"runtime_changed,omitempty"`
	FinalGate          bool    `json:"final_gate,omitempty"`
	Region             *Region `json:"region,omitempty"`
}

type Region struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Scale  float64 `json:"scale,omitempty"`
}

// EvidencePlan is the renderer-neutral plan produced by the FastPath planner.
type EvidencePlan struct {
	Structural    bool     `json:"structural"`
	Diagnostics   bool     `json:"diagnostics"`
	Accessibility bool     `json:"accessibility"`
	Fonts         bool     `json:"fonts"`
	Pixels        bool     `json:"pixels"`
	BrowserTruth  bool     `json:"browser_truth"`
	Reasons       []string `json:"reasons,omitempty"`
}

// ValidationResult stores only compact facts in Axiom state. Heavy screenshots,
// DOM snapshots and VLM artifacts must remain in the execution plane and be
// referenced separately when persistence is added.
type ValidationResult struct {
	BlockingFindings    int      `json:"blocking_findings"`
	HighFindings        int      `json:"high_findings"`
	MissingEvidence     []string `json:"missing_evidence,omitempty"`
	VisualRegions       int      `json:"visual_regions,omitempty"`
	VisualFindings      int      `json:"visual_findings,omitempty"`
	PixelEvidence       bool     `json:"pixel_evidence,omitempty"`
	DiagnosticsComplete bool     `json:"diagnostics_complete"`
	Summary             string   `json:"summary,omitempty"`
}

type Decision string

const (
	DecisionPass        Decision = "pass"
	DecisionRepair      Decision = "repair"
	DecisionRecollect   Decision = "recollect"
	DecisionPixels      Decision = "pixels"
	DecisionSemantic    Decision = "semantic"
	DecisionHumanReview Decision = "human_review"
)

// Executor is the only execution-plane contract the Axiom workflow needs.
// Implementations may call FastCDP/WGGo/engine directly in-process.
type Executor interface {
	PlanEvidence(context.Context, Change) (EvidencePlan, error)
	CollectVerify(context.Context, Change, EvidencePlan) (ValidationResult, error)
	Decide(context.Context, Change, EvidencePlan, ValidationResult) (Decision, error)
}

// DesignPolishRequest encapsulates parameters for a multi-step design polish workflow.
type DesignPolishRequest struct {
	Intent        string   `json:"intent"`
	Target        string   `json:"target,omitempty"`
	MaxIterations int      `json:"max_iterations,omitempty"`
	ProtectedAxes []string `json:"protected_axes,omitempty"`
	Profile       string   `json:"profile,omitempty"`
}

// PolishIteration records the outcome of one repair and re-verification cycle.
type PolishIteration struct {
	Iteration      int     `json:"iteration"`
	HypothesisID   string  `json:"hypothesis_id,omitempty"`
	FindingsCount  int     `json:"findings_count"`
	HardViolations int     `json:"hard_violations"`
	Score          float64 `json:"score"`
	Accepted       bool    `json:"accepted"`
	Rationale      string  `json:"rationale"`
}

// DesignPolishResult records final quality, iteration history and convergence status.
type DesignPolishResult struct {
	InitialScore      float64           `json:"initial_score"`
	FinalScore        float64           `json:"final_score"`
	AcceptedCount     int               `json:"accepted_count"`
	TotalIterations   int               `json:"total_iterations"`
	Converged         bool              `json:"converged"`
	RemainingFindings int               `json:"remaining_findings"`
	Summary           string            `json:"summary"`
	Iterations        []PolishIteration `json:"iterations,omitempty"`
}

// DesignPolishExecutor drives the execution steps of a DesignPolishRun.
type DesignPolishExecutor interface {
	InspectBaseline(context.Context, DesignPolishRequest) (PolishIteration, error)
	StepPolish(context.Context, DesignPolishRequest, int) (PolishIteration, error)
	ConcludePolish(context.Context, DesignPolishRequest, []PolishIteration) (DesignPolishResult, error)
}

// CandidateComparisonRequest encapsulates parameters for a candidate ranking workflow.
type CandidateComparisonRequest struct {
	BaselineID    string   `json:"baseline_id"`
	CandidateIDs  []string `json:"candidate_ids"`
	ProtectedAxes []string `json:"protected_axes,omitempty"`
}

// CandidateRank records an evaluated variant's position and constraints outcome.
type CandidateRank struct {
	CandidateID       string   `json:"candidate_id"`
	Rank              int      `json:"rank"`
	Score             float64  `json:"score"`
	PassedConstraints bool     `json:"passed_constraints"`
	RegressedAxes     []string `json:"regressed_axes,omitempty"`
	Rationale         string   `json:"rationale"`
}

// ComparisonRunResult records the winner and ranked variants.
type ComparisonRunResult struct {
	WinnerID string          `json:"winner_id"`
	Rankings []CandidateRank `json:"rankings"`
	Summary  string          `json:"summary"`
}

// CandidateComparisonExecutor drives the execution steps of a CandidateComparisonRun.
type CandidateComparisonExecutor interface {
	EvaluateCandidates(context.Context, CandidateComparisonRequest) ([]CandidateRank, error)
	ConcludeComparison(context.Context, CandidateComparisonRequest, []CandidateRank) (ComparisonRunResult, error)
}

type Budget struct {
	MaxCost           float64       `json:"max_cost,omitempty"`
	MaxTokens         int64         `json:"max_tokens,omitempty"`
	MaxDuration       time.Duration `json:"max_duration,omitempty"`
	MaxLLMCalls       int           `json:"max_llm_calls,omitempty"`
	MaxSearchQueries  int           `json:"max_search_queries,omitempty"`
	MaxBrowserFetches int           `json:"max_browser_fetches,omitempty"`
}

type Usage struct {
	Cost           float64       `json:"cost,omitempty"`
	Tokens         int64         `json:"tokens,omitempty"`
	ActiveDuration time.Duration `json:"active_duration,omitempty"`
	LLMCalls       int           `json:"llm_calls,omitempty"`
	SearchQueries  int           `json:"search_queries,omitempty"`
	BrowserFetches int           `json:"browser_fetches,omitempty"`
}

type HistoryEntry struct {
	Sequence uint64         `json:"sequence"`
	At       time.Time      `json:"at"`
	Type     string         `json:"type"`
	NodeID   string         `json:"node_id,omitempty"`
	Message  string         `json:"message,omitempty"`
	Data     map[string]any `json:"data,omitempty"`
}

type Run struct {
	ID          string           `json:"id"`
	Status      string           `json:"status"`
	PlanID      string           `json:"plan_id"`
	PlanVersion string           `json:"plan_version"`
	PlanDigest  string           `json:"plan_digest"`
	Change      Change           `json:"change"`
	Evidence    EvidencePlan     `json:"evidence"`
	Validation  ValidationResult `json:"validation"`
	Decision    Decision         `json:"decision,omitempty"`
	Usage       Usage            `json:"usage"`
	Failure     string           `json:"failure,omitempty"`
	History     []HistoryEntry   `json:"history,omitempty"`
}

type DesignPolishRun struct {
	ID      string              `json:"id"`
	Status  string              `json:"status"`
	PlanID  string              `json:"plan_id"`
	Request DesignPolishRequest `json:"request"`
	Result  DesignPolishResult  `json:"result"`
	Usage   Usage               `json:"usage"`
	Failure string              `json:"failure,omitempty"`
	History []HistoryEntry      `json:"history,omitempty"`
}

type CandidateComparisonRun struct {
	ID      string                     `json:"id"`
	Status  string                     `json:"status"`
	PlanID  string                     `json:"plan_id"`
	Request CandidateComparisonRequest `json:"request"`
	Result  ComparisonRunResult        `json:"result"`
	Usage   Usage                      `json:"usage"`
	Failure string                     `json:"failure,omitempty"`
	History []HistoryEntry             `json:"history,omitempty"`
}
