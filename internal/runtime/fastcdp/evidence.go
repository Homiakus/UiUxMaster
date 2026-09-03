package fastcdp

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"time"
)

var ErrEpochChanged = errors.New("fastcdp: render epoch changed during evidence capture")

// EvidenceRequest describes the minimum browser evidence required by a
// verifier. Snapshot and Region are optional independently, but at least one
// must be requested. RequireAfter is the last accepted render epoch; when
// WaitForNewEpoch is true the collector blocks until a strictly newer epoch is
// observed before capture begins. MaxEpochRetries defaults to 2; a negative
// value disables retry and fails immediately on an in-flight epoch change.
type EvidenceRequest struct {
	RequireAfter    uint64
	WaitForNewEpoch bool
	Snapshot        *SnapshotOptions
	Region          *CaptureRegionOptions
	MaxEpochRetries int
}

type EvidenceTiming struct {
	WaitEpoch time.Duration
	Snapshot  time.Duration
	Pixels    time.Duration
	Total     time.Duration
	Retries   int
}

type CollectedEvidence struct {
	Epoch        uint64
	Snapshot     *Snapshot
	RGBA         *image.RGBA
	CaptureStats CaptureStats
	Timing       EvidenceTiming
}

// CollectEvidence captures all requested evidence against one stable render
// epoch. Snapshot and pixel capture are issued concurrently because Connection
// safely multiplexes CDP requests. If a newer render epoch arrives while the
// operations are in flight, every result from that attempt is discarded and a
// bounded retry starts from the newer state.
func (p *WarmPage) CollectEvidence(ctx context.Context, conn *Connection, req EvidenceRequest) (CollectedEvidence, error) {
	if p == nil || p.Epoch == nil || p.Session.SessionID == "" {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: warm page is not ready for evidence collection")
	}
	if conn == nil {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: evidence collection requires connection")
	}
	if req.Snapshot == nil && req.Region == nil {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: evidence request must include snapshot and/or region")
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
		if _, err := p.Epoch.WaitAfter(ctx, req.RequireAfter); err != nil {
			return CollectedEvidence{}, err
		}
		timing.WaitEpoch = time.Since(waitStarted)
	} else if current := p.Epoch.Current(); current < req.RequireAfter {
		return CollectedEvidence{}, fmt.Errorf("fastcdp: current render epoch %d is older than required %d", current, req.RequireAfter)
	}

	for attempt := 0; attempt <= retries; attempt++ {
		epochBefore := p.Epoch.Current()
		result, err := p.captureStableAttempt(ctx, conn, req)
		if err != nil {
			return CollectedEvidence{}, err
		}
		epochAfter := p.Epoch.Current()
		if epochAfter == epochBefore {
			result.Epoch = epochAfter
			timing.Snapshot += result.Timing.Snapshot
			timing.Pixels += result.Timing.Pixels
			timing.Retries = attempt
			timing.Total = time.Since(started)
			result.Timing = timing
			return result, nil
		}

		timing.Snapshot += result.Timing.Snapshot
		timing.Pixels += result.Timing.Pixels
		if attempt == retries {
			timing.Retries = attempt
			timing.Total = time.Since(started)
			return CollectedEvidence{}, fmt.Errorf("%w: before=%d after=%d retries=%d", ErrEpochChanged, epochBefore, epochAfter, attempt)
		}
	}
	panic("unreachable")
}

func (p *WarmPage) captureStableAttempt(ctx context.Context, conn *Connection, req EvidenceRequest) (CollectedEvidence, error) {
	var (
		result   CollectedEvidence
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

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

	wg.Wait()
	mu.Lock()
	err := firstErr
	mu.Unlock()
	if err != nil {
		return CollectedEvidence{}, err
	}
	return result, nil
}
