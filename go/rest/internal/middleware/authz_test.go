package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kaitoimai/go-sample/rest/internal/auth"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
	"github.com/kaitoimai/go-sample/rest/internal/testutil"
	"github.com/ogen-go/ogen/middleware"
)

// TestAuthzMiddleware_Handle_Success tests successful authorization with valid roles
func TestAuthzMiddleware_Handle_Success(t *testing.T) {
	tests := []struct {
		name        string
		operationID string
		role        string
	}{
		{
			name:        "v1GetHello with user role",
			operationID: "v1GetHello",
			role:        auth.RoleUser,
		},
		{
			name:        "v1GetHello with admin role",
			operationID: "v1GetHello",
			role:        auth.RoleAdmin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Claimsを作成してContextに保存
			claims := &auth.Claims{
				UserID: "user123",
				Role:   tt.role,
			}
			ctx := auth.NewContext(context.Background(), claims)

			// リクエスト作成
			rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			req := middleware.Request{
				Context:     ctx,
				Raw:         rawReq,
				OperationID: tt.operationID,
			}

			// 次のミドルウェアをモック
			nextCalled := false
			next := func(req middleware.Request) (middleware.Response, error) {
				nextCalled = true
				return middleware.Response{}, nil
			}

			// ミドルウェア実行
			m := NewAuthzMiddleware()
			_, err = m.Handle(req, next)

			// 検証
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if !nextCalled {
				t.Fatal("next middleware was not called")
			}
		})
	}
}

// TestAuthzMiddleware_Handle_Forbidden_InsufficientRole tests authorization failure due to insufficient role
func TestAuthzMiddleware_Handle_Forbidden_InsufficientRole(t *testing.T) {
	// adminのみアクセス可能なエンドポイントを追加（テスト用）
	// 注: 実際のauthorizeRoleMapには存在しないが、概念的なテスト
	// 実際には v1GetHello は user/admin 両方許可されているため、
	// 存在しないロールでテストする

	tests := []struct {
		name        string
		operationID string
		role        string
	}{
		{
			name:        "v1GetHello with invalid role",
			operationID: "v1GetHello",
			role:        "invalid_role", // user/admin以外のロール
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Claimsを作成してContextに保存
			claims := &auth.Claims{
				UserID: "user123",
				Role:   tt.role,
			}
			ctx := auth.NewContext(context.Background(), claims)

			// リクエスト作成
			rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			req := middleware.Request{
				Context:     ctx,
				Raw:         rawReq,
				OperationID: tt.operationID,
			}

			next := func(req middleware.Request) (middleware.Response, error) {
				t.Fatal("next should not be called")
				return middleware.Response{}, nil
			}

			// ミドルウェア実行
			m := NewAuthzMiddleware()
			_, err = m.Handle(req, next)

			// 403エラーの検証
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var forbidden *myerrors.ForbiddenError
			if !errors.As(err, &forbidden) {
				t.Fatalf("expected ForbiddenError, got %T", err)
			}

			if forbidden.Error() != "この操作を実行する権限がありません" {
				t.Errorf("expected error message 'この操作を実行する権限がありません', got %q", forbidden.Error())
			}
		})
	}
}

// TestAuthzMiddleware_Handle_Forbidden_UndefinedOperationID tests authorization failure due to undefined operationID
func TestAuthzMiddleware_Handle_Forbidden_UndefinedOperationID(t *testing.T) {
	// マッピング未定義のoperationID
	operationID := "v1UndefinedOperation"

	// Claimsを作成してContextに保存
	claims := &auth.Claims{
		UserID: "user123",
		Role:   auth.RoleAdmin, // adminロールでもマッピング未定義なら拒否
	}
	ctx := auth.NewContext(context.Background(), claims)

	// リクエスト作成
	rawReq, err := http.NewRequest(http.MethodGet, "/v1/undefined", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req := middleware.Request{
		Context:     ctx,
		Raw:         rawReq,
		OperationID: operationID,
	}

	next := func(req middleware.Request) (middleware.Response, error) {
		t.Fatal("next should not be called")
		return middleware.Response{}, nil
	}

	// ミドルウェア実行
	m := NewAuthzMiddleware()
	_, err = m.Handle(req, next)

	// 403エラーの検証
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var forbidden *myerrors.ForbiddenError
	if !errors.As(err, &forbidden) {
		t.Fatalf("expected ForbiddenError, got %T", err)
	}

	expectedError := "この操作を実行する権限がありません（ロールマッピング未定義）"
	if forbidden.Error() != expectedError {
		t.Errorf("expected error message %q, got %q", expectedError, forbidden.Error())
	}
}

// TestAuthzMiddleware_Handle_Unauthorized_MissingClaims tests authorization failure due to missing claims
func TestAuthzMiddleware_Handle_Unauthorized_MissingClaims(t *testing.T) {
	// Claimsなしのリクエスト
	rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	req := middleware.Request{
		Context:     context.Background(), // Claimsを保存していない
		Raw:         rawReq,
		OperationID: "v1GetHello",
	}

	next := func(req middleware.Request) (middleware.Response, error) {
		t.Fatal("next should not be called")
		return middleware.Response{}, nil
	}

	// ミドルウェア実行
	m := NewAuthzMiddleware()
	_, err = m.Handle(req, next)

	// 401エラーの検証
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var unauthorized *myerrors.UnauthorizedError
	if !errors.As(err, &unauthorized) {
		t.Fatalf("expected UnauthorizedError, got %T", err)
	}

	if unauthorized.Error() != "認証情報が見つかりません" {
		t.Errorf("expected error message '認証情報が見つかりません', got %q", unauthorized.Error())
	}
}

// TestAuthzMiddleware_Integration tests the integration of AuthnMiddleware and AuthzMiddleware
func TestAuthzMiddleware_Integration(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		role          string
		operationID   string
		expectSuccess bool
	}{
		{
			name:          "admin can access v1GetHello",
			userID:        "admin123",
			role:          auth.RoleAdmin,
			operationID:   "v1GetHello",
			expectSuccess: true,
		},
		{
			name:          "user can access v1GetHello",
			userID:        "user456",
			role:          auth.RoleUser,
			operationID:   "v1GetHello",
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// JWT生成
			token := generateTestJWTForAuthz(t, tt.userID, tt.role)

			// リクエスト作成
			rawReq, err := http.NewRequest(http.MethodGet, "/v1/hello", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			rawReq.Header.Set("Authorization", "Bearer "+token)

			req := middleware.Request{
				Context:     context.Background(),
				Raw:         rawReq,
				OperationID: tt.operationID,
			}

			// 次のミドルウェアをモック
			nextCalled := false
			next := func(req middleware.Request) (middleware.Response, error) {
				nextCalled = true
				return middleware.Response{}, nil
			}

			// AuthnMiddleware実行
			authnMiddleware := NewAuthnMiddleware()
			_, err = authnMiddleware.Handle(req, func(req middleware.Request) (middleware.Response, error) {
				// AuthzMiddleware実行
				authzMiddleware := NewAuthzMiddleware()
				return authzMiddleware.Handle(req, next)
			})

			// 検証
			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !nextCalled {
					t.Fatal("next middleware was not called")
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			}
		})
	}
}

// --- Helper functions ---

// generateTestJWTForAuthz creates a test JWT token for authz tests
func generateTestJWTForAuthz(t *testing.T, userID, role string) string {
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
