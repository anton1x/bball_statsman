package pubsub

import (
	"sync"

	"bball_statsman_backend/internal/domain"
)

// Event is what gets broadcast to subscribers after an operation batch is applied.
type Event struct {
	VideoURL   string                  `json:"videoUrl"`
	Operations []domain.VideoOperation `json:"operations"`
	Version    int64                   `json:"version"`
}

type subscriber struct {
	ch chan Event
}

// Broker is an in-memory pub/sub for video state change events.
type Broker struct {
	mu   sync.Mutex
	subs map[string][]*subscriber // keyed by video URL
}

func NewBroker() *Broker {
	return &Broker{subs: make(map[string][]*subscriber)}
}

// Subscribe returns a channel that receives events for the given video URL.
// Call the returned cancel func to unsubscribe.
func (b *Broker) Subscribe(videoURL string) (<-chan Event, func()) {
	sub := &subscriber{ch: make(chan Event, 16)}

	b.mu.Lock()
	b.subs[videoURL] = append(b.subs[videoURL], sub)
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subs[videoURL]
		next := list[:0]
		for _, s := range list {
			if s != sub {
				next = append(next, s)
			}
		}
		b.subs[videoURL] = next
		close(sub.ch)
	}

	return sub.ch, cancel
}

// Publish sends an event to all subscribers of the given video URL.
func (b *Broker) Publish(videoURL string, ev Event) {
	b.mu.Lock()
	list := make([]*subscriber, len(b.subs[videoURL]))
	copy(list, b.subs[videoURL])
	b.mu.Unlock()

	for _, sub := range list {
		select {
		case sub.ch <- ev:
		default:
			// slow subscriber — drop rather than block
		}
	}
}
