package fastcdp

import (
	"context"
	"fmt"
)

type BrowserContextID string
type TargetID string
type SessionID string

type PageSpec struct {
	URL     string
	Width   int
	Height  int
	Context BrowserContextID
}

type PageSession struct {
	TargetID  TargetID
	SessionID SessionID
	ContextID BrowserContextID
}

func (c *Connection) CreateBrowserContext(ctx context.Context) (BrowserContextID, error) {
	var result struct {
		BrowserContextID string `json:"browserContextId"`
	}
	if err := c.Call(ctx, "", "Target.createBrowserContext", map[string]any{"disposeOnDetach": true}, &result); err != nil {
		return "", err
	}
	if result.BrowserContextID == "" {
		return "", fmt.Errorf("fastcdp: Target.createBrowserContext returned empty id")
	}
	return BrowserContextID(result.BrowserContextID), nil
}

func (c *Connection) DisposeBrowserContext(ctx context.Context, id BrowserContextID) error {
	if id == "" {
		return nil
	}
	return c.Call(ctx, "", "Target.disposeBrowserContext", map[string]any{"browserContextId": string(id)}, nil)
}

func (c *Connection) CreatePage(ctx context.Context, spec PageSpec) (PageSession, error) {
	if spec.URL == "" {
		spec.URL = "about:blank"
	}
	params := map[string]any{"url": spec.URL}
	if spec.Context != "" {
		params["browserContextId"] = string(spec.Context)
	}
	if spec.Width > 0 {
		params["width"] = spec.Width
	}
	if spec.Height > 0 {
		params["height"] = spec.Height
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	if err := c.Call(ctx, "", "Target.createTarget", params, &created); err != nil {
		return PageSession{}, err
	}
	if created.TargetID == "" {
		return PageSession{}, fmt.Errorf("fastcdp: Target.createTarget returned empty target id")
	}

	var attached struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.Call(ctx, "", "Target.attachToTarget", map[string]any{
		"targetId": created.TargetID,
		"flatten":  true,
	}, &attached); err != nil {
		_ = c.CloseTarget(context.Background(), TargetID(created.TargetID))
		return PageSession{}, err
	}
	if attached.SessionID == "" {
		_ = c.CloseTarget(context.Background(), TargetID(created.TargetID))
		return PageSession{}, fmt.Errorf("fastcdp: Target.attachToTarget returned empty session id")
	}
	return PageSession{
		TargetID:  TargetID(created.TargetID),
		SessionID: SessionID(attached.SessionID),
		ContextID: spec.Context,
	}, nil
}

func (c *Connection) CloseTarget(ctx context.Context, id TargetID) error {
	if id == "" {
		return nil
	}
	var result struct {
		Success bool `json:"success"`
	}
	if err := c.Call(ctx, "", "Target.closeTarget", map[string]any{"targetId": string(id)}, &result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("fastcdp: target %q did not close", id)
	}
	return nil
}

func (c *Connection) EnablePageDomains(ctx context.Context, session SessionID) error {
	for _, method := range []string{"Runtime.enable", "Page.enable", "DOM.enable"} {
		if err := c.Call(ctx, string(session), method, nil, nil); err != nil {
			return fmt.Errorf("fastcdp: %s: %w", method, err)
		}
	}
	return nil
}
