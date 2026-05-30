package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// LoadEnv loads an environment variable with a fallback default value
func LoadEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// LoadEnvInt loads an integer environment variable with a fallback default value
func LoadEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// LoadEnvDuration loads a duration environment variable with a fallback default value
func LoadEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// MustLoadEnv loads an environment variable and panics if it's not set
func MustLoadEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
