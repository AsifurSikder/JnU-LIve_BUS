package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	sharedredis "github.com/mdasifurrahmansikder/jnu-live-bus/shared/redis"
)

// Feature: university-bus-tracker, Task 4.9: Integration tests for Route Service

var testDB *sql.DB

func setupTestDB(t *testing.T) *sql.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := testDB.PingContext(ctx); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Clean up test data
	cleanupTestData(t, testDB)

	return testDB
}

func cleanupTestData(t *testing.T, db *sql.DB) {
	queries := []string{
		"DELETE FROM route_assignments",
		"DELETE FROM stops",
		"DELETE FROM routes",
		"DELETE FROM buses",
		"DELETE FROM users WHERE role='driver'",
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			t.Logf("Cleanup warning: %v", err)
		}
	}
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/routes", getRoutesHandler)
	router.POST("/routes", createRouteHandler)
	router.PUT("/routes/:id", updateRouteHandler)
	router.DELETE("/routes/:id", deleteRouteHandler)
	router.POST("/routes/:id/assign", assignBusHandler)

	return router
}

func TestRouteServiceIntegration(t *testing.T) {
	testDB = setupTestDB(t)
	defer testDB.Close()

	// Set global db for handlers
	db = testDB

	// Setup Redis if available
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL != "" {
		var err error
		redisClient, err = sharedredis.NewClient(sharedredis.DefaultConfig(redisURL))
		if err != nil {
			t.Logf("Redis not available: %v", err)
		} else {
			defer redisClient.Close()
			pubsub = sharedredis.NewPubSubManager(redisClient)
			defer pubsub.Close()
		}
	}

	router := setupTestRouter()

	t.Run("CreateRoute", func(t *testing.T) {
		testCreateRoute(t, router)
	})

	t.Run("GetRoutes", func(t *testing.T) {
		testGetRoutes(t, router)
	})

	t.Run("UpdateRoute", func(t *testing.T) {
		testUpdateRoute(t, router)
	})

	t.Run("DuplicateRouteName", func(t *testing.T) {
		testDuplicateRouteName(t, router)
	})

	t.Run("DeleteRoute", func(t *testing.T) {
		testDeleteRoute(t, router)
	})

	t.Run("AssignBus", func(t *testing.T) {
		testAssignBus(t, router)
	})

	t.Run("DeleteRouteWithActiveBus", func(t *testing.T) {
		testDeleteRouteWithActiveBus(t, router)
	})
}

func testCreateRoute(t *testing.T, router *gin.Engine) {
	reqBody := CreateRouteRequest{
		Name: "Test Route 1",
		Stops: []StopCreate{
			{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
			{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var route Route
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if route.ID == "" {
		t.Error("Expected route ID to be set")
	}
	if route.Name != "Test Route 1" {
		t.Errorf("Expected name 'Test Route 1', got '%s'", route.Name)
	}
	if len(route.Stops) != 2 {
		t.Errorf("Expected 2 stops, got %d", len(route.Stops))
	}
}

func testGetRoutes(t *testing.T, router *gin.Engine) {
	req := httptest.NewRequest(http.MethodGet, "/routes", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		return
	}

	var response struct {
		Routes []Route `json:"routes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response.Routes) == 0 {
		t.Error("Expected at least one route")
	}
}

func testUpdateRoute(t *testing.T, router *gin.Engine) {
	// First create a route
	createReq := CreateRouteRequest{
		Name: "Route to Update",
		Stops: []StopCreate{
			{Name: "Stop A", Latitude: 23.7104, Longitude: 90.4074},
			{Name: "Stop B", Latitude: 23.7200, Longitude: 90.4100},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdRoute Route
	json.Unmarshal(w.Body.Bytes(), &createdRoute)

	// Now update it
	updateReq := UpdateRouteRequest{
		Name: "Updated Route Name",
		Stops: []StopCreate{
			{Name: "Stop X", Latitude: 23.7300, Longitude: 90.4200},
			{Name: "Stop Y", Latitude: 23.7400, Longitude: 90.4300},
			{Name: "Stop Z", Latitude: 23.7500, Longitude: 90.4400},
		},
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, "/routes/"+createdRoute.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var updatedRoute Route
	json.Unmarshal(w.Body.Bytes(), &updatedRoute)

	if updatedRoute.Name != "Updated Route Name" {
		t.Errorf("Expected updated name, got '%s'", updatedRoute.Name)
	}
	if len(updatedRoute.Stops) != 3 {
		t.Errorf("Expected 3 stops after update, got %d", len(updatedRoute.Stops))
	}
}

func testDuplicateRouteName(t *testing.T, router *gin.Engine) {
	// Create first route
	reqBody := CreateRouteRequest{
		Name: "Duplicate Test Route",
		Stops: []StopCreate{
			{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
			{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("First route creation failed: %d", w.Code)
		return
	}

	// Try to create second route with same name
	req = httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate name, got %d", w.Code)
	}
}

func testDeleteRoute(t *testing.T, router *gin.Engine) {
	// Create a route
	reqBody := CreateRouteRequest{
		Name: "Route to Delete",
		Stops: []StopCreate{
			{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
			{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createdRoute Route
	json.Unmarshal(w.Body.Bytes(), &createdRoute)

	// Delete the route
	req = httptest.NewRequest(http.MethodDelete, "/routes/"+createdRoute.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	// Verify it's deleted
	req = httptest.NewRequest(http.MethodDelete, "/routes/"+createdRoute.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted route, got %d", w.Code)
	}
}

func testAssignBus(t *testing.T, router *gin.Engine) {
	// Create test data
	var routeID, busID, driverID string

	// Create route
	_, err := testDB.Exec(`INSERT INTO routes (id, name) VALUES (gen_random_uuid(), 'Assignment Test Route') RETURNING id`)
	if err != nil {
		t.Fatalf("Failed to create test route: %v", err)
	}
	err = testDB.QueryRow(`SELECT id FROM routes WHERE name='Assignment Test Route'`).Scan(&routeID)
	if err != nil {
		t.Fatalf("Failed to get route ID: %v", err)
	}

	// Create bus
	err = testDB.QueryRow(`INSERT INTO buses (license_plate, capacity) VALUES ('TEST-123', 50) RETURNING id`).Scan(&busID)
	if err != nil {
		t.Fatalf("Failed to create test bus: %v", err)
	}

	// Create driver
	err = testDB.QueryRow(`INSERT INTO users (username, password_hash, role, full_name) VALUES ('test_driver', 'hash', 'driver', 'Test Driver') RETURNING id`).Scan(&driverID)
	if err != nil {
		t.Fatalf("Failed to create test driver: %v", err)
	}

	// Assign bus
	reqBody := AssignBusRequest{
		BusID:    busID,
		DriverID: driverID,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/routes/"+routeID+"/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		return
	}

	var response AssignmentResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.RouteID != routeID {
		t.Errorf("Expected route ID %s, got %s", routeID, response.RouteID)
	}
	if response.BusID != busID {
		t.Errorf("Expected bus ID %s, got %s", busID, response.BusID)
	}
}

func testDeleteRouteWithActiveBus(t *testing.T, router *gin.Engine) {
	// Create route with assigned bus
	var routeID, busID, driverID string

	err := testDB.QueryRow(`INSERT INTO routes (name) VALUES ('Route with Bus') RETURNING id`).Scan(&routeID)
	if err != nil {
		t.Fatalf("Failed to create route: %v", err)
	}

	err = testDB.QueryRow(`INSERT INTO buses (license_plate, capacity) VALUES ('BUS-456', 40) RETURNING id`).Scan(&busID)
	if err != nil {
		t.Fatalf("Failed to create bus: %v", err)
	}

	err = testDB.QueryRow(`INSERT INTO users (username, password_hash, role, full_name) VALUES ('driver2', 'hash', 'driver', 'Driver 2') RETURNING id`).Scan(&driverID)
	if err != nil {
		t.Fatalf("Failed to create driver: %v", err)
	}

	_, err = testDB.Exec(`INSERT INTO route_assignments (route_id, bus_id, driver_id) VALUES ($1, $2, $3)`, routeID, busID, driverID)
	if err != nil {
		t.Fatalf("Failed to create assignment: %v", err)
	}

	// Try to delete route
	req := httptest.NewRequest(http.MethodDelete, "/routes/"+routeID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", w.Code)
	}
}

func TestRouteValidation(t *testing.T) {
	testDB = setupTestDB(t)
	defer testDB.Close()
	db = testDB

	router := setupTestRouter()

	t.Run("InvalidRouteName", func(t *testing.T) {
		reqBody := CreateRouteRequest{
			Name: "", // Empty name
			Stops: []StopCreate{
				{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
				{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
	})

	t.Run("TooFewStops", func(t *testing.T) {
		reqBody := CreateRouteRequest{
			Name: "Route with Few Stops",
			Stops: []StopCreate{
				{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
	})

	t.Run("InvalidCoordinates", func(t *testing.T) {
		reqBody := CreateRouteRequest{
			Name: "Route with Invalid Coords",
			Stops: []StopCreate{
				{Name: "Stop 1", Latitude: 91.0, Longitude: 90.4074}, // Invalid latitude
				{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
			},
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("Expected status 422, got %d", w.Code)
		}
	})
}

// Test Redis pub/sub if available
func TestRouteEventPublishing(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL not set, skipping Redis pub/sub test")
	}

	testDB = setupTestDB(t)
	defer testDB.Close()
	db = testDB

	var err error
	redisClient, err = sharedredis.NewClient(sharedredis.DefaultConfig(redisURL))
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	pubsub = sharedredis.NewPubSubManager(redisClient)
	defer pubsub.Close()

	// Subscribe to route updates
	eventReceived := make(chan RouteUpdateEvent, 1)
	handler := func(channel string, payload []byte) error {
		var event RouteUpdateEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return err
		}
		eventReceived <- event
		return nil
	}

	if err := pubsub.Subscribe("route:updates", handler); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	router := setupTestRouter()

	// Create a route
	reqBody := CreateRouteRequest{
		Name: "PubSub Test Route",
		Stops: []StopCreate{
			{Name: "Stop 1", Latitude: 23.7104, Longitude: 90.4074},
			{Name: "Stop 2", Latitude: 23.7200, Longitude: 90.4100},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create route: %d", w.Code)
	}

	// Wait for event
	select {
	case event := <-eventReceived:
		if event.Action != "created" {
			t.Errorf("Expected action 'created', got '%s'", event.Action)
		}
		if event.Route == nil {
			t.Error("Expected route data in event")
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for route event")
	}
}
