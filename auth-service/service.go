package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	
	sharedJWT "github.com/university-bus-tracker/shared/jwt"
	sharedRedis "github.com/university-bus-tracker/shared/redis"
)

type AuthService struct {
	db          *Database
	jwtManager  *sharedJWT.Manager
	config      *Config
	redisClient *sharedRedis.Client
	keyHelper   *sharedRedis.KeyHelper
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8,max=100"`
	Role     string `json:"role" binding:"required,oneof=driver admin"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	Role      string    `json:"role"`
	UserID    string    `json:"userId"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewAuthService(db *Database, config *Config) *AuthService {
	jwtConfig := &sharedJWT.Config{
		Secret:            config.JWTSecret,
		DriverTokenExpiry: config.DriverTokenExpiry,
		AdminTokenExpiry:  config.AdminTokenExpiry,
	}

	service := &AuthService{
		db:         db,
		jwtManager: sharedJWT.NewManager(jwtConfig),
		config:     config,
		keyHelper:  sharedRedis.NewKeyHelper(),
	}

	// Initialize Redis if URL is provided
	if config.RedisURL != "" {
		redisConfig := sharedRedis.DefaultConfig(config.RedisURL)
		redisClient, err := sharedRedis.NewClient(redisConfig)
		if err != nil {
			fmt.Printf("Warning: Failed to connect to Redis: %v\n", err)
		} else {
			service.redisClient = redisClient
		}
	}

	return service
}

func (s *AuthService) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid request body", err.Error())
		return
	}

	// Check rate limit
	if s.redisClient != nil {
		clientIP := c.ClientIP()
		if blocked, err := s.checkRateLimit(c.Request.Context(), clientIP, req.Role); err != nil {
			fmt.Printf("Rate limit check error: %v\n", err)
		} else if blocked {
			s.respondError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many failed login attempts", nil)
			return
		}
	}

	// Get user from database
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user, err := s.db.GetUserByUsernameAndRole(ctx, req.Username, req.Role)
	if err != nil {
		fmt.Printf("Database error: %v\n", err)
		s.respondError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable", nil)
		return
	}

	if user == nil {
		// Record failed attempt
		if s.redisClient != nil {
			s.recordFailedAttempt(c.Request.Context(), c.ClientIP())
		}
		s.respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials", nil)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// Record failed attempt
		if s.redisClient != nil {
			s.recordFailedAttempt(c.Request.Context(), c.ClientIP())
		}
		s.respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials", nil)
		return
	}

	// Generate JWT token
	role := sharedJWT.Role(req.Role)
	token, expiresAt, err := s.jwtManager.GenerateToken(user.ID, role)
	if err != nil {
		fmt.Printf("JWT generation error: %v\n", err)
		s.respondError(c, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate token", nil)
		return
	}

	// Clear failed attempts on successful login
	if s.redisClient != nil {
		s.clearFailedAttempts(c.Request.Context(), c.ClientIP())
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		Role:      req.Role,
		UserID:    user.ID,
	})
}

func (s *AuthService) checkRateLimit(ctx context.Context, ip, role string) (bool, error) {
	key := s.keyHelper.AuthRateLimit(ip)
	
	// Get current count
	countStr, err := s.redisClient.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		return false, err
	}

	var count int
	if countStr != "" {
		fmt.Sscanf(countStr, "%d", &count)
	}

	// Check threshold based on role
	maxAttempts := s.config.RateLimitMaxDriver
	if role == "admin" {
		maxAttempts = s.config.RateLimitMaxAdmin
	}

	return count >= maxAttempts, nil
}

func (s *AuthService) recordFailedAttempt(ctx context.Context, ip string) {
	key := s.keyHelper.AuthRateLimit(ip)
	
	// Get current count
	countStr, err := s.redisClient.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		return
	}

	var count int
	if countStr != "" {
		fmt.Sscanf(countStr, "%d", &count)
	}

	count++
	
	// Set with TTL
	s.redisClient.Set(ctx, key, fmt.Sprintf("%d", count), s.config.RateLimitWindow)
}

func (s *AuthService) clearFailedAttempts(ctx context.Context, ip string) {
	key := s.keyHelper.AuthRateLimit(ip)
	s.redisClient.Del(ctx, key)
}

func (s *AuthService) respondError(c *gin.Context, status int, code, message string, details interface{}) {
	c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			Details:   details,
			Timestamp: time.Now(),
		},
	})
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string, cost int) (string, error) {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	
	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}
	return nil
}
