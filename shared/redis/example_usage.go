// +build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/university-bus-tracker/shared/redis"
)

// LocationUpdate represents a bus location update
type LocationUpdate struct {
	BusID     string    `json:"busId"`
	RouteID   string    `json:"routeId"`
	RouteName string    `json:"routeName"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	// Get Redis URL from environment
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Create Redis client
	cfg := redis.DefaultConfig(redisURL)
	client, err := redis.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	log.Println("Connected to Redis successfully")

	// Create key helper
	keyHelper := redis.NewKeyHelper()

	// Create pub/sub manager
	pubsubMgr := redis.NewPubSubManager(client)
	defer pubsubMgr.Close()

	// Subscribe to bus location updates
	err = pubsubMgr.Subscribe(redis.ChannelBusLocationUpdates, func(channel string, payload []byte) error {
		var update LocationUpdate
		if err := json.Unmarshal(payload, &update); err != nil {
			return fmt.Errorf("failed to unmarshal location update: %w", err)
		}

		log.Printf("Received location update: Bus %s at (%.6f, %.6f) on route %s",
			update.BusID, update.Latitude, update.Longitude, update.RouteName)

		// Store last known position in Redis
		ctx := context.Background()
		locationKey := keyHelper.BusLocation(update.BusID)

		err := client.HSet(ctx, locationKey,
			"latitude", fmt.Sprintf("%.8f", update.Latitude),
			"longitude", fmt.Sprintf("%.8f", update.Longitude),
			"timestamp", update.Timestamp.Format(time.RFC3339),
			"routeId", update.RouteID,
			"routeName", update.RouteName,
		)
		if err != nil {
			return fmt.Errorf("failed to store location: %w", err)
		}

		// Set TTL to 30 minutes
		err = client.Expire(ctx, locationKey, 30*time.Minute)
		if err != nil {
			return fmt.Errorf("failed to set TTL: %w", err)
		}

		// Add to active buses set
		activeBusesKey := keyHelper.ActiveBuses()
		err = client.SAdd(ctx, activeBusesKey, update.BusID)
		if err != nil {
			return fmt.Errorf("failed to add to active buses: %w", err)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to location updates: %v", err)
	}

	log.Println("Subscribed to bus location updates")

	// Subscribe to route updates
	err = pubsubMgr.Subscribe(redis.ChannelRouteUpdates, func(channel string, payload []byte) error {
		var update map[string]interface{}
		if err := json.Unmarshal(payload, &update); err != nil {
			return fmt.Errorf("failed to unmarshal route update: %w", err)
		}

		log.Printf("Received route update: %v", update)
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to route updates: %v", err)
	}

	log.Println("Subscribed to route updates")

	// Create health checker
	healthChecker := redis.NewHealthChecker(client)

	// Set up HTTP server with health check endpoints
	http.HandleFunc("/health/redis", redis.HealthCheckHandler(healthChecker))
	http.HandleFunc("/health/redis/detailed", redis.DetailedHealthCheckHandler(healthChecker))

	// Example endpoint to publish a location update
	http.HandleFunc("/publish/location", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var update LocationUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		update.Timestamp = time.Now()

		ctx := r.Context()
		err := pubsubMgr.Publish(ctx, redis.ChannelBusLocationUpdates, update)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to publish: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "published",
		})
	})

	// Example endpoint to get active buses
	http.HandleFunc("/buses/active", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		activeBusesKey := keyHelper.ActiveBuses()

		buses, err := client.SMembers(ctx, activeBusesKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get active buses: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"activeBuses": buses,
			"count":       len(buses),
		})
	})

	// Example endpoint to get bus location
	http.HandleFunc("/buses/location/", func(w http.ResponseWriter, r *http.Request) {
		busID := r.URL.Path[len("/buses/location/"):]
		if busID == "" {
			http.Error(w, "Bus ID required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		locationKey := keyHelper.BusLocation(busID)

		location, err := client.HGetAll(ctx, locationKey)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get location: %v", err), http.StatusInternalServerError)
			return
		}

		if len(location) == 0 {
			http.Error(w, "Bus not found or inactive", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(location)
	})

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Println("Starting HTTP server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Shutdown complete")
}
