package security

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"backend-go/pkg/config"
)

func TestJWT(t *testing.T) {
	// Setup
	config.AppConfig.JWTSecret = "test-secret"
	config.AppConfig.JWTIssuer = "test-issuer"
	config.AppConfig.JWTAudience = "test-audience"
	userID := "user-123"
	email := "test@test.com"
	roleID := "admin"
	permissions := []Permission{
		{Feature: "user", View: true, Create: true},
	}

	t.Run("Generate and Validate Token Success", func(t *testing.T) {
		token, err := GenerateToken(userID, email, roleID, permissions, 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, roleID, claims.RoleID)
		assert.Equal(t, permissions, claims.Permissions)
		assert.Equal(t, 1, claims.SessionVersion)
	})

	t.Run("Validate Token - Invalid Token String", func(t *testing.T) {
		claims, err := ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Validate Token - Wrong Secret", func(t *testing.T) {
		token, _ := GenerateToken(userID, email, roleID, permissions, 1)
		
		// Change secret temporarily
		originalSecret := config.AppConfig.JWTSecret
		config.AppConfig.JWTSecret = "wrong-secret"
		defer func() { config.AppConfig.JWTSecret = originalSecret }()

		claims, err := ValidateToken(token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("Validate Token - Expired", func(t *testing.T) {
		// Manually create an expired token
		claims := &JWTClaims{
			UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))

		resultClaims, err := ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, resultClaims)
	})

	t.Run("Generate Token with custom expiry", func(t *testing.T) {
		orig := config.AppConfig.JWTAccessExpiry
		config.AppConfig.JWTAccessExpiry = "30m"
		defer func() { config.AppConfig.JWTAccessExpiry = orig }()

		token, err := GenerateToken(userID, email, roleID, permissions, 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(token)
		assert.NoError(t, err)
		assert.Equal(t, config.AppConfig.JWTIssuer, claims.Issuer)
		assert.Contains(t, claims.Audience, config.AppConfig.JWTAudience)
	})

	t.Run("Generate Refresh Token with custom expiry", func(t *testing.T) {
		orig := config.AppConfig.JWTRefreshExpiry
		config.AppConfig.JWTRefreshExpiry = "72h"
		defer func() { config.AppConfig.JWTRefreshExpiry = orig }()

		token, err := GenerateRefreshToken(userID, email, roleID)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ValidateToken(token)
		assert.NoError(t, err)
		assert.Empty(t, claims.Permissions)
	})

	t.Run("Validate Token - Invalid but no error", func(t *testing.T) {
		origParse := jwtParseWithClaims
		defer func() { jwtParseWithClaims = origParse }()

		jwtParseWithClaims = func(tokenString string, claims jwt.Claims, keyFunc jwt.Keyfunc, options ...jwt.ParserOption) (*jwt.Token, error) {
			return &jwt.Token{Valid: false, Claims: &JWTClaims{}}, nil
		}

		claims, err := ValidateToken("any-token")
		assert.ErrorIs(t, err, jwt.ErrSignatureInvalid)
		assert.Nil(t, claims)
	})
}

func TestPermissionBitset(t *testing.T) {
	perms := []Permission{
		{Feature: "user", View: true, Create: true},
		{Feature: "product", View: true, Create: false, Delete: true, Activate: true},
		{Feature: "dashboard", View: false, Create: false, Delete: false, Activate: false},
	}

	bitset := CompilePermissions(perms)

	t.Run("HasPermission returns true for allowed actions", func(t *testing.T) {
		assert.True(t, bitset.HasPermission("user", "view"))
		assert.True(t, bitset.HasPermission("user", "create"))
		assert.True(t, bitset.HasPermission("product", "view"))
		assert.True(t, bitset.HasPermission("product", "delete"))
		assert.True(t, bitset.HasPermission("product", "activate"))
	})

	t.Run("HasPermission returns false for denied actions", func(t *testing.T) {
		assert.False(t, bitset.HasPermission("user", "delete"))
		assert.False(t, bitset.HasPermission("user", "activate"))
		assert.False(t, bitset.HasPermission("product", "create"))
	})

	t.Run("HasPermission returns false for unknown feature", func(t *testing.T) {
		assert.False(t, bitset.HasPermission("settings", "view"))
	})

	t.Run("HasPermission returns false for all-false feature", func(t *testing.T) {
		assert.False(t, bitset.HasPermission("dashboard", "view"))
		assert.False(t, bitset.HasPermission("dashboard", "create"))
		assert.False(t, bitset.HasPermission("dashboard", "delete"))
		assert.False(t, bitset.HasPermission("dashboard", "activate"))
	})

	t.Run("HasPermission returns false for invalid action", func(t *testing.T) {
		assert.False(t, bitset.HasPermission("user", "invalid_action"))
	})
}

func TestPassword(t *testing.T) {
	password := "secret123"
	
	t.Run("Hash and Check Success", func(t *testing.T) {
		hash, err := HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		
		assert.True(t, CheckPasswordHash(password, hash))
	})
	
	t.Run("Check Failure", func(t *testing.T) {
		hash, _ := HashPassword(password)
		assert.False(t, CheckPasswordHash("wrong-password", hash))
	})
}
