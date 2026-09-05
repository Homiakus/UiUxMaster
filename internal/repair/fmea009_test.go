package repair

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/design"
	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/memory"
	"github.com/Homiakus/UiUxMaster/internal/verifier"
)

type mockL3RepairCollector struct{}

func (m *mockL3RepairCollector) Collect(_ context.Context, req engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	return repairPacketFromSource(req, "L3", "mock-independent-truthpath"), nil
}

func repairPacketFromSource(req engine.ValidationRequest, tier, name string) evidence.Packet {
	htmlStr := string(req.HTML)
	cssStr := string(req.CSS)
	packet := evidence.Packet{
		RunID: req.RunID,
		Viewport: evidence.Viewport{
			Width:       1280,
			Height:      800,
			DeviceScale: 1,
		},
		Renderer: evidence.RendererRef{Tier: tier, Name: name, Version: "test-v1"},
		Diagnostics: &evidence.DiagnosticsEvidence{Complete: true},
	}

	if strings.Contains(strings.ToLower(htmlStr), "<h1") {
		packet.Elements = append(packet.Elements, evidence.ElementRef{
			ID: "h1-main", Tag: "h1", Role: "heading", Name: "Main Page Title", Visible: true,
		})
	}
	if strings.Contains(strings.ToLower(htmlStr), "<button") {
		name := ""
		if strings.Contains(htmlStr, `aria-label="Action Button"`) {
			name = "Action Button"
		}
		packet.Elements = append(packet.Elements, evidence.ElementRef{
			ID: "cta-btn", Tag: "button", Role: "button", Name: name, Visible: true, Clickable: true,
		})
	}

	contentWidth := 1200.0
	if strings.Contains(cssStr, "width: 2000px") && !strings.Contains(cssStr, "max-width: 100vw") {
		contentWidth = 2000
	}
	packet.Documents = []evidence.DocumentMetrics{{ContentWidth: contentWidth, ContentHeight: 800}}
	return packet
}

type passHeldOutProbe struct{}

func (passHeldOutProbe) Evaluate(_ context.Context, req HeldOutEvaluationRequest) error {
	if req.Candidate.Renderer.Tier != "L3" {
		return errors.New("candidate is not L3 evidence")
	}
	return nil
}

type rejectOverflowHiddenProbe struct{}

func (rejectOverflowHiddenProbe) Evaluate(_ context.Context, req HeldOutEvaluationRequest) error {
	// Private requirement: clipping/hidden overflow is not an acceptable substitute
	// for preserving access to wide content. The optimization loop never sees this
	// predicate and therefore cannot train directly against it.
	if strings.Contains(strings.ToLower(req.CandidateCSS), "overflow-x: hidden") {
		return errors.New("private protected-content probe rejected overflow clipping")
	}
	return nil
}

type stubbornOverflowCollector struct{}

func (stubbornOverflowCollector) Collect(_ context.Context, req engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "stubborn-browser"},
		Viewport: evidence.Viewport{Width: 1280, Height: 800},
		Documents: []evidence.DocumentMetrics{{ContentWidth: 2000, ContentHeight: 800}},
		Elements: []evidence.ElementRef{
			{ID: "h1", Tag: "h1", Role: "heading", Name: "Title", Visible: true},
			{ID: "button", Tag: "button", Role: "button", Name: "Action", Visible: true, Clickable: true},
		},
	}, nil
}

type recordingComparator struct {
	inner design.Comparator
	last  design.ComparisonRequest
}

func (c *recordingComparator) Compare(ctx context.Context, req design.ComparisonRequest) (design.CandidateComparison, error) {
	c.last = req
	return c.inner.Compare(ctx, req)
}

func TestFMEA009FindingStateDigestIgnoresEphemeralRunID(t *testing.T) {
	left := []design.Finding{{
		ID: "finding:overflow:iter-1", RuleID: "RULE-RESP-001", Axis: "responsive", Category: "overflow",
		Title: "Horizontal viewport overflow by 720.0px", Severity: evidence.SeverityCritical, HardConstraint: true,
	}}
	right := []design.Finding{{
		ID: "finding:overflow:iter-2", RuleID: "RULE-RESP-001", Axis: "responsive", Category: "overflow",
		Title: "Horizontal viewport overflow by 720.0px", Severity: evidence.SeverityCritical, HardConstraint: true,
	}}
	if findingStateDigest(left) != findingStateDigest(right) {
		t.Fatalf("semantic finding state must be stable across iteration RunIDs")
	}
}

func TestFMEA009SharedCollectorIsNotIndependent(t *testing.T) {
	shared := &mockL3RepairCollector{}
	optimization := &engine.Pipeline{Collector: shared}
	finalPipeline := &engine.Pipeline{Collector: shared}
	gate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(passHeldOutProbe{}))
	if gate.IndependentFrom(optimization) {
		t.Fatalf("different Pipeline wrappers over the same collector are not independent")
	}
}

func TestFMEA009HeldOutRejectsVisibleScoreImprovement(t *testing.T) {
	optimization := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	finalPipeline := &engine.Pipeline{
		Collector: &mockL3RepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	gate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(rejectOverflowHiddenProbe{}))
	repairEngine := NewWithFinalGate(optimization, gate)
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID:         "fmea009-hidden-requirement",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography", "interaction"},
		RiskClass:     RepairRiskCritical,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if !result.CandidateImproved {
		t.Fatalf("adversarial fixture requires an optimization-score improvement")
	}
	if result.Passed {
		t.Fatalf("held-out regression must veto completion despite visible score improvement")
	}
	if !result.FinalGate.Independent {
		t.Fatalf("expected independent final pipeline to execute")
	}
	if result.FinalGate.HeldOut.Total != 1 || result.FinalGate.HeldOut.Failed != 1 {
		t.Fatalf("held-out report = %#v", result.FinalGate.HeldOut)
	}
	if result.Metrics.RegressionEscapes != 1 || result.Metrics.HeldOutEscapeRate != 1 {
		t.Fatalf("held-out metrics = %#v", result.Metrics)
	}
	if !containsString(result.FinalGate.ReasonCodes, "held_out_regression_escape") {
		t.Fatalf("final reasons = %#v", result.FinalGate.ReasonCodes)
	}
}

func TestFMEA009MemoryAdmissionRequiresIndependentPass(t *testing.T) {
	optimization := &engine.Pipeline{
		Collector: &mockRepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	finalPipeline := &engine.Pipeline{
		Collector: &mockL3RepairCollector{},
		VerPolicy: verifier.DefaultPolicy(),
	}
	gate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(passHeldOutProbe{}))
	store := memory.NewEpMemoryStore()
	repairEngine := NewWithMemoryAndFinalGate(optimization, gate, store)
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID:         "fmea009-memory-admission",
		ProjectID:     "fmea009-project",
		HTML:          faultyHTML,
		CSS:           faultyCSS,
		Profile:       design.FindProfile("saas-modern"),
		ProtectedAxes: []string{"accessibility", "responsive", "typography", "interaction"},
		RiskClass:     RepairRiskCritical,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if !result.Passed || !result.FinalGate.Independent {
		t.Fatalf("expected independent completion PASS: %#v", result.FinalGate)
	}
	if result.FinalGate.EvidenceTier != "L3" || result.FinalGate.HeldOut.Failed != 0 {
		t.Fatalf("final evidence = %#v", result.FinalGate)
	}
	if !result.MemoryAdmitted || result.AdmittedAtoms == 0 {
		t.Fatalf("memory must admit only independently proven repair, got admitted=%v atoms=%d", result.MemoryAdmitted, result.AdmittedAtoms)
	}

	ns, err := memory.NewProjectKnowledgeNamespace("fmea009-project")
	if err != nil {
		t.Fatal(err)
	}
	query, err := store.Query(context.Background(), memory.QueryRequest{Namespace: ns})
	if err != nil {
		t.Fatal(err)
	}
	if query.Total == 0 {
		t.Fatalf("expected independently proven repair lesson in project memory")
	}
}

func TestFMEA009HeldOutFailureBlocksMemoryPoisoning(t *testing.T) {
	optimization := &engine.Pipeline{Collector: &mockRepairCollector{}, VerPolicy: verifier.DefaultPolicy()}
	finalPipeline := &engine.Pipeline{Collector: &mockL3RepairCollector{}, VerPolicy: verifier.DefaultPolicy()}
	gate := NewPipelineFinalGate(finalPipeline, NewPrivateHeldOutSuite(rejectOverflowHiddenProbe{}))
	store := memory.NewEpMemoryStore()
	repairEngine := NewWithMemoryAndFinalGate(optimization, gate, store)
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID: "fmea009-memory-block", ProjectID: "fmea009-blocked", HTML: faultyHTML, CSS: faultyCSS,
		Profile: design.FindProfile("saas-modern"), RiskClass: RepairRiskCritical,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if result.Passed || result.MemoryAdmitted || result.AdmittedAtoms != 0 {
		t.Fatalf("held-out-rejected candidate must not be promoted to memory: %#v", result)
	}
	ns, err := memory.NewProjectKnowledgeNamespace("fmea009-blocked")
	if err != nil {
		t.Fatal(err)
	}
	query, err := store.Query(context.Background(), memory.QueryRequest{Namespace: ns})
	if err != nil {
		t.Fatal(err)
	}
	if query.Total != 0 {
		t.Fatalf("blocked repair leaked %d atom(s) into memory", query.Total)
	}
}

func TestFMEA009RepeatedFindingStateEscalates(t *testing.T) {
	optimization := &engine.Pipeline{Collector: stubbornOverflowCollector{}, VerPolicy: verifier.DefaultPolicy()}
	repairEngine := New(optimization)

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID: "fmea009-oscillation",
		HTML: `<html><body><h1>Title</h1><button aria-label="Action">Action</button></body></html>`,
		CSS: `body { width: 2000px; }`,
		Profile: design.FindProfile("saas-modern"),
		MaxIterations: 5,
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if result.TerminationReason != "repeated_finding_state" {
		t.Fatalf("termination = %q", result.TerminationReason)
	}
	if result.Metrics.OscillationCount == 0 || !result.EscalationRequired || result.Passed {
		t.Fatalf("oscillation must fail closed and escalate: %#v", result)
	}
}

func TestFMEA009OptimizationComparisonUsesImmutableOriginalBaseline(t *testing.T) {
	optimization := &engine.Pipeline{Collector: &mockRepairCollector{}, VerPolicy: verifier.DefaultPolicy()}
	recorder := &recordingComparator{inner: design.NewComparator()}
	repairEngine := New(optimization)
	repairEngine.Comparator = recorder
	faultyHTML, faultyCSS := repairFixture()

	result, err := repairEngine.RunRepairLoop(context.Background(), RepairLoopRequest{
		RunID: "fmea009-baseline-integrity", HTML: faultyHTML, CSS: faultyCSS,
		Profile: design.FindProfile("saas-modern"),
	})
	if err != nil {
		t.Fatalf("RunRepairLoop: %v", err)
	}
	if recorder.last.BaselineCritique == nil {
		t.Fatalf("comparator did not receive baseline critique")
	}
	if got := len(recorder.last.BaselineCritique.Findings); got != result.InitialFindings {
		t.Fatalf("baseline critique mutated: comparator=%d original=%d", got, result.InitialFindings)
	}
	if len(recorder.last.CandidateCritique.Findings) >= len(recorder.last.BaselineCritique.Findings) {
		t.Fatalf("fixture must reach an improved candidate for baseline-integrity proof")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
