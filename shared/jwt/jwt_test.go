package jwt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	config := DefaultConfig("test-secret-key")
	manager := NewManager(config)

	tests := []struct {
		name        string
		userID      string
		role        Role
		expectError bool
	}{
		{
			name:        "valid driver token",
			userID:      "user-123",
			role:        RoleDriver,
			expectError: false,
		},
		{
			name:        "valid admin token",
			userID:      "admin-456",
			role:        RoleAdmin,
			expectError: false,
		},
		{
			name:        "empty userID",
			userID:      "",
			role:        RoleDriver,
			expectError: true,
		},
		{
			name:        "invalid role",
			userID:      "user-123",
			role:        Role("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := manager.GenerateToken(tt.userID, tt.role)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, token)
			assert.True(t, expiresAt.After(time.Now()))

			// Verify expiry time based on role
			expectedExpiry := manager.GetExpiryForRole(tt.role)
			actualExpiry := expiresAt.Sub(time.Now())
			assert.InDelta(t, expectedExpiry.Seconds(), actualExpiry.Seconds(), 5.0)
		})
	}
}

func TestValidateToken(t *testing.T) {
	config := DefaultConfig("test-secret-key")
	manager := NewManager(config)

	// Generate a valid token
	validToken, _, err := manager.GenerateToken("user-123", RoleDriver)
	require.NoError(t, err)

	// Generate an expired token
	expiredConfig := &Config{
		Secret:            "test-secret-key",
		DriverTokenExpiry: -1 * time.Hour, // Already expired
		AdminTokenExpiry:  8 * time.Hour,
	}
	expiredManager := NewManager(expiredConfig)
	expiredToken, _, err := expiredManager.GenerateToken("user-456", RoleDriver)
	require.NoError(t, err)

	// Generate token with wrong secret
	wrongSecretManager := NewManager(DefaultConfig("wrong-secret"))
	wrongSecretToken, _, err := wrongSecretManager.GenerateToken("user-789", RoleAdmin)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError error
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: nil,
		},
		{
			name:        "expired token",
			token:       expiredToken,
			expectError: ErrExpiredToken,
		},
		{
			name:        "invalid signature",
			token:       wrongSecretToken,
			expectError: ErrInvalidSignature,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: ErrInvalidToken,
		},
		{
			name:        "malformed token",
			token:       "not.a.valid.token",
			expectError: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(tt.token)

			if tt.expectError != nil {
				assert.ErrorIs(t, err, tt.expectError)
				assert.Nil(t, claims)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, "user-123", claims.UserID)
			assert.Equal(t, RoleDriver, claims.Role)
		})
	}
}

func TestRoleBasedExpiry(t *testing.T) {
	config := DefaultConfig("test-secret-key")
	manager := NewManager(config)

	// Test driver token expiry (12 hours)
	driverToken, driverExpiry, err := manager.GenerateToken("driver-1", RoleDriver)
	require.NoError(t, err)

	driverClaims, err := manager.ValidateToken(driverToken)
	require.NoError(t, err)

	expectedDriverExpiry := time.Now().Add(12 * time.Hour)
	assert.InDelta(t, expectedDriverExpiry.Unix(), driverExpiry.Unix(), 5.0)
	assert.InDelta(t, expectedDriverExpiry.Unix(), driverClaims.ExpiresAt.Unix(), 5.0)

	// Test admin token expiry (8 hours)
	adminToken, adminExpiry, err := manager.GenerateToken("admin-1", RoleAdmin)
	require.NoError(t, err)

	adminClaims, err := manager.ValidateToken(adminToken)
	require.NoError(t, err)

	expectedAdminExpiry := time.Now().Add(8 * time.Hour)
	assert.InDelta(t, expectedAdminExpiry.Unix(), adminExpiry.Unix(), 5.0)
	assert.InDelta(t, expectedAdminExpiry.Unix(), adminClaims.ExpiresAt.Unix(), 5.0)
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		expected    string
		expectError bool
	}{
		{
			name:        "valid bearer token",
			header:      "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectError: false,
		},
		{
			name:        "empty header",
			header:      "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "missing Bearer prefix",
			header:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Bearer without token",
			header:      "Bearer ",
			expected:    "",
			expectError: true,
		},
		{
			name:        "wrong prefix",
			header:      "Basic eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractTokenFromHeader(tt.header)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, token)
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	config := DefaultConfig("test-secret-key")
	manager := NewManager(config)

	// Valid token (not expired)
	validToken, _, err := manager.GenerateToken("user-123", RoleDriver)
	require.NoError(t, err)
	validClaims, err := manager.ValidateToken(validToken)
	require.NoError(t, err)
	assert.False(t, IsTokenExpired(validClaims))

	// Expired token
	expiredConfig := &Config{
		Secret:            "test-secret-key",
		DriverTokenExpiry: -1 * time.Hour,
		AdminTokenExpiry:  8 * time.Hour,
	}
	expiredManager := NewManager(expiredConfig)
	expiredToken, _, err := expiredManager.GenerateToken("user-456", RoleDriver)
	require.NoError(t, err)
	expiredClaims, _ := expiredManager.ParseToken(expiredToken)
	assert.True(t, IsTokenExpired(expiredClaims))
}
