package fastcdp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrClosed = errors.New("fastcdp: connection closed")

// Transport is the only capability the raw CDP core needs from a WebSocket
// implementation. Read and Write must honor context cancellation.
type Transport interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
	Close() error
}

// ProtocolError mirrors the stable error envelope used by the Chrome DevTools
// Protocol without leaking a third-party CDP client type into UiUxMaster.
type ProtocolError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("fastcdp: protocol error %d: %s", e.Code, e.Message)
}

// Event is a renderer-neutral CDP event envelope. Consumers decode Params only
// for events they explicitly subscribe to.
type Event struct {
	Method    string
	SessionID string
	Params    json.RawMessage
}

type wireMessage struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *ProtocolError  `json:"error,omitempty"`
}

type callResult struct {
	msg wireMessage
	err error
}

type subscriber struct {
	id uint64
	ch chan Event
}

// Connection multiplexes concurrent commands and asynchronous events over one
// resident CDP transport. It owns exactly one read loop; callers may Call from
// arbitrary goroutines.
type Connection struct {
	transport Transport
	nextID    atomic.Int64
	nextSubID atomic.Uint64

	mu      sync.Mutex
	pending map[int64]chan callResult

	eventsMu sync.RWMutex
	events   map[string]map[uint64]chan Event

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  atomic.Pointer[closeError]
}

type closeError struct{ err error }

func NewConnection(transport Transport) *Connection {
	c := &Connection{
		transport: transport,
		pending:   make(map[int64]chan callResult),
		events:    make(map[string]map[uint64]chan Event),
		closed:    make(chan struct{}),
	}
	go c.readLoop()
	return c
}

func (c *Connection) Done() <-chan struct{} { return c.closed }

func (c *Connection) Err() error {
	if p := c.closeErr.Load(); p != nil {
		return p.err
	}
	return nil
}

// Call sends one CDP method and waits for the matching response. SessionID is
// optional and supports flattened Target sessions without a second connection.
func (c *Connection) Call(ctx context.Context, sessionID, method string, params any, out any) error {
	if method == "" {
		return errors.New("fastcdp: method is required")
	}
	select {
	case <-c.closed:
		return c.closedError()
	default:
	}

	id := c.nextID.Add(1)
	paramsRaw, err := marshalParams(params)
	if err != nil {
		return fmt.Errorf("fastcdp: marshal %s params: %w", method, err)
	}
	msg := wireMessage{ID: id, Method: method, SessionID: sessionID, Params: paramsRaw}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("fastcdp: marshal %s request: %w", method, err)
	}

	resultCh := make(chan callResult, 1)
	c.mu.Lock()
	select {
	case <-c.closed:
		c.mu.Unlock()
		return c.closedError()
	default:
		c.pending[id] = resultCh
	}
	c.mu.Unlock()

	if err := c.transport.Write(ctx, payload); err != nil {
		c.removePending(id)
		return fmt.Errorf("fastcdp: write %s: %w", method, err)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			return result.err
		}
		if result.msg.Error != nil {
			return result.msg.Error
		}
		if out == nil || len(result.msg.Result) == 0 {
			return nil
		}
		if err := json.Unmarshal(result.msg.Result, out); err != nil {
			return fmt.Errorf("fastcdp: decode %s result: %w", method, err)
		}
		return nil
	case <-ctx.Done():
		c.removePending(id)
		return ctx.Err()
	case <-c.closed:
		c.removePending(id)
		return c.closedError()
	}
}

// Subscribe registers a bounded, non-blocking event subscriber. Slow consumers
// cannot stall the CDP read loop; if the buffer is full, the newest event is
// dropped and callers should rely on explicit state/epoch checks before PASS.
func (c *Connection) Subscribe(method string, buffer int) (<-chan Event, func()) {
	if buffer < 1 {
		buffer = 1
	}
	id := c.nextSubID.Add(1)
	ch := make(chan Event, buffer)
	c.eventsMu.Lock()
	if c.events[method] == nil {
		c.events[method] = make(map[uint64]chan Event)
	}
	c.events[method][id] = ch
	c.eventsMu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			c.eventsMu.Lock()
			if subscribers := c.events[method]; subscribers != nil {
				delete(subscribers, id)
				if len(subscribers) == 0 {
					delete(c.events, method)
				}
			}
			c.eventsMu.Unlock()
		})
	}
	return ch, cancel
}

func (c *Connection) Close() error {
	c.shutdown(ErrClosed)
	return c.transport.Close()
}

func (c *Connection) readLoop() {
	for {
		payload, err := c.transport.Read(context.Background())
		if err != nil {
			c.shutdown(fmt.Errorf("fastcdp: read: %w", err))
			return
		}
		var msg wireMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			// A malformed transport frame is a connection-level integrity failure;
			// continuing could mis-correlate command responses.
			c.shutdown(fmt.Errorf("fastcdp: decode frame: %w", err))
			return
		}
		if msg.ID != 0 {
			c.deliverResponse(msg)
			continue
		}
		if msg.Method != "" {
			c.publish(Event{Method: msg.Method, SessionID: msg.SessionID, Params: msg.Params})
		}
	}
}

func (c *Connection) deliverResponse(msg wireMessage) {
	c.mu.Lock()
	ch := c.pending[msg.ID]
	delete(c.pending, msg.ID)
	c.mu.Unlock()
	if ch != nil {
		ch <- callResult{msg: msg}
	}
}

func (c *Connection) publish(event Event) {
	c.eventsMu.RLock()
	subscribers := make([]chan Event, 0, len(c.events[event.Method]))
	for _, ch := range c.events[event.Method] {
		subscribers = append(subscribers, ch)
	}
	c.eventsMu.RUnlock()
	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (c *Connection) removePending(id int64) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *Connection) shutdown(err error) {
	if err == nil {
		err = ErrClosed
	}
	c.closeOnce.Do(func() {
		c.closeErr.Store(&closeError{err: err})
		close(c.closed)

		c.mu.Lock()
		pending := c.pending
		c.pending = make(map[int64]chan callResult)
		c.mu.Unlock()
		for _, ch := range pending {
			ch <- callResult{err: err}
		}
	})
}

func (c *Connection) closedError() error {
	if err := c.Err(); err != nil {
		return err
	}
	return ErrClosed
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
