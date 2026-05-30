# Redis Package

This package provides Redis client wrapper, pub/sub management, key helpers, and health check functionality for the University Bus Tracker system.

## Features

- **Connection Pooling**: Efficient connection management with configurable pool size
- **Pub/Sub Manager**: Easy-to-use pub/sub with multiple handlers per channel
- **Key Helpers**: Consistent key naming across the application
- **Health Checks**: Basic and detailed health check functionality

## Usage

### Creating a Redis Client

```go
package main

import (
    "github.com/university-bus-tracker/shared/redis"
)

func main() {
    // Create client with default configuration
    cfg := redis.DefaultConfig("redis://localhost:6379")
    client, err := redis.NewClient(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Use the client
    ctx := context.Background()
    err = client.Set(ctx, "key", "value", 5*time.Minute)
}
```

### Using Pub/Sub

```go
// Create pub/sub manager
pubsubMgr := redis.NewPubSubManager(client)
defer pubsubMgr.Close()

// Subscribe to a channel
err := pubsubMgr.Subscribe(redis.ChannelBusLocationUpdates, func(channel string, payload []byte) error {
    fmt.Printf("Received message on %s: %s\n", channel, string(payload))
    return nil
})

// Publish a message
type LocationUpdate struct {
    BusID     string  `json:"busId"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}

update := LocationUpdate{
    BusID:     "bus-123",
    Latitude:  23.7104,
    Longitude: 90.4074,
}

err = pubsubMgr.Publish(ctx, redis.ChannelBusLocationUpdates, update)
```

### Using Key Helpers

```go
keyHelper := redis.NewKeyHelper()

// Get consistent key names
locationKey := keyHelper.BusLocation("bus-123")
activeBusesKey := keyHelper.ActiveBuses()
rateLimitKey := keyHelper.DriverRateLimit("driver-456")

// Use with Redis client
err := client.HSet(ctx, locationKey, "latitude", "23.7104", "longitude", "90.4074")
err = client.SAdd(ctx, activeBusesKey, "bus-123")
```

### Health Checks

```go
// Create health checker
healthChecker := redis.NewHealthChecker(client)

// Basic health check
status := healthChecker.CheckWithTimeout(3 * time.Second)
fmt.Println(status.String())

// Detailed health check
detailedStatus, err := healthChecker.DetailedCheck(ctx)
if err != nil {
    fmt.Printf("Health check failed: %v\n", err)
} else {
    fmt.Printf("Pool stats: %+v\n", detailedStatus.PoolStats)
}
```

## Channel Names

The package defines standard channel names for pub/sub:

- `bus:location:updates` - For broadcasting bus location updates
- `route:updates` - For broadcasting route changes

## Key Naming Conventions

All keys follow consistent naming patterns:

- `bus:location:{busId}` - Last known location for a bus
- `buses:active` - Set of active bus IDs
- `ratelimit:driver:{driverId}` - Rate limit tracking for driver GPS updates
- `ratelimit:auth:{ip}` - Rate limit tracking for authentication attempts
- `ratelimit:jwt:{ip}` - Rate limit tracking for JWT validation failures
- `session:driver:{driverId}` - Driver session data
- `cache:route:{routeId}` - Cached route data
- `cache:routes:all` - Cached list of all routes

## Configuration

The `Config` struct allows customization of connection parameters:

```go
cfg := &redis.Config{
    URL:             "redis://localhost:6379",
    MaxRetries:      3,
    PoolSize:        10,
    MinIdleConns:    5,
    ConnMaxIdleTime: 5 * time.Minute,
    DialTimeout:     5 * time.Second,
    ReadTimeout:     3 * time.Second,
    WriteTimeout:    3 * time.Second,
}
```

## Error Handling

All functions return errors that should be checked. The pub/sub manager logs handler errors but continues processing to ensure resilience.

## Thread Safety

- The `Client` is thread-safe and can be used concurrently
- The `PubSubManager` is thread-safe with internal locking
- The `KeyHelper` is stateless and thread-safe
