package main

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	DriverTokenExpiry time.Duration
	AdminTokenExpiry  time.Duration
	BcryptCost        int
	Environment       string
	RedisURL          string
	RateLimitWindow   time.Duration
	RateLimitMaxAdmin int
	RateLimitMaxDriver int
}

func LoadConfig() *Config {
	port := getEnv("PORT", "8081")
	databaseURL := getEnv("DATABASE_URL", "")
	jwtSecret := getEnv("JWT_SECRET", "")
	environment := getEnv("ENVIRONMENT", "development")
	redisURL := getEnv("REDIS_URL", "")

	if jwtSecret == "" {
		panic("JWT_SECRET environment variable is required")
	}

	if databaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	driverTokenExpiry := getDurationEnv("DRIVER_TOKEN_EXPIRY", 12*time.Hour)
	adminTokenExpiry := getDurationEnv("ADMIN_TOKEN_EXPIRY", 8*time.Hour)
	bcryptCost := getIntEnv("BCRYPT_COST", 12)
	rateLimitWindow := getDurationEnv("RATE_LIMIT_WINDOW", 60*time.Second)
	rateLimitMaxAdmin := getIntEnv("RATE_LIMIT_MAX_ADMIN", 5)
	rateLimitMaxDriver := getIntEnv("RATE_LIMIT_MAX_DRIVER", 10)

	return &Config{
		Port:              port,
		DatabaseURL:       databaseURL,
		JWTSecret:         jwtSecret,
		DriverTokenExpiry: driverTokenExpiry,
		AdminTokenExpiry:  adminTokenExpiry,
		BcryptCost:        bcryptCost,
		Environment:       environment,
		RedisURL:          redisURL,
		RateLimitWindow:   rateLimitWindow,
		RateLimitMaxAdmin: rateLimitMaxAdmin,
		RateLimitMaxDriver: rateLimitMaxDriver,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
