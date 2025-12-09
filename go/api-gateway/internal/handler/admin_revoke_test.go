package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-gateway/pkg/logger"
)

// Mock SessionRepository for AdminRevoke tests
type mockAdminSessionRepository struct {
	setRevokedTimeFunc func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error
}

func (m *mockAdminSessionRepository) SetRevokedTime(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
	if m.setRevokedTimeFunc != nil {
		return m.setRevokedTimeFunc(ctx, userID, revokedTime, expiration)
	}
	return nil
}

func (m *mockAdminSessionRepository) GetRevokedTime(ctx context.Context, userID string) (time.Time, error) {
	return time.Time{}, nil
}

func (m *mockAdminSessionRepository) DeleteRevokedTime(ctx context.Context, userID string) error {
	return nil
}

func TestNewAdminRevokeHandler(t *testing.T) {
	repo := &mockAdminSessionRepository{}

	tests := []struct {
		name   string
		config AdminRevokeConfig
		want   *AdminRevokeHandler
	}{
		{
			name: "デフォルト値が設定される",
			config: AdminRevokeConfig{
				Repository: repo,
				APIKey:     "test-api-key",
			},
			want: &AdminRevokeHandler{
				repository:    repo,
				apiKey:        "test-api-key",
				jwtExpiration: 10 * time.Hour,
			},
		},
		{
			name: "カスタム設定が適用される",
			config: AdminRevokeConfig{
				Repository:    repo,
				APIKey:        "custom-key",
				JWTExpiration: 5 * time.Hour,
			},
			want: &AdminRevokeHandler{
				repository:    repo,
				apiKey:        "custom-key",
				jwtExpiration: 5 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewAdminRevokeHandler(tt.config)
			if got.apiKey != tt.want.apiKey {
				t.Errorf("apiKey = %v, want %v", got.apiKey, tt.want.apiKey)
			}
			if got.jwtExpiration != tt.want.jwtExpiration {
				t.Errorf("jwtExpiration = %v, want %v", got.jwtExpiration, tt.want.jwtExpiration)
			}
		})
	}
}

func TestAdminRevokeHandler_ServeHTTP_Success(t *testing.T) {
	var capturedUserID string
	var capturedExpiration time.Duration

	repo := &mockAdminSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			capturedUserID = userID
			capturedExpiration = expiration
			return nil
		},
	}

	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository:    repo,
		APIKey:        "test-api-key",
		JWTExpiration: 10 * time.Hour,
		Logger:        logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	body := RevokeRequest{UserID: "user_123"}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/revoke", bytes.NewReader(bodyJSON))
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusOK)
	}

	if capturedUserID != "user_123" {
		t.Errorf("captured user_id = %v, want %v", capturedUserID, "user_123")
	}

	if capturedExpiration != 10*time.Hour {
		t.Errorf("captured expiration = %v, want %v", capturedExpiration, 10*time.Hour)
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["success"] != true {
		t.Errorf("response success = %v, want true", response["success"])
	}

	if response["user_id"] != "user_123" {
		t.Errorf("response user_id = %v, want user_123", response["user_id"])
	}
}

func TestAdminRevokeHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	repo := &mockAdminSessionRepository{}
	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository: repo,
		APIKey:     "test-api-key",
		Logger:     logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/revoke", nil)
			req.Header.Set("X-API-Key", "test-api-key")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status code = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}

func TestAdminRevokeHandler_ServeHTTP_Unauthorized(t *testing.T) {
	repo := &mockAdminSessionRepository{}
	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository: repo,
		APIKey:     "correct-api-key",
		Logger:     logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "APIキーが存在しない",
			apiKey: "",
		},
		{
			name:   "APIキーが間違っている",
			apiKey: "wrong-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := RevokeRequest{UserID: "user_123"}
			bodyJSON, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/v1/revoke", bytes.NewReader(bodyJSON))
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status code = %v, want %v", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAdminRevokeHandler_ServeHTTP_BadRequest(t *testing.T) {
	repo := &mockAdminSessionRepository{}
	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository: repo,
		APIKey:     "test-api-key",
		Logger:     logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	tests := []struct {
		name string
		body string
	}{
		{
			name: "不正なJSON",
			body: "{invalid json}",
		},
		{
			name: "user_idが空",
			body: `{"user_id": ""}`,
		},
		{
			name: "user_idが存在しない",
			body: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/revoke", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("X-API-Key", "test-api-key")
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status code = %v, want %v", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestAdminRevokeHandler_ServeHTTP_InternalServerError(t *testing.T) {
	repo := &mockAdminSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			return fmt.Errorf("redis connection error")
		},
	}

	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository: repo,
		APIKey:     "test-api-key",
		Logger:     logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	body := RevokeRequest{UserID: "user_123"}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/v1/revoke", bytes.NewReader(bodyJSON))
	req.Header.Set("X-API-Key", "test-api-key")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %v, want %v", rec.Code, http.StatusInternalServerError)
	}
}

func TestAdminRevokeHandler_authenticate(t *testing.T) {
	handler := &AdminRevokeHandler{
		apiKey: "correct-key",
	}

	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "正しいAPIキー",
			apiKey:  "correct-key",
			wantErr: false,
		},
		{
			name:    "間違ったAPIキー",
			apiKey:  "wrong-key",
			wantErr: true,
		},
		{
			name:    "APIキーが空",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/revoke", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			err := handler.authenticate(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// race検証用のテスト
func TestAdminRevokeHandler_Race(t *testing.T) {
	repo := &mockAdminSessionRepository{
		setRevokedTimeFunc: func(ctx context.Context, userID string, revokedTime time.Time, expiration time.Duration) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	handler := NewAdminRevokeHandler(AdminRevokeConfig{
		Repository: repo,
		APIKey:     "test-api-key",
		Logger:     logger.New(logger.Config{Level: logger.LevelInfo, Format: "json"}),
	})

	// 並行してリクエストを送信
	for i := 0; i < 10; i++ {
		go func(id int) {
			body := RevokeRequest{UserID: fmt.Sprintf("user_%d", id)}
			bodyJSON, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/v1/revoke", bytes.NewReader(bodyJSON))
			req.Header.Set("X-API-Key", "test-api-key")
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
}
