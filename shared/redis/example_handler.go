package redis

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthCheckHandler creates an HTTP handler for Redis health checks
// This can be used in any service that needs to expose a Redis health endpoint
func HealthCheckHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Perform health check with 3-second timeout
		status := checker.CheckWithTimeout(3 * time.Second)

		// Set appropriate status code
		statusCode := http.StatusOK
		if !status.Healthy {
			statusCode = http.StatusServiceUnavailable
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(status)
	}
}

// DetailedHealthCheckHandler creates an HTTP handler for detailed Redis health checks
// This provides more information including pool stats and Redis server info
func DetailedHealthCheckHandler(checker *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Perform detailed health check
		status, err := checker.DetailedCheck(ctx)

		// Set appropriate status code
		statusCode := http.StatusOK
		if err != nil || !status.Healthy {
			statusCode = http.StatusServiceUnavailable
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(status)
	}
}

// Example usage in a service:
//
// func main() {
//     cfg := redis.DefaultConfig(os.Getenv("REDIS_URL"))
//     client, err := redis.NewClient(cfg)
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer client.Close()
//
//     checker := redis.NewHealthChecker(client)
//
//     http.HandleFunc("/health/redis", redis.HealthCheckHandler(checker))
//     http.HandleFunc("/health/redis/detailed", redis.DetailedHealthCheckHandler(checker))
//
//     log.Fatal(http.ListenAndServe(":8080", nil))
// }
