package service

import (
	"context"
	"encoding/json"
	"sync"
)

// UpdateEvent is emitted after a target receives a new health check.
type UpdateEvent struct {
	TargetID int `json:"target_id"`
}

// EventBroker fan-outs polling updates to connected SSE clients.
type EventBroker struct {
	mu          sync.Mutex
	subscribers map[chan []byte]struct{}
}

func NewEventBroker() *EventBroker {
	return &EventBroker{subscribers: make(map[chan []byte]struct{})}
}

func (b *EventBroker) Publish(event UpdateEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for subscriber := range b.subscribers {
		select {
		case subscriber <- payload:
		default:
		}
	}
}

func (b *EventBroker) Subscribe(ctx context.Context) <-chan []byte {
	updates := make(chan []byte, 1)
	b.mu.Lock()
	b.subscribers[updates] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		b.mu.Lock()
		delete(b.subscribers, updates)
		close(updates)
		b.mu.Unlock()
	}()
	return updates
}
