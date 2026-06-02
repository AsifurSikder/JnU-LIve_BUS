package jwt

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: university-bus-tracker, Property 1: JWT generation with role-based expiry
// Validates: Requirements 3.1, 4.1
// For any valid user credentials with a specified role (driver or admin), the generated JWT
// SHALL contain the correct role claim, have an expiry time matching the role-specific duration
// (12 hours for drivers, 8 hours for admins), and be verifiable with the signing secret.
func TestProperty_JWTGenerationWithRoleBasedExpiry(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 1000

	properties := gopter.NewProperties(parameters)

	// Create a manager for testing
	config := DefaultConfig("test-secret-for-property-testing")
	manager := NewManager(config)

	// Generator for user IDs (non-empty strings, 1-50 chars)
	userIDGen := gen.Identifier().Map(func(s string) string {
		// Ensure it's not empty and not too long
		if len(s) == 0 {
			return "user-id"
		}
		if len(s) > 50 {
			return s[:50]
		}
		return s
	})

	// Generator for roles (driver or admin)
	roleGen := gen.OneConstOf(RoleDriver, RoleAdmin)

	properties.Property("JWT contains correct role and expiry", prop.ForAll(
		func(userID string, role Role) bool {
			// Generate JWT token
			token, expiresAt, err := manager.GenerateToken(userID, role)
			if err != nil {
				t.Logf("Failed to generate token: %v", err)
				return false
			}

			// Verify token is not empty
			if token == "" {
				t.Logf("Generated token is empty")
				return false
			}

			// Verify expiresAt is in the future
			if !expiresAt.After(time.Now()) {
				t.Logf("expiresAt is not in the future: %v", expiresAt)
				return false
			}

			// Validate token and extract claims
			claims, err := manager.ValidateToken(token)
			if err != nil {
				t.Logf("Failed to validate token: %v", err)
				return false
			}

			// Verify userID matches
			if claims.UserID != userID {
				t.Logf("UserID mismatch: expected %s, got %s", userID, claims.UserID)
				return false
			}

			// Verify role matches
			if claims.Role != role {
				t.Logf("Role mismatch: expected %s, got %s", role, claims.Role)
				return false
			}

			// Verify expiry time matches role-specific duration
			expectedDuration := manager.GetExpiryForRole(role)
			actualDuration := expiresAt.Sub(time.Now())
			
			// Allow 10 second tolerance for test execution time
			tolerance := 10 * time.Second
			minDuration := expectedDuration - tolerance
			maxDuration := expectedDuration + tolerance

			if actualDuration < minDuration || actualDuration > maxDuration {
				t.Logf("Expiry duration out of range: expected ~%v, got %v", expectedDuration, actualDuration)
				return false
			}

			// Verify signature by attempting to validate with correct secret
			_, err = manager.ValidateToken(token)
			if err != nil {
				t.Logf("Token validation failed with correct secret: %v", err)
				return false
			}

			// Verify signature by attempting to validate with wrong secret
			wrongManager := NewManager(DefaultConfig("wrong-secret"))
			_, err = wrongManager.ValidateToken(token)
			if err == nil {
				t.Logf("Token validation succeeded with wrong secret (should have failed)")
				return false
			}

			return true
		},
		userIDGen,
		roleGen,
	))

	properties.Property("Driver tokens expire in 12 hours", prop.ForAll(
		func(userID string) bool {
			token, expiresAt, err := manager.GenerateToken(userID, RoleDriver)
			if err != nil {
				return false
			}

			// Validate token to get claims
			claims, err := manager.ValidateToken(token)
			if err != nil {
				return false
			}

			// Check that expiry is approximately 12 hours from now
			expectedExpiry := time.Now().Add(12 * time.Hour)
			actualExpiry := expiresAt
			
			// Allow 10 second tolerance
			diff := actualExpiry.Sub(expectedExpiry)
			if diff < -10*time.Second || diff > 10*time.Second {
				t.Logf("Driver token expiry mismatch: expected ~%v, got %v (diff: %v)", 
					expectedExpiry, actualExpiry, diff)
				return false
			}

			// Also verify claims expiry matches
			diff = claims.ExpiresAt.Sub(expectedExpiry)
			if diff < -10*time.Second || diff > 10*time.Second {
				return false
			}

			return true
		},
		userIDGen,
	))

	properties.Property("Admin tokens expire in 8 hours", prop.ForAll(
		func(userID string) bool {
			token, expiresAt, err := manager.GenerateToken(userID, RoleAdmin)
			if err != nil {
				return false
			}

			// Validate token to get claims
			claims, err := manager.ValidateToken(token)
			if err != nil {
				return false
			}

			// Check that expiry is approximately 8 hours from now
			expectedExpiry := time.Now().Add(8 * time.Hour)
			actualExpiry := expiresAt
			
			// Allow 10 second tolerance
			diff := actualExpiry.Sub(expectedExpiry)
			if diff < -10*time.Second || diff > 10*time.Second {
				t.Logf("Admin token expiry mismatch: expected ~%v, got %v (diff: %v)", 
					expectedExpiry, actualExpiry, diff)
				return false
			}

			// Also verify claims expiry matches
			diff = claims.ExpiresAt.Sub(expectedExpiry)
			if diff < -10*time.Second || diff > 10*time.Second {
				return false
			}

			return true
		},
		userIDGen,
	))

	properties.Property("Tokens with invalid signatures are rejected", prop.ForAll(
		func(userID string, role Role) bool {
			// Generate token with one secret
			token, _, err := manager.GenerateToken(userID, role)
			if err != nil {
				return false
			}

			// Try to validate with different secret
			differentManager := NewManager(DefaultConfig("different-secret-key"))
			_, err = differentManager.ValidateToken(token)
			
			// Should always return an error
			return err != nil
		},
		userIDGen,
		roleGen,
	))

	properties.TestingRun(t)
}

// Feature: university-bus-tracker, Property 1 (Edge Cases)
// Test edge cases for JWT generation
func TestProperty_JWTGenerationEdgeCases(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	config := DefaultConfig("test-secret")
	manager := NewManager(config)

	// Test that empty userID always fails
	properties.Property("Empty userID always fails", prop.ForAll(
		func(role Role) bool {
			_, _, err := manager.GenerateToken("", role)
			return err != nil
		},
		gen.OneConstOf(RoleDriver, RoleAdmin),
	))

	// Test that invalid role always fails
	properties.Property("Invalid role always fails", prop.ForAll(
		func(userID string) bool {
			_, _, err := manager.GenerateToken(userID, Role("invalid"))
			return err != nil
		},
		gen.Identifier(),
	))

	properties.TestingRun(t)
}
