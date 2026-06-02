package main

import "time"

// Route represents a bus route with ordered stops
type Route struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Stops              []Stop    `json:"stops"`
	AssignedBusID      *string   `json:"assigned_bus_id,omitempty"`
	AssignedDriverName *string   `json:"assigned_driver_name,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Stop represents a waypoint on a route
type Stop struct {
	ID        string    `json:"id"`
	RouteID   string    `json:"route_id"`
	Name      string    `json:"name"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	StopOrder int       `json:"stop_order"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRouteRequest is the request body for creating a route
type CreateRouteRequest struct {
	Name  string       `json:"name" binding:"required,min=1,max=100"`
	Stops []StopCreate `json:"stops" binding:"required,min=2,max=50,dive"`
}

// UpdateRouteRequest is the request body for updating a route
type UpdateRouteRequest struct {
	Name  string       `json:"name" binding:"required,min=1,max=100"`
	Stops []StopCreate `json:"stops" binding:"required,min=2,max=50,dive"`
}

// StopCreate represents a stop to be created
type StopCreate struct {
	Name      string  `json:"name" binding:"required,min=1,max=100"`
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
}

// AssignBusRequest is the request body for assigning a bus to a route
type AssignBusRequest struct {
	BusID    string `json:"busId" binding:"required"`
	DriverID string `json:"driverId" binding:"required"`
}

// AssignmentResponse is the response for bus assignment
type AssignmentResponse struct {
	RouteID    string    `json:"routeId"`
	BusID      string    `json:"busId"`
	DriverID   string    `json:"driverId"`
	AssignedAt time.Time `json:"assignedAt"`
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

// RouteUpdateEvent represents a route change event for pub/sub
type RouteUpdateEvent struct {
	Action  string `json:"action"` // "created", "updated", "deleted"
	RouteID string `json:"routeId"`
	Route   *Route `json:"route,omitempty"`
}
