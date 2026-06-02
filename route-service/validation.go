package main

import (
	"errors"
	"fmt"
)

var (
	ErrRouteNameTooShort     = errors.New("route name must be at least 1 character")
	ErrRouteNameTooLong      = errors.New("route name must not exceed 100 characters")
	ErrTooFewStops           = errors.New("route must have at least 2 stops")
	ErrTooManyStops          = errors.New("route must not exceed 50 stops")
	ErrStopNameTooShort      = errors.New("stop name must be at least 1 character")
	ErrStopNameTooLong       = errors.New("stop name must not exceed 100 characters")
	ErrInvalidLatitude       = errors.New("latitude must be between -90 and 90")
	ErrInvalidLongitude      = errors.New("longitude must be between -180 and 180")
)

// ValidateRouteName validates route name constraints
func ValidateRouteName(name string) error {
	if len(name) < 1 {
		return ErrRouteNameTooShort
	}
	if len(name) > 100 {
		return ErrRouteNameTooLong
	}
	return nil
}

// ValidateStopList validates the number of stops
func ValidateStopList(stops []StopCreate) error {
	if len(stops) < 2 {
		return ErrTooFewStops
	}
	if len(stops) > 50 {
		return ErrTooManyStops
	}
	return nil
}

// ValidateStopName validates stop name constraints
func ValidateStopName(name string) error {
	if len(name) < 1 {
		return ErrStopNameTooShort
	}
	if len(name) > 100 {
		return ErrStopNameTooLong
	}
	return nil
}

// ValidateCoordinates validates latitude and longitude
func ValidateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return ErrInvalidLatitude
	}
	if lon < -180 || lon > 180 {
		return ErrInvalidLongitude
	}
	return nil
}

// ValidateStop validates a single stop
func ValidateStop(stop StopCreate) error {
	if err := ValidateStopName(stop.Name); err != nil {
		return err
	}
	if err := ValidateCoordinates(stop.Latitude, stop.Longitude); err != nil {
		return err
	}
	return nil
}

// ValidateRouteData validates complete route data
func ValidateRouteData(name string, stops []StopCreate) error {
	// Validate route name
	if err := ValidateRouteName(name); err != nil {
		return fmt.Errorf("invalid route name: %w", err)
	}

	// Validate stop list size
	if err := ValidateStopList(stops); err != nil {
		return fmt.Errorf("invalid stop list: %w", err)
	}

	// Validate each stop
	for i, stop := range stops {
		if err := ValidateStop(stop); err != nil {
			return fmt.Errorf("invalid stop at index %d: %w", i, err)
		}
	}

	return nil
}
