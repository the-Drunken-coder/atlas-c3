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

func (h *Hub) Publish(_ context.Context, event model.EventEnvelope) error {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for ch := range h.subscribers {
        select {
        case ch <- event:
        default:
        }
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
