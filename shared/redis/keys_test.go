package redis

import (
	"testing"
)

func TestKeyHelper_BusLocation(t *testing.T) {
	helper := NewKeyHelper()
	
	tests := []struct {
		name   string
		busID  string
		want   string
	}{
		{
			name:  "standard bus ID",
			busID: "bus-123",
			want:  "bus:location:bus-123",
		},
		{
			name:  "UUID bus ID",
			busID: "550e8400-e29b-41d4-a716-446655440000",
			want:  "bus:location:550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:  "empty bus ID",
			busID: "",
			want:  "bus:location:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helper.BusLocation(tt.busID)
			if got != tt.want {
				t.Errorf("BusLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyHelper_ActiveBuses(t *testing.T) {
	helper := NewKeyHelper()
	want := "buses:active"
	
	got := helper.ActiveBuses()
	if got != want {
		t.Errorf("ActiveBuses() = %v, want %v", got, want)
	}
}

func TestKeyHelper_DriverRateLimit(t *testing.T) {
	helper := NewKeyHelper()
	
	tests := []struct {
		name     string
		driverID string
		want     string
	}{
		{
			name:     "standard driver ID",
			driverID: "driver-456",
			want:     "ratelimit:driver:driver-456",
		},
		{
			name:     "UUID driver ID",
			driverID: "660e8400-e29b-41d4-a716-446655440000",
			want:     "ratelimit:driver:660e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helper.DriverRateLimit(tt.driverID)
			if got != tt.want {
				t.Errorf("DriverRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyHelper_AuthRateLimit(t *testing.T) {
	helper := NewKeyHelper()
	
	tests := []struct {
		name string
		ip   string
		want string
	}{
		{
			name: "IPv4 address",
			ip:   "192.168.1.1",
			want: "ratelimit:auth:192.168.1.1",
		},
		{
			name: "IPv6 address",
			ip:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want: "ratelimit:auth:2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		},
		{
			name: "localhost",
			ip:   "127.0.0.1",
			want: "ratelimit:auth:127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helper.AuthRateLimit(tt.ip)
			if got != tt.want {
				t.Errorf("AuthRateLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyHelper_JWTRateLimit(t *testing.T) {
	helper := NewKeyHelper()
	
	ip := "192.168.1.1"
	want := "ratelimit:jwt:192.168.1.1"
	
	got := helper.JWTRateLimit(ip)
	if got != want {
		t.Errorf("JWTRateLimit() = %v, want %v", got, want)
	}
}

func TestKeyHelper_SessionKey(t *testing.T) {
	helper := NewKeyHelper()
	
	driverID := "driver-789"
	want := "session:driver:driver-789"
	
	got := helper.SessionKey(driverID)
	if got != want {
		t.Errorf("SessionKey() = %v, want %v", got, want)
	}
}

func TestKeyHelper_RouteCache(t *testing.T) {
	helper := NewKeyHelper()
	
	routeID := "route-abc"
	want := "cache:route:route-abc"
	
	got := helper.RouteCache(routeID)
	if got != want {
		t.Errorf("RouteCache() = %v, want %v", got, want)
	}
}

func TestKeyHelper_AllRoutesCache(t *testing.T) {
	helper := NewKeyHelper()
	want := "cache:routes:all"
	
	got := helper.AllRoutesCache()
	if got != want {
		t.Errorf("AllRoutesCache() = %v, want %v", got, want)
	}
}

func TestChannelConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "bus location updates channel",
			value: ChannelBusLocationUpdates,
			want:  "bus:location:updates",
		},
		{
			name:  "route updates channel",
			value: ChannelRouteUpdates,
			want:  "route:updates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("Channel constant = %v, want %v", tt.value, tt.want)
			}
		})
	}
}
