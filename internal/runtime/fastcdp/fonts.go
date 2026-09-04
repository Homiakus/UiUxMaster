package fastcdp

import (
	"context"
	"fmt"
)

type FontFaceState struct {
	Family  string `json:"family"`
	Style   string `json:"style"`
	Weight  string `json:"weight"`
	Stretch string `json:"stretch"`
	Status  string `json:"status"`
}

type FontState struct {
	Status    string          `json:"status"`
	Faces     []FontFaceState `json:"faces,omitempty"`
	Total     int             `json:"total"`
	Truncated bool            `json:"truncated,omitempty"`
}

// CaptureFontState inspects the standard FontFaceSet through Runtime.evaluate.
// It is pull-based so the CSS domain does not stay enabled on resident pages.
func (c *Connection) CaptureFontState(ctx context.Context, sessionID string, maxFaces int) (FontState, error) {
	if maxFaces <= 0 {
		maxFaces = 128
	}
	expression := fmt.Sprintf(`(() => {
  const all = Array.from(document.fonts || []);
  const faces = all.slice(0, %d).map(f => ({
    family: String(f.family || ''),
    style: String(f.style || ''),
    weight: String(f.weight || ''),
    stretch: String(f.stretch || ''),
    status: String(f.status || '')
  }));
  return {
    status: document.fonts ? String(document.fonts.status || '') : 'unsupported',
    faces,
    total: all.length,
    truncated: all.length > faces.length
  };
})()`, maxFaces)

	var response struct {
		Result struct {
			Type  string    `json:"type"`
			Value FontState `json:"value"`
		} `json:"result"`
		ExceptionDetails any `json:"exceptionDetails,omitempty"`
	}
	if err := c.Call(ctx, sessionID, "Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
	}, &response); err != nil {
		return FontState{}, err
	}
	if response.ExceptionDetails != nil {
		return FontState{}, fmt.Errorf("fastcdp: font state evaluation raised an exception")
	}
	return response.Result.Value, nil
}
