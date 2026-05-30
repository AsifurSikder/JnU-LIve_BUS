package redis

import "fmt"

// Channel names for pub/sub
const (
	ChannelBusLocationUpdates = "bus:location:updates"
	ChannelRouteUpdates       = "route:updates"
)

// KeyHelper provides consistent Redis key naming
type KeyHelper struct{}

// NewKeyHelper creates a new key helper
func NewKeyHelper() *KeyHelper {
	return &KeyHelper{}
}

// BusLocation returns the key for a bus's last known location
// Format: bus:location:{busId}
func (k *KeyHelper) BusLocation(busID string) string {
	return fmt.Sprintf("bus:location:%s", busID)
}

// ActiveBuses returns the key for the set of active buses
// Format: buses:active
func (k *KeyHelper) ActiveBuses() string {
	return "buses:active"
}

// DriverRateLimit returns the key for driver GPS update rate limiting
// Format: ratelimit:driver:{driverId}
func (k *KeyHelper) DriverRateLimit(driverID string) string {
	return fmt.Sprintf("ratelimit:driver:%s", driverID)
}

// AuthRateLimit returns the key for authentication rate limiting by IP
// Format: ratelimit:auth:{ip}
func (k *KeyHelper) AuthRateLimit(ip string) string {
	return fmt.Sprintf("ratelimit:auth:%s", ip)
}

// JWTRateLimit returns the key for JWT validation rate limiting by IP
// Format: ratelimit:jwt:{ip}
func (k *KeyHelper) JWTRateLimit(ip string) string {
	return fmt.Sprintf("ratelimit:jwt:%s", ip)
}

// SessionKey returns the key for a driver session
// Format: session:driver:{driverId}
func (k *KeyHelper) SessionKey(driverID string) string {
	return fmt.Sprintf("session:driver:%s", driverID)
}

// RouteCache returns the key for cached route data
// Format: cache:route:{routeId}
func (k *KeyHelper) RouteCache(routeID string) string {
	return fmt.Sprintf("cache:route:%s", routeID)
}

// AllRoutesCache returns the key for cached list of all routes
// Format: cache:routes:all
func (k *KeyHelper) AllRoutesCache() string {
	return "cache:routes:all"
}
