package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// テスト用のRSAキーペアを生成
func generateTestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// 公開鍵をPEM形式に変換
func publicKeyToPEM(pub *rsa.PublicKey) (string, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})
	return string(pubPEM), nil
}

// テスト用のJWTトークンを生成
func generateTestToken(privateKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	return token.SignedString(privateKey)
}

func TestNewJWTMiddleware(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	config := JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
		SkipValidation: false,
		RequiredClaims: []string{"sub", "iss"},
	}

	middleware := NewJWTMiddleware(config)

	if middleware == nil {
		t.Fatal("NewJWTMiddleware returned nil")
	}

	if middleware.config.SkipValidation {
		t.Error("SkipValidation should be false")
	}

	if len(middleware.config.PublicKeys) != 1 {
		t.Errorf("expected 1 public key, got %d", len(middleware.config.PublicKeys))
	}

	// テスト用に privateKey を使用して検証
	_ = privateKey
}

func TestNewJWTMiddlewareFromPEMs(t *testing.T) {
	_, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("failed to convert public key to PEM: %v", err)
	}

	t.Run("valid PEM", func(t *testing.T) {
		middleware, err := NewJWTMiddlewareFromPEMs(
			map[string]string{"test-kid": publicKeyPEM},
			false,
			[]string{"sub"},
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if middleware == nil {
			t.Fatal("middleware is nil")
		}

		if len(middleware.config.PublicKeys) != 1 {
			t.Errorf("expected 1 public key, got %d", len(middleware.config.PublicKeys))
		}
	})

	t.Run("invalid PEM", func(t *testing.T) {
		_, err := NewJWTMiddlewareFromPEMs(
			map[string]string{"test-kid": "invalid-pem"},
			false,
			nil,
		)

		if err == nil {
			t.Error("expected error for invalid PEM, got nil")
		}
	})

	t.Run("skip validation", func(t *testing.T) {
		middleware, err := NewJWTMiddlewareFromPEMs(
			nil,
			true,
			[]string{"sub"},
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !middleware.config.SkipValidation {
			t.Error("SkipValidation should be true")
		}
	})
}

func TestJWTMiddleware_Process_Success(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
		RequiredClaims: []string{"sub", "iss"},
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	tokenString, err := generateTestToken(privateKey, "test-kid", claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	ctx := context.Background()
	resultCtx, err := middleware.Process(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// コンテキストからクレームを取得
	resultClaims, ok := GetClaimsFromContext(resultCtx)
	if !ok {
		t.Fatal("claims not found in context")
	}

	if resultClaims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", resultClaims["sub"])
	}

	if resultClaims["iss"] != "test-issuer" {
		t.Errorf("expected iss=test-issuer, got %v", resultClaims["iss"])
	}
}

func TestJWTMiddleware_Process_MissingAuthorizationHeader(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.Background()

	_, err := middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for missing authorization header")
	}
}

func TestJWTMiddleware_Process_InvalidAuthorizationFormat(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	ctx := context.Background()

	_, err := middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for invalid authorization format")
	}
}

func TestJWTMiddleware_Process_SkipValidation(t *testing.T) {
	middleware := NewJWTMiddleware(JWTConfig{
		SkipValidation: true,
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer fake-token")

	ctx := context.Background()
	resultCtx, err := middleware.Process(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims, ok := GetClaimsFromContext(resultCtx)
	if !ok {
		t.Fatal("claims not found in context")
	}

	if claims["skip_validation"] != true {
		t.Error("skip_validation claim should be true")
	}
}

func TestJWTMiddleware_Process_InvalidToken(t *testing.T) {
	_, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")

	ctx := context.Background()
	_, err = middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestJWTMiddleware_Process_ExpiredToken(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(-time.Hour).Unix(), // 1時間前に期限切れ
	}

	tokenString, err := generateTestToken(privateKey, "test-kid", claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	ctx := context.Background()
	_, err = middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTMiddleware_Process_MissingKid(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// kidヘッダーなしでトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// kidを設定しない
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	ctx := context.Background()
	_, err = middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for missing kid")
	}
}

func TestJWTMiddleware_Process_UnknownKid(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	// 存在しないkidでトークンを生成
	tokenString, err := generateTestToken(privateKey, "unknown-kid", claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	ctx := context.Background()
	_, err = middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for unknown kid")
	}
}

func TestJWTMiddleware_Process_MissingRequiredClaims(t *testing.T) {
	privateKey, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"test-kid": publicKey,
		},
		RequiredClaims: []string{"sub", "role"},
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"exp": time.Now().Add(time.Hour).Unix(),
		// "role" が欠落
	}

	tokenString, err := generateTestToken(privateKey, "test-kid", claims)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	ctx := context.Background()
	_, err = middleware.Process(ctx, req)

	if err == nil {
		t.Fatal("expected error for missing required claim")
	}
}

func TestJWTMiddleware_Process_MultiplePublicKeys(t *testing.T) {
	privateKey1, publicKey1, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair 1: %v", err)
	}

	privateKey2, publicKey2, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair 2: %v", err)
	}

	middleware := NewJWTMiddleware(JWTConfig{
		PublicKeys: map[string]*rsa.PublicKey{
			"kid-1": publicKey1,
			"kid-2": publicKey2,
		},
	})

	// kid-1でトークンを生成
	claims1 := jwt.MapClaims{
		"sub": "user1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString1, err := generateTestToken(privateKey1, "kid-1", claims1)
	if err != nil {
		t.Fatalf("failed to generate token 1: %v", err)
	}

	// kid-2でトークンを生成
	claims2 := jwt.MapClaims{
		"sub": "user2",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenString2, err := generateTestToken(privateKey2, "kid-2", claims2)
	if err != nil {
		t.Fatalf("failed to generate token 2: %v", err)
	}

	ctx := context.Background()

	// kid-1のトークンを検証
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.Header.Set("Authorization", "Bearer "+tokenString1)
	resultCtx1, err := middleware.Process(ctx, req1)
	if err != nil {
		t.Fatalf("unexpected error for token 1: %v", err)
	}
	claims1Result, ok := GetClaimsFromContext(resultCtx1)
	if !ok || claims1Result["sub"] != "user1" {
		t.Error("claims1 not found or invalid")
	}

	// kid-2のトークンを検証
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Authorization", "Bearer "+tokenString2)
	resultCtx2, err := middleware.Process(ctx, req2)
	if err != nil {
		t.Fatalf("unexpected error for token 2: %v", err)
	}
	claims2Result, ok := GetClaimsFromContext(resultCtx2)
	if !ok || claims2Result["sub"] != "user2" {
		t.Error("claims2 not found or invalid")
	}
}

func TestGetClaimsFromContext(t *testing.T) {
	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "test-issuer",
	}

	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

	resultClaims, ok := GetClaimsFromContext(ctx)

	if !ok {
		t.Fatal("claims not found in context")
	}

	if resultClaims["sub"] != "user123" {
		t.Errorf("expected sub=user123, got %v", resultClaims["sub"])
	}

	if resultClaims["iss"] != "test-issuer" {
		t.Errorf("expected iss=test-issuer, got %v", resultClaims["iss"])
	}
}

func TestGetClaimsFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	_, ok := GetClaimsFromContext(ctx)

	if ok {
		t.Error("expected claims not found, but got ok=true")
	}
}

func TestParsePublicKeyFromPEM(t *testing.T) {
	_, publicKey, err := generateTestKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	publicKeyPEM, err := publicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("failed to convert public key to PEM: %v", err)
	}

	t.Run("valid PEM", func(t *testing.T) {
		parsedKey, err := parsePublicKeyFromPEM(publicKeyPEM)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if parsedKey == nil {
			t.Fatal("parsed key is nil")
		}

		if parsedKey.N.Cmp(publicKey.N) != 0 {
			t.Error("parsed key does not match original key")
		}
	})

	t.Run("invalid PEM", func(t *testing.T) {
		_, err := parsePublicKeyFromPEM("invalid-pem-data")
		if err == nil {
			t.Error("expected error for invalid PEM")
		}
	})

	t.Run("empty PEM", func(t *testing.T) {
		_, err := parsePublicKeyFromPEM("")
		if err == nil {
			t.Error("expected error for empty PEM")
		}
	})
}
