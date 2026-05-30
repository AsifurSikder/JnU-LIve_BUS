# JWT Utilities

This package provides JWT (JSON Web Token) generation, validation, and parsing utilities for the University Bus Tracker system.

## Features

- **Role-based token expiry**: Different expiry times for driver (12 hours) and admin (8 hours) tokens
- **Token generation**: Create signed JWT tokens with user ID and role claims
- **Token validation**: Validate tokens with signature verification and expiry checking
- **Token parsing**: Parse tokens without validation (useful for debugging)
- **Configuration loading**: Load JWT configuration with secret and expiry settings

## Usage

### Generate a Token

```go
import (
    "time"
    "github.com/university-bus-tracker/shared/jwt"
)

// Configuration
secret := "your-secret-key"
driverExpiry := 12 * time.Hour
adminExpiry := 8 * time.Hour

// Generate a driver token
tokenString, expiryTime, err := jwt.GenerateToken(
    "user-123",      // User ID
    "driver",        // Role
    secret,
    driverExpiry,
    adminExpiry,
)
if err != nil {
    // Handle error
}

// Generate an admin token
tokenString, expiryTime, err := jwt.GenerateToken(
    "admin-456",     // User ID
    "admin",         // Role
    secret,
    driverExpiry,
    adminExpiry,
)
```

### Validate a Token

```go
import "github.com/university-bus-tracker/shared/jwt"

claims, err := jwt.ValidateToken(tokenString, secret)
if err != nil {
    switch err {
    case jwt.ErrExpiredToken:
        // Token has expired
    case jwt.ErrInvalidSignature:
        // Invalid signature
    case jwt.ErrMalformedToken:
        // Malformed token
    default:
        // Other validation error
    }
    return
}

// Access claims
userID := claims.Sub
role := claims.Role
expiryTimestamp := claims.Exp
issuedAtTimestamp := claims.Iat
```

### Parse a Token (Without Validation)

```go
import "github.com/university-bus-tracker/shared/jwt"

// Parse token without validating signature or expiry
claims, err := jwt.ParseToken(tokenString)
if err != nil {
    // Handle parsing error
}

// Access claims (same as ValidateToken)
userID := claims.Sub
role := claims.Role
```

### Load JWT Configuration

```go
import (
    "time"
    "github.com/university-bus-tracker/shared/jwt"
    "github.com/university-bus-tracker/shared/config"
)

// Load secret from environment
secret := config.MustLoadEnv("JWT_SECRET")

// Create configuration
jwtConfig, err := jwt.LoadJWTConfig(
    secret,
    12 * time.Hour, // Driver token expiry
    8 * time.Hour,  // Admin token expiry
)
if err != nil {
    // Handle error
}

// Use configuration
tokenString, expiryTime, err := jwt.GenerateToken(
    userID,
    role,
    jwtConfig.Secret,
    jwtConfig.DriverTokenExpiry,
    jwtConfig.AdminTokenExpiry,
)
```

## JWT Claims Structure

The JWT tokens contain the following claims:

```json
{
  "sub": "user-uuid",           // Subject (User ID)
  "role": "driver|admin",       // User role
  "exp": 1234567890,            // Expiry timestamp (Unix)
  "iat": 1234567890             // Issued at timestamp (Unix)
}
```

## Error Handling

The package provides specific error types for different validation failures:

- `ErrInvalidToken`: Generic invalid token error
- `ErrExpiredToken`: Token has expired
- `ErrMalformedToken`: Token is malformed
- `ErrInvalidSignature`: Token signature is invalid

## Security Considerations

1. **Secret Key**: Always use a strong, randomly generated secret key (at least 32 bytes)
2. **HTTPS**: Always transmit tokens over HTTPS in production
3. **Storage**: Store tokens securely on the client side (encrypted SharedPreferences on Android, secure storage on web)
4. **Expiry**: Tokens automatically expire based on role (12h for drivers, 8h for admins)
5. **Validation**: Always validate tokens on the server side before processing requests

## Testing

Run the unit tests:

```bash
go test -v ./jwt/...
```

Run tests with coverage:

```bash
go test -v -cover ./jwt/...
```

## Requirements

This package validates the following requirements:

- **Requirement 3.1**: Driver authentication with JWT (12-hour expiry)
- **Requirement 4.1**: Admin authentication with JWT (8-hour expiry)

## Dependencies

- `github.com/golang-jwt/jwt/v5`: JWT implementation for Go
