// +build integration

package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// Note: These integration tests require a running Redis instance
// Set REDIS_URL environment variable or use default redis://localhost:6379
// To run: go test -tags=integration -v ./redis/

func getTestRedisURL() string {
	// In a real integration test, you would use testcontainers-go
	// For now, we'll use a default local Redis instance
	return "redis://localhost:6379/15" // Use DB 15 for testing
}

func TestClient_Integration(t *testing.T) {
	cfg := DefaultConfig(getTestRedisURL())
	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("Set and Get", func(t *testing.T) {
		key := "test:key:1"
		value := "test-value"

		// Set value
		err := client.Set(ctx, key, value, 1*time.Minute)
		if err != nil {
			t.Fatalf("Failed to set key: %v", err)
		}

		// Get value
		got, err := client.Get(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get key: %v", err)
		}

		if got != value {
			t.Errorf("Get() = %v, want %v", got, value)
		}

		// Cleanup
		client.Del(ctx, key)
	})

	t.Run("Hash operations", func(t *testing.T) {
		key := "test:hash:1"

		// Set hash fields
		err := client.HSet(ctx, key, "field1", "value1", "field2", "value2")
		if err != nil {
			t.Fatalf("Failed to set hash: %v", err)
		}

		// Get all hash fields
		fields, err := client.HGetAll(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get hash: %v", err)
		}

		if len(fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(fields))
		}

		if fields["field1"] != "value1" {
			t.Errorf("field1 = %v, want value1", fields["field1"])
		}

		// Cleanup
		client.Del(ctx, key)
	})

	t.Run("Set operations", func(t *testing.T) {
		key := "test:set:1"

		// Add members
		err := client.SAdd(ctx, key, "member1", "member2", "member3")
		if err != nil {
			t.Fatalf("Failed to add to set: %v", err)
		}

		// Get members
		members, err := client.SMembers(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get set members: %v", err)
		}

		if len(members) != 3 {
			t.Errorf("Expected 3 members, got %d", len(members))
		}

		// Remove member
		err = client.SRem(ctx, key, "member2")
		if err != nil {
			t.Fatalf("Failed to remove from set: %v", err)
		}

		members, err = client.SMembers(ctx, key)
		if err != nil {
			t.Fatalf("Failed to get set members: %v", err)
		}

		if len(members) != 2 {
			t.Errorf("Expected 2 members after removal, got %d", len(members))
		}

		// Cleanup
		client.Del(ctx, key)
	})

	t.Run("TTL and Expiry", func(t *testing.T) {
		key := "test:ttl:1"

		// Set with TTL
		err := client.Set(ctx, key, "value", 2*time.Second)
		if err != nil {
			t.Fatalf("Failed to set key with TTL: %v", err)
		}

		// Check exists
		count, err := client.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected key to exist, got count %d", count)
		}

		// Wait for expiry
		time.Sleep(3 * time.Second)

		// Check not exists
		count, err = client.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}

		if count != 0 {
			t.Errorf("Expected key to be expired, got count %d", count)
		}
	})
}

func TestPubSubManager_Integration(t *testing.T) {
	cfg := DefaultConfig(getTestRedisURL())
	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	ctx := context.Background()
	pubsubMgr := NewPubSubManager(client)
	defer pubsubMgr.Close()

	t.Run("Subscribe and Publish", func(t *testing.T) {
		channel := "test:channel:1"
		received := make(chan string, 1)

		// Subscribe
		err := pubsubMgr.Subscribe(channel, func(ch string, payload []byte) error {
			received <- string(payload)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Give subscription time to establish
		time.Sleep(100 * time.Millisecond)

		// Publish
		message := "test-message"
		err = pubsubMgr.Publish(ctx, channel, message)
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		// Wait for message
		select {
		case msg := <-received:
			if msg != message {
				t.Errorf("Received message = %v, want %v", msg, message)
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for message")
		}

		// Cleanup
		pubsubMgr.Unsubscribe(channel)
	})

	t.Run("Publish JSON", func(t *testing.T) {
		channel := "test:channel:2"
		received := make(chan map[string]interface{}, 1)

		// Subscribe
		err := pubsubMgr.Subscribe(channel, func(ch string, payload []byte) error {
			var data map[string]interface{}
			if err := json.Unmarshal(payload, &data); err != nil {
				return err
			}
			received <- data
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe: %v", err)
		}

		// Give subscription time to establish
		time.Sleep(100 * time.Millisecond)

		// Publish JSON
		data := map[string]interface{}{
			"busId":     "bus-123",
			"latitude":  23.7104,
			"longitude": 90.4074,
		}

		err = pubsubMgr.Publish(ctx, channel, data)
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		// Wait for message
		select {
		case msg := <-received:
			if msg["busId"] != "bus-123" {
				t.Errorf("busId = %v, want bus-123", msg["busId"])
			}
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for message")
		}

		// Cleanup
		pubsubMgr.Unsubscribe(channel)
	})

	t.Run("Multiple Handlers", func(t *testing.T) {
		channel := "test:channel:3"
		received1 := make(chan string, 1)
		received2 := make(chan string, 1)

		// Subscribe with first handler
		err := pubsubMgr.Subscribe(channel, func(ch string, payload []byte) error {
			received1 <- string(payload)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe with handler 1: %v", err)
		}

		// Subscribe with second handler
		err = pubsubMgr.Subscribe(channel, func(ch string, payload []byte) error {
			received2 <- string(payload)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to subscribe with handler 2: %v", err)
		}

		// Give subscriptions time to establish
		time.Sleep(100 * time.Millisecond)

		// Publish
		message := "multi-handler-test"
		err = pubsubMgr.Publish(ctx, channel, message)
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		// Both handlers should receive the message
		timeout := time.After(2 * time.Second)
		receivedCount := 0

		for receivedCount < 2 {
			select {
			case msg := <-received1:
				if msg != message {
					t.Errorf("Handler 1 received = %v, want %v", msg, message)
				}
				receivedCount++
			case msg := <-received2:
				if msg != message {
					t.Errorf("Handler 2 received = %v, want %v", msg, message)
				}
				receivedCount++
			case <-timeout:
				t.Errorf("Timeout: only %d handlers received message", receivedCount)
				return
			}
		}

		// Cleanup
		pubsubMgr.Unsubscribe(channel)
	})
}

func TestHealthChecker_Integration(t *testing.T) {
	cfg := DefaultConfig(getTestRedisURL())
	client, err := NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
		return
	}
	defer client.Close()

	checker := NewHealthChecker(client)

	t.Run("Basic Health Check", func(t *testing.T) {
		status := checker.CheckWithTimeout(3 * time.Second)

		if !status.Healthy {
			t.Errorf("Expected healthy status, got unhealthy: %s", status.Error)
		}

		if status.ResponseTime == 0 {
			t.Error("Expected non-zero response time")
		}
	})

	t.Run("Detailed Health Check", func(t *testing.T) {
		ctx := context.Background()
		status, err := checker.DetailedCheck(ctx)

		if err != nil {
			t.Fatalf("DetailedCheck failed: %v", err)
		}

		if !status.Healthy {
			t.Errorf("Expected healthy status, got unhealthy: %s", status.Error)
		}

		if status.PoolStats == nil {
			t.Error("Expected pool stats to be populated")
		}

		if status.Info == "" {
			t.Error("Expected Redis info to be populated")
		}
	})
}
