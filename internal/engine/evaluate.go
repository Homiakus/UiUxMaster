package engine

import (
	"sort"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

type AxisSummary struct {
	AxisID       string            `json:"axis_id"`
	Findings     int               `json:"findings"`
	HighestLevel evidence.Severity `json:"highest_severity"`
}

type Report struct {
	RunID           string        `json:"run_id"`
	Axes             []AxisSummary `json:"axes"`
	BlockingFindings int           `json:"blocking_findings"`
	HighFindings     int           `json:"high_findings"`
	MissingEvidence  []string      `json:"missing_evidence,omitempty"`
	RecommendedNext string        `json:"recommended_next"`
}

var severityRank = map[evidence.Severity]int{
	evidence.SeverityInfo:0,evidence.SeverityLow:1,evidence.SeverityMedium:2,evidence.SeverityHigh:3,evidence.SeverityCritical:4,
}

// Evaluate preserves the original complete-deterministic synthesis contract for
// callers that do not have an explicit evidence plan.
func Evaluate(packet evidence.Packet) Report {
	return evaluate(packet, evidenceplan.Plan{
		Structural: true,
		Accessibility: true,
		Diagnostics: packet.Renderer.Tier == "L2",
	})
}

// EvaluateForPlan validates completeness against the evidence that was actually
// requested. This is critical for the sub-millisecond quick-structural loop:
// intentionally omitted AX/font evidence must not be mistaken for collection
// failure, while final/typography plans remain fail-conservative.
func EvaluateForPlan(packet evidence.Packet, plan evidenceplan.Plan) Report {
	return evaluate(packet, plan)
}

func evaluate(packet evidence.Packet, plan evidenceplan.Plan) Report {
	axisByID:=map[string]*AxisSummary{}
	for _,axis:=range design.DefaultRubric(){axisByID[axis.ID]=&AxisSummary{AxisID:axis.ID,HighestLevel:evidence.SeverityInfo}}
	report:=Report{RunID:packet.RunID}
	for _,finding:=range packet.VisualFindings{
		summary,ok:=axisByID[finding.Axis];if !ok{continue};summary.Findings++;if severityRank[finding.Severity]>severityRank[summary.HighestLevel]{summary.HighestLevel=finding.Severity}
		switch finding.Severity{case evidence.SeverityCritical:report.BlockingFindings++;case evidence.SeverityHigh:report.HighFindings++}
	}
	for _,issue:=range packet.RuntimeIssues{if issue.Severity==evidence.SeverityCritical{report.BlockingFindings++}else if issue.Severity==evidence.SeverityHigh{report.HighFindings++}}
	for _,axis:=range design.DefaultRubric(){report.Axes=append(report.Axes,*axisByID[axis.ID])}

	if plan.Accessibility&&packet.AriaSnapshot==""{report.MissingEvidence=append(report.MissingEvidence,"accessibility snapshot")}
	if plan.Structural&&len(packet.Elements)==0{report.MissingEvidence=append(report.MissingEvidence,"DOM geometry/computed-style evidence")}
	if plan.Fonts&&packet.Fonts==nil{report.MissingEvidence=append(report.MissingEvidence,"font state")}
	if plan.Diagnostics{
		if packet.Diagnostics==nil{report.MissingEvidence=append(report.MissingEvidence,"runtime diagnostics")}else if !packet.Diagnostics.Complete{report.MissingEvidence=append(report.MissingEvidence,"complete runtime diagnostics")}
	}

	hasPixels:=packet.Pixels!=nil||packet.ScreenshotPath!=""
	needsPixelInspection:=len(packet.VisualRegions)>0&&len(packet.VisualFindings)==0
	if plan.Pixels&&!hasPixels{report.MissingEvidence=append(report.MissingEvidence,"rendered region pixels")}else if needsPixelInspection&&!hasPixels{report.MissingEvidence=append(report.MissingEvidence,"rendered region pixels")}
	sort.Strings(report.MissingEvidence)

	switch{
	case report.BlockingFindings>0:report.RecommendedNext="repair blocking correctness/accessibility/runtime defects before aesthetic refinement"
	case report.HighFindings>0:report.RecommendedNext="repair grounded high-severity defects, then rerender the affected regions"
	case len(report.MissingEvidence)>0:report.RecommendedNext="collect the cheapest missing deterministic evidence before escalating fidelity"
	case needsPixelInspection:report.RecommendedNext="inspect suspicious regions with a local visual critic using cropped pixel evidence"
	default:report.RecommendedNext="deterministic evidence is clean; escalate to pixel or semantic comparison only when change risk or policy requires it"
	}
	return report
}
