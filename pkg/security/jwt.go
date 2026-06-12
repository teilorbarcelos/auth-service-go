package security

import (
	"time"

	"backend-go/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

type Permission struct {
	Feature  string `json:"feature"`
	View     bool   `json:"view"`
	Create   bool   `json:"create"`
	Delete   bool   `json:"delete"`
	Activate bool   `json:"activate"`
}

type PermissionBitset map[string]uint8

const (
	PermView    uint8 = 1 << 0
	PermCreate  uint8 = 1 << 1
	PermDelete  uint8 = 1 << 2
	PermActivate uint8 = 1 << 3
)

func CompilePermissions(perms []Permission) PermissionBitset {
	bitset := make(PermissionBitset, len(perms))
	for _, p := range perms {
		var bits uint8
		if p.View {
			bits |= PermView
		}
		if p.Create {
			bits |= PermCreate
		}
		if p.Delete {
			bits |= PermDelete
		}
		if p.Activate {
			bits |= PermActivate
		}
		bitset[p.Feature] = bits
	}
	return bitset
}

func (pb PermissionBitset) HasPermission(feature string, action string) bool {
	bits, ok := pb[feature]
	if !ok {
		return false
	}
	switch action {
	case "view":
		return bits&PermView != 0
	case "create":
		return bits&PermCreate != 0
	case "delete":
		return bits&PermDelete != 0
	case "activate":
		return bits&PermActivate != 0
	}
	return false
}

type JWTClaims struct {
	UserID         string       `json:"id"`
	Email          string       `json:"email"`
	RoleID         string       `json:"roleId"`
	SessionVersion int          `json:"sessionVersion"`
	Permissions    []Permission `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

var jwtParseWithClaims = jwt.ParseWithClaims

func GenerateToken(userID, email, roleID string, permissions []Permission, sessionVersion int) (string, error) {
	expiry := 15 * time.Minute
	if d, err := time.ParseDuration(config.AppConfig.JWTAccessExpiry); err == nil {
		expiry = d
	}
	return generateToken(userID, email, roleID, permissions, sessionVersion, expiry)
}

func GenerateRefreshToken(userID, email, roleID string) (string, error) {
	expiry := 7 * 24 * time.Hour
	if d, err := time.ParseDuration(config.AppConfig.JWTRefreshExpiry); err == nil {
		expiry = d
	}
	return generateToken(userID, email, roleID, nil, 0, expiry)
}

func generateToken(userID, email, roleID string, permissions []Permission, sessionVersion int, duration time.Duration) (string, error) {
	claims := &JWTClaims{
		UserID:         userID,
		Email:          email,
		RoleID:         roleID,
		SessionVersion: sessionVersion,
		Permissions:    permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    config.AppConfig.JWTIssuer,
			Audience:  jwt.ClaimStrings{config.AppConfig.JWTAudience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwtParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
