package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Test fixtures
const (
	testUserID   = "4d890d9a-cea8-4c5b-9135-bc858a6bd246"
	testKID      = "test-key-id"
	testIssuer   = "test-issuer"
	testAudience = "test-audience"
)

// generateTestRSAKey generates a test RSA key pair
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	return privateKey
}

// createTempKeyFile creates a temporary PKCS#8 format key file
func createTempKeyFile(t *testing.T, privateKey *rsa.PrivateKey) string {
	t.Helper()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private_key.pem")

	keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal PKCS8: %v", err)
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	if err := pem.Encode(file, pemBlock); err != nil {
		t.Fatalf("Failed to encode PEM: %v", err)
	}

	return keyPath
}

// TestValidateConfig tests the validateConfig function
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name: "valid config",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: nil,
		},
		{
			name: "empty kid",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      "",
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: ErrEmptyKID,
		},
		{
			name: "empty issuer",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   "",
				Audience: testAudience,
			},
			wantErr: ErrEmptyIssuer,
		},
		{
			name: "empty audience",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: "",
			},
			wantErr: ErrEmptyAudience,
		},
		{
			name: "empty user id",
			config: Config{
				UserID:   "",
				Role:     RoleUser,
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: ErrEmptyUserID,
		},
		{
			name: "zero duration",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: 0,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: ErrInvalidDuration,
		},
		{
			name: "negative duration",
			config: Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: -1 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: ErrInvalidDuration,
		},
		{
			name: "invalid role",
			config: Config{
				UserID:   testUserID,
				Role:     "invalid",
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			},
			wantErr: ErrInvalidRole,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIsValidRole tests the isValidRole function
func TestIsValidRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{name: "admin role", role: RoleAdmin, want: true},
		{name: "user role", role: RoleUser, want: true},
		{name: "invalid role", role: "invalid", want: false},
		{name: "empty role", role: "", want: false},
		{name: "uppercase role", role: "ADMIN", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidRole(tt.role); got != tt.want {
				t.Errorf("isValidRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoadPrivateKey tests loading PKCS#8 format keys
func TestLoadPrivateKey(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	keyPath := createTempKeyFile(t, privateKey)

	loadedKey, err := loadPrivateKey(keyPath)
	if err != nil {
		t.Fatalf("loadPrivateKey() error = %v", err)
	}

	if loadedKey.N.Cmp(privateKey.N) != 0 {
		t.Error("Loaded key does not match original key")
	}
}

// TestLoadPrivateKey_FileNotFound tests error handling for missing files
func TestLoadPrivateKey_FileNotFound(t *testing.T) {
	_, err := loadPrivateKey("/nonexistent/path/key.pem")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read private key file") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestLoadPrivateKey_InvalidPEM tests error handling for invalid PEM data
func TestLoadPrivateKey_InvalidPEM(t *testing.T) {
	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "invalid.pem")

	if err := os.WriteFile(invalidKeyPath, []byte("not a valid pem file"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := loadPrivateKey(invalidKeyPath)
	if err == nil {
		t.Error("Expected error for invalid PEM data, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode PEM block") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestGenerateJWT tests JWT generation
func TestGenerateJWT(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	now := time.Date(2024, 12, 5, 12, 0, 0, 0, time.UTC)

	cfg := Config{
		UserID:   testUserID,
		Role:     RoleUser,
		KID:      testKID,
		Duration: 15 * time.Minute,
		Issuer:   testIssuer,
		Audience: testAudience,
	}

	tokenString, claims, err := GenerateJWT(cfg, privateKey, now)
	if err != nil {
		t.Fatalf("generateJWT() error = %v", err)
	}

	// Verify token is not empty
	if tokenString == "" {
		t.Error("Generated token is empty")
	}

	// Verify claims
	if claims.UserID != testUserID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, testUserID)
	}
	if claims.Role != RoleUser {
		t.Errorf("claims.Role = %v, want %v", claims.Role, RoleUser)
	}
	if claims.Issuer != testIssuer {
		t.Errorf("claims.Issuer = %v, want %v", claims.Issuer, testIssuer)
	}
	if claims.Subject != testUserID {
		t.Errorf("claims.Subject = %v, want %v", claims.Subject, testUserID)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != testAudience {
		t.Errorf("claims.Audience = %v, want [%v]", claims.Audience, testAudience)
	}

	// Verify timestamps
	if !claims.IssuedAt.Equal(now) {
		t.Errorf("claims.IssuedAt = %v, want %v", claims.IssuedAt, now)
	}
	if !claims.NotBefore.Equal(now) {
		t.Errorf("claims.NotBefore = %v, want %v", claims.NotBefore, now)
	}
	expectedExpiry := now.Add(15 * time.Minute)
	if !claims.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("claims.ExpiresAt = %v, want %v", claims.ExpiresAt, expectedExpiry)
	}

	// Parse and verify token (without validating expiration for testing purposes)
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return &privateKey.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	// Verify kid in header
	if kid, ok := token.Header["kid"].(string); !ok || kid != testKID {
		t.Errorf("token.Header[kid] = %v, want %v", token.Header["kid"], testKID)
	}

	// Verify algorithm
	if alg, ok := token.Header["alg"].(string); !ok || alg != "RS256" {
		t.Errorf("token.Header[alg] = %v, want RS256", token.Header["alg"])
	}
}

// TestGenerateJWT_AllRoles tests JWT generation with all role types
func TestGenerateJWT_AllRoles(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	now := time.Now()

	roles := []string{RoleAdmin, RoleUser}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			cfg := Config{
				UserID:   testUserID,
				Role:     role,
				KID:      testKID,
				Duration: 15 * time.Minute,
				Issuer:   testIssuer,
				Audience: testAudience,
			}

			tokenString, claims, err := GenerateJWT(cfg, privateKey, now)
			if err != nil {
				t.Fatalf("generateJWT() error = %v", err)
			}

			if tokenString == "" {
				t.Error("Generated token is empty")
			}

			if claims.Role != role {
				t.Errorf("claims.Role = %v, want %v", claims.Role, role)
			}
		})
	}
}

// TestGenerateJWT_DifferentDurations tests JWT generation with various durations
func TestGenerateJWT_DifferentDurations(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	now := time.Date(2024, 12, 5, 12, 0, 0, 0, time.UTC)

	durations := []time.Duration{
		1 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		24 * time.Hour,
	}

	for _, duration := range durations {
		t.Run(duration.String(), func(t *testing.T) {
			cfg := Config{
				UserID:   testUserID,
				Role:     RoleUser,
				KID:      testKID,
				Duration: duration,
				Issuer:   testIssuer,
				Audience: testAudience,
			}

			_, claims, err := GenerateJWT(cfg, privateKey, now)
			if err != nil {
				t.Fatalf("generateJWT() error = %v", err)
			}

			expectedExpiry := now.Add(duration)
			if !claims.ExpiresAt.Equal(expectedExpiry) {
				t.Errorf("claims.ExpiresAt = %v, want %v", claims.ExpiresAt, expectedExpiry)
			}
		})
	}
}

// TestGenerateJWT_TokenVerification tests that generated tokens can be verified
func TestGenerateJWT_TokenVerification(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	now := time.Now()

	cfg := Config{
		UserID:   testUserID,
		Role:     RoleAdmin,
		KID:      testKID,
		Duration: 15 * time.Minute,
		Issuer:   testIssuer,
		Audience: testAudience,
	}

	tokenString, _, err := GenerateJWT(cfg, privateKey, now)
	if err != nil {
		t.Fatalf("generateJWT() error = %v", err)
	}

	// Parse and verify token
	parsedClaims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, parsedClaims, func(token *jwt.Token) (any, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return &privateKey.PublicKey, nil
	})
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if !token.Valid {
		t.Error("Token is not valid")
	}

	// Verify all claims match
	if parsedClaims.UserID != testUserID {
		t.Errorf("Parsed UserID = %v, want %v", parsedClaims.UserID, testUserID)
	}
	if parsedClaims.Role != RoleAdmin {
		t.Errorf("Parsed Role = %v, want %v", parsedClaims.Role, RoleAdmin)
	}
}
