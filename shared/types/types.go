package types

import "time"

// User represents a user in the system (driver or admin)
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash in JSON
	Role         string    `json:"role"` // "driver" or "admin"
	FullName     string    `json:"fullName,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Route represents a bus route with ordered stops
type Route struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Stops              []Stop    `json:"stops"`
	AssignedBusID      *string   `json:"assignedBusId,omitempty"`
	AssignedDriverName *string   `json:"assignedDriverName,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Stop represents a waypoint on a route
type Stop struct {
	ID        string  `json:"id"`
	RouteID   string  `json:"routeId"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Order     int     `json:"order"`
}

// Bus represents a physical bus
type Bus struct {
	ID           string    `json:"id"`
	LicensePlate string    `json:"licensePlate"`
	Capacity     int       `json:"capacity,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// RouteAssignment represents a bus/driver assignment to a route
type RouteAssignment struct {
	ID         string    `json:"id"`
	RouteID    string    `json:"routeId"`
	BusID      string    `json:"busId"`
	DriverID   string    `json:"driverId"`
	AssignedAt time.Time `json:"assignedAt"`
}

// LocationUpdate represents a GPS coordinate update from a driver
type LocationUpdate struct {
	BusID      string    `json:"busId" binding:"required"`
	Latitude   float64   `json:"latitude" binding:"required"`
	Longitude  float64   `json:"longitude" binding:"required"`
	Timestamp  time.Time `json:"timestamp" binding:"required"`
	Accuracy   *float64  `json:"accuracy,omitempty"`
	Speed      *float64  `json:"speed,omitempty"`
}

// LocationBroadcast represents a location update broadcast to WebSocket clients
type LocationBroadcast struct {
	Type       string    `json:"type"` // "location_update"
	BusID      string    `json:"busId"`
	RouteID    string    `json:"routeId"`
	RouteName  string    `json:"routeName"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Timestamp  time.Time `json:"timestamp"`
	DriverName *string   `json:"driverName,omitempty"` // Only for admin clients
}

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	Sub  string `json:"sub"`  // User ID
	Role string `json:"role"` // "driver" or "admin"
	Exp  int64  `json:"exp"`  // Expiry timestamp
	Iat  int64  `json:"iat"`  // Issued at timestamp
}

// ErrorResponse represents a standardized error response
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
