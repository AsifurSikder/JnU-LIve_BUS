package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// PubSubManager manages Redis pub/sub operations
type PubSubManager struct {
	client       *Client
	subscriptions map[string]*redis.PubSub
	handlers     map[string][]MessageHandler
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// MessageHandler is a function that handles incoming pub/sub messages
type MessageHandler func(channel string, payload []byte) error

// NewPubSubManager creates a new pub/sub manager
func NewPubSubManager(client *Client) *PubSubManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &PubSubManager{
		client:        client,
		subscriptions: make(map[string]*redis.PubSub),
		handlers:      make(map[string][]MessageHandler),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Subscribe subscribes to a channel and registers a message handler
func (m *PubSubManager) Subscribe(channel string, handler MessageHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add handler to the list
	m.handlers[channel] = append(m.handlers[channel], handler)

	// If already subscribed, just add the handler
	if _, exists := m.subscriptions[channel]; exists {
		return nil
	}

	// Create new subscription
	pubsub := m.client.rdb.Subscribe(m.ctx, channel)

	// Test subscription
	_, err := pubsub.Receive(m.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel %s: %w", channel, err)
	}

	m.subscriptions[channel] = pubsub

	// Start listening in a goroutine
	m.wg.Add(1)
	go m.listen(channel, pubsub)

	return nil
}

// listen listens for messages on a channel and dispatches to handlers
func (m *PubSubManager) listen(channel string, pubsub *redis.PubSub) {
	defer m.wg.Done()

	ch := pubsub.Channel()

	for {
		select {
		case <-m.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			m.mu.RLock()
			handlers := m.handlers[channel]
			m.mu.RUnlock()

			// Call all handlers for this channel
			for _, handler := range handlers {
				if err := handler(msg.Channel, []byte(msg.Payload)); err != nil {
					// Log error but continue processing
					fmt.Printf("Error handling message on channel %s: %v\n", channel, err)
				}
			}
		}
	}
}

// Publish publishes a message to a channel
func (m *PubSubManager) Publish(ctx context.Context, channel string, message interface{}) error {
	var payload string

	switch v := message.(type) {
	case string:
		payload = v
	case []byte:
		payload = string(v)
	default:
		// Marshal to JSON for complex types
		data, err := json.Marshal(message)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		payload = string(data)
	}

	return m.client.rdb.Publish(ctx, channel, payload).Err()
}

// Unsubscribe unsubscribes from a channel
func (m *PubSubManager) Unsubscribe(channel string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pubsub, exists := m.subscriptions[channel]
	if !exists {
		return nil
	}

	if err := pubsub.Unsubscribe(m.ctx, channel); err != nil {
		return fmt.Errorf("failed to unsubscribe from channel %s: %w", channel, err)
	}

	if err := pubsub.Close(); err != nil {
		return fmt.Errorf("failed to close subscription for channel %s: %w", channel, err)
	}

	delete(m.subscriptions, channel)
	delete(m.handlers, channel)

	return nil
}

// Close closes all subscriptions and stops the manager
func (m *PubSubManager) Close() error {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for channel, pubsub := range m.subscriptions {
		if err := pubsub.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close subscription for channel %s: %w", channel, err))
		}
	}

	// Wait for all listeners to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All listeners finished
	case <-time.After(5 * time.Second):
		errs = append(errs, fmt.Errorf("timeout waiting for listeners to finish"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing pub/sub manager: %v", errs)
	}

	return nil
}

// GetSubscribedChannels returns a list of currently subscribed channels
func (m *PubSubManager) GetSubscribedChannels() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	channels := make([]string, 0, len(m.subscriptions))
	for channel := range m.subscriptions {
		channels = append(channels, channel)
	}

	return channels
}
