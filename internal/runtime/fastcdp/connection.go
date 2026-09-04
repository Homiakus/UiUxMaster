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
	id      uint64
	ch      chan Event
	dropped atomic.Uint64
}

// EventSubscription is a bounded non-blocking subscription. Dropped reports
// events that could not be delivered because the consumer was slower than the
// CDP producer. Correctness-sensitive collectors must treat a changed drop
// count as incomplete evidence rather than silently passing validation.
type EventSubscription struct {
	Events <-chan Event
	sub    *subscriber
	cancel func()
}

func (s *EventSubscription) Dropped() uint64 {
	if s == nil || s.sub == nil {
		return 0
	}
	return s.sub.dropped.Load()
}

func (s *EventSubscription) Close() {
	if s != nil && s.cancel != nil {
		s.cancel()
	}
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
	events   map[string]map[uint64]*subscriber

	closeOnce sync.Once
	closed    chan struct{}
	closeErr  atomic.Pointer[closeError]
}

type closeError struct{ err error }

func NewConnection(transport Transport) *Connection {
	c := &Connection{
		transport: transport,
		pending:   make(map[int64]chan callResult),
		events:    make(map[string]map[uint64]*subscriber),
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

// Subscribe registers a bounded, non-blocking event subscriber. It preserves
// the original lightweight API; correctness-sensitive code should prefer
// SubscribeObserved so event loss can be detected.
func (c *Connection) Subscribe(method string, buffer int) (<-chan Event, func()) {
	sub := c.SubscribeObserved(method, buffer)
	return sub.Events, sub.Close
}

// SubscribeObserved registers a bounded event subscriber and exposes a drop
// counter. The CDP read loop never blocks on a slow consumer.
func (c *Connection) SubscribeObserved(method string, buffer int) *EventSubscription {
	if buffer < 1 {
		buffer = 1
	}
	id := c.nextSubID.Add(1)
	sub := &subscriber{id: id, ch: make(chan Event, buffer)}
	c.eventsMu.Lock()
	if c.events[method] == nil {
		c.events[method] = make(map[uint64]*subscriber)
	}
	c.events[method][id] = sub
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
	return &EventSubscription{Events: sub.ch, sub: sub, cancel: cancel}
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
	subscribers := make([]*subscriber, 0, len(c.events[event.Method]))
	for _, sub := range c.events[event.Method] {
		subscribers = append(subscribers, sub)
	}
	c.eventsMu.RUnlock()
	for _, sub := range subscribers {
		select {
		case sub.ch <- event:
		default:
			sub.dropped.Add(1)
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
