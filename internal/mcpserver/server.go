package mcpserver

import (
	"context"
	"fmt"

	"github.com/Homiakus/UiUxMaster/internal/critic"
	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
	"github.com/Homiakus/UiUxMaster/internal/invalidation"
	"github.com/Homiakus/UiUxMaster/internal/repair"
	"github.com/Homiakus/UiUxMaster/internal/runtime/playwright"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.3.0"

type RubricInput struct{}

type RubricOutput struct {
	Principles []string      `json:"principles"`
	Axes       []design.Axis `json:"axes"`
}

type EvaluateEvidenceInput struct {
	Packet evidence.Packet `json:"packet" jsonschema:"normalized evidence collected from a rendered UI validation run"`
}

type EvaluateEvidenceOutput struct {
	Report engine.Report `json:"report"`
}

type PlanValidationInput struct {
	RunID          string   `json:"run_id,omitempty"`
	ChangedFiles   []string `json:"changed_files,omitempty"`
	ChangedTokens  []string `json:"changed_tokens,omitempty"`
	Intent         string   `json:"intent,omitempty"`
	FinalGate      bool     `json:"final_gate,omitempty"`
	ForceWholeSite bool     `json:"force_whole_site,omitempty"`
	TargetRoutes   []string `json:"target_routes,omitempty"`
	Viewports      []string `json:"viewports,omitempty"`
	Themes         []string `json:"themes,omitempty"`
}

type PlanValidationOutput struct {
	Scope           invalidation.ValidationScope `json:"scope"`
	Assessment      fidelity.Assessment          `json:"assessment"`
	Plan            engine.ValidationPlan        `json:"plan"`
	RecommendedTier string                       `json:"recommended_tier"`
}

type CaptureInput struct {
	RunID          string               `json:"run_id,omitempty"`
	URL            string               `json:"url,omitempty"`
	HTML           string               `json:"html,omitempty"`
	CSS            string               `json:"css,omitempty"`
	BaseURL        string               `json:"base_url,omitempty"`
	Intent         string               `json:"intent,omitempty"`
	FinalGate      bool                 `json:"final_gate,omitempty"`
	ChangedFiles   []string             `json:"changed_files,omitempty"`
	ChangedTokens  []string             `json:"changed_tokens,omitempty"`
	TargetRoutes   []string             `json:"target_routes,omitempty"`
	Viewports      []string             `json:"viewports,omitempty"`
	Themes         []string             `json:"themes,omitempty"`
	Region         *evidenceplan.Region `json:"region,omitempty"`
}

type CaptureOutput struct {
	Packet    evidence.Packet          `json:"packet"`
	Report    engine.Report            `json:"report"`
	Telemetry engine.PipelineTelemetry `json:"telemetry"`
}

type InspectLayoutInput struct {
	Packet evidence.Packet  `json:"packet"`
	Policy *verifier.Policy `json:"policy,omitempty"`
}

type InspectLayoutOutput struct {
	Issues []evidence.RuntimeIssue `json:"issues"`
	Passed bool                    `json:"passed"`
}

type InspectAccessibilityInput struct {
	Packet evidence.Packet `json:"packet"`
}

type InspectAccessibilityOutput struct {
	Issues []evidence.RuntimeIssue      `json:"issues"`
	Nodes  []evidence.AccessibilityNode `json:"nodes"`
	Passed bool                         `json:"passed"`
}

type CritiquePageInput struct {
	Packet        evidence.Packet `json:"packet"`
	Profile       string          `json:"profile,omitempty"`
	Level         string          `json:"level,omitempty"`
	ProtectedAxes []string        `json:"protected_axes,omitempty"`
}

type CritiquePageOutput struct {
	Pass       design.CritiquePass       `json:"pass"`
	Findings   []design.Finding          `json:"findings"`
	Hypotheses []design.RepairHypothesis `json:"hypotheses"`
	Passed     bool                      `json:"passed"`
}

type CompareCandidatesInput struct {
	BaselinePacket       evidence.Packet `json:"baseline_packet"`
	CandidatePacket      evidence.Packet `json:"candidate_packet"`
	ProtectedAxes        []string        `json:"protected_axes,omitempty"`
	MaxAllowedRegression float64         `json:"max_allowed_regression,omitempty"`
}

type CompareCandidatesOutput struct {
	Comparison design.CandidateComparison `json:"comparison"`
	Winner     string                     `json:"winner"`
	Passed     bool                       `json:"passed"`
}

type RecommendRepairsInput struct {
	HTML          string   `json:"html"`
	CSS           string   `json:"css"`
	Profile       string   `json:"profile,omitempty"`
	ProtectedAxes []string `json:"protected_axes,omitempty"`
	MaxIterations int      `json:"max_iterations,omitempty"`
	ProjectID     string   `json:"project_id,omitempty"`
}

type RecommendRepairsOutput struct {
	Result repair.RepairLoopResult `json:"result"`
}

type RunScenarioInput struct {
	Scenario playwright.Scenario `json:"scenario"`
}

type RunScenarioOutput struct {
	Valid       bool   `json:"valid"`
	ActionCount int    `json:"action_count"`
	Summary     string `json:"summary"`
}

// Config configures the MCP server with optional pipeline dependencies.
type Config struct {
	Pipeline     *engine.Pipeline
	Critic       *critic.LocalSemanticCritic
	Comparator   *design.RelativeComparator
	RepairEngine *repair.HostRepairEngine
}

// New creates the protocol adapter with all canonical UI/UX validation tools.
func New(cfg Config) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "uiuxmaster",
		Version: version,
	}, nil)

	pipeline := cfg.Pipeline
	if pipeline == nil {
		pipeline = &engine.Pipeline{
			VerPolicy: verifier.DefaultPolicy(),
		}
	}

	criticInst := cfg.Critic
	if criticInst == nil {
		criticInst = critic.New()
	}
	comparatorInst := cfg.Comparator
	if comparatorInst == nil {
		comparatorInst = design.NewComparator()
	}
	repairInst := cfg.RepairEngine
	if repairInst == nil {
		repairInst = repair.New(pipeline)
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_get_rubric",
		Description: "Return UiUxMaster's canonical design-quality axes and governing principles before a design critique or polish pass.",
	}, getRubric)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_evaluate_evidence",
		Description: "Synthesize grounded browser/visual evidence, identify missing verification signals, and choose the cheapest useful next validation step without inventing defects.",
	}, evaluateEvidence)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_plan_validation",
		Description: "Resolve source changes to an authoritative validation scope, calculate fidelity risk, and plan the optimal execution tier (L0/L1/L2/L3).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input PlanValidationInput) (*mcp.CallToolResult, PlanValidationOutput, error) {
		req := engine.ValidationRequest{
			RunID:          input.RunID,
			ChangedFiles:   input.ChangedFiles,
			ChangedTokens:  input.ChangedTokens,
			Intent:         evidenceplan.Intent(input.Intent),
			FinalGate:      input.FinalGate,
			ForceWholeSite: input.ForceWholeSite,
			TargetRoutes:   input.TargetRoutes,
			Viewports:      input.Viewports,
			Themes:         input.Themes,
		}

		req, scope, err := engine.PlanScope(ctx, req, pipeline.Resolver, pipeline.Policy)
		if err != nil {
			return nil, PlanValidationOutput{}, err
		}

		features := fidelity.ScanSourceFeatures(fidelity.SourceInput{
			DynamicDependencyUnresolved: scope.Widened,
		})
		fidelityCaps := fidelity.RendererCapabilities{
			Name:            pipeline.Capabilities.Name,
			BrowserAccurate: pipeline.Capabilities.BrowserAccurate,
			Supported:       make(map[fidelity.Feature]bool),
		}
		for _, fn := range pipeline.Capabilities.FeatureNames {
			fidelityCaps.Supported[fidelity.Feature(fn)] = true
		}
		assessment := fidelity.Assess(features, fidelityCaps)
		plan := engine.PlanValidationRoute(req, assessment, pipeline.Capabilities)

		return nil, PlanValidationOutput{
			Scope:           scope,
			Assessment:      assessment,
			Plan:            plan,
			RecommendedTier: string(plan.Route.Tier),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_capture",
		Description: "Execute canonical UI/UX validation pipeline from source/page change to verified evidence packet and repair recommendations.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input CaptureInput) (*mcp.CallToolResult, CaptureOutput, error) {
		targetRoutes := input.TargetRoutes
		if input.URL != "" && len(targetRoutes) == 0 {
			targetRoutes = []string{input.URL}
		}

		req := engine.ValidationRequest{
			RunID:         input.RunID,
			HTML:          []byte(input.HTML),
			CSS:           []byte(input.CSS),
			BaseURL:       input.BaseURL,
			Intent:        evidenceplan.Intent(input.Intent),
			FinalGate:     input.FinalGate,
			ChangedFiles:  input.ChangedFiles,
			ChangedTokens: input.ChangedTokens,
			TargetRoutes:  targetRoutes,
			Viewports:     input.Viewports,
			Themes:        input.Themes,
			Region:        input.Region,
		}

		res, err := pipeline.Execute(ctx, req)
		if err != nil {
			return nil, CaptureOutput{}, err
		}

		return nil, CaptureOutput{
			Packet:    res.Packet,
			Report:    res.Report,
			Telemetry: res.Telemetry,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_inspect_layout",
		Description: "Inspect layout geometry, overflow, clipping, overlap, and target size against deterministic verifier rules.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input InspectLayoutInput) (*mcp.CallToolResult, InspectLayoutOutput, error) {
		policy := verifier.DefaultPolicy()
		if input.Policy != nil {
			policy = *input.Policy
		}
		res := verifier.Verify(input.Packet, policy)

		hasBlocking := false
		for _, issue := range res.Issues {
			if issue.Severity == evidence.SeverityCritical || issue.Severity == evidence.SeverityHigh {
				hasBlocking = true
				break
			}
		}

		return nil, InspectLayoutOutput{
			Issues: res.Issues,
			Passed: !hasBlocking,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_inspect_accessibility",
		Description: "Inspect accessibility nodes, missing accessible names, interactive roles, and ARIA state anomalies.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input InspectAccessibilityInput) (*mcp.CallToolResult, InspectAccessibilityOutput, error) {
		issues := verifier.VerifyAccessibility(input.Packet)

		hasBlocking := false
		for _, issue := range issues {
			if issue.Severity == evidence.SeverityCritical || issue.Severity == evidence.SeverityHigh {
				hasBlocking = true
				break
			}
		}

		return nil, InspectAccessibilityOutput{
			Issues: issues,
			Nodes:  input.Packet.Accessibility,
			Passed: !hasBlocking,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_critique_page",
		Description: "Perform hierarchical semantic design critique over rendered UI evidence, generating grounded findings and repair hypotheses.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input CritiquePageInput) (*mcp.CallToolResult, CritiquePageOutput, error) {
		profile := design.FindProfile(input.Profile)
		if profile.ID == "" {
			profile = design.FindProfile("saas-modern")
		}
		level := design.HierarchyLevel(input.Level)
		if level == "" {
			level = design.LevelPage
		}
		pass, err := criticInst.Critique(ctx, critic.CritiqueRequest{
			Level:         level,
			Profile:       profile,
			Packet:        input.Packet,
			ProtectedAxes: input.ProtectedAxes,
		})
		if err != nil {
			return nil, CritiquePageOutput{}, err
		}
		passed := len(pass.Findings) == 0
		return nil, CritiquePageOutput{
			Pass:       pass,
			Findings:   pass.Findings,
			Hypotheses: pass.Hypotheses,
			Passed:     passed,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_compare_candidates",
		Description: "Perform pairwise A/B comparison between baseline and candidate UI revisions across 10 canonical rubric axes with hard constraint gates.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input CompareCandidatesInput) (*mcp.CallToolResult, CompareCandidatesOutput, error) {
		cmp, err := comparatorInst.Compare(ctx, design.ComparisonRequest{
			BaselinePacket:       input.BaselinePacket,
			CandidatePacket:      input.CandidatePacket,
			ProtectedAxes:        input.ProtectedAxes,
			MaxAllowedRegression: input.MaxAllowedRegression,
		})
		if err != nil {
			return nil, CompareCandidatesOutput{}, err
		}
		return nil, CompareCandidatesOutput{
			Comparison: cmp,
			Winner:     cmp.PreferredCandidate,
			Passed:     cmp.PassedConstraints,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_recommend_repairs",
		Description: "Execute autonomous closed-loop repair: generate concrete CSS/HTML patches for discovered defects and verify resolution.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input RecommendRepairsInput) (*mcp.CallToolResult, RecommendRepairsOutput, error) {
		profile := design.FindProfile(input.Profile)
		if profile.ID == "" {
			profile = design.FindProfile("saas-modern")
		}
		res, err := repairInst.RunRepairLoop(ctx, repair.RepairLoopRequest{
			HTML:          input.HTML,
			CSS:           input.CSS,
			Profile:       profile,
			ProtectedAxes: input.ProtectedAxes,
			MaxIterations: input.MaxIterations,
			ProjectID:     input.ProjectID,
		})
		if err != nil {
			return nil, RecommendRepairsOutput{}, err
		}
		return nil, RecommendRepairsOutput{
			Result: res,
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_run_scenario",
		Description: "Validate and inspect interactive user playthrough scenarios (click, fill, wait, scroll, resize) with deterministic controls.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input RunScenarioInput) (*mcp.CallToolResult, RunScenarioOutput, error) {
		err := playwright.ValidateScenario(input.Scenario)
		if err != nil {
			return nil, RunScenarioOutput{
				Valid:   false,
				Summary: fmt.Sprintf("scenario validation failed: %v", err),
			}, nil
		}
		return nil, RunScenarioOutput{
			Valid:       true,
			ActionCount: len(input.Scenario.Actions),
			Summary:     fmt.Sprintf("scenario %q with %d action(s) is valid for execution", input.Scenario.ID, len(input.Scenario.Actions)),
		}, nil
	})

	return server
}

func getRubric(context.Context, *mcp.CallToolRequest, RubricInput) (*mcp.CallToolResult, RubricOutput, error) {
	return nil, RubricOutput{
		Principles: []string{
			"code is a hypothesis; render is evidence; interaction is the result",
			"hierarchy before pixels",
			"progressive visual attention: page to section to component to element to pixels",
			"relative preference is stronger than a single absolute beauty score",
			"localize a defect before proposing a repair",
			"keep deterministic, accessibility, interaction, regression, and aesthetic verifiers independent",
			"do not optimize only for visible tests; verify user intent with perturbed scenarios",
		},
		Axes: design.DefaultRubric(),
	}, nil
}

func evaluateEvidence(_ context.Context, _ *mcp.CallToolRequest, input EvaluateEvidenceInput) (*mcp.CallToolResult, EvaluateEvidenceOutput, error) {
	return nil, EvaluateEvidenceOutput{Report: engine.Evaluate(input.Packet)}, nil
}
