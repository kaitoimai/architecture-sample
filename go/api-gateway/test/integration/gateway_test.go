package integration

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/handler"
	"api-gateway/internal/routing"
	"api-gateway/internal/transport"
)

// TestGateway_EndToEnd はエンドツーエンドの統合テスト
func TestGateway_EndToEnd(t *testing.T) {
	// モックバックエンドサーバの起動
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// リクエストパスを確認
		if r.URL.Path == "/api/v1/users" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`))
			return
		}

		if r.URL.Path == "/api/v1/users/123" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":123,"name":"Alice"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer backendServer.Close()

	// ルーターの設定
	router := routing.NewRouter()
	backendURL, _ := url.Parse(backendServer.URL)

	// ルート1: /api/v1/users (GET, POST)
	route1 := &routing.Route{
		Path:    "/api/v1/users",
		Methods: []string{http.MethodGet, http.MethodPost},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	}
	router.AddRoute(route1)

	// ルート2: /api/v1/users/:id (GET)
	route2 := &routing.Route{
		Path:    "/api/v1/users/:id",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   20,
	}
	router.AddRoute(route2)

	// Gatewayの初期化
	transporter := transport.NewHTTPTransporter()
	gateway := handler.NewGateway(router, transporter, nil, slog.Default())

	// テストケース1: GETリクエスト - ユーザー一覧
	t.Run("GET /api/v1/users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		expected := `{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`
		if string(body) != expected {
			t.Errorf("expected body %s, got %s", expected, string(body))
		}
	})

	// テストケース2: GETリクエスト - パスパラメータ
	t.Run("GET /api/v1/users/:id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		expected := `{"id":123,"name":"Alice"}`
		if string(body) != expected {
			t.Errorf("expected body %s, got %s", expected, string(body))
		}
	})

	// テストケース3: 404エラー
	t.Run("GET /not-found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	// テストケース4: メソッド不許可
	t.Run("DELETE /api/v1/users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	// テストケース5: OPTIONSリクエスト
	t.Run("OPTIONS /api/v1/users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}

// TestGateway_MultipleBackends は複数バックエンドの統合テスト
func TestGateway_MultipleBackends(t *testing.T) {
	// モックバックエンドサーバ1: ユーザーサービス
	userService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"user"}`))
	}))
	defer userService.Close()

	// モックバックエンドサーバ2: 商品サービス
	productService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"product"}`))
	}))
	defer productService.Close()

	// ルーターの設定
	router := routing.NewRouter()
	userURL, _ := url.Parse(userService.URL)
	productURL, _ := url.Parse(productService.URL)

	// ユーザーサービスのルート
	router.AddRoute(&routing.Route{
		Path:    "/api/v1/users",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     userURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	})

	// 商品サービスのルート
	router.AddRoute(&routing.Route{
		Path:    "/api/v1/products",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     productURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	})

	// Gatewayの初期化
	transporter := transport.NewHTTPTransporter()
	gateway := handler.NewGateway(router, transporter, nil, slog.Default())

	// ユーザーサービスへのリクエスト
	t.Run("User Service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		expected := `{"service":"user"}`
		if string(body) != expected {
			t.Errorf("expected body %s, got %s", expected, string(body))
		}
	})

	// 商品サービスへのリクエスト
	t.Run("Product Service", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
		w := httptest.NewRecorder()

		gateway.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body, _ := io.ReadAll(w.Body)
		expected := `{"service":"product"}`
		if string(body) != expected {
			t.Errorf("expected body %s, got %s", expected, string(body))
		}
	})
}

// TestGateway_WithCustomHeaders はカスタムヘッダーの統合テスト
func TestGateway_WithCustomHeaders(t *testing.T) {
	// モックバックエンドサーバ
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ヘッダーの確認
		customHeader := r.Header.Get("X-Custom-Header")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"custom_header":"` + customHeader + `"}`))
	}))
	defer backendServer.Close()

	// ルーターの設定
	router := routing.NewRouter()
	backendURL, _ := url.Parse(backendServer.URL)

	router.AddRoute(&routing.Route{
		Path:    "/api/v1/test",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	})

	// Gatewayの初期化
	transporter := transport.NewHTTPTransporter()
	gateway := handler.NewGateway(router, transporter, nil, slog.Default())

	// カスタムヘッダー付きリクエスト
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	expected := `{"custom_header":"test-value"}`
	if string(body) != expected {
		t.Errorf("expected body %s, got %s", expected, string(body))
	}
}
