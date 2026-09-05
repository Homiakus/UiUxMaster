package engine_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

type weakTruthPathCollector struct{}

func (weakTruthPathCollector) Collect(_ context.Context, req engine.ValidationRequest, _ engine.ValidationPlan) (evidence.Packet, error) {
	return evidence.Packet{
		RunID: req.RunID,
		Renderer: evidence.RendererRef{Tier: "L2", Name: "weak-custom-collector"},
	}, nil
}

func TestPipelineRejectsWeakCustomCollectorBeforeVerifier(t *testing.T) {
	pipeline := engine.Pipeline{Collector: weakTruthPathCollector{}}
	_, err := pipeline.Execute(context.Background(), engine.ValidationRequest{
		RunID: "pipeline-required-l3",
		Need:  engine.EvidenceNeed{CleanState: true},
	})
	if !errors.Is(err, engine.ErrInsufficientEvidenceTier) {
		t.Fatalf("Pipeline.Execute error = %v, want ErrInsufficientEvidenceTier", err)
	}
}
