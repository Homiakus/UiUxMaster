package fastcdp

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"
)

var ErrEpochChanged = errors.New("fastcdp: render token changed during evidence capture")

type EvidenceRequest struct {
	RequireAfter       uint64
	ExpectedRevision   string
	WaitForNewEpoch    bool
	Snapshot           *SnapshotOptions
	Region             *CaptureRegionOptions
	Accessibility      bool
	AccessibilityDepth int
	Fonts              bool
	MaxFonts           int
	DiagnosticsSince   *DiagnosticMark
	MaxEpochRetries    int
}

type EvidenceTiming struct {
	WaitEpoch     time.Duration
	Snapshot      time.Duration
	Pixels        time.Duration
	Accessibility time.Duration
	Fonts         time.Duration
	Diagnostics   time.Duration
	Total         time.Duration
	Retries       int
}

type CollectedEvidence struct {
	Epoch         uint64
	Revision      string
	Snapshot      *Snapshot
	RGBA          *image.RGBA
	CaptureStats  CaptureStats
	Accessibility *AXTree
	Fonts         *FontState
	Diagnostics   *DiagnosticSnapshot
	Timing        EvidenceTiming
}

func (p *WarmPage) CollectEvidence(ctx context.Context, conn *Connection, req EvidenceRequest) (CollectedEvidence, error) {
	if p == nil || p.Epoch == nil || p.Session.SessionID == "" {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: warm page is not ready for evidence collection")
	}
	if conn == nil {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: evidence collection requires connection")
	}
	if req.Snapshot == nil && req.Region == nil && !req.Accessibility && !req.Fonts && req.DiagnosticsSince == nil {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: evidence request is empty")
	}
	if req.DiagnosticsSince != nil && p.Diagnostics == nil {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: diagnostics were requested but disabled for this warm page")
	}

	retries := req.MaxEpochRetries
	if retries == 0 {
		retries = 2
	} else if retries < 0 {
		retries = 0
	}
	started := time.Now()
	var timing EvidenceTiming
	if req.WaitForNewEpoch {
		waitStarted := time.Now()
		if _, err := p.Epoch.WaitAfterRevision(ctx, req.RequireAfter, req.ExpectedRevision); err != nil {
			return CollectedEvidence{}, err
		}
		timing.WaitEpoch = time.Since(waitStarted)
	} else {
		if _, err := p.Epoch.ValidateCurrentRevision(req.RequireAfter, req.ExpectedRevision); err != nil {
			return CollectedEvidence{}, err
		}
	}

	for attempt := 0; attempt <= retries; attempt++ {
		tokenBefore := p.Epoch.CurrentToken()
		if req.ExpectedRevision != "" && tokenBefore.Revision != req.ExpectedRevision {
			return CollectedEvidence{}, &RevisionMismatchError{Expected: req.ExpectedRevision, Observed: tokenBefore}
		}
		result, err := p.captureStableAttempt(ctx, conn, req)
		if err != nil {
			return CollectedEvidence{}, err
		}
		tokenAfter := p.Epoch.CurrentToken()
		accumulateTiming(&timing, result.Timing)
		if tokenAfter == tokenBefore {
			if req.ExpectedRevision != "" && tokenAfter.Revision != req.ExpectedRevision {
				return CollectedEvidence{}, &RevisionMismatchError{Expected: req.ExpectedRevision, Observed: tokenAfter}
			}
			result.Epoch = tokenAfter.Epoch
			result.Revision = tokenAfter.Revision
			timing.Retries = attempt
			timing.Total = time.Since(started)
			result.Timing = timing
			return result, nil
		}
		if attempt == retries {
			timing.Retries = attempt
			timing.Total = time.Since(started)
			return CollectedEvidence{}, fmt.Errorf("%w: before=%+v after=%+v retries=%d", ErrEpochChanged, tokenBefore, tokenAfter, attempt)
		}
		if req.ExpectedRevision != "" && tokenAfter.Revision != req.ExpectedRevision {
			return CollectedEvidence{}, &RevisionMismatchError{Expected: req.ExpectedRevision, Observed: tokenAfter}
		}
	}
	panic("unreachable")
}

func accumulateTiming(dst *EvidenceTiming, src EvidenceTiming) {
	dst.Snapshot += src.Snapshot
	dst.Pixels += src.Pixels
	dst.Accessibility += src.Accessibility
	dst.Fonts += src.Fonts
	dst.Diagnostics += src.Diagnostics
}

func (p *WarmPage) captureStableAttempt(ctx context.Context, conn *Connection, req EvidenceRequest) (CollectedEvidence, error) {
	var result CollectedEvidence
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
	}

	if req.Snapshot != nil {
		options := *req.Snapshot
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			snapshot, err := conn.CaptureSnapshot(ctx, string(p.Session.SessionID), options)
			duration := time.Since(started)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			result.Snapshot = &snapshot
			result.Timing.Snapshot = duration
			mu.Unlock()
		}()
	}
	if req.Region != nil {
		options := *req.Region
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			img, stats, err := conn.CaptureRegionRGBA(ctx, string(p.Session.SessionID), options)
			duration := time.Since(started)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			result.RGBA = img
			result.CaptureStats = stats
			result.Timing.Pixels = duration
			mu.Unlock()
		}()
	}
	if req.Accessibility {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			tree, err := conn.CaptureAXTree(ctx, string(p.Session.SessionID), req.AccessibilityDepth)
			duration := time.Since(started)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			result.Accessibility = &tree
			result.Timing.Accessibility = duration
			mu.Unlock()
		}()
	}
	if req.Fonts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			started := time.Now()
			fonts, err := conn.CaptureFontState(ctx, string(p.Session.SessionID), req.MaxFonts)
			duration := time.Since(started)
			if err != nil {
				setErr(err)
				return
			}
			mu.Lock()
			result.Fonts = &fonts
			result.Timing.Fonts = duration
			mu.Unlock()
		}()
	}

	wg.Wait()
	mu.Lock()
	err := firstErr
	mu.Unlock()
	if err != nil {
		return CollectedEvidence{}, err
	}

	if req.DiagnosticsSince != nil {
		started := time.Now()
		if err := p.Diagnostics.Barrier(ctx, conn); err != nil {
			return CollectedEvidence{}, err
		}
		through := p.Diagnostics.Mark()
		diagnostics := p.Diagnostics.SnapshotRange(*req.DiagnosticsSince, through)
		result.Diagnostics = &diagnostics
		result.Timing.Diagnostics = time.Since(started)
	}
	return result, nil
}
