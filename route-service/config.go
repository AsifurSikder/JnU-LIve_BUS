package main

import (
	"os"
	"time"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	Environment string
}

func LoadConfig() *Config {
	port := getEnv("PORT", "8083")
	databaseURL := getEnv("DATABASE_URL", "")
	redisURL := getEnv("REDIS_URL", "")
	environment := getEnv("ENVIRONMENT", "development")

	if databaseURL == "" {
		panic("DATABASE_URL environment variable is required")
	}

	return &Config{
		Port:        port,
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
		Environment: environment,
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
		var intValue int
		if _, err := time.ParseDuration(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
