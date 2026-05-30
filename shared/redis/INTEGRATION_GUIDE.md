# Redis Integration Guide

This guide explains how to integrate the Redis infrastructure into each microservice of the University Bus Tracker system.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Service-Specific Integration](#service-specific-integration)
3. [Common Patterns](#common-patterns)
4. [Testing](#testing)
5. [Troubleshooting](#troubleshooting)

## Quick Start

### Installation

Add the shared module to your service:

```bash
# In your service directory (e.g., location-service/)
go mod edit -require=github.com/university-bus-tracker/shared@latest
go mod edit -replace=github.com/university-bus-tracker/shared=../shared
go mod tidy
```

### Basic Setup

```go
package main

import (
    "log"
    "os"
    
    "github.com/university-bus-tracker/shared/redis"
)

func main() {
    // Create Redis client
    cfg := redis.DefaultConfig(os.Getenv("REDIS_URL"))
    client, err := redis.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Create pub/sub manager
    pubsubMgr := redis.NewPubSubManager(client)
    defer pubsubMgr.Close()
    
    // Create key helper
    keyHelper := redis.NewKeyHelper()
    
    // Your service logic here...
}
```

## Service-Specific Integration

### Location Service

The Location Service uses Redis for:
- Storing last known bus positions (with 30-minute TTL)
- Publishing location updates to subscribers
- Rate limiting GPS updates per driver
- Tracking active buses

```go
package main

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/university-bus-tracker/shared/redis"
)

type LocationService struct {
    redisClient *redis.Client
    pubsubMgr   *redis.PubSubManager
    keyHelper   *redis.KeyHelper
}

func NewLocationService(redisURL string) (*LocationService, error) {
    cfg := redis.DefaultConfig(redisURL)
    client, err := redis.NewClient(cfg)
    if err != nil {
        return nil, err
    }
    
    return &LocationService{
        redisClient: client,
        pubsubMgr:   redis.NewPubSubManager(client),
        keyHelper:   redis.NewKeyHelper(),
    }, nil
}

func (s *LocationService) StoreLocation(ctx context.Context, busID string, lat, lon float64, routeID, routeName string) error {
    // Check rate limit
    rateLimitKey := s.keyHelper.DriverRateLimit(busID)
    exists, err := s.redisClient.Exists(ctx, rateLimitKey)
    if err != nil {
        return err
    }
    if exists > 0 {
        return ErrRateLimitExceeded
    }
    
    // Store location
    locationKey := s.keyHelper.BusLocation(busID)
    err = s.redisClient.HSet(ctx, locationKey,
        "latitude", lat,
        "longitude", lon,
        "timestamp", time.Now().Format(time.RFC3339),
        "routeId", routeID,
        "routeName", routeName,
    )
    if err != nil {
        return err
    }
    
    // Set TTL
    err = s.redisClient.Expire(ctx, locationKey, 30*time.Minute)
    if err != nil {
        return err
    }
    
    // Add to active buses
    activeBusesKey := s.keyHelper.ActiveBuses()
    err = s.redisClient.SAdd(ctx, activeBusesKey, busID)
    if err != nil {
        return err
    }
    
    // Set rate limit (5 seconds)
    err = s.redisClient.Set(ctx, rateLimitKey, time.Now().Unix(), 5*time.Second)
    if err != nil {
        return err
    }
    
    // Publish update
    update := map[string]interface{}{
        "busId":     busID,
        "latitude":  lat,
        "longitude": lon,
        "routeId":   routeID,
        "routeName": routeName,
        "timestamp": time.Now().Format(time.RFC3339),
    }
    
    return s.pubsubMgr.Publish(ctx, redis.ChannelBusLocationUpdates, update)
}
```

### Route Service

The Route Service uses Redis for:
- Publishing route changes to subscribers
- Caching route data (optional)

```go
package main

import (
    "context"
    
    "github.com/university-bus-tracker/shared/redis"
)

type RouteService struct {
    redisClient *redis.Client
    pubsubMgr   *redis.PubSubManager
    keyHelper   *redis.KeyHelper
}

func (s *RouteService) PublishRouteUpdate(ctx context.Context, action string, route interface{}) error {
    update := map[string]interface{}{
        "action": action, // "created", "updated", "deleted"
        "route":  route,
    }
    
    return s.pubsubMgr.Publish(ctx, redis.ChannelRouteUpdates, update)
}

func (s *RouteService) CreateRoute(ctx context.Context, route *Route) error {
    // ... database operations ...
    
    // Publish route creation
    return s.PublishRouteUpdate(ctx, "created", route)
}

func (s *RouteService) UpdateRoute(ctx context.Context, route *Route) error {
    // ... database operations ...
    
    // Invalidate cache
    cacheKey := s.keyHelper.RouteCache(route.ID)
    s.redisClient.Del(ctx, cacheKey)
    
    // Publish route update
    return s.PublishRouteUpdate(ctx, "updated", route)
}

func (s *RouteService) DeleteRoute(ctx context.Context, routeID string) error {
    // ... database operations ...
    
    // Invalidate cache
    cacheKey := s.keyHelper.RouteCache(routeID)
    s.redisClient.Del(ctx, cacheKey)
    
    // Publish route deletion
    return s.PublishRouteUpdate(ctx, "deleted", map[string]string{"routeId": routeID})
}
```

### Auth Service

The Auth Service uses Redis for:
- Rate limiting failed authentication attempts by IP

```go
package main

import (
    "context"
    "fmt"
    "strconv"
    "time"
    
    "github.com/university-bus-tracker/shared/redis"
)

type AuthService struct {
    redisClient *redis.Client
    keyHelper   *redis.KeyHelper
}

func (s *AuthService) CheckAuthRateLimit(ctx context.Context, ip string, maxAttempts int) error {
    rateLimitKey := s.keyHelper.AuthRateLimit(ip)
    
    countStr, err := s.redisClient.Get(ctx, rateLimitKey)
    if err != nil && err.Error() != "redis: nil" {
        return err
    }
    
    count := 0
    if countStr != "" {
        count, _ = strconv.Atoi(countStr)
    }
    
    if count >= maxAttempts {
        return fmt.Errorf("rate limit exceeded: %d attempts", count)
    }
    
    return nil
}

func (s *AuthService) IncrementAuthFailures(ctx context.Context, ip string) error {
    rateLimitKey := s.keyHelper.AuthRateLimit(ip)
    
    countStr, err := s.redisClient.Get(ctx, rateLimitKey)
    if err != nil && err.Error() != "redis: nil" {
        return err
    }
    
    count := 0
    if countStr != "" {
        count, _ = strconv.Atoi(countStr)
    }
    
    count++
    return s.redisClient.Set(ctx, rateLimitKey, strconv.Itoa(count), 60*time.Second)
}

func (s *AuthService) ResetAuthFailures(ctx context.Context, ip string) error {
    rateLimitKey := s.keyHelper.AuthRateLimit(ip)
    return s.redisClient.Del(ctx, rateLimitKey)
}
```

### API Gateway

The API Gateway uses Redis for:
- Rate limiting failed JWT validation attempts by IP

```go
package main

import (
    "context"
    "strconv"
    "time"
    
    "github.com/university-bus-tracker/shared/redis"
)

type APIGateway struct {
    redisClient *redis.Client
    keyHelper   *redis.KeyHelper
}

func (g *APIGateway) CheckJWTRateLimit(ctx context.Context, ip string) (bool, error) {
    rateLimitKey := g.keyHelper.JWTRateLimit(ip)
    
    countStr, err := g.redisClient.Get(ctx, rateLimitKey)
    if err != nil && err.Error() != "redis: nil" {
        return false, err
    }
    
    if countStr == "" {
        return true, nil
    }
    
    count, _ := strconv.Atoi(countStr)
    return count < 10, nil // Max 10 failed attempts
}

func (g *APIGateway) IncrementJWTFailures(ctx context.Context, ip string) error {
    rateLimitKey := g.keyHelper.JWTRateLimit(ip)
    
    countStr, err := g.redisClient.Get(ctx, rateLimitKey)
    if err != nil && err.Error() != "redis: nil" {
        return err
    }
    
    count := 0
    if countStr != "" {
        count, _ = strconv.Atoi(countStr)
    }
    
    count++
    return g.redisClient.Set(ctx, rateLimitKey, strconv.Itoa(count), 60*time.Second)
}
```

## Common Patterns

### Health Check Endpoint

Add to all services:

```go
import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/university-bus-tracker/shared/redis"
)

func setupHealthChecks(router *gin.Engine, redisClient *redis.Client) {
    healthChecker := redis.NewHealthChecker(redisClient)
    
    router.GET("/health/redis", gin.WrapF(redis.HealthCheckHandler(healthChecker)))
    router.GET("/health/redis/detailed", gin.WrapF(redis.DetailedHealthCheckHandler(healthChecker)))
}
```

### Graceful Shutdown

```go
func main() {
    // ... setup ...
    
    // Create shutdown channel
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    
    // Start server in goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Wait for shutdown signal
    <-quit
    log.Println("Shutting down...")
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Close Redis connections
    pubsubMgr.Close()
    redisClient.Close()
    
    // Shutdown HTTP server
    server.Shutdown(ctx)
}
```

## Testing

### Unit Tests

Test key generation and business logic without Redis:

```go
func TestKeyGeneration(t *testing.T) {
    helper := redis.NewKeyHelper()
    
    busID := "bus-123"
    key := helper.BusLocation(busID)
    
    expected := "bus:location:bus-123"
    if key != expected {
        t.Errorf("Expected %s, got %s", expected, key)
    }
}
```

### Integration Tests

Use testcontainers-go for integration tests:

```go
// +build integration

func TestRedisIntegration(t *testing.T) {
    // Start Redis container
    ctx := context.Background()
    redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
        },
        Started: true,
    })
    if err != nil {
        t.Fatal(err)
    }
    defer redisContainer.Terminate(ctx)
    
    // Get connection details
    host, _ := redisContainer.Host(ctx)
    port, _ := redisContainer.MappedPort(ctx, "6379")
    redisURL := fmt.Sprintf("redis://%s:%s", host, port.Port())
    
    // Run tests with real Redis
    cfg := redis.DefaultConfig(redisURL)
    client, err := redis.NewClient(cfg)
    if err != nil {
        t.Fatal(err)
    }
    defer client.Close()
    
    // ... test logic ...
}
```

## Troubleshooting

### Connection Issues

```go
// Enable debug logging
cfg := redis.DefaultConfig(redisURL)
cfg.DialTimeout = 10 * time.Second
cfg.ReadTimeout = 5 * time.Second
cfg.WriteTimeout = 5 * time.Second

client, err := redis.NewClient(cfg)
if err != nil {
    log.Printf("Redis connection error: %v", err)
    // Check: Is Redis running? Is URL correct? Network connectivity?
}
```

### Pub/Sub Not Receiving Messages

1. Ensure subscription is established before publishing
2. Add a small delay after subscribing: `time.Sleep(100 * time.Millisecond)`
3. Check that channel names match exactly
4. Verify handler errors are not silently failing

### Memory Issues

Monitor Redis memory usage:

```go
healthChecker := redis.NewHealthChecker(client)
status, _ := healthChecker.DetailedCheck(ctx)
log.Printf("Pool stats: %+v", status.PoolStats)
```

Set appropriate TTLs on all keys to prevent memory leaks.

### Rate Limiting Not Working

Ensure TTLs are set correctly:

```go
// Set value with TTL
client.Set(ctx, key, value, 60*time.Second)

// Or set TTL separately
client.Set(ctx, key, value, 0)
client.Expire(ctx, key, 60*time.Second)
```

## Environment Variables

Each service should accept these environment variables:

```bash
REDIS_URL=redis://localhost:6379
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5
REDIS_MAX_RETRIES=3
```

Example configuration loading:

```go
func loadRedisConfig() *redis.Config {
    cfg := redis.DefaultConfig(os.Getenv("REDIS_URL"))
    
    if poolSize := os.Getenv("REDIS_POOL_SIZE"); poolSize != "" {
        if size, err := strconv.Atoi(poolSize); err == nil {
            cfg.PoolSize = size
        }
    }
    
    // ... load other config values ...
    
    return cfg
}
```
