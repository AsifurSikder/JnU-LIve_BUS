package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port := getEnv("PORT", "8080")
	authURL := getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
	locationURL := getEnv("LOCATION_SERVICE_URL", "http://localhost:8082")
	routeURL := getEnv("ROUTE_SERVICE_URL", "http://localhost:8083")
	jwtSecret := getEnv("JWT_SECRET", "")

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Public routes — no auth needed
	router.POST("/auth/login", proxyTo(authURL))

	// Protected routes — JWT required
	protected := router.Group("/")
	protected.Use(authMiddleware(jwtSecret))
	{
		// Location routes
		protected.POST("/location/update", proxyTo(locationURL))
		protected.GET("/location/buses", proxyTo(locationURL))

		// Route management
		protected.GET("/routes", proxyTo(routeURL))
		protected.POST("/routes", proxyTo(routeURL))
		protected.GET("/routes/:id/stops", proxyTo(routeURL))
		protected.POST("/routes/:id/stops", proxyTo(routeURL))
	}

	// WebSocket — no auth in header possible, skip JWT for now
	router.GET("/ws/location", proxyTo(locationURL))

	server := &http.Server{Addr: ":" + port, Handler: router}

	go func() {
		log.Printf("API Gateway starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")
}

func proxyTo(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		remote, err := url.Parse(target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid target"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Director = func(req *http.Request) {
			req.Header = c.Request.Header
			req.Host = remote.Host
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
