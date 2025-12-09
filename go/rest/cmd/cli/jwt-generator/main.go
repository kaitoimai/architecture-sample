package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT payload structure
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// Config holds the configuration for JWT generation
type Config struct {
	UserID         string
	Role           string
	PrivateKeyPath string
	KID            string
	Duration       time.Duration
	Issuer         string
	Audience       string
}

var (
	ErrEmptyKID        = errors.New("kid cannot be empty")
	ErrEmptyIssuer     = errors.New("issuer cannot be empty")
	ErrEmptyAudience   = errors.New("audience cannot be empty")
	ErrEmptyUserID     = errors.New("user-id cannot be empty")
	ErrInvalidDuration = errors.New("duration must be positive")
	ErrInvalidRole     = errors.New("invalid role")
)

func main() {
	// CLI flags
	userID := flag.String("user-id", "test-user-123", "User ID to include in JWT")
	role := flag.String("role", RoleUser, fmt.Sprintf("User role (options: %s, %s)", RoleAdmin, RoleUser))
	privateKeyPath := flag.String("private-key", ".keys/private_key.pem", "Path to RSA private key")
	kid := flag.String("kid", "", "Key ID (kid) for JWT header (required)")
	duration := flag.Duration("duration", 15*time.Minute, "Token expiration duration (e.g., 15m, 1h, 24h)")
	issuer := flag.String("issuer", "go-sample-api", "Token issuer")
	audience := flag.String("audience", "go-sample-api", "Token audience")
	flag.Parse()

	cfg := Config{
		UserID:         *userID,
		Role:           *role,
		PrivateKeyPath: *privateKeyPath,
		KID:            *kid,
		Duration:       *duration,
		Issuer:         *issuer,
		Audience:       *audience,
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	// Load private key
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Failed to load private key: %v", err)
	}

	// Generate JWT token
	tokenString, claims, err := GenerateJWT(cfg, privateKey, time.Now())
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	// Output token
	printToken(tokenString, cfg.KID, claims)
}

// validateConfig validates the configuration
func validateConfig(cfg Config) error {
	if cfg.KID == "" {
		return ErrEmptyKID
	}
	if cfg.Issuer == "" {
		return ErrEmptyIssuer
	}
	if cfg.Audience == "" {
		return ErrEmptyAudience
	}
	if cfg.UserID == "" {
		return ErrEmptyUserID
	}
	if cfg.Duration <= 0 {
		return ErrInvalidDuration
	}
	if !isValidRole(cfg.Role) {
		return fmt.Errorf("%w: %s", ErrInvalidRole, cfg.Role)
	}
	return nil
}

// isValidRole checks if the role is valid
func isValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser
}

// GenerateJWT generates a JWT token with the given configuration
func GenerateJWT(cfg Config, privateKey *rsa.PrivateKey, now time.Time) (string, Claims, error) {
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

// printToken prints the generated token and its details
func printToken(tokenString, kid string, claims Claims) {
	fmt.Println("JWT Token generated successfully!")
	fmt.Println()
	fmt.Println("Token:")
	fmt.Println(tokenString)
	fmt.Println()
	fmt.Println("Header:")
	fmt.Printf("  Algorithm:  RS256\n")
	fmt.Printf("  Type:       JWT\n")
	fmt.Printf("  Key ID:     %s\n", kid)
	fmt.Println()
	fmt.Println("Claims:")
	fmt.Printf("  User ID:    %s\n", claims.UserID)
	fmt.Printf("  Role:       %s\n", claims.Role)
	fmt.Printf("  Issuer:     %s\n", claims.Issuer)
	fmt.Printf("  Audience:   %s\n", claims.Audience)
	fmt.Printf("  Issued At:  %s\n", claims.IssuedAt.Format(time.RFC3339))
	fmt.Printf("  Expires At: %s\n", claims.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("  Not Before: %s\n", claims.NotBefore.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  curl -H \"Authorization: Bearer %s\" http://localhost:8080/v1/hello\n", tokenString)
}

// loadPrivateKey loads RSA private key from PEM file in PKCS#8 format
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse PKCS#8 format (BEGIN PRIVATE KEY)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA private key")
	}

	return rsaKey, nil
}
