package fastcdp

import "github.com/Homiakus/UiUxMaster/internal/evidenceplan"

type PlannedRequestOptions struct {
	RequireAfter uint64
	WaitForNewEpoch bool
	DiagnosticsSince *DiagnosticMark
	MaxEpochRetries int
}

// RequestFromPlan is the FastCDP adapter for the vendor-neutral evidence plan.
// Diagnostics are included only when a cycle mark is supplied; callers should
// mark the warm page immediately before applying the change under validation.
func RequestFromPlan(plan evidenceplan.Plan, options PlannedRequestOptions) EvidenceRequest {
	req:=EvidenceRequest{RequireAfter:options.RequireAfter,WaitForNewEpoch:options.WaitForNewEpoch,MaxEpochRetries:options.MaxEpochRetries}
	if plan.Structural { snapshot:=DefaultSnapshotOptions(); req.Snapshot=&snapshot }
	if plan.Accessibility { req.Accessibility=true }
	if plan.Fonts { req.Fonts=true }
	if plan.Diagnostics { req.DiagnosticsSince=options.DiagnosticsSince }
	if plan.Pixels {
		// Region coordinates are intentionally carried by the plan so browser
		// adapters never infer a full-page screenshot when only one ROI is needed.
		// A missing region remains nil and is rejected by the caller/router rather
		// than silently escalating to whole-page pixels.
		//
		// This branch is completed below only for a concrete planned region.
	}
	return req
}

func RequestFromSignals(signals evidenceplan.Signals, options PlannedRequestOptions) EvidenceRequest {
	plan:=evidenceplan.Build(signals)
	req:=RequestFromPlan(plan,options)
	if plan.Pixels && signals.Region!=nil {
		scale:=signals.Region.Scale;if scale==0{scale=1}
		req.Region=&CaptureRegionOptions{X:signals.Region.X,Y:signals.Region.Y,Width:signals.Region.Width,Height:signals.Region.Height,Scale:scale,OptimizeForSpeed:true}
	}
	return req
}
