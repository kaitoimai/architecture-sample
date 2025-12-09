package auth_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-gateway/internal/middleware/auth"

	"github.com/golang-jwt/jwt/v5"
)

// mockSessionRepository はSessionRepositoryのモック実装
type mockSessionRepository struct {
	getRevokedTimeFunc    func(ctx context.Context, userID string) (time.Time, error)
	setRevokedTimeFunc    func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error
	deleteRevokedTimeFunc func(ctx context.Context, userID string) error
}

func (m *mockSessionRepository) GetRevokedTime(ctx context.Context, userID string) (time.Time, error) {
	if m.getRevokedTimeFunc != nil {
		return m.getRevokedTimeFunc(ctx, userID)
	}
	return time.Time{}, nil
}

func (m *mockSessionRepository) SetRevokedTime(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
	if m.setRevokedTimeFunc != nil {
		return m.setRevokedTimeFunc(ctx, userID, revokedTime, expiration)
	}
	return nil
}

func (m *mockSessionRepository) DeleteRevokedTime(ctx context.Context, userID string) error {
	if m.deleteRevokedTimeFunc != nil {
		return m.deleteRevokedTimeFunc(ctx, userID)
	}
	return nil
}

func TestNewRevokeMiddleware(t *testing.T) {
	repo := &mockSessionRepository{}
	logger := slog.Default()

	tests := []struct {
		name   string
		config auth.RevokeConfig
	}{
		{
			name: "デフォルト設定",
			config: auth.RevokeConfig{
				Repository: repo,
			},
		},
		{
			name: "カスタム設定",
			config: auth.RevokeConfig{
				Repository:     repo,
				UserIDClaim:    "custom_sub",
				IssuedAtClaim:  "custom_iat",
				FailOpen:       true,
				Logger:         logger,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := auth.NewRevokeMiddleware(tt.config)
			if middleware == nil {
				t.Fatal("NewRevokeMiddleware returned nil")
			}
		})
	}
}

func TestRevokeMiddleware_Process_Success(t *testing.T) {
	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			// 失効時刻が設定されていない
			return time.Time{}, nil
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	// コンテキストにClaimsを設定
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user123",
		"iat": float64(now.Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	newCtx, err := middleware.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v, want nil", err)
	}
	if newCtx == nil {
		t.Error("Process() returned nil context")
	}
}

func TestRevokeMiddleware_Process_TokenRevoked(t *testing.T) {
	now := time.Now()
	revokedTime := now.Add(1 * time.Hour) // トークン発行後に失効

	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			return revokedTime, nil
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	// コンテキストにClaimsを設定（発行時刻が失効時刻より前）
	claims := jwt.MapClaims{
		"sub": "user123",
		"iat": float64(now.Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.Process(ctx, req)
	if err == nil {
		t.Error("Process() error = nil, want error (token revoked)")
	}
}

func TestRevokeMiddleware_Process_TokenNotRevoked(t *testing.T) {
	now := time.Now()
	revokedTime := now.Add(-1 * time.Hour) // トークン発行前に失効

	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			return revokedTime, nil
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	// コンテキストにClaimsを設定（発行時刻が失効時刻より後）
	claims := jwt.MapClaims{
		"sub": "user123",
		"iat": float64(now.Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v, want nil", err)
	}
}

func TestRevokeMiddleware_Process_NoClaims(t *testing.T) {
	repo := &mockSessionRepository{}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	// Claimsが存在しないコンテキスト
	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	newCtx, err := middleware.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v, want nil (should skip)", err)
	}
	if newCtx == nil {
		t.Error("Process() returned nil context")
	}
}

func TestRevokeMiddleware_Process_FailOpen(t *testing.T) {
	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			// Redis接続エラーをシミュレート
			return time.Time{}, fmt.Errorf("redis connection error")
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
		FailOpen:   true, // エラー時は通過
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"iat": float64(time.Now().Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v, want nil (fail-open mode)", err)
	}
}

func TestRevokeMiddleware_Process_FailClose(t *testing.T) {
	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			// Redis接続エラーをシミュレート
			return time.Time{}, fmt.Errorf("redis connection error")
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
		FailOpen:   false, // エラー時は拒否
	})

	claims := jwt.MapClaims{
		"sub": "user123",
		"iat": float64(time.Now().Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.Process(ctx, req)
	if err == nil {
		t.Error("Process() error = nil, want error (fail-close mode)")
	}
}

func TestRevokeMiddleware_Process_InvalidUserID(t *testing.T) {
	repo := &mockSessionRepository{}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "ユーザーIDが存在しない",
			claims: jwt.MapClaims{
				"iat": float64(time.Now().Unix()),
			},
		},
		{
			name: "ユーザーIDが文字列でない",
			claims: jwt.MapClaims{
				"sub": 12345,
				"iat": float64(time.Now().Unix()),
			},
		},
		{
			name: "ユーザーIDが空文字列",
			claims: jwt.MapClaims{
				"sub": "",
				"iat": float64(time.Now().Unix()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			_, err := middleware.Process(ctx, req)
			if err == nil {
				t.Error("Process() error = nil, want error (invalid user id)")
			}
		})
	}
}

func TestRevokeMiddleware_Process_InvalidIssuedAt(t *testing.T) {
	repo := &mockSessionRepository{}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "発行時刻が存在しない",
			claims: jwt.MapClaims{
				"sub": "user123",
			},
		},
		{
			name: "発行時刻が無効な型",
			claims: jwt.MapClaims{
				"sub": "user123",
				"iat": "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, tt.claims)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			_, err := middleware.Process(ctx, req)
			if err == nil {
				t.Error("Process() error = nil, want error (invalid issued at)")
			}
		})
	}
}

func TestRevokeMiddleware_Process_CustomClaims(t *testing.T) {
	now := time.Now()
	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			return time.Time{}, nil
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository:    repo,
		UserIDClaim:   "custom_user_id",
		IssuedAtClaim: "custom_issued_at",
	})

	claims := jwt.MapClaims{
		"custom_user_id":   "user123",
		"custom_issued_at": float64(now.Unix()),
	}
	ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	_, err := middleware.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v, want nil", err)
	}
}

func TestRevokeMiddleware_Process_IssuedAtTypes(t *testing.T) {
	repo := &mockSessionRepository{
		getRevokedTimeFunc: func(ctx context.Context, userID string) (time.Time, error) {
			return time.Time{}, nil
		},
	}

	middleware := auth.NewRevokeMiddleware(auth.RevokeConfig{
		Repository: repo,
	})

	now := time.Now()

	tests := []struct {
		name    string
		iatType any
		wantErr bool
	}{
		{
			name:    "float64型",
			iatType: float64(now.Unix()),
			wantErr: false,
		},
		{
			name:    "int64型",
			iatType: now.Unix(),
			wantErr: false,
		},
		{
			name:    "int型",
			iatType: int(now.Unix()),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := jwt.MapClaims{
				"sub": "user123",
				"iat": tt.iatType,
			}
			ctx := context.WithValue(context.Background(), auth.ClaimsContextKey, claims)
			req := httptest.NewRequest(http.MethodGet, "/test", nil)

			_, err := middleware.Process(ctx, req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
