package handler_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-gateway/internal/handler"

	"github.com/golang-jwt/jwt/v5"
)

// mockSessionRepository はSessionRepositoryのモック実装
type mockSessionRepository struct {
	setRevokedTimeFunc    func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error
	getRevokedTimeFunc    func(ctx context.Context, userID string) (time.Time, error)
	deleteRevokedTimeFunc func(ctx context.Context, userID string) error
}

func (m *mockSessionRepository) SetRevokedTime(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
	if m.setRevokedTimeFunc != nil {
		return m.setRevokedTimeFunc(ctx, userID, revokedTime, expiration)
	}
	return nil
}

func (m *mockSessionRepository) GetRevokedTime(ctx context.Context, userID string) (time.Time, error) {
	if m.getRevokedTimeFunc != nil {
		return m.getRevokedTimeFunc(ctx, userID)
	}
	return time.Time{}, nil
}

func (m *mockSessionRepository) DeleteRevokedTime(ctx context.Context, userID string) error {
	if m.deleteRevokedTimeFunc != nil {
		return m.deleteRevokedTimeFunc(ctx, userID)
	}
	return nil
}

func TestNewLogoutHandler(t *testing.T) {
	repo := &mockSessionRepository{}
	logger := slog.Default()

	tests := []struct {
		name   string
		config handler.LogoutConfig
	}{
		{
			name: "デフォルト設定",
			config: handler.LogoutConfig{
				Repository: repo,
			},
		},
		{
			name: "カスタム設定",
			config: handler.LogoutConfig{
				Repository:    repo,
				UserIDClaim:   "custom_sub",
				JWTExpiration: 1 * time.Hour,
				Logger:        logger,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handler.NewLogoutHandler(tt.config)
			if handler == nil {
				t.Fatal("NewLogoutHandler returned nil")
			}
		})
	}
}

func TestLogoutHandler_ServeHTTP_Success(t *testing.T) {
	repo := &mockSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			if userID != "user123" {
				t.Errorf("SetRevokedTime called with userID = %s, want user123", userID)
			}
			if expiration != 10*time.Hour {
				t.Errorf("SetRevokedTime called with expiration = %v, want 10h", expiration)
			}
			return nil
		},
	}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	// JWTトークンの生成（検証なしでOK）
	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user123",
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestLogoutHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	repo := &mockSessionRepository{}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	tests := []struct {
		name   string
		method string
	}{
		{
			name:   "GETメソッド",
			method: http.MethodGet,
		},
		{
			name:   "POSTメソッド",
			method: http.MethodPost,
		},
		{
			name:   "PUTメソッド",
			method: http.MethodPut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/v1/logout", nil)
			w := httptest.NewRecorder()

			logoutHandler.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestLogoutHandler_ServeHTTP_MissingAuthHeader(t *testing.T) {
	repo := &mockSessionRepository{}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	// Authorizationヘッダーなし
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogoutHandler_ServeHTTP_InvalidAuthHeader(t *testing.T) {
	repo := &mockSessionRepository{}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "Bearerプレフィックスなし",
			header: "token123",
		},
		{
			name:   "空のトークン",
			header: "Bearer ",
		},
		{
			name:   "不正な形式",
			header: "Basic token123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			logoutHandler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestLogoutHandler_ServeHTTP_InvalidToken(t *testing.T) {
	repo := &mockSessionRepository{}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogoutHandler_ServeHTTP_InvalidClaims(t *testing.T) {
	repo := &mockSessionRepository{}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "ユーザーIDなし",
			claims: jwt.MapClaims{
				"iat": time.Now().Unix(),
			},
		},
		{
			name: "ユーザーIDが文字列でない",
			claims: jwt.MapClaims{
				"sub": 12345,
				"iat": time.Now().Unix(),
			},
		},
		{
			name: "ユーザーIDが空文字列",
			claims: jwt.MapClaims{
				"sub": "",
				"iat": time.Now().Unix(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodNone, tt.claims)
			tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

			req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			w := httptest.NewRecorder()

			logoutHandler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestLogoutHandler_ServeHTTP_RedisError(t *testing.T) {
	repo := &mockSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			return fmt.Errorf("redis connection error")
		},
	}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository: repo,
	})

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user123",
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestLogoutHandler_ServeHTTP_CustomClaims(t *testing.T) {
	repo := &mockSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			if userID != "custom-user-456" {
				t.Errorf("SetRevokedTime called with userID = %s, want custom-user-456", userID)
			}
			return nil
		},
	}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository:  repo,
		UserIDClaim: "custom_user_id",
	})

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"custom_user_id": "custom-user-456",
		"iat":            time.Now().Unix(),
	})
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestLogoutHandler_ServeHTTP_CustomExpiration(t *testing.T) {
	customExpiration := 2 * time.Hour
	repo := &mockSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			if expiration != customExpiration {
				t.Errorf("SetRevokedTime called with expiration = %v, want %v", expiration, customExpiration)
			}
			return nil
		},
	}

	logoutHandler := handler.NewLogoutHandler(handler.LogoutConfig{
		Repository:    repo,
		JWTExpiration: customExpiration,
	})

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "user123",
		"iat": time.Now().Unix(),
	})
	tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodDelete, "/v1/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	logoutHandler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("ServeHTTP() status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
