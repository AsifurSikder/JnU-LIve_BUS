package main

import "time"

// LocationUpdate represents a GPS coordinate update from a driver
type LocationUpdate struct {
	BusID     string  `json:"busId" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Timestamp string  `json:"timestamp" binding:"required"` // ISO 8601 UTC
	Accuracy  float64 `json:"accuracy,omitempty"`           // meters
	Speed     float64 `json:"speed,omitempty"`              // m/s
	RouteID   string  `json:"routeId,omitempty"`
	RouteName string  `json:"routeName,omitempty"`
	DriverID  string  `json:"driverId,omitempty"`
	DriverName string `json:"driverName,omitempty"`
}

// BroadcastMessage is the message sent to WebSocket clients
type BroadcastMessage struct {
	Type       string  `json:"type"` // "location_update"
	BusID      string  `json:"busId"`
	RouteID    string  `json:"routeId,omitempty"`
	RouteName  string  `json:"routeName,omitempty"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Timestamp  string  `json:"timestamp"`
	DriverName string  `json:"driverName,omitempty"` // Only for admin clients
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Config holds location service configuration
type Config struct {
	Port                 string
	RedisURL             string
	MaxDriverConnections int
	MaxRiderConnections  int
	LocationTTL          time.Duration
	BroadcastTimeout     time.Duration
	MinUpdateInterval    time.Duration
	MaxTimestampAge      time.Duration
}
