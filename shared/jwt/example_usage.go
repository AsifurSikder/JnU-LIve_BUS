package jwt

import (
	"fmt"
	"time"
)

// ExampleGenerateAndValidateToken demonstrates basic JWT generation and validation
func ExampleGenerateAndValidateToken() {
	// Configuration
	secret := "my-super-secret-key-change-in-production"
	driverExpiry := 12 * time.Hour
	adminExpiry := 8 * time.Hour

	// Generate a driver token
	userID := "driver-123"
	role := "driver"

	tokenString, expiryTime, err := GenerateToken(userID, role, secret, driverExpiry, adminExpiry)
	if err != nil {
		fmt.Printf("Error generating token: %v\n", err)
		return
	}

	fmt.Printf("Generated token: %s\n", tokenString)
	fmt.Printf("Token expires at: %s\n", expiryTime.Format(time.RFC3339))

	// Validate the token
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		fmt.Printf("Error validating token: %v\n", err)
		return
	}

	fmt.Printf("Token is valid!\n")
	fmt.Printf("User ID: %s\n", claims.Sub)
	fmt.Printf("Role: %s\n", claims.Role)
	fmt.Printf("Expires at: %s\n", time.Unix(claims.Exp, 0).Format(time.RFC3339))
	fmt.Printf("Issued at: %s\n", time.Unix(claims.Iat, 0).Format(time.RFC3339))
}

// ExampleHandleExpiredToken demonstrates handling expired tokens
func ExampleHandleExpiredToken() {
	secret := "my-super-secret-key"

	// Simulate an expired token (this would normally come from a client)
	expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyIsInJvbGUiOiJkcml2ZXIiLCJleHAiOjE2MDAwMDAwMDAsImlhdCI6MTYwMDAwMDAwMH0.invalid"

	claims, err := ValidateToken(expiredToken, secret)
	if err != nil {
		switch err {
		case ErrExpiredToken:
			fmt.Println("Token has expired. Please log in again.")
		case ErrInvalidSignature:
			fmt.Println("Invalid token signature. Authentication failed.")
		case ErrMalformedToken:
			fmt.Println("Malformed token. Please provide a valid token.")
		default:
			fmt.Printf("Token validation failed: %v\n", err)
		}
		return
	}

	fmt.Printf("Token is valid for user: %s\n", claims.Sub)
}

// ExampleLoadConfigAndGenerateToken demonstrates loading configuration and generating tokens
func ExampleLoadConfigAndGenerateToken() {
	// In production, load secret from environment variable
	// secret := config.MustLoadEnv("JWT_SECRET")
	secret := "production-secret-key"

	// Load JWT configuration
	jwtConfig, err := LoadJWTConfig(secret, 12*time.Hour, 8*time.Hour)
	if err != nil {
		fmt.Printf("Error loading JWT config: %v\n", err)
		return
	}

	// Generate tokens for different roles
	driverToken, _, err := GenerateToken(
		"driver-456",
		"driver",
		jwtConfig.Secret,
		jwtConfig.DriverTokenExpiry,
		jwtConfig.AdminTokenExpiry,
	)
	if err != nil {
		fmt.Printf("Error generating driver token: %v\n", err)
		return
	}

	adminToken, _, err := GenerateToken(
		"admin-789",
		"admin",
		jwtConfig.Secret,
		jwtConfig.DriverTokenExpiry,
		jwtConfig.AdminTokenExpiry,
	)
	if err != nil {
		fmt.Printf("Error generating admin token: %v\n", err)
		return
	}

	fmt.Printf("Driver token: %s\n", driverToken)
	fmt.Printf("Admin token: %s\n", adminToken)

	// Validate driver token
	driverClaims, err := ValidateToken(driverToken, jwtConfig.Secret)
	if err != nil {
		fmt.Printf("Error validating driver token: %v\n", err)
		return
	}

	// Validate admin token
	adminClaims, err := ValidateToken(adminToken, jwtConfig.Secret)
	if err != nil {
		fmt.Printf("Error validating admin token: %v\n", err)
		return
	}

	// Compare expiry times
	driverExpiryTime := time.Unix(driverClaims.Exp, 0)
	adminExpiryTime := time.Unix(adminClaims.Exp, 0)

	fmt.Printf("Driver token expires at: %s\n", driverExpiryTime.Format(time.RFC3339))
	fmt.Printf("Admin token expires at: %s\n", adminExpiryTime.Format(time.RFC3339))
	fmt.Printf("Driver token expires %v after admin token\n", driverExpiryTime.Sub(adminExpiryTime))
}

// ExampleMiddlewareUsage demonstrates how to use JWT validation in a middleware
func ExampleMiddlewareUsage() {
	secret := "middleware-secret-key"

	// Simulate receiving a token from Authorization header
	authHeader := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

	// Extract token from header (remove "Bearer " prefix)
	var tokenString string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenString = authHeader[7:]
	} else {
		fmt.Println("Invalid Authorization header format")
		return
	}

	// Validate token
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		switch err {
		case ErrExpiredToken:
			fmt.Println("HTTP 401: Token expired")
		case ErrInvalidSignature, ErrMalformedToken:
			fmt.Println("HTTP 401: Invalid token")
		default:
			fmt.Println("HTTP 401: Authentication failed")
		}
		return
	}

	// Check role-based access
	requiredRole := "admin"
	if claims.Role != requiredRole {
		fmt.Printf("HTTP 403: Forbidden. Required role: %s, got: %s\n", requiredRole, claims.Role)
		return
	}

	// Token is valid and user has required role
	fmt.Printf("Access granted for user %s with role %s\n", claims.Sub, claims.Role)
}
