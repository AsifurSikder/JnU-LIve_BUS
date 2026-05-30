package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Route struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Stop struct {
	ID        string  `json:"id"`
	RouteID   string  `json:"route_id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	StopOrder int     `json:"stop_order"`
}

var db *sql.DB

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := getEnv("PORT", "8083")
	dbURL := getEnv("DATABASE_URL", "")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
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

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "route-service"})
	})
	router.GET("/routes", getRoutes)
	router.POST("/routes", createRoute)
	router.GET("/routes/:id/stops", getStops)
	router.POST("/routes/:id/stops", createStop)

	server := &http.Server{Addr: ":" + port, Handler: router}

	go func() {
		log.Printf("Route Service starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	server.Shutdown(shutCtx)
}

func getRoutes(c *gin.Context) {
	rows, err := db.QueryContext(c.Request.Context(),
		`SELECT id, name, created_at FROM routes ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	routes := []Route{}
	for rows.Next() {
		var r Route
		rows.Scan(&r.ID, &r.Name, &r.CreatedAt)
		routes = append(routes, r)
	}
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func createRoute(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	var r Route
	err := db.QueryRowContext(c.Request.Context(),
		`INSERT INTO routes (name) VALUES ($1) RETURNING id, name, created_at`,
		body.Name,
	).Scan(&r.ID, &r.Name, &r.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, r)
}

func getStops(c *gin.Context) {
	routeID := c.Param("id")
	rows, err := db.QueryContext(c.Request.Context(),
		`SELECT id, route_id, name, latitude, longitude, stop_order FROM stops WHERE route_id=$1 ORDER BY stop_order`,
		routeID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	stops := []Stop{}
	for rows.Next() {
		var s Stop
		rows.Scan(&s.ID, &s.RouteID, &s.Name, &s.Latitude, &s.Longitude, &s.StopOrder)
		stops = append(stops, s)
	}
	c.JSON(http.StatusOK, gin.H{"stops": stops})
}

func createStop(c *gin.Context) {
	routeID := c.Param("id")
	var body struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		StopOrder int     `json:"stop_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var s Stop
	err := db.QueryRowContext(c.Request.Context(),
		`INSERT INTO stops (route_id, name, latitude, longitude, stop_order) VALUES ($1,$2,$3,$4,$5) RETURNING id, route_id, name, latitude, longitude, stop_order`,
		routeID, body.Name, body.Latitude, body.Longitude, body.StopOrder,
	).Scan(&s.ID, &s.RouteID, &s.Name, &s.Latitude, &s.Longitude, &s.StopOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
