package mcpserver

import (
	"context"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const version = "0.1.0-dev"

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

// New creates the protocol adapter. Domain logic deliberately lives outside
// this package so UiUxMaster can also be reused by CLI/CI and future transports.
func New() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "uiuxmaster",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_get_rubric",
		Description: "Return UiUxMaster's canonical design-quality axes and governing principles before a design critique or polish pass.",
	}, getRubric)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "uiux_evaluate_evidence",
		Description: "Synthesize grounded browser/visual evidence, identify missing verification signals, and choose the cheapest useful next validation step without inventing defects.",
	}, evaluateEvidence)

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
