package fastcdp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

const defaultWarmPages = 2

type PagePoolConfig struct {
	MaxPages int
	Page     PageSpec
}

type WarmPage struct {
	Session PageSession
	Epoch   *EpochGate
	Bridge  *EpochBridge
}

// PagePool owns one isolated browser context and a bounded set of warm pages.
// A page is created lazily, then reused without navigation while healthy.
type PagePool struct {
	conn      *Connection
	contextID BrowserContextID
	spec      PageSpec
	max       int

	inUse  chan struct{}
	idle   chan *WarmPage
	closed atomic.Bool

	mu    sync.Mutex
	pages map[TargetID]*WarmPage
}

type PageLease struct {
	pool *PagePool
	page *WarmPage
	once sync.Once
}

func NewPagePool(ctx context.Context, conn *Connection, config PagePoolConfig) (*PagePool, error) {
	if conn == nil {
		return nil, fmt.Errorf("fastcdp: page pool requires connection")
	}
	maxPages := config.MaxPages
	if maxPages <= 0 {
		maxPages = defaultWarmPages
	}
	contextID, err := conn.CreateBrowserContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("fastcdp: create page-pool context: %w", err)
	}
	return &PagePool{
		conn:      conn,
		contextID: contextID,
		spec:      config.Page,
		max:       maxPages,
		inUse:     make(chan struct{}, maxPages),
		idle:      make(chan *WarmPage, maxPages),
		pages:     make(map[TargetID]*WarmPage),
	}, nil
}

func (p *PagePool) ContextID() BrowserContextID { return p.contextID }
func (p *PagePool) MaxPages() int               { return p.max }

// Acquire leases a warm page, creating one lazily when all existing pages are
// already leased and the configured bound has spare capacity.
func (p *PagePool) Acquire(ctx context.Context) (*PageLease, error) {
	if p.closed.Load() {
		return nil, ErrClosed
	}
	select {
	case p.inUse <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if p.closed.Load() {
		<-p.inUse
		return nil, ErrClosed
	}

	select {
	case page := <-p.idle:
		return &PageLease{pool: p, page: page}, nil
	default:
	}

	page, err := p.createWarmPage(ctx)
	if err != nil {
		<-p.inUse
		return nil, err
	}
	return &PageLease{pool: p, page: page}, nil
}

func (p *PagePool) createWarmPage(ctx context.Context) (*WarmPage, error) {
	spec := p.spec
	spec.Context = p.contextID
	session, err := p.conn.CreatePage(ctx, spec)
	if err != nil {
		return nil, err
	}
	cleanupTarget := true
	defer func() {
		if cleanupTarget {
			_ = p.conn.CloseTarget(context.Background(), session.TargetID)
		}
	}()

	if err := p.conn.EnablePageDomains(ctx, session.SessionID); err != nil {
		return nil, err
	}
	if err := p.conn.SetViewport(ctx, session.SessionID, spec.Width, spec.Height, spec.DPR); err != nil {
		return nil, fmt.Errorf("fastcdp: set page viewport: %w", err)
	}
	gate := NewEpochGate()
	bridge, err := p.conn.InstallEpochBridge(ctx, session.SessionID, gate)
	if err != nil {
		return nil, err
	}
	page := &WarmPage{Session: session, Epoch: gate, Bridge: bridge}

	p.mu.Lock()
	if p.closed.Load() {
		p.mu.Unlock()
		bridge.Close()
		return nil, ErrClosed
	}
	p.pages[session.TargetID] = page
	p.mu.Unlock()
	cleanupTarget = false
	return page, nil
}

func (l *PageLease) Page() *WarmPage {
	if l == nil {
		return nil
	}
	return l.page
}

// Release returns a healthy page to the warm pool. It is idempotent.
func (l *PageLease) Release() {
	if l == nil || l.pool == nil || l.page == nil {
		return
	}
	l.once.Do(func() {
		p := l.pool
		if p.closed.Load() {
			l.page.Bridge.Close()
		} else {
			p.idle <- l.page
		}
		<-p.inUse
	})
}

// Discard removes a stale/corrupt page and closes only its target. The next
// Acquire lazily creates a replacement, preserving browser/context state.
func (l *PageLease) Discard(ctx context.Context) error {
	if l == nil || l.pool == nil || l.page == nil {
		return nil
	}
	var discardErr error
	l.once.Do(func() {
		p := l.pool
		l.page.Bridge.Close()
		p.mu.Lock()
		delete(p.pages, l.page.Session.TargetID)
		p.mu.Unlock()
		if !p.closed.Load() {
			discardErr = p.conn.CloseTarget(ctx, l.page.Session.TargetID)
		}
		<-p.inUse
	})
	return discardErr
}

// Close disposes the pool browser context, which closes every page belonging to
// it, including currently leased pages. Local epoch consumers are stopped first.
func (p *PagePool) Close(ctx context.Context) error {
	if p == nil || !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.mu.Lock()
	pages := make([]*WarmPage, 0, len(p.pages))
	for _, page := range p.pages {
		pages = append(pages, page)
	}
	p.pages = make(map[TargetID]*WarmPage)
	p.mu.Unlock()
	for _, page := range pages {
		page.Bridge.Close()
	}
	return p.conn.DisposeBrowserContext(ctx, p.contextID)
}
