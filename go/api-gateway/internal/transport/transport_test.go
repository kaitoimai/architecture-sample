package transport

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPTransporter(t *testing.T) {
	transporter := NewHTTPTransporter()

	if transporter == nil {
		t.Fatal("NewHTTPTransporter returned nil")
	}

	if transporter.ErrorHandler == nil {
		t.Error("ErrorHandler should not be nil")
	}
}

func TestNewBackend(t *testing.T) {
	t.Run("valid URL", func(t *testing.T) {
		backend, err := NewBackend("https://example.com", 30*time.Second)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if backend == nil {
			t.Fatal("backend is nil")
		}

		if backend.URL.Scheme != "https" {
			t.Errorf("expected scheme https, got %s", backend.URL.Scheme)
		}

		if backend.URL.Host != "example.com" {
			t.Errorf("expected host example.com, got %s", backend.URL.Host)
		}

		if backend.Timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", backend.Timeout)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := NewBackend("://invalid-url", 30*time.Second)

		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})
}

func TestBackend_AddHeader(t *testing.T) {
	backend, err := NewBackend("https://example.com", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backend.AddHeader("X-Custom-Header", "test-value")
	backend.AddHeader("Authorization", "Bearer token")

	if backend.Headers["X-Custom-Header"] != "test-value" {
		t.Errorf("expected X-Custom-Header=test-value, got %s", backend.Headers["X-Custom-Header"])
	}

	if backend.Headers["Authorization"] != "Bearer token" {
		t.Errorf("expected Authorization=Bearer token, got %s", backend.Headers["Authorization"])
	}
}

func TestHTTPTransporter_Transport_Success(t *testing.T) {
	// バックエンドサーバーのモック
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストの検証
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/test" {
			t.Errorf("expected path /api/test, got %s", r.URL.Path)
		}

		if r.URL.Query().Get("param") != "value" {
			t.Errorf("expected query param=value, got %s", r.URL.Query().Get("param"))
		}

		// レスポンスを返す
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer backendServer.Close()

	backend, err := NewBackend(backendServer.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	transporter := NewHTTPTransporter()

	// クライアントリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/api/test?param=value", strings.NewReader("test body"))
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status":"success"}` {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestHTTPTransporter_Transport_WithCustomHeaders(t *testing.T) {
	// バックエンドサーバーのモック
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// カスタムヘッダーの検証
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected X-Custom-Header=custom-value, got %s", r.Header.Get("X-Custom-Header"))
		}

		if r.Header.Get("X-API-Key") != "secret-key" {
			t.Errorf("expected X-API-Key=secret-key, got %s", r.Header.Get("X-API-Key"))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer backendServer.Close()

	backend, err := NewBackend(backendServer.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	backend.AddHeader("X-Custom-Header", "custom-value")
	backend.AddHeader("X-API-Key", "secret-key")

	transporter := NewHTTPTransporter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPTransporter_Transport_WithTimeout(t *testing.T) {
	// 遅いバックエンドサーバー
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer backendServer.Close()

	backend, err := NewBackend(backendServer.URL, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	transporter := NewHTTPTransporter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	// タイムアウトエラーは発生しないが、リバースプロキシがエラーを処理する
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := w.Result()
	defer resp.Body.Close()

	// タイムアウトの場合、502 Bad Gatewayが返される
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", resp.StatusCode)
	}
}

func TestHTTPTransporter_Transport_NilBackend(t *testing.T) {
	transporter := NewHTTPTransporter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err := transporter.Transport(ctx, w, req, nil)

	if err == nil {
		t.Fatal("expected error for nil backend")
	}
}

func TestHTTPTransporter_Transport_NilBackendURL(t *testing.T) {
	transporter := NewHTTPTransporter()

	backend := &Backend{
		URL:     nil,
		Timeout: 30 * time.Second,
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err := transporter.Transport(ctx, w, req, backend)

	if err == nil {
		t.Fatal("expected error for nil backend URL")
	}
}

func TestHTTPTransporter_Transport_PathCombination(t *testing.T) {
	// バックエンドサーバーのモック
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// パスの結合を検証
		if r.URL.Path != "/api/v1/users/123" {
			t.Errorf("expected path /api/v1/users/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backendServer.Close()

	// バックエンドURLに /api/v1 が含まれている
	backend, err := NewBackend(backendServer.URL+"/api/v1", 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	transporter := NewHTTPTransporter()

	// クライアントリクエストのパスは /users/123
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPTransporter_Transport_PreservesRequestBody(t *testing.T) {
	expectedBody := `{"name":"test","value":123}`

	// バックエンドサーバーのモック
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != expectedBody {
			t.Errorf("expected body %s, got %s", expectedBody, string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backendServer.Close()

	backend, err := NewBackend(backendServer.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	transporter := NewHTTPTransporter()

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(expectedBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHTTPTransporter_Transport_PreservesResponseHeaders(t *testing.T) {
	// バックエンドサーバーのモック
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Response", "custom-value")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backendServer.Close()

	backend, err := NewBackend(backendServer.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	transporter := NewHTTPTransporter()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("X-Custom-Response") != "custom-value" {
		t.Errorf("expected X-Custom-Response=custom-value, got %s", resp.Header.Get("X-Custom-Response"))
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", resp.Header.Get("Content-Type"))
	}
}

func TestHTTPTransporter_CustomErrorHandler(t *testing.T) {
	// エラーを返すバックエンドサーバー
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// サーバーをすぐに閉じてエラーを発生させる
	}))
	backendServer.Close() // すぐに閉じる

	backend, err := NewBackend(backendServer.URL, 30*time.Second)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	customErrorHandlerCalled := false
	transporter := &HTTPTransporter{
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			customErrorHandlerCalled = true
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("custom error"))
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	ctx := context.Background()
	err = transporter.Transport(ctx, w, req, backend)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !customErrorHandlerCalled {
		t.Error("custom error handler was not called")
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "custom error" {
		t.Errorf("unexpected body: %s", string(body))
	}
}
