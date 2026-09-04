package fidelity_test

import (
	"errors"
	"testing"

	"github.com/Homiakus/UiUxMaster/internal/fidelity"
)

func TestCalibrationMatrix_StaticLayout(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	// Normal run
	if !matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassStaticLayout, false) {
		t.Errorf("expected L1 to legally pass static layout in normal run")
	}
	if !matrix.CanLegallyPass(fidelity.TierL2, fidelity.EvidenceClassStaticLayout, false) {
		t.Errorf("expected L2 to legally pass static layout")
	}
	if !matrix.CanLegallyPass(fidelity.TierL3, fidelity.EvidenceClassStaticLayout, false) {
		t.Errorf("expected L3 to legally pass static layout")
	}

	// Final gate run: L1 prohibited, L2/L3 required
	if matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassStaticLayout, true) {
		t.Errorf("expected L1 to be prohibited on final gate")
	}
	if !matrix.CanLegallyPass(fidelity.TierL2, fidelity.EvidenceClassStaticLayout, true) {
		t.Errorf("expected L2 to legally pass static layout on final gate")
	}
}

func TestCalibrationMatrix_Typography(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	// L1 is always prohibited from typography pass
	if matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassTypography, false) {
		t.Errorf("L1 must not legally pass typography")
	}
	if matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassTypography, true) {
		t.Errorf("L1 must not legally pass typography on final gate")
	}

	// L2 and L3 can pass
	if !matrix.CanLegallyPass(fidelity.TierL2, fidelity.EvidenceClassTypography, false) {
		t.Errorf("L2 should pass typography")
	}
	if !matrix.CanLegallyPass(fidelity.TierL3, fidelity.EvidenceClassTypography, false) {
		t.Errorf("L3 should pass typography")
	}
}

func TestCalibrationMatrix_Interactive(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	// L1 is always prohibited from interactive pass
	if matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassInteractive, false) {
		t.Errorf("L1 must not legally pass interactive")
	}

	// L2 and L3 can pass
	if !matrix.CanLegallyPass(fidelity.TierL2, fidelity.EvidenceClassInteractive, false) {
		t.Errorf("L2 should pass interactive")
	}
	if !matrix.CanLegallyPass(fidelity.TierL3, fidelity.EvidenceClassInteractive, true) {
		t.Errorf("L3 should pass interactive on final gate")
	}
}

func TestCalibrationMatrix_CrossBrowserRelease(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	// Only L3 may pass cross-browser release
	if matrix.CanLegallyPass(fidelity.TierL0, fidelity.EvidenceClassCrossBrowserRelease, false) {
		t.Errorf("L0 must not pass cross-browser release")
	}
	if matrix.CanLegallyPass(fidelity.TierL1, fidelity.EvidenceClassCrossBrowserRelease, false) {
		t.Errorf("L1 must not pass cross-browser release")
	}
	if matrix.CanLegallyPass(fidelity.TierL2, fidelity.EvidenceClassCrossBrowserRelease, false) {
		t.Errorf("L2 must not pass cross-browser release")
	}
	if !matrix.CanLegallyPass(fidelity.TierL3, fidelity.EvidenceClassCrossBrowserRelease, false) {
		t.Errorf("L3 must legally pass cross-browser release")
	}
}

func TestCalibrationMatrix_ValidateLegalPass(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	// Legal pass
	err := matrix.ValidateLegalPass(fidelity.TierL2, fidelity.EvidenceClassTypography, false)
	if err != nil {
		t.Fatalf("unexpected error on legal pass: %v", err)
	}

	// Illegal pass
	err = matrix.ValidateLegalPass(fidelity.TierL1, fidelity.EvidenceClassTypography, false)
	if err == nil {
		t.Fatal("expected ErrIllegalPass, got nil")
	}
	if !errors.Is(err, fidelity.ErrIllegalPass) {
		t.Fatalf("expected ErrIllegalPass, got %v", err)
	}
}

func TestCalibrationCorpus_EscalationTiers(t *testing.T) {
	matrix := fidelity.DefaultCalibrationMatrix()

	corpus := []struct {
		class     fidelity.EvidenceClass
		finalGate bool
		wantTier  fidelity.Tier
	}{
		{fidelity.EvidenceClassStaticLayout, false, fidelity.TierL1},
		{fidelity.EvidenceClassStaticLayout, true, fidelity.TierL2},
		{fidelity.EvidenceClassTypography, false, fidelity.TierL2},
		{fidelity.EvidenceClassTypography, true, fidelity.TierL2},
		{fidelity.EvidenceClassInteractive, false, fidelity.TierL2},
		{fidelity.EvidenceClassInteractive, true, fidelity.TierL3},
		{fidelity.EvidenceClassCrossBrowserRelease, false, fidelity.TierL3},
		{fidelity.EvidenceClassCrossBrowserRelease, true, fidelity.TierL3},
	}

	for _, tc := range corpus {
		got := matrix.RequiredEscalationTier(tc.class, tc.finalGate)
		if got != tc.wantTier {
			t.Errorf("RequiredEscalationTier(%s, finalGate=%v) = %s, want %s", tc.class, tc.finalGate, got, tc.wantTier)
		}
	}
}
