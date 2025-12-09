package testutil

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT payload structure for testing
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTConfig holds the configuration for JWT generation
type JWTConfig struct {
	UserID   string
	Role     string
	KID      string
	Duration time.Duration
	Issuer   string
	Audience string
}

// GenerateJWT generates a JWT token with the given configuration for testing
func GenerateJWT(cfg JWTConfig, privateKey *rsa.PrivateKey, now time.Time) (string, Claims, error) {
	claims := Claims{
		UserID: cfg.UserID,
		Role:   cfg.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   cfg.UserID,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.Duration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        fmt.Sprintf("%d", now.Unix()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = cfg.KID

	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", Claims{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, claims, nil
}
