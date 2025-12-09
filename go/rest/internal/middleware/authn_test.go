package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kaitoimai/go-sample/rest/internal/auth"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
	"github.com/kaitoimai/go-sample/rest/internal/testutil"
	"github.com/ogen-go/ogen/middleware"
)

// TestAuthnMiddleware_Handle_Success tests successful authentication with valid JWT
func TestAuthnMiddleware_Handle_Success(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		role     string
		wantRole string
	}{
		{
			name:     "valid JWT with admin role",
			userID:   "user123",
			role:     auth.RoleAdmin,
			wantRole: auth.RoleAdmin,
		},
		{
			name:     "valid JWT with user role",
			userID:   "user456",
			role:     auth.RoleUser,
			wantRole: auth.RoleUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// JWT生成
			token := generateTestJWT(t, tt.userID, tt.role)

			// リクエスト作成
			req := createTestRequest(t, token)

			// 次のミドルウェアをモック
			nextCalled := false
			var capturedContext context.Context
			next := func(req middleware.Request) (middleware.Response, error) {
				nextCalled = true
				capturedContext = req.Context
				return middleware.Response{}, nil
			}

			// ミドルウェア実行
			m := NewAuthnMiddleware()
			_, err := m.Handle(req, next)

			// 検証
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if !nextCalled {
				t.Fatal("next middleware was not called")
			}

			// Contextからclaimsを取得して検証
			claims := auth.FromContext(capturedContext)
			if claims == nil {
				t.Fatal("claims not found in context")
			}

			if claims.UserID != tt.userID {
				t.Errorf("expected UserID %q, got %q", tt.userID, claims.UserID)
			}

			if claims.Role != tt.wantRole {
				t.Errorf("expected Role %q, got %q", tt.wantRole, claims.Role)
			}
		})
	}
}

// TestAuthnMiddleware_Handle_MissingAuthHeader tests missing Authorization header
func TestAuthnMiddleware_Handle_MissingAuthHeader(t *testing.T) {
	// Authorizationヘッダーなしのリクエスト
	rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req := middleware.Request{
		Context:     context.Background(),
		Raw:         rawReq,
		OperationID: "v1GetHello",
	}

	next := func(req middleware.Request) (middleware.Response, error) {
		t.Fatal("next should not be called")
		return middleware.Response{}, nil
	}

	m := NewAuthnMiddleware()
	_, err = m.Handle(req, next)

	// 401エラーの検証
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var unauthorized *myerrors.UnauthorizedError
	if !errors.As(err, &unauthorized) {
		t.Fatalf("expected UnauthorizedError, got %T", err)
	}

	if unauthorized.Error() != "認証トークンが必要です" {
		t.Errorf("expected error message '認証トークンが必要です', got %q", unauthorized.Error())
	}
}

// TestAuthnMiddleware_Handle_InvalidBearerFormat tests invalid Bearer format
func TestAuthnMiddleware_Handle_InvalidBearerFormat(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedError string
	}{
		{
			name:          "missing Bearer prefix",
			authHeader:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
			expectedError: "認証形式が不正です",
		},
		{
			name:          "wrong prefix",
			authHeader:    "Basic dXNlcjpwYXNz",
			expectedError: "認証形式が不正です",
		},
		{
			name:          "Bearer with empty token",
			authHeader:    "Bearer ",
			expectedError: "認証トークンが空です",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			rawReq.Header.Set("Authorization", tt.authHeader)

			req := middleware.Request{
				Context:     context.Background(),
				Raw:         rawReq,
				OperationID: "v1GetHello",
			}

			next := func(req middleware.Request) (middleware.Response, error) {
				t.Fatal("next should not be called")
				return middleware.Response{}, nil
			}

			m := NewAuthnMiddleware()
			_, err = m.Handle(req, next)

			// 401エラーの検証
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unauthorized *myerrors.UnauthorizedError
			if !errors.As(err, &unauthorized) {
				t.Fatalf("expected UnauthorizedError, got %T", err)
			}

			if unauthorized.Error() != tt.expectedError {
				t.Errorf("expected error message %q, got %q", tt.expectedError, unauthorized.Error())
			}
		})
	}
}

// TestAuthnMiddleware_Handle_InvalidJWTFormat tests invalid JWT format
func TestAuthnMiddleware_Handle_InvalidJWTFormat(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectedError string
	}{
		{
			name:          "JWT with only 2 segments",
			token:         "header.payload",
			expectedError: "トークンの解析に失敗しました",
		},
		{
			name:          "JWT with 4 segments",
			token:         "header.payload.signature.extra",
			expectedError: "トークンの解析に失敗しました",
		},
		{
			name:          "JWT with invalid base64 payload",
			token:         "header.!!!invalid!!!.signature",
			expectedError: "トークンの解析に失敗しました",
		},
		{
			name:          "JWT with invalid JSON payload",
			token:         createInvalidJSONJWT(t),
			expectedError: "トークンの解析に失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(t, tt.token)

			next := func(req middleware.Request) (middleware.Response, error) {
				t.Fatal("next should not be called")
				return middleware.Response{}, nil
			}

			m := NewAuthnMiddleware()
			_, err := m.Handle(req, next)

			// 401エラーの検証
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unauthorized *myerrors.UnauthorizedError
			if !errors.As(err, &unauthorized) {
				t.Fatalf("expected UnauthorizedError, got %T", err)
			}

			if unauthorized.Error() != tt.expectedError {
				t.Errorf("expected error message %q, got %q", tt.expectedError, unauthorized.Error())
			}
		})
	}
}

// TestAuthnMiddleware_Handle_InvalidRole tests invalid role in JWT
func TestAuthnMiddleware_Handle_InvalidRole(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{
			name: "empty role",
			role: "",
		},
		{
			name: "unknown role",
			role: "unknown",
		},
		{
			name: "uppercase Admin",
			role: "Admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := generateTestJWT(t, "user123", tt.role)
			req := createTestRequest(t, token)

			next := func(req middleware.Request) (middleware.Response, error) {
				t.Fatal("next should not be called")
				return middleware.Response{}, nil
			}

			m := NewAuthnMiddleware()
			_, err := m.Handle(req, next)

			// 401エラーの検証
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var unauthorized *myerrors.UnauthorizedError
			if !errors.As(err, &unauthorized) {
				t.Fatalf("expected UnauthorizedError, got %T", err)
			}

			if unauthorized.Error() != "無効なロールです" {
				t.Errorf("expected error message '無効なロールです', got %q", unauthorized.Error())
			}
		})
	}
}

// TestExtractClaims tests the extractClaims function directly
func TestExtractClaims(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantError bool
		wantRole  string
	}{
		{
			name:      "valid JWT",
			token:     generateTestJWT(t, "user123", auth.RoleAdmin),
			wantError: false,
			wantRole:  auth.RoleAdmin,
		},
		{
			name:      "JWT with less than 3 segments",
			token:     "header.payload",
			wantError: true,
		},
		{
			name:      "JWT with more than 3 segments",
			token:     "header.payload.signature.extra",
			wantError: true,
		},
		{
			name:      "JWT with invalid base64",
			token:     "header.!!!.signature",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := extractClaims(tt.token)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if claims == nil {
				t.Fatal("expected claims, got nil")
			}

			if claims.Role != tt.wantRole {
				t.Errorf("expected role %q, got %q", tt.wantRole, claims.Role)
			}
		})
	}
}

// --- Helper functions ---

// generateTestJWT creates a test JWT token using testutil
func generateTestJWT(t *testing.T, userID, role string) string {
	t.Helper()

	// テスト用のRSA秘密鍵を生成
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// testutilのConfigを作成
	cfg := testutil.JWTConfig{
		UserID:   userID,
		Role:     role,
		KID:      "test-key-id",
		Duration: 15 * time.Minute,
		Issuer:   "test-issuer",
		Audience: "test-audience",
	}

	// JWT生成
	tokenString, _, err := testutil.GenerateJWT(cfg, privateKey, time.Now())
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	return tokenString
}

// createInvalidJSONJWT creates a JWT with invalid JSON in the payload
func createInvalidJSONJWT(t *testing.T) string {
	t.Helper()

	header := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9" // dummy header
	invalidPayload := base64.RawURLEncoding.EncodeToString([]byte("{invalid json"))
	signature := "dummy-signature"

	return header + "." + invalidPayload + "." + signature
}

// createTestRequest creates a test middleware.Request with the given JWT token
func createTestRequest(t *testing.T, token string) middleware.Request {
	t.Helper()

	rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	rawReq.Header.Set("Authorization", "Bearer "+token)

	return middleware.Request{
		Context:     context.Background(),
		Raw:         rawReq,
		OperationID: "v1GetHello",
	}
}
