package controlplane

import (
	"context"
	"time"
)

// Change is the compact run-level intent handed to the durable control plane.
// It deliberately avoids CDP/renderer implementation types.
type Change struct {
	Intent             string `json:"intent,omitempty"`
	Risk               string `json:"risk,omitempty"`
	CustomFontsChanged bool   `json:"custom_fonts_changed,omitempty"`
	SemanticsChanged   bool   `json:"semantics_changed,omitempty"`
	InteractionChanged bool   `json:"interaction_changed,omitempty"`
	RuntimeChanged     bool   `json:"runtime_changed,omitempty"`
	FinalGate          bool   `json:"final_gate,omitempty"`
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
	BlockingFindings   int      `json:"blocking_findings"`
	HighFindings       int      `json:"high_findings"`
	MissingEvidence    []string `json:"missing_evidence,omitempty"`
	VisualRegions      int      `json:"visual_regions,omitempty"`
	DiagnosticsComplete bool    `json:"diagnostics_complete"`
	Summary            string   `json:"summary,omitempty"`
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
	ID         string           `json:"id"`
	Status     string           `json:"status"`
	PlanID     string           `json:"plan_id"`
	PlanVersion string          `json:"plan_version"`
	PlanDigest string           `json:"plan_digest"`
	Change     Change           `json:"change"`
	Evidence   EvidencePlan     `json:"evidence"`
	Validation ValidationResult `json:"validation"`
	Decision   Decision         `json:"decision,omitempty"`
	Usage      Usage            `json:"usage"`
	Failure    string           `json:"failure,omitempty"`
	History    []HistoryEntry   `json:"history,omitempty"`
}
