package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBroker(t *testing.T) {
	broker := NewBroker()
	require.NotNil(t, broker)
	assert.NotNil(t, broker.subscribers)
	assert.Equal(t, 16, broker.bufferSize)
	assert.False(t, broker.closed)
}

func TestBroker_Subscribe(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	t.Run("subscribe to topic", func(t *testing.T) {
		sub := broker.Subscribe("test-namespace")
		require.NotNil(t, sub)
		assert.Equal(t, "test-namespace", sub.topic)
		assert.Same(t, broker, sub.broker)
		assert.NotNil(t, sub.ch)
	})

	t.Run("subscribe to wildcard", func(t *testing.T) {
		sub := broker.Subscribe("*")
		require.NotNil(t, sub)
		assert.Equal(t, "*", sub.topic)
	})

	t.Run("multiple subscriptions same topic", func(t *testing.T) {
		sub1 := broker.Subscribe("same-topic")
		sub2 := broker.Subscribe("same-topic")
		require.NotNil(t, sub1)
		require.NotNil(t, sub2)
		// Each subscription should have its own channel
		assert.NotEqual(t, sub1, sub2)
	})

	t.Run("subscribe to closed broker returns nil", func(t *testing.T) {
		closedBroker := NewBroker()
		closedBroker.Close()
		sub := closedBroker.Subscribe("test")
		assert.Nil(t, sub)
	})
}

func TestSubscription_Events(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	sub := broker.Subscribe("test")
	events := sub.Events()
	assert.NotNil(t, events)
}

func TestSubscription_Unsubscribe(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	sub := broker.Subscribe("test-topic")
	require.NotNil(t, sub)

	// Verify subscription exists
	stats := broker.Stats()
	assert.Equal(t, 1, stats["test-topic"])

	// Unsubscribe
	sub.Unsubscribe()

	// Verify subscription removed
	stats = broker.Stats()
	assert.Equal(t, 0, stats["test-topic"])
}

func TestBroker_Publish(t *testing.T) {
	t.Run("publish to exact match", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		sub := broker.Subscribe("ns/table")
		event := CatalogEvent{
			Type:      TableCreated,
			Namespace: "ns",
			Table:     "table",
		}

		broker.Publish(event)

		select {
		case received := <-sub.Events():
			assert.Equal(t, TableCreated, received.Type)
			assert.Equal(t, "ns", received.Namespace)
			assert.Equal(t, "table", received.Table)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("publish to namespace wildcard", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		sub := broker.Subscribe("ns")
		event := CatalogEvent{
			Type:      TableCreated,
			Namespace: "ns",
			Table:     "table",
		}

		broker.Publish(event)

		select {
		case received := <-sub.Events():
			assert.Equal(t, TableCreated, received.Type)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("publish to global wildcard", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		sub := broker.Subscribe("*")
		event := CatalogEvent{
			Type:      NamespaceCreated,
			Namespace: "any-ns",
		}

		broker.Publish(event)

		select {
		case received := <-sub.Events():
			assert.Equal(t, NamespaceCreated, received.Type)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for event")
		}
	})

	t.Run("publish to closed broker", func(t *testing.T) {
		broker := NewBroker()
		broker.Close()

		event := CatalogEvent{Type: TableCreated}
		// Should not panic
		broker.Publish(event)
	})

	t.Run("non-blocking on full channel", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		sub := broker.Subscribe("test")

		// Fill the channel buffer
		for i := 0; i < 20; i++ {
			broker.Publish(CatalogEvent{Type: TableCreated, Namespace: "test"})
		}

		// Should not block
		done := make(chan bool)
		go func() {
			broker.Publish(CatalogEvent{Type: TableCreated, Namespace: "test"})
			done <- true
		}()

		select {
		case <-done:
			// Success - didn't block
		case <-time.After(time.Second):
			t.Fatal("publish blocked on full channel")
		}

		sub.Unsubscribe()
	})

	t.Run("multiple subscribers receive event", func(t *testing.T) {
		broker := NewBroker()
		defer broker.Close()

		sub1 := broker.Subscribe("test")
		sub2 := broker.Subscribe("test")

		event := CatalogEvent{Type: TableCreated, Namespace: "test"}
		broker.Publish(event)

		received1 := false
		received2 := false

		for i := 0; i < 2; i++ {
			select {
			case <-sub1.Events():
				received1 = true
			case <-sub2.Events():
				received2 = true
			case <-time.After(time.Second):
				t.Fatal("timeout")
			}
		}

		assert.True(t, received1)
		assert.True(t, received2)
	})
}

func TestBroker_Close(t *testing.T) {
	broker := NewBroker()

	sub1 := broker.Subscribe("topic1")
	sub2 := broker.Subscribe("topic2")
	require.NotNil(t, sub1)
	require.NotNil(t, sub2)

	broker.Close()

	// Channels should be closed
	_, ok := <-sub1.Events()
	assert.False(t, ok, "channel should be closed")

	_, ok = <-sub2.Events()
	assert.False(t, ok, "channel should be closed")

	// Stats should be empty
	stats := broker.Stats()
	assert.Empty(t, stats)
}

func TestBroker_Stats(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	// Initially empty
	stats := broker.Stats()
	assert.Empty(t, stats)

	// Add subscriptions
	sub1 := broker.Subscribe("topic1")
	sub2 := broker.Subscribe("topic1")
	sub3 := broker.Subscribe("topic2")

	stats = broker.Stats()
	assert.Equal(t, 2, stats["topic1"])
	assert.Equal(t, 1, stats["topic2"])

	// Remove subscription
	sub1.Unsubscribe()
	stats = broker.Stats()
	assert.Equal(t, 1, stats["topic1"])

	sub2.Unsubscribe()
	sub3.Unsubscribe()
}

func TestBroker_Concurrent(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Publishers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				event := CatalogEvent{
					Type:      TableCreated,
					Namespace: "test",
					Table:     "table",
				}
				broker.Publish(event)
			}
		}(g)
	}

	// Subscribers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				sub := broker.Subscribe("test")
				if sub != nil {
					sub.Unsubscribe()
				}
			}
		}(g)
	}

	// Stat readers
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = broker.Stats()
			}
		}(g)
	}

	wg.Wait()
}

func TestBroker_TopicMatching(t *testing.T) {
	broker := NewBroker()
	defer broker.Close()

	// Subscribe to different patterns
	wildcardSub := broker.Subscribe("*")
	nsSub := broker.Subscribe("myns")
	tableSub := broker.Subscribe("myns/mytable")

	tests := []struct {
		name          string
		event         CatalogEvent
		expectWildcard bool
		expectNs      bool
		expectTable   bool
	}{
		{
			name:          "namespace event",
			event:         CatalogEvent{Type: NamespaceCreated, Namespace: "myns"},
			expectWildcard: true,
			expectNs:      true,
			expectTable:   false,
		},
		{
			name:          "table event",
			event:         CatalogEvent{Type: TableCreated, Namespace: "myns", Table: "mytable"},
			expectWildcard: true,
			expectNs:      true,
			expectTable:   true,
		},
		{
			name:          "different namespace event",
			event:         CatalogEvent{Type: NamespaceCreated, Namespace: "otherns"},
			expectWildcard: true,
			expectNs:      false,
			expectTable:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker.Publish(tt.event)

			checkReceived := func(sub *Subscription, expected bool, name string) {
				select {
				case <-sub.Events():
					assert.True(t, expected, "unexpected event for %s", name)
				case <-time.After(50 * time.Millisecond):
					assert.False(t, expected, "expected event for %s but didn't receive", name)
				}
			}

			checkReceived(wildcardSub, tt.expectWildcard, "wildcard")
			checkReceived(nsSub, tt.expectNs, "namespace")
			checkReceived(tableSub, tt.expectTable, "table")
		})
	}

	wildcardSub.Unsubscribe()
	nsSub.Unsubscribe()
	tableSub.Unsubscribe()
}
