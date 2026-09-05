package dispatcher

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/runtime/fastrender"
	"github.com/Homiakus/UiUxMaster/internal/visualdiff"
)

var protectedBaselineComparator = visualdiff.NewComparator()

func compareProtectedBaseline(packet *evidence.Packet, req engine.ValidationRequest, caps fastrender.Capabilities, ev fastrender.Evidence, theme string) error {
	if req.BaselineRGBA == nil {
		return nil
	}
	if req.BaselineEnvironment == nil {
		return fmt.Errorf("%w: baseline environment identity is required", visualdiff.ErrBaselineIncompatible)
	}
	if ev.RGBA == nil {
		return fmt.Errorf("visualdiff: candidate renderer returned no RGBA")
	}
	if strings.TrimSpace(theme) == "" {
		theme = "default"
	}

	candidateEnv := evidence.RenderEnvironmentIdentity{
		SchemaVersion:   evidence.RenderEnvironmentSchemaVersion,
		RendererName:    caps.Name,
		RendererVersion: caps.Version,
		WorkerVersion:   "in-process:" + caps.Version,
		BrowserFamily:   "synthetic",
		BrowserEngine:   caps.Name,
		BrowserVersion:  caps.Version,
		Platform:        runtime.GOOS + "/" + runtime.GOARCH,
		ViewportWidth:   packet.Viewport.Width,
		ViewportHeight:  packet.Viewport.Height,
		DeviceScale:     packet.Viewport.DeviceScale,
		Theme:           theme,
		FontSetDigest:   req.FontSetDigest,
		Locale:          req.Locale,
		Timezone:        req.Timezone,
		FixtureRevision: req.FixtureRevision,
	}

	baselineKey, err := req.BaselineEnvironment.CanonicalKey()
	if err != nil {
		return fmt.Errorf("%w: baseline identity invalid: %v", visualdiff.ErrBaselineIncompatible, err)
	}
	baselineID := strings.TrimSpace(req.BaselineID)
	if baselineID == "" {
		baselineID = "inline-protected-baseline"
	}
	ref := visualdiff.BaselineReference{
		ID:             baselineID,
		Environment:    req.BaselineEnvironment.Normalize(),
		EnvironmentKey: baselineKey,
		DigestSHA256:   strings.TrimSpace(req.BaselineDigest),
		Protected:      true,
	}

	// L1 boxes are the current semantic ownership evidence available to masks.
	// They are projected locally for gate validation; packet element population
	// remains owned by the existing collector path.
	elements := make([]evidence.ElementRef, 0, len(ev.Boxes))
	for i, box := range ev.Boxes {
		elements = append(elements, evidence.ElementRef{
			ID:      fmt.Sprintf("box-%d", i),
			Role:    box.Kind,
			Visible: true,
			Bounds: evidence.Rect{
				X:      float64(box.Bounds.Min.X),
				Y:      float64(box.Bounds.Min.Y),
				Width:  float64(box.Bounds.Dx()),
				Height: float64(box.Bounds.Dy()),
			},
		})
	}

	cmp, err := protectedBaselineComparator.CompareBaseline(visualdiff.ComparisonRequest{
		Baseline:             ref,
		BaselineImage:        req.BaselineRGBA,
		CandidateEnvironment: candidateEnv,
		CandidateImage:       ev.RGBA,
		Elements:             elements,
		Masks:                req.BaselineMasks,
		Options:              visualdiff.Options{ChannelTolerance: req.Tolerance},
	})
	if err != nil {
		return err
	}

	attestedEnv := candidateEnv.Normalize()
	packet.Environment = &attestedEnv
	packet.RuntimeIssues = append(packet.RuntimeIssues, evidence.RuntimeIssue{
		Code:     "BASELINE_ENVIRONMENT_ATTESTED",
		Message:  "protected visual baseline environment matched candidate before pixel comparison",
		Severity: evidence.SeverityInfo,
		Details: map[string]string{
			"baseline_id":                baselineID,
			"baseline_environment_key":   cmp.BaselineEnvironmentKey,
			"candidate_environment_key":  cmp.CandidateEnvironmentKey,
			"comparison_digest":          cmp.ComparisonDigest,
			"masked_pixels":              fmt.Sprintf("%d", cmp.PixelResult.MaskedPixels),
		},
	})

	diffRes := cmp.PixelResult
	if diffRes.ChangedPixels == 0 {
		return nil
	}
	diffBounds := evidence.Rect{
		X:      float64(diffRes.Bounds.Min.X),
		Y:      float64(diffRes.Bounds.Min.Y),
		Width:  float64(diffRes.Bounds.Dx()),
		Height: float64(diffRes.Bounds.Dy()),
	}
	packet.VisualRegions = append(packet.VisualRegions, evidence.VisualRegion{
		ID:            "visualdiff-changed-roi",
		Bounds:        diffBounds,
		ChangedPixels: int64(diffRes.ChangedPixels),
		DiffRatio:     diffRes.ChangeRatio,
	})
	packet.VisualFindings = append(packet.VisualFindings, evidence.VisualFinding{
		ID:          fmt.Sprintf("finding:visualdiff:%s", req.RunID),
		Axis:        "visual_regression",
		Title:       "Visual difference detected against compatible protected baseline",
		Description: fmt.Sprintf("%d changed pixels (ratio: %.4f, max delta: %d)", diffRes.ChangedPixels, diffRes.ChangeRatio, diffRes.MaxDelta),
		Severity:    evidence.SeverityMedium,
		Confidence:  1.0,
		Source:      "pixel_diff",
		RegionID:    "visualdiff-changed-roi",
		Evidence:    []string{cmp.BaselineEnvironmentKey, cmp.ComparisonDigest},
	})
	return nil
}
