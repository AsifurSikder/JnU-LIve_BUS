package redis

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents the health status of Redis
type HealthStatus struct {
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// HealthChecker provides health check functionality for Redis
type HealthChecker struct {
	client *Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client) *HealthChecker {
	return &HealthChecker{
		client: client,
	}
}

// Check performs a health check on the Redis connection
func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
	start := time.Now()
	status := &HealthStatus{
		Timestamp: start,
	}

	err := h.client.HealthCheck(ctx)
	status.ResponseTime = time.Since(start)

	if err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status
	}

	status.Healthy = true
	return status
}

// CheckWithTimeout performs a health check with a timeout
func (h *HealthChecker) CheckWithTimeout(timeout time.Duration) *HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return h.Check(ctx)
}

// DetailedCheck performs a more comprehensive health check
func (h *HealthChecker) DetailedCheck(ctx context.Context) (*DetailedHealthStatus, error) {
	start := time.Now()
	status := &DetailedHealthStatus{
		Timestamp: start,
	}

	// Check basic connectivity
	if err := h.client.HealthCheck(ctx); err != nil {
		status.Healthy = false
		status.Error = err.Error()
		status.ResponseTime = time.Since(start)
		return status, err
	}

	status.Healthy = true
	status.ResponseTime = time.Since(start)

	// Get Redis info
	info, err := h.client.rdb.Info(ctx, "server", "memory", "stats").Result()
	if err != nil {
		status.InfoError = err.Error()
	} else {
		status.Info = info
	}

	// Get connection pool stats
	poolStats := h.client.rdb.PoolStats()
	status.PoolStats = &PoolStats{
		Hits:       poolStats.Hits,
		Misses:     poolStats.Misses,
		Timeouts:   poolStats.Timeouts,
		TotalConns: poolStats.TotalConns,
		IdleConns:  poolStats.IdleConns,
		StaleConns: poolStats.StaleConns,
	}

	return status, nil
}

// DetailedHealthStatus represents detailed health information
type DetailedHealthStatus struct {
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
	InfoError    string        `json:"info_error,omitempty"`
	Info         string        `json:"info,omitempty"`
	PoolStats    *PoolStats    `json:"pool_stats,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// PoolStats represents connection pool statistics
type PoolStats struct {
	Hits       uint32 `json:"hits"`
	Misses     uint32 `json:"misses"`
	Timeouts   uint32 `json:"timeouts"`
	TotalConns uint32 `json:"total_conns"`
	IdleConns  uint32 `json:"idle_conns"`
	StaleConns uint32 `json:"stale_conns"`
}

// String returns a human-readable string representation of the health status
func (h *HealthStatus) String() string {
	if h.Healthy {
		return fmt.Sprintf("Redis is healthy (response time: %v)", h.ResponseTime)
	}
	return fmt.Sprintf("Redis is unhealthy: %s", h.Error)
}
