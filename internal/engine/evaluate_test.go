package engine

import (
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/evidenceplan"
)

func TestEvaluateRequestsCheapEvidenceBeforePixels(t *testing.T) {
	report := Evaluate(evidence.Packet{RunID:"run-1"})
	if len(report.MissingEvidence)!=2{t.Fatalf("expected accessibility + structural evidence only, got %v",report.MissingEvidence)}
	for _,missing:=range report.MissingEvidence{if missing=="rendered screenshot"||missing=="rendered region pixels"{t.Fatalf("clean structural path must not demand pixels: %v",report.MissingEvidence)}}
	if report.RecommendedNext!="collect the cheapest missing deterministic evidence before escalating fidelity"{t.Fatalf("unexpected next step: %q",report.RecommendedNext)}
}

func TestEvaluateForQuickStructuralDoesNotRequireAXOrFonts(t *testing.T) {
	packet:=evidence.Packet{
		Renderer:evidence.RendererRef{Tier:"L2"},
		Elements:[]evidence.ElementRef{{ID:"main",Tag:"main",Visible:true}},
		Diagnostics:&evidence.DiagnosticsEvidence{Complete:true},
	}
	report:=EvaluateForPlan(packet,evidenceplan.Build(evidenceplan.Signals{Intent:evidenceplan.IntentQuickStructural}))
	if len(report.MissingEvidence)!=0{t.Fatalf("quick structural unexpectedly requires optional evidence: %v",report.MissingEvidence)}
}

func TestEvaluateForTypographyRequiresAXFontsAndDiagnostics(t *testing.T) {
	packet:=evidence.Packet{Renderer:evidence.RendererRef{Tier:"L2"},Elements:[]evidence.ElementRef{{ID:"body",Tag:"body",Visible:true}}}
	plan:=evidenceplan.Build(evidenceplan.Signals{Intent:evidenceplan.IntentTypography})
	report:=EvaluateForPlan(packet,plan)
	want:=map[string]bool{"accessibility snapshot":true,"font state":true,"runtime diagnostics":true}
	if len(report.MissingEvidence)!=3{t.Fatalf("missing = %v",report.MissingEvidence)}
	for _,item:=range report.MissingEvidence{if !want[item]{t.Fatalf("unexpected missing evidence %q in %v",item,report.MissingEvidence)}}
}

func TestEvaluateCleanDeterministicPacketDoesNotRequireScreenshot(t *testing.T) {
	report:=Evaluate(evidence.Packet{RunID:"clean",AriaSnapshot:"- button: Publish",Elements:[]evidence.ElementRef{{ID:"publish",Role:"button",Name:"Publish",Visible:true}}})
	if len(report.MissingEvidence)!=0{t.Fatalf("unexpected missing evidence: %v",report.MissingEvidence)}
	if report.RecommendedNext!="deterministic evidence is clean; escalate to pixel or semantic comparison only when change risk or policy requires it"{t.Fatalf("unexpected next step: %q",report.RecommendedNext)}
}

func TestEvaluateL2RequiresRuntimeDiagnostics(t *testing.T) {
	packet:=evidence.Packet{Renderer:evidence.RendererRef{Tier:"L2"},AriaSnapshot:"- button: Publish",Elements:[]evidence.ElementRef{{ID:"publish",Role:"button",Visible:true}}}
	report:=Evaluate(packet)
	if len(report.MissingEvidence)!=1||report.MissingEvidence[0]!="runtime diagnostics"{t.Fatalf("missing = %v",report.MissingEvidence)}
	packet.Diagnostics=&evidence.DiagnosticsEvidence{Complete:false,DroppedMethods:[]string{"Runtime.consoleAPICalled"}}
	report=Evaluate(packet)
	if len(report.MissingEvidence)!=1||report.MissingEvidence[0]!="complete runtime diagnostics"{t.Fatalf("missing = %v",report.MissingEvidence)}
	packet.Diagnostics=&evidence.DiagnosticsEvidence{Complete:true}
	report=Evaluate(packet)
	if len(report.MissingEvidence)!=0{t.Fatalf("complete L2 evidence still missing: %v",report.MissingEvidence)}
}

func TestEvaluatePrioritizesBlockingDefects(t *testing.T) {
	report:=Evaluate(evidence.Packet{RunID:"run-2",AriaSnapshot:"- button: Publish",Elements:[]evidence.ElementRef{{ID:"publish",Role:"button",Name:"Publish",Visible:true}},RuntimeIssues:[]evidence.RuntimeIssue{{Code:"page-error",Message:"uncaught exception",Severity:evidence.SeverityCritical}}})
	if report.BlockingFindings!=1{t.Fatalf("expected one blocking finding, got %d",report.BlockingFindings)}
	if report.RecommendedNext!="repair blocking correctness/accessibility/runtime defects before aesthetic refinement"{t.Fatalf("unexpected next step: %q",report.RecommendedNext)}
}

func TestEvaluateRequestsPixelsOnlyForLocalizedVisualRegion(t *testing.T) {
	report:=Evaluate(evidence.Packet{RunID:"run-3",AriaSnapshot:"- heading: Example",Elements:[]evidence.ElementRef{{ID:"hero",Role:"heading",Name:"Example",Visible:true}},VisualRegions:[]evidence.VisualRegion{{ID:"region-1",DiffRatio:0.12}}})
	if len(report.MissingEvidence)!=1||report.MissingEvidence[0]!="rendered region pixels"{t.Fatalf("missing evidence = %v",report.MissingEvidence)}
}

func TestEvaluateEscalatesLocalizedUnexplainedRegionsWithPixels(t *testing.T) {
	report:=Evaluate(evidence.Packet{RunID:"run-4",AriaSnapshot:"- heading: Example",Elements:[]evidence.ElementRef{{ID:"hero",Role:"heading",Name:"Example",Visible:true}},VisualRegions:[]evidence.VisualRegion{{ID:"region-1",DiffRatio:0.12}},Pixels:&evidence.PixelEvidence{Width:320,Height:180,DigestSHA256:"abc"}})
	if report.RecommendedNext!="inspect suspicious regions with a local visual critic using cropped pixel evidence"{t.Fatalf("unexpected next step: %q",report.RecommendedNext)}
}
