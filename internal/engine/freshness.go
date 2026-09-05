package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Homiakus/UiUxMaster/internal/evidence"
)

var ErrRevisionAttestation = errors.New("engine: render revision attestation failed")

// ValidateRevisionAttestation is the canonical pre-verifier guard for collectors
// that provide render freshness provenance. FastCDP packets with a requested
// SourceDigest must carry a matching expected/observed revision and epoch.
func ValidateRevisionAttestation(req ValidationRequest, packet evidence.Packet) error {
	expected := strings.TrimSpace(req.SourceDigest)
	freshness := packet.Freshness

	if freshness == nil {
		if expected != "" && strings.EqualFold(strings.TrimSpace(packet.Renderer.Name), "fastcdp") {
			return fmt.Errorf("%w: FastCDP packet omitted freshness for requested revision %q", ErrRevisionAttestation, expected)
		}
		return nil
	}
	if freshness.Epoch != packet.Epoch {
		return fmt.Errorf("%w: packet epoch=%d freshness epoch=%d", ErrRevisionAttestation, packet.Epoch, freshness.Epoch)
	}
	declaredExpected := strings.TrimSpace(freshness.ExpectedRevision)
	observed := strings.TrimSpace(freshness.ObservedRevision)
	if expected != "" && declaredExpected != expected {
		return fmt.Errorf("%w: request expected=%q packet expected=%q", ErrRevisionAttestation, expected, declaredExpected)
	}
	if declaredExpected != "" && observed != declaredExpected {
		return fmt.Errorf("%w: expected=%q observed=%q epoch=%d", ErrRevisionAttestation, declaredExpected, observed, freshness.Epoch)
	}
	if expected != "" && observed != expected {
		return fmt.Errorf("%w: request expected=%q observed=%q epoch=%d", ErrRevisionAttestation, expected, observed, freshness.Epoch)
	}
	return nil
}
