// Package events provides an in-process publish/subscribe hub used to stream
// scan progress to API clients. It is intentionally transport-agnostic: the
// API layer adapts it to Server-Sent Events and WebSocket frames.
package events

import (
	"sync"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// subscriberBuffer is how many events may queue for a slow client before we
// start dropping. Progress is periodic, so dropping the oldest is acceptable
// and prevents one stalled client from blocking the whole run.
const subscriberBuffer = 256

type subscriber struct {
	id int64
	ch chan *domain.Event
}

// Hub fans out run events to any number of subscribers.
type Hub struct {
	mu          sync.RWMutex
	subs        map[domain.ID]map[int64]*subscriber
	nextID      int64
	lastEventID int64
	dropped     int64
}

// NewHub constructs an empty hub.
func NewHub() *Hub {
	return &Hub{subs: make(map[domain.ID]map[int64]*subscriber)}
}

// Subscribe registers a listener for a single run. The returned cancel func
// must be called to release resources; the channel is closed by cancel.
func (h *Hub) Subscribe(runID domain.ID) (<-chan *domain.Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	s := &subscriber{id: h.nextID, ch: make(chan *domain.Event, subscriberBuffer)}
	if h.subs[runID] == nil {
		h.subs[runID] = make(map[int64]*subscriber)
	}
	h.subs[runID][s.id] = s

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			if m, ok := h.subs[runID]; ok {
				delete(m, s.id)
				if len(m) == 0 {
					delete(h.subs, runID)
				}
			}
			h.mu.Unlock()
			close(s.ch)
		})
	}
	return s.ch, cancel
}

// Publish delivers an event to all subscribers of its run. Delivery is
// non-blocking; if a subscriber's buffer is full the event is dropped for that
// subscriber only. Publish assigns a monotonic global event ID when unset.
func (h *Hub) Publish(evt *domain.Event) {
	h.mu.Lock()
	if evt.ID == 0 {
		h.lastEventID++
		evt.ID = h.lastEventID
	} else if evt.ID > h.lastEventID {
		h.lastEventID = evt.ID
	}
	if evt.At.IsZero() {
		evt.At = time.Now().UTC()
	}
	// Snapshot subscribers so we can send without holding the write lock.
	targets := make([]*subscriber, 0, len(h.subs[evt.RunID]))
	for _, s := range h.subs[evt.RunID] {
		targets = append(targets, s)
	}
	h.mu.Unlock()

	for _, s := range targets {
		select {
		case s.ch <- evt:
		default:
			h.mu.Lock()
			h.dropped++
			h.mu.Unlock()
		}
	}
}

// Dropped returns the number of events dropped due to slow subscribers.
func (h *Hub) Dropped() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.dropped
}

// SubscriberCount returns the number of active subscribers for a run.
func (h *Hub) SubscriberCount(runID domain.ID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[runID])
}
