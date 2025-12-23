package events

import (
	"log/slog"
	"sync"
)

// Broker is an in-memory event broker for pub/sub.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan CatalogEvent]struct{}
	bufferSize  int
	closed      bool
}

// NewBroker creates a new event broker.
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string]map[chan CatalogEvent]struct{}),
		bufferSize:  16, // Buffer size for each subscription channel
	}
}

// Subscription represents an active subscription that can be used for cleanup.
type Subscription struct {
	topic  string
	ch     chan CatalogEvent
	broker *Broker
}

// Events returns the channel for receiving events.
func (s *Subscription) Events() <-chan CatalogEvent {
	return s.ch
}

// Unsubscribe removes this subscription and closes the channel.
func (s *Subscription) Unsubscribe() {
	s.broker.unsubscribe(s.topic, s.ch)
}

// Subscribe creates a subscription to a topic.
// Topic can be:
//   - "*" for all events
//   - "namespace" for events in a specific namespace
//   - "namespace/table" for events on a specific table
func (b *Broker) Subscribe(topic string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	ch := make(chan CatalogEvent, b.bufferSize)

	if b.subscribers[topic] == nil {
		b.subscribers[topic] = make(map[chan CatalogEvent]struct{})
	}
	b.subscribers[topic][ch] = struct{}{}

	slog.Debug("new subscription", "topic", topic)
	return &Subscription{
		topic:  topic,
		ch:     ch,
		broker: b,
	}
}

// unsubscribe removes a subscription (internal method).
func (b *Broker) unsubscribe(topic string, ch chan CatalogEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[topic]; ok {
		if _, exists := subs[ch]; exists {
			delete(subs, ch)
			close(ch)
			slog.Debug("unsubscribed", "topic", topic)
		}
		// Clean up empty topic map
		if len(subs) == 0 {
			delete(b.subscribers, topic)
		}
	}
}

// Publish sends an event to all matching subscribers.
// Matches subscribers for:
//   - The exact topic (namespace/table)
//   - The namespace wildcard (namespace)
//   - The global wildcard ("*")
func (b *Broker) Publish(event CatalogEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return
	}

	topic := event.Topic()
	namespace := event.Namespace

	// Collect all matching channels
	var channels []chan CatalogEvent

	// Exact topic match
	if subs, ok := b.subscribers[topic]; ok {
		for ch := range subs {
			channels = append(channels, ch)
		}
	}

	// Namespace wildcard (if topic is namespace/table)
	if namespace != "" && topic != namespace {
		if subs, ok := b.subscribers[namespace]; ok {
			for ch := range subs {
				channels = append(channels, ch)
			}
		}
	}

	// Global wildcard
	if subs, ok := b.subscribers["*"]; ok {
		for ch := range subs {
			channels = append(channels, ch)
		}
	}

	// Send to all matching subscribers (non-blocking)
	for _, ch := range channels {
		select {
		case ch <- event:
			// Sent successfully
		default:
			// Channel full, skip to avoid blocking
			slog.Warn("event dropped, subscriber channel full",
				"event_type", event.Type,
				"topic", topic,
			)
		}
	}

	slog.Debug("event published",
		"event_type", event.Type,
		"topic", topic,
		"subscribers", len(channels),
	)
}

// Close closes the broker and all subscriptions.
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true

	for topic, subs := range b.subscribers {
		for ch := range subs {
			close(ch)
		}
		delete(b.subscribers, topic)
	}

	slog.Info("event broker closed")
}

// Stats returns broker statistics.
func (b *Broker) Stats() map[string]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	stats := make(map[string]int)
	for topic, subs := range b.subscribers {
		stats[topic] = len(subs)
	}
	return stats
}
