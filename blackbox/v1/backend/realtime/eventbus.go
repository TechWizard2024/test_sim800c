package realtime

import (
	"sync"
)

type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type EventBus struct {
	mu sync.RWMutex
	subs []chan Event
}

func NewEventBus() *EventBus {
	return &EventBus{subs: make([]chan Event, 0)}
}

func (b *EventBus) Subscribe(buffer int) chan Event {
	ch := make(chan Event, buffer)
	b.mu.Lock()
	b.subs = append(b.subs, ch)
	b.mu.Unlock()
	return ch
}

func (b *EventBus) Publish(evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- evt:
		default:
			// drop on backpressure
		}
	}
}

