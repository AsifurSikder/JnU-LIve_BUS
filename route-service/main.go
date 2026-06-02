package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	sharedredis "github.com/mdasifurrahmansikder/jnu-live-bus/shared/redis"
)

var (
	db          *sql.DB
	redisClient *sharedredis.Client
	pubsub      *sharedredis.PubSubManager
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	config := LoadConfig()

	var err error
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize Redis if configured
	if config.RedisURL != "" {
		redisClient, err = sharedredis.NewClient(sharedredis.DefaultConfig(config.RedisURL))
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v", err)
		} else {
			log.Println("Connected to Redis")
			pubsub = sharedredis.NewPubSubManager(redisClient)
			defer pubsub.Close()
		}
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "route-service"})
	})
	router.GET("/routes", getRoutesHandler)
	router.POST("/routes", createRouteHandler)
	router.PUT("/routes/:id", updateRouteHandler)
	router.DELETE("/routes/:id", deleteRouteHandler)
	router.POST("/routes/:id/assign", assignBusHandler)

	server := &http.Server{Addr: ":" + config.Port, Handler: router}

	go func() {
		log.Printf("Route Service starting on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if redisClient != nil {
		redisClient.Close()
	}
	server.Shutdown(shutCtx)
}

// getRoutesHandler fetches all routes with stops and assignments
func getRoutesHandler(c *gin.Context) {
	routes, err := getAllRoutes(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

// getAllRoutes retrieves all routes with stops and bus assignments
func getAllRoutes(ctx context.Context) ([]Route, error) {
	// Query routes with optional bus and driver info
	query := `
		SELECT 
			r.id, r.name, r.created_at, r.updated_at,
			ra.bus_id, u.full_name as driver_name
		FROM routes r
		LEFT JOIN route_assignments ra ON r.id = ra.route_id
		LEFT JOIN users u ON ra.driver_id = u.id
		ORDER BY r.created_at DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query routes: %w", err)
	}
	defer rows.Close()

	routeMap := make(map[string]*Route)
	var routeOrder []string

	for rows.Next() {
		var r Route
		var busID, driverName sql.NullString

		err := rows.Scan(&r.ID, &r.Name, &r.CreatedAt, &r.UpdatedAt, &busID, &driverName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}

		if busID.Valid {
			r.AssignedBusID = &busID.String
		}
		if driverName.Valid {
			r.AssignedDriverName = &driverName.String
		}

		routeMap[r.ID] = &r
		routeOrder = append(routeOrder, r.ID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating routes: %w", err)
	}

	// Fetch stops for all routes
	if len(routeMap) > 0 {
		routeIDs := make([]string, 0, len(routeMap))
		for id := range routeMap {
			routeIDs = append(routeIDs, id)
		}

		stopsQuery := `
			SELECT id, route_id, name, latitude, longitude, stop_order, created_at
			FROM stops
			WHERE route_id = ANY($1)
			ORDER BY route_id, stop_order
		`

		stopsRows, err := db.QueryContext(ctx, stopsQuery, routeIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to query stops: %w", err)
		}
		defer stopsRows.Close()

		for stopsRows.Next() {
			var s Stop
			err := stopsRows.Scan(&s.ID, &s.RouteID, &s.Name, &s.Latitude, &s.Longitude, &s.StopOrder, &s.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("failed to scan stop: %w", err)
			}

			if route, ok := routeMap[s.RouteID]; ok {
				route.Stops = append(route.Stops, s)
			}
		}

		if err = stopsRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating stops: %w", err)
		}
	}

	// Convert map to ordered slice
	routes := make([]Route, 0, len(routeOrder))
	for _, id := range routeOrder {
		if route, ok := routeMap[id]; ok {
			if route.Stops == nil {
				route.Stops = []Stop{}
			}
			routes = append(routes, *route)
		}
	}

	return routes, nil
}

// createRouteHandler creates a new route with stops
func createRouteHandler(c *gin.Context) {
	var req CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Validate route data
	if err := ValidateRouteData(req.Name, req.Stops); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Check for duplicate route name
	var exists bool
	err := db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM routes WHERE name=$1)", req.Name).Scan(&exists)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if exists {
		respondError(c, http.StatusConflict, "duplicate_route_name", "Route name already exists")
		return
	}

	// Begin transaction
	tx, err := db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	defer tx.Rollback()

	// Insert route
	var route Route
	err = tx.QueryRowContext(c.Request.Context(),
		`INSERT INTO routes (name) VALUES ($1) RETURNING id, name, created_at, updated_at`,
		req.Name,
	).Scan(&route.ID, &route.Name, &route.CreatedAt, &route.UpdatedAt)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Insert stops
	route.Stops = make([]Stop, 0, len(req.Stops))
	for i, stopReq := range req.Stops {
		var stop Stop
		err = tx.QueryRowContext(c.Request.Context(),
			`INSERT INTO stops (route_id, name, latitude, longitude, stop_order) 
			 VALUES ($1, $2, $3, $4, $5) 
			 RETURNING id, route_id, name, latitude, longitude, stop_order, created_at`,
			route.ID, stopReq.Name, stopReq.Latitude, stopReq.Longitude, i,
		).Scan(&stop.ID, &stop.RouteID, &stop.Name, &stop.Latitude, &stop.Longitude, &stop.StopOrder, &stop.CreatedAt)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "database_error", err.Error())
			return
		}
		route.Stops = append(route.Stops, stop)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Publish route creation event
	publishRouteEvent(c.Request.Context(), "created", route.ID, &route)

	c.JSON(http.StatusCreated, route)
}

// updateRouteHandler updates an existing route
func updateRouteHandler(c *gin.Context) {
	routeID := c.Param("id")

	var req UpdateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Validate route data
	if err := ValidateRouteData(req.Name, req.Stops); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Check if route exists
	var existingName string
	err := db.QueryRowContext(c.Request.Context(), "SELECT name FROM routes WHERE id=$1", routeID).Scan(&existingName)
	if err == sql.ErrNoRows {
		respondError(c, http.StatusNotFound, "route_not_found", "Route not found")
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Check for duplicate route name (if name changed)
	if req.Name != existingName {
		var exists bool
		err := db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM routes WHERE name=$1)", req.Name).Scan(&exists)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "database_error", err.Error())
			return
		}
		if exists {
			respondError(c, http.StatusConflict, "duplicate_route_name", "Route name already exists")
			return
		}
	}

	// Begin transaction
	tx, err := db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	defer tx.Rollback()

	// Update route
	var route Route
	err = tx.QueryRowContext(c.Request.Context(),
		`UPDATE routes SET name=$1, updated_at=NOW() WHERE id=$2 RETURNING id, name, created_at, updated_at`,
		req.Name, routeID,
	).Scan(&route.ID, &route.Name, &route.CreatedAt, &route.UpdatedAt)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Delete existing stops
	_, err = tx.ExecContext(c.Request.Context(), "DELETE FROM stops WHERE route_id=$1", routeID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Insert new stops
	route.Stops = make([]Stop, 0, len(req.Stops))
	for i, stopReq := range req.Stops {
		var stop Stop
		err = tx.QueryRowContext(c.Request.Context(),
			`INSERT INTO stops (route_id, name, latitude, longitude, stop_order) 
			 VALUES ($1, $2, $3, $4, $5) 
			 RETURNING id, route_id, name, latitude, longitude, stop_order, created_at`,
			route.ID, stopReq.Name, stopReq.Latitude, stopReq.Longitude, i,
		).Scan(&stop.ID, &stop.RouteID, &stop.Name, &stop.Latitude, &stop.Longitude, &stop.StopOrder, &stop.CreatedAt)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "database_error", err.Error())
			return
		}
		route.Stops = append(route.Stops, stop)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Publish route update event
	publishRouteEvent(c.Request.Context(), "updated", route.ID, &route)

	c.JSON(http.StatusOK, route)
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(c *gin.Context) {
	routeID := c.Param("id")

	// Check if route exists
	var exists bool
	err := db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM routes WHERE id=$1)", routeID).Scan(&exists)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if !exists {
		respondError(c, http.StatusNotFound, "route_not_found", "Route not found")
		return
	}

	// Check if route has active bus assigned
	var hasAssignment bool
	err = db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM route_assignments WHERE route_id=$1)", routeID).Scan(&hasAssignment)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if hasAssignment {
		respondError(c, http.StatusConflict, "route_has_active_bus", "Cannot delete route with active bus assignment")
		return
	}

	// Delete route (stops will cascade)
	_, err = db.ExecContext(c.Request.Context(), "DELETE FROM routes WHERE id=$1", routeID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	// Publish route deletion event
	publishRouteEvent(c.Request.Context(), "deleted", routeID, nil)

	c.Status(http.StatusNoContent)
}

// assignBusHandler assigns a bus to a route
func assignBusHandler(c *gin.Context) {
	routeID := c.Param("id")

	var req AssignBusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusUnprocessableEntity, "validation_error", err.Error())
		return
	}

	// Check if route exists
	var routeExists bool
	err := db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM routes WHERE id=$1)", routeID).Scan(&routeExists)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if !routeExists {
		respondError(c, http.StatusNotFound, "route_not_found", "Route not found")
		return
	}

	// Check if bus exists
	var busExists bool
	err = db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM buses WHERE id=$1)", req.BusID).Scan(&busExists)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if !busExists {
		respondError(c, http.StatusNotFound, "bus_not_found", "Bus not found")
		return
	}

	// Check if driver exists
	var driverExists bool
	err = db.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND role='driver')", req.DriverID).Scan(&driverExists)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}
	if !driverExists {
		respondError(c, http.StatusNotFound, "driver_not_found", "Driver not found")
		return
	}

	// Upsert route assignment
	var assignedAt time.Time
	err = db.QueryRowContext(c.Request.Context(),
		`INSERT INTO route_assignments (route_id, bus_id, driver_id, assigned_at) 
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (route_id) DO UPDATE 
		 SET bus_id=$2, driver_id=$3, assigned_at=NOW()
		 RETURNING assigned_at`,
		routeID, req.BusID, req.DriverID,
	).Scan(&assignedAt)
	if err != nil {
		// Handle unique constraint on bus_id
		if strings.Contains(err.Error(), "route_assignments_bus_id_key") {
			respondError(c, http.StatusConflict, "bus_already_assigned", "Bus is already assigned to another route")
			return
		}
		respondError(c, http.StatusInternalServerError, "database_error", err.Error())
		return
	}

	response := AssignmentResponse{
		RouteID:    routeID,
		BusID:      req.BusID,
		DriverID:   req.DriverID,
		AssignedAt: assignedAt,
	}

	c.JSON(http.StatusOK, response)
}

// publishRouteEvent publishes a route change event to Redis
func publishRouteEvent(ctx context.Context, action, routeID string, route *Route) {
	if pubsub == nil {
		return
	}

	event := RouteUpdateEvent{
		Action:  action,
		RouteID: routeID,
		Route:   route,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal route event: %v", err)
		return
	}

	if err := pubsub.Publish(ctx, "route:updates", data); err != nil {
		log.Printf("Failed to publish route event: %v", err)
	}
}

// respondError sends a standardized error response
func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Timestamp: time.Now(),
		},
	})
}
