package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
	ErrMissingClaims    = errors.New("missing required claims")
)

// Role represents user roles in the system
type Role string

const (
	RoleDriver Role = "driver"
	RoleAdmin  Role = "admin"
)

// Claims represents JWT claims structure
type Claims struct {
	UserID string `json:"sub"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT configuration
type Config struct {
	Secret            string
	DriverTokenExpiry time.Duration // 12 hours
	AdminTokenExpiry  time.Duration // 8 hours
}

// DefaultConfig returns a Config with default values
func DefaultConfig(secret string) *Config {
	return &Config{
		Secret:            secret,
		DriverTokenExpiry: 12 * time.Hour,
		AdminTokenExpiry:  8 * time.Hour,
	}
}

// Manager handles JWT operations
type Manager struct {
	config *Config
}

// NewManager creates a new JWT manager
func NewManager(config *Config) *Manager {
	return &Manager{
		config: config,
	}
}

// GenerateToken generates a JWT token for a user with role-based expiry
func (m *Manager) GenerateToken(userID string, role Role) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, errors.New("userID cannot be empty")
	}

	if role != RoleDriver && role != RoleAdmin {
		return "", time.Time{}, fmt.Errorf("invalid role: %s", role)
	}

	now := time.Now()
	var expiresAt time.Time

	// Role-based expiry
	switch role {
	case RoleDriver:
		expiresAt = now.Add(m.config.DriverTokenExpiry)
	case RoleAdmin:
		expiresAt = now.Add(m.config.AdminTokenExpiry)
	}

	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.config.Secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate required claims
	if claims.UserID == "" || claims.Role == "" {
		return nil, ErrMissingClaims
	}

	return claims, nil
}

// ParseToken parses a token without validation (useful for debugging)
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	return claims, nil
}

// GetExpiryForRole returns the expiry duration for a given role
func (m *Manager) GetExpiryForRole(role Role) time.Duration {
	switch role {
	case RoleDriver:
		return m.config.DriverTokenExpiry
	case RoleAdmin:
		return m.config.AdminTokenExpiry
	default:
		return 0
	}
}

// IsTokenExpired checks if a token is expired without full validation
func IsTokenExpired(claims *Claims) bool {
	return claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now())
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
// Expected format: "Bearer <token>"
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", errors.New("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is empty")
	}

	return token, nil
}
