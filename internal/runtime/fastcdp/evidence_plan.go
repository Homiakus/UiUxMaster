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
	req := EvidenceRequest{RequireAfter: options.RequireAfter, WaitForNewEpoch: options.WaitForNewEpoch, MaxEpochRetries: options.MaxEpochRetries}
	if plan.Structural {
		snapshot := DefaultSnapshotOptions()
		req.Snapshot = &snapshot
	}
	if plan.Accessibility {
		req.Accessibility = true
	}
	if plan.Fonts {
		req.Fonts = true
	}
	if plan.Diagnostics {
		req.DiagnosticsSince = options.DiagnosticsSince
	}
	if plan.Pixels && plan.Region != nil {
		scale := plan.Region.Scale
		if scale == 0 {
			scale = 1
		}
		req.Region = &CaptureRegionOptions{
			X:                plan.Region.X,
			Y:                plan.Region.Y,
			Width:            plan.Region.Width,
			Height:           plan.Region.Height,
			Scale:            scale,
			OptimizeForSpeed: true,
		}
	}
	return req
}

func RequestFromSignals(signals evidenceplan.Signals, options PlannedRequestOptions) EvidenceRequest {
	plan := evidenceplan.Build(signals)
	return RequestFromPlan(plan, options)
}
