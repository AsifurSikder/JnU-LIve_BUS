package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	sharedredis "github.com/mdasifurrahmansikder/jnu-live-bus/shared/redis"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	redisClient *sharedredis.Client
	pubsub      *sharedredis.PubSubManager
	hub         *Hub
	config      *Config
}

func NewHandler(redisClient *sharedredis.Client, pubsub *sharedredis.PubSubManager, hub *Hub, config *Config) *Handler {
	return &Handler{
		redisClient: redisClient,
		pubsub:      pubsub,
		hub:         hub,
		config:      config,
	}
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var update LocationUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Validate GPS payload
	if err := ValidateGPSPayload(&update, h.config.MaxTimestampAge); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Check rate limit
	rateLimitKey := fmt.Sprintf("ratelimit:driver:%s", update.BusID)
	exists, err := h.redisClient.Exists(c.Request.Context(), rateLimitKey)
	if err != nil {
		log.Printf("Rate limit check error: %v", err)
	} else if exists > 0 {
		respondError(c, http.StatusTooManyRequests, "rate_limit_exceeded", 
			fmt.Sprintf("Updates must be at least %v apart", h.config.MinUpdateInterval))
		return
	}

	// Set rate limit
	if err := h.redisClient.Set(c.Request.Context(), rateLimitKey, time.Now().Unix(), h.config.MinUpdateInterval); err != nil {
		log.Printf("Failed to set rate limit: %v", err)
	}

	// Store location in Redis with hash
	locationKey := fmt.Sprintf("bus:location:%s", update.BusID)
	fields := []interface{}{
		"latitude", update.Latitude,
		"longitude", update.Longitude,
		"timestamp", update.Timestamp,
	}
	if update.RouteID != "" {
		fields = append(fields, "routeId", update.RouteID)
	}
	if update.RouteName != "" {
		fields = append(fields, "routeName", update.RouteName)
	}
	if update.DriverID != "" {
		fields = append(fields, "driverId", update.DriverID)
	}
	if update.DriverName != "" {
		fields = append(fields, "driverName", update.DriverName)
	}
	if update.Speed > 0 {
		fields = append(fields, "speed", fmt.Sprintf("%.2f", update.Speed))
	}
	if update.Accuracy > 0 {
		fields = append(fields, "accuracy", fmt.Sprintf("%.2f", update.Accuracy))
	}

	if err := h.redisClient.HSet(c.Request.Context(), locationKey, fields...); err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", "Failed to store location")
		return
	}

	// Set TTL on location
	if err := h.redisClient.Expire(c.Request.Context(), locationKey, h.config.LocationTTL); err != nil {
		log.Printf("Failed to set TTL: %v", err)
	}

	// Add bus to active set
	if err := h.redisClient.SAdd(c.Request.Context(), "buses:active", update.BusID); err != nil {
		log.Printf("Failed to add bus to active set: %v", err)
	}

	// Publish to Redis pub/sub channel
	if err := h.pubsub.Publish(c.Request.Context(), "bus:location:updates", update); err != nil {
		log.Printf("Failed to publish location update: %v", err)
	}

	// Broadcast to WebSocket clients
	broadcastMsg := BroadcastMessage{
		Type:       "location_update",
		BusID:      update.BusID,
		RouteID:    update.RouteID,
		RouteName:  update.RouteName,
		Latitude:   update.Latitude,
		Longitude:  update.Longitude,
		Timestamp:  update.Timestamp,
		DriverName: update.DriverName,
	}
	msgData, _ := json.Marshal(broadcastMsg)
	h.hub.Broadcast(msgData)

	// Count broadcasted clients
	broadcastCount := h.hub.ClientCount()

	c.JSON(http.StatusOK, gin.H{
		"status":        "accepted",
		"broadcastedTo": broadcastCount,
	})
}

func (h *Handler) GetAllBuses(c *gin.Context) {
	ctx := c.Request.Context()

	// Get active buses
	busIDs, err := h.redisClient.SMembers(ctx, "buses:active")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", "Failed to fetch buses")
		return
	}

	buses := make([]LocationUpdate, 0, len(busIDs))
	for _, busID := range busIDs {
		locationKey := fmt.Sprintf("bus:location:%s", busID)
		data, err := h.redisClient.HGetAll(ctx, locationKey)
		if err != nil || len(data) == 0 {
			continue
		}

		update := LocationUpdate{
			BusID:      busID,
			Latitude:   parseFloat(data["latitude"]),
			Longitude:  parseFloat(data["longitude"]),
			Timestamp:  data["timestamp"],
			RouteID:    data["routeId"],
			RouteName:  data["routeName"],
			DriverID:   data["driverId"],
			DriverName: data["driverName"],
			Speed:      parseFloat(data["speed"]),
			Accuracy:   parseFloat(data["accuracy"]),
		}

		buses = append(buses, update)
	}

	c.JSON(http.StatusOK, gin.H{"buses": buses, "count": len(buses)})
}

func (h *Handler) WebSocketHandler(c *gin.Context) {
	// Check connection limit
	if h.hub.ClientCount() >= h.config.MaxRiderConnections {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Maximum number of connections reached",
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	h.hub.Register(client)

	// Send current bus positions immediately on connect
	ctx := context.Background()
	busIDs, err := h.redisClient.SMembers(ctx, "buses:active")
	if err == nil {
		for _, busID := range busIDs {
			locationKey := fmt.Sprintf("bus:location:%s", busID)
			data, err := h.redisClient.HGetAll(ctx, locationKey)
			if err != nil || len(data) == 0 {
				continue
			}

			broadcastMsg := BroadcastMessage{
				Type:       "location_update",
				BusID:      busID,
				RouteID:    data["routeId"],
				RouteName:  data["routeName"],
				Latitude:   parseFloat(data["latitude"]),
				Longitude:  parseFloat(data["longitude"]),
				Timestamp:  data["timestamp"],
				DriverName: data["driverName"],
			}

			msgData, _ := json.Marshal(broadcastMsg)
			select {
			case client.send <- msgData:
			default:
				// Channel full, skip
			}
		}
	}

	// Write pump: sends messages from channel to WebSocket
	go func() {
		defer func() {
			h.hub.Unregister(client)
			conn.Close()
		}()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-client.send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				// Send ping to keep connection alive
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read pump: keeps connection alive, detects disconnect
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			h.hub.Unregister(client)
			break
		}
	}
}

func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
		},
	})
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
