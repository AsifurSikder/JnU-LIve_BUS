package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	sharedredis "github.com/mdasifurrahmansikder/jnu-live-bus/shared/redis"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := loadConfig()

	// Initialize Redis
	redisClient, err := sharedredis.NewClient(sharedredis.DefaultConfig(config.RedisURL))
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.HealthCheck(ctx); err != nil {
		log.Fatalf("Redis health check failed: %v", err)
	}
	log.Println("Connected to Redis")

	// Initialize pub/sub
	pubsub := sharedredis.NewPubSubManager(redisClient)
	defer pubsub.Close()

	// Initialize WebSocket hub
	hub := NewHub()
	go hub.Run()

	// Initialize handler
	handler := NewHandler(redisClient, pubsub, hub, config)

	// Setup router
	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":         "healthy",
			"service":        "location-service",
			"activeClients": hub.ClientCount(),
		})
	})
	router.POST("/location/update", handler.UpdateLocation)
	router.GET("/location/buses", handler.GetAllBuses)
	router.GET("/ws/location", handler.WebSocketHandler)

	server := &http.Server{Addr: ":" + config.Port, Handler: router}

	go func() {
		log.Printf("Location Service starting on port %s", config.Port)
		log.Printf("Config: MaxDrivers=%d, MaxRiders=%d, LocationTTL=%v, MinUpdateInterval=%v",
			config.MaxDriverConnections, config.MaxRiderConnections, config.LocationTTL, config.MinUpdateInterval)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Location Service...")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	server.Shutdown(shutCtx)
	log.Println("Location service stopped")
}

func loadConfig() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8082"),
		RedisURL:             getEnv("REDIS_URL", "redis://localhost:6379"),
		MaxDriverConnections: getEnvInt("MAX_DRIVER_CONNECTIONS", 30),
		MaxRiderConnections:  getEnvInt("MAX_RIDER_CONNECTIONS", 300),
		LocationTTL:          getEnvDuration("LOCATION_TTL", 30*time.Minute),
		BroadcastTimeout:     getEnvDuration("BROADCAST_TIMEOUT", 2*time.Second),
		MinUpdateInterval:    getEnvDuration("MIN_UPDATE_INTERVAL", 5*time.Second),
		MaxTimestampAge:      getEnvDuration("MAX_TIMESTAMP_AGE", 5*time.Minute),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
