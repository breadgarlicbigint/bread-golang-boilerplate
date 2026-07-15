package handler

import (
	"sync"

	"github.com/google/uuid"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/realtime"
)

// chanClient is a pkg/realtime.Client backed by a buffered channel — used by
// both the WebSocket and SSE handlers to bridge Hub.Publish into whichever
// transport-specific write loop is actually pumping bytes to the socket.
//
// Close and Send are mutex-guarded together so a Publish racing a client
// disconnect can never send on (or double-close) an already-closed channel:
// once closed=true, Send short-circuits instead of touching the channel.
type chanClient struct {
	id     string
	mu     sync.Mutex
	send   chan realtime.Event
	closed bool
}

func newChanClient(bufSize int) *chanClient {
	return &chanClient{id: uuid.New().String(), send: make(chan realtime.Event, bufSize)}
}

func (c *chanClient) ID() string { return c.id }

func (c *chanClient) Send(evt realtime.Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- evt:
		return true
	default:
		return false
	}
}

func (c *chanClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.send)
}

var _ realtime.Client = (*chanClient)(nil)
