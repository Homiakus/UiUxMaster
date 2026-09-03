package fastcdp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

const defaultCDPReadLimit int64 = 128 << 20

// DialOptions controls only the WebSocket transport boundary. Browser process,
// target/session lifecycle and evidence policies remain separate layers.
type DialOptions struct {
	HTTPClient *http.Client
	Header     http.Header
	ReadLimit  int64
}

// Dial connects to an already-running browser/target CDP WebSocket endpoint and
// starts the multiplexed raw-CDP read loop. It does not launch Chromium.
func Dial(ctx context.Context, endpoint string, options DialOptions) (*Connection, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("fastcdp: websocket endpoint is required")
	}
	conn, response, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{
		HTTPClient: options.HTTPClient,
		HTTPHeader: options.Header,
	})
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("fastcdp: websocket dial failed with HTTP %s: %w", response.Status, err)
		}
		return nil, fmt.Errorf("fastcdp: websocket dial: %w", err)
	}
	limit := options.ReadLimit
	if limit == 0 {
		limit = defaultCDPReadLimit
	}
	conn.SetReadLimit(limit)
	transport := &websocketTransport{conn: conn}
	return NewConnection(transport), nil
}

type websocketTransport struct {
	conn *websocket.Conn
	once sync.Once
	err  error
}

func (t *websocketTransport) Read(ctx context.Context) ([]byte, error) {
	typ, payload, err := t.conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	if typ != websocket.MessageText && typ != websocket.MessageBinary {
		return nil, fmt.Errorf("fastcdp: unsupported websocket message type %s", typ)
	}
	return payload, nil
}

func (t *websocketTransport) Write(ctx context.Context, payload []byte) error {
	return t.conn.Write(ctx, websocket.MessageText, payload)
}

func (t *websocketTransport) Close() error {
	t.once.Do(func() {
		t.err = t.conn.CloseNow()
	})
	return t.err
}
