package fastcdp

import (
	"context"
	"errors"
	"fmt"
)

type RuntimeConfig struct {
	Browser BrowserConfig
	Dial    DialOptions
	Pages   PagePoolConfig
}

type ResidentRuntime struct {
	Browser *BrowserProcess
	Conn    *Connection
	Pages   *PagePool
}

type BrowserVersion struct {
	ProtocolVersion string `json:"protocolVersion"`
	Product         string `json:"product"`
	Revision        string `json:"revision"`
	UserAgent       string `json:"userAgent"`
	JSVersion       string `json:"jsVersion"`
}

// StartResidentRuntime launches one browser process, dials its browser-level CDP
// websocket and creates one isolated bounded warm-page pool. Browser launch and
// navigation are therefore outside the per-verification hot path.
func StartResidentRuntime(ctx context.Context, config RuntimeConfig) (*ResidentRuntime, error) {
	browser, err := LaunchBrowser(ctx, config.Browser)
	if err != nil {
		return nil, err
	}
	conn, err := Dial(ctx, browser.Endpoint, config.Dial)
	if err != nil {
		_ = browser.Close()
		return nil, err
	}
	pool, err := NewPagePool(ctx, conn, config.Pages)
	if err != nil {
		_ = conn.Close()
		_ = browser.Close()
		return nil, err
	}
	return &ResidentRuntime{Browser: browser, Conn: conn, Pages: pool}, nil
}

func (r *ResidentRuntime) Version(ctx context.Context) (BrowserVersion, error) {
	if r == nil || r.Conn == nil {
		return BrowserVersion{}, fmt.Errorf("fastcdp: runtime is not connected")
	}
	var version BrowserVersion
	if err := r.Conn.Call(ctx, "", "Browser.getVersion", nil, &version); err != nil {
		return BrowserVersion{}, err
	}
	if version.Product == "" || version.ProtocolVersion == "" {
		return BrowserVersion{}, fmt.Errorf("fastcdp: incomplete Browser.getVersion response")
	}
	return version, nil
}

func (r *ResidentRuntime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var errs []error
	if r.Pages != nil {
		if err := r.Pages.Close(ctx); err != nil && !errors.Is(err, ErrClosed) {
			errs = append(errs, err)
		}
	}
	if r.Conn != nil {
		if err := r.Conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.Browser != nil {
		if err := r.Browser.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
