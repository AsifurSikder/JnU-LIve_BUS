package main

import (
	"fmt"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: university-bus-tracker, Property 6: Route Data Validation
// Validates: Requirements 5.1, 5.2, 5.3
// For any route data with a name and list of stops, the validation logic SHALL accept
// the data if and only if: the name is 1-100 characters and unique, the stop list contains
// 2-50 stops, and each stop has a name (1-100 characters) and valid coordinates
// (latitude -90 to 90, longitude -180 to 180). Invalid data SHALL be rejected with HTTP status 422.
func TestProperty_RouteDataValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 1000

	properties := gopter.NewProperties(parameters)

	// Generator for route names
	validRouteNameGen := gen.RegexMatch("[a-zA-Z0-9 -]{1,100}")
	
	// Generator for stop counts
	validStopCountGen := gen.IntRange(2, 50)

	// Generator for stop names
	validStopNameGen := gen.RegexMatch("[a-zA-Z0-9 -]{1,100}")
	
	// Generator for coordinates
	validLatGen := gen.Float64Range(-90, 90)
	validLonGen := gen.Float64Range(-180, 180)

	_ = validStopNameGen
	_ = validLatGen
	_ = validLonGen

	properties.Property("Valid route data is accepted", prop.ForAll(
		func(name string, stopCount int) bool {
			// Generate valid stops
			stops := make([]StopCreate, stopCount)
			for i := 0; i < stopCount; i++ {
				stops[i] = StopCreate{
					Name:      fmt.Sprintf("Stop %c", rune('A'+i)),
					Latitude:  float64(i%90) - 45,    // -45 to 44
					Longitude: float64(i%180) - 90,   // -90 to 89
				}
			}

			err := ValidateRouteData(name, stops)
			if err != nil {
				t.Logf("Valid data rejected: %v", err)
				return false
			}
			return true
		},
		validRouteNameGen,
		validStopCountGen,
	))

	properties.Property("Route name too short is rejected", prop.ForAll(
		func() bool {
			err := ValidateRouteName("")
			return err != nil && err == ErrRouteNameTooShort
		},
	))

	properties.Property("Route name too long is rejected", prop.ForAll(
		func() bool {
			longName := make([]byte, 101)
			for i := range longName {
				longName[i] = 'A'
			}
			err := ValidateRouteName(string(longName))
			return err != nil && err == ErrRouteNameTooLong
		},
	))

	properties.Property("Too few stops is rejected", prop.ForAll(
		func(count int) bool {
			if count >= 2 {
				return true // Skip valid cases
			}
			stops := make([]StopCreate, count)
			err := ValidateStopList(stops)
			return err != nil && err == ErrTooFewStops
		},
		gen.IntRange(0, 1),
	))

	properties.Property("Too many stops is rejected", prop.ForAll(
		func(count int) bool {
			if count <= 50 {
				return true // Skip valid cases
			}
			stops := make([]StopCreate, count)
			err := ValidateStopList(stops)
			return err != nil && err == ErrTooManyStops
		},
		gen.IntRange(51, 100),
	))

	properties.Property("Invalid latitude is rejected", prop.ForAll(
		func(lat float64) bool {
			if lat >= -90 && lat <= 90 {
				return true // Skip valid cases
			}
			err := ValidateCoordinates(lat, 0)
			return err != nil && err == ErrInvalidLatitude
		},
		gen.OneGenOf(
			gen.Float64Range(-1000, -90.1),
			gen.Float64Range(90.1, 1000),
		),
	))

	properties.Property("Invalid longitude is rejected", prop.ForAll(
		func(lon float64) bool {
			if lon >= -180 && lon <= 180 {
				return true // Skip valid cases
			}
			err := ValidateCoordinates(0, lon)
			return err != nil && err == ErrInvalidLongitude
		},
		gen.OneGenOf(
			gen.Float64Range(-1000, -180.1),
			gen.Float64Range(180.1, 1000),
		),
	))

	properties.Property("Boundary coordinates are valid", prop.ForAll(
		func() bool {
			// Test all boundary cases
			boundaries := []struct {
				lat, lon float64
			}{
				{-90, -180}, {-90, 180},
				{90, -180}, {90, 180},
				{0, 0},
				{-90, 0}, {90, 0},
				{0, -180}, {0, 180},
			}

			for _, b := range boundaries {
				if err := ValidateCoordinates(b.lat, b.lon); err != nil {
					t.Logf("Boundary coordinates rejected: lat=%f, lon=%f, err=%v", b.lat, b.lon, err)
					return false
				}
			}
			return true
		},
	))

	properties.TestingRun(t)
}

// Test individual validation functions
func TestProperty_RouteValidationComponents(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	properties.Property("Route names 1-100 chars are valid", prop.ForAll(
		func(length int) bool {
			name := make([]byte, length)
			for i := range name {
				name[i] = 'A'
			}
			err := ValidateRouteName(string(name))
			return err == nil
		},
		gen.IntRange(1, 100),
	))

	properties.Property("Stop lists 2-50 items are valid", prop.ForAll(
		func(count int) bool {
			stops := make([]StopCreate, count)
			err := ValidateStopList(stops)
			return err == nil
		},
		gen.IntRange(2, 50),
	))

	properties.Property("Stop names 1-100 chars are valid", prop.ForAll(
		func(length int) bool {
			name := make([]byte, length)
			for i := range name {
				name[i] = 'A'
			}
			err := ValidateStopName(string(name))
			return err == nil
		},
		gen.IntRange(1, 100),
	))

	properties.Property("Coordinates in range are valid", prop.ForAll(
		func(lat, lon float64) bool {
			err := ValidateCoordinates(lat, lon)
			return err == nil
		},
		gen.Float64Range(-90, 90),
		gen.Float64Range(-180, 180),
	))

	properties.TestingRun(t)
}
