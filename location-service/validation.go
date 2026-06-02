package main

import (
	"errors"
	"time"
)

var (
	ErrMissingBusID     = errors.New("busId is required")
	ErrMissingLatitude  = errors.New("latitude is required")
	ErrMissingLongitude = errors.New("longitude is required")
	ErrMissingTimestamp = errors.New("timestamp is required")
	ErrInvalidLatitude  = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude = errors.New("longitude must be between -180 and 180")
	ErrInvalidTimestamp = errors.New("timestamp must be in ISO 8601 format")
	ErrStaleTimestamp   = errors.New("timestamp is too old (> 5 minutes)")
)

// ValidateGPSPayload validates a GPS location update
func ValidateGPSPayload(update *LocationUpdate, maxAge time.Duration) error {
	// Check required fields
	if update.BusID == "" {
		return ErrMissingBusID
	}
	if update.Latitude == 0 && update.Longitude == 0 {
		// Both being exactly 0 is suspicious but technically valid
		// We'll allow it but check the range
	}
	if update.Timestamp == "" {
		return ErrMissingTimestamp
	}

	// Validate coordinate ranges
	if update.Latitude < -90 || update.Latitude > 90 {
		return ErrInvalidLatitude
	}
	if update.Longitude < -180 || update.Longitude > 180 {
		return ErrInvalidLongitude
	}

	// Validate timestamp format and age
	timestamp, err := time.Parse(time.RFC3339, update.Timestamp)
	if err != nil {
		return ErrInvalidTimestamp
	}

	age := time.Since(timestamp)
	if age > maxAge {
		return ErrStaleTimestamp
	}

	// Future timestamps are also suspicious
	if age < -1*time.Minute {
		return ErrInvalidTimestamp
	}

	return nil
}

// ValidateCoordinates validates latitude and longitude ranges
func ValidateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return ErrInvalidLongitude
	}
	return nil
}
