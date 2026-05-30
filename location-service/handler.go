package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type LocationUpdate struct {
	BusID     string  `json:"bus_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	DriverID  string  `json:"driver_id"`
	RouteID   string  `json:"route_id"`
	Speed     float64 `json:"speed"`
	Timestamp string  `json:"timestamp"`
}

type Handler struct {
	rdb *redis.Client
	hub *Hub
}

func NewHandler(rdb *redis.Client, hub *Hub) *Handler {
	return &Handler{rdb: rdb, hub: hub}
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var update LocationUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if update.BusID == "" || update.Latitude == 0 || update.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bus_id, latitude, longitude required"})
		return
	}

	update.Timestamp = time.Now().Format(time.RFC3339)

	data, _ := json.Marshal(update)
	ctx := context.Background()

	// Store in Redis, expires after 2 minutes (bus went offline)
	if err := h.rdb.Set(ctx, "bus:location:"+update.BusID, data, 2*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	// Broadcast to all connected rider apps instantly
	msg, _ := json.Marshal(gin.H{"type": "location_update", "data": update})
	h.hub.Broadcast(msg)

	c.JSON(http.StatusOK, gin.H{"status": "ok", "bus_id": update.BusID})
}

func (h *Handler) GetAllBuses(c *gin.Context) {
	ctx := context.Background()
	keys, err := h.rdb.Keys(ctx, "bus:location:*").Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch"})
		return
	}

	buses := []LocationUpdate{}
	for _, key := range keys {
		data, err := h.rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var loc LocationUpdate
		if json.Unmarshal([]byte(data), &loc) == nil {
			buses = append(buses, loc)
		}
	}
	c.JSON(http.StatusOK, gin.H{"buses": buses, "count": len(buses)})
}

func (h *Handler) WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{conn: conn, send: make(chan []byte, 256)}
	h.hub.register <- client

	// Send current bus positions immediately on connect
	ctx := context.Background()
	if keys, err := h.rdb.Keys(ctx, "bus:location:*").Result(); err == nil {
		for _, key := range keys {
			if data, err := h.rdb.Get(ctx, key).Result(); err == nil {
				var loc LocationUpdate
				if json.Unmarshal([]byte(data), &loc) == nil {
					msg, _ := json.Marshal(gin.H{"type": "location_update", "data": loc})
					client.send <- msg
				}
			}
		}
	}

	// Write pump: sends messages from channel to WebSocket
	go func() {
		defer func() {
			h.hub.unregister <- client
			conn.Close()
		}()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
	}()

	// Read pump: keeps connection alive, detects disconnect
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			h.hub.unregister <- client
			break
		}
	}
}
