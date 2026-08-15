package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type Publisher interface {
	Publish(context.Context, model.EventEnvelope) error
	Subscribe() (<-chan model.EventEnvelope, func())
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan model.EventEnvelope]struct{}
	counter     atomic.Uint64
}

func NewHub() *Hub {
	return &Hub{subscribers: map[chan model.EventEnvelope]struct{}{}}
}

func (h *Hub) NextID() string {
	return fmt.Sprintf("evt-%d-%d", time.Now().UTC().UnixNano(), h.counter.Add(1))
}

// Publish fans an event out to every subscriber. Subscribers that cannot
// receive immediately (their bounded buffer is full) are evicted: the hub
// closes their channel and removes them. The SSE handler treats the closed
// channel as a stream-loss signal and returns, forcing the client to
// reconnect and re-snapshot. Silent drops were rejected because they let
// clients diverge from server state without any signal.
func (h *Hub) Publish(_ context.Context, event model.EventEnvelope) error {
	// Take an exclusive lock: we may need to mutate h.subscribers below to
	// evict slow channels, and an RLock would deadlock on that mutation.
	h.mu.Lock()
	defer h.mu.Unlock()
	var slow []chan model.EventEnvelope
	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			slow = append(slow, ch)
		}
	}
	for _, ch := range slow {
		delete(h.subscribers, ch)
		close(ch)
	}
	return nil
}

func (h *Hub) Subscribe() (<-chan model.EventEnvelope, func()) {
	ch := make(chan model.EventEnvelope, 32)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		// The publisher may have already evicted this channel (slow
		// subscriber path); only close+delete if it is still registered.
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

func EncodeSSE(event model.EventEnvelope) ([]byte, error) {
	raw, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", event.Type, event.EventID, raw)), nil
}
