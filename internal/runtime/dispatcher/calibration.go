package dispatcher

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/engine"
	"github.com/Homiakus/UiUxMaster/internal/evidence"
	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

// calibrationEnvironmentProvider is intentionally tiny so runtime adapters can
// attest exact identity without exposing vendor-specific types to engine.
type calibrationEnvironmentProvider interface {
	CalibrationEnvironment(context.Context) (fidelity.CalibrationEnvironment, error)
}

var _ engine.CalibrationContextProvider = (*Dispatcher)(nil)

// CalibrationContext binds the actual approximate renderer that produced packet
// to the exact configured TruthPath oracle. Any missing identity fails closed at
// PassAuthority rather than silently treating an old parity result as current.
func (d *Dispatcher) CalibrationContext(ctx context.Context, req engine.ValidationRequest, _ engine.ValidationPlan, packet evidence.Packet) (fidelity.CalibrationContext, error) {
	if d == nil {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: calibration context requires dispatcher")
	}
	strength, err := engine.PacketEvidenceStrength(packet.Renderer.Tier)
	if err != nil {
		return fidelity.CalibrationContext{}, err
	}
	if strength != engine.StrengthFastRender && strength != engine.StrengthFastBrowser {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: calibration context only applies to L1/L2, got %q", packet.Renderer.Tier)
	}

	var approx fidelity.CalibrationEnvironment
	switch strength {
	case engine.StrengthFastRender:
		if d.l1 == nil {
			return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: L1 renderer unavailable for calibration identity")
		}
		caps := d.l1.Capabilities()
		approx = fidelity.CalibrationEnvironment{
			RendererName:    firstNonEmpty(packet.Renderer.Name, caps.Name),
			RendererVersion: firstNonEmpty(packet.Renderer.Version, caps.Version),
			FidelityID:      packet.Renderer.FidelityID,
			RuntimeVersion:  caps.Version,
		}
	case engine.StrengthFastBrowser:
		if provider, ok := d.l2.(calibrationEnvironmentProvider); ok {
			approx, err = provider.CalibrationEnvironment(ctx)
			if err != nil {
				return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: L2 calibration identity: %w", err)
			}
		} else {
			approx = fidelity.CalibrationEnvironment{
				RendererName:    packet.Renderer.Name,
				RendererVersion: packet.Renderer.Version,
				FidelityID:      packet.Renderer.FidelityID,
			}
		}
	}

	provider, ok := d.l3.(calibrationEnvironmentProvider)
	if !ok || provider == nil {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: TruthPath calibration identity provider unavailable")
	}
	truth, err := provider.CalibrationEnvironment(ctx)
	if err != nil {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: TruthPath calibration identity: %w", err)
	}

	platform := runtime.GOOS + "/" + runtime.GOARCH
	if approx.Platform == "" {
		approx.Platform = platform
	}
	if truth.Platform == "" {
		truth.Platform = platform
	}
	approx.ViewportWidth = packet.Viewport.Width
	approx.ViewportHeight = packet.Viewport.Height
	approx.DeviceScale = packet.Viewport.DeviceScale
	approx.ColorScheme = packet.Viewport.ColorScheme
	truth.ViewportWidth = packet.Viewport.Width
	truth.ViewportHeight = packet.Viewport.Height
	truth.DeviceScale = packet.Viewport.DeviceScale
	truth.ColorScheme = packet.Viewport.ColorScheme
	if len(req.Themes) > 0 {
		approx.ColorScheme = req.Themes[0]
		truth.ColorScheme = req.Themes[0]
	}

	if err := approx.Validate(); err != nil {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: approximate calibration identity invalid: %w", err)
	}
	if err := truth.Validate(); err != nil {
		return fidelity.CalibrationContext{}, fmt.Errorf("dispatcher: TruthPath calibration identity invalid: %w", err)
	}
	return fidelity.CalibrationContext{Approx: approx, Truth: truth}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
