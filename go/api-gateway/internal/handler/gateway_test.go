package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"api-gateway/internal/config"
	"api-gateway/internal/routing"
	"api-gateway/internal/transport"
)

// mockTransporter はテスト用のTransporter実装
type mockTransporter struct {
	transportFunc func(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *transport.Backend) error
}

func (m *mockTransporter) Transport(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *transport.Backend) error {
	if m.transportFunc != nil {
		return m.transportFunc(ctx, w, req, backend)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"success"}`))
	return nil
}

func TestNewGateway(t *testing.T) {
	router := routing.NewRouter()
	transporter := &mockTransporter{}
	logger := slog.Default()

	gateway := NewGateway(router, transporter, nil, logger)

	if gateway == nil {
		t.Fatal("NewGateway returned nil")
	}

	if gateway.router == nil {
		t.Error("router is nil")
	}

	if gateway.transporter == nil {
		t.Error("transporter is nil")
	}

	if gateway.logger == nil {
		t.Error("logger is nil")
	}
}

func TestNewGatewayWithNilLogger(t *testing.T) {
	router := routing.NewRouter()
	transporter := &mockTransporter{}

	gateway := NewGateway(router, transporter, nil, nil)

	if gateway == nil {
		t.Fatal("NewGateway returned nil")
	}

	if gateway.logger == nil {
		t.Error("logger should be set to default logger")
	}
}

func TestGateway_ServeHTTP_OptionsRequest(t *testing.T) {
	router := routing.NewRouter()
	transporter := &mockTransporter{}
	gateway := NewGateway(router, transporter, nil, slog.Default())

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestGateway_ServeHTTP_RouteNotFound(t *testing.T) {
	router := routing.NewRouter()
	transporter := &mockTransporter{}
	gateway := NewGateway(router, transporter, nil, slog.Default())

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
}

func TestGateway_ServeHTTP_Success(t *testing.T) {
	// ルーターの準備
	router := routing.NewRouter()
	backendURL, _ := url.Parse("http://backend.example.com")
	route := &routing.Route{
		Path:    "/api/v1/users",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	}
	router.AddRoute(route)

	// トランスポーターの準備
	transporter := &mockTransporter{
		transportFunc: func(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *transport.Backend) error {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"users":[]}`))
			return nil
		},
	}

	gateway := NewGateway(router, transporter, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	expected := `{"users":[]}`
	if string(body) != expected {
		t.Errorf("expected body %s, got %s", expected, string(body))
	}
}

func TestGateway_ServeHTTP_MethodNotAllowed(t *testing.T) {
	// ルーターの準備（POSTのみ許可）
	router := routing.NewRouter()
	backendURL, _ := url.Parse("http://backend.example.com")
	route := &routing.Route{
		Path:    "/api/v1/users",
		Methods: []string{http.MethodPost},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	}
	router.AddRoute(route)

	transporter := &mockTransporter{}
	gateway := NewGateway(router, transporter, nil, slog.Default())

	// GETリクエスト（許可されていない）
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestGateway_ServeHTTP_TransportError(t *testing.T) {
	// ルーターの準備
	router := routing.NewRouter()
	backendURL, _ := url.Parse("http://backend.example.com")
	route := &routing.Route{
		Path:    "/api/v1/users",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	}
	router.AddRoute(route)

	// エラーを返すトランスポーター
	transporter := &mockTransporter{
		transportFunc: func(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *transport.Backend) error {
			return http.ErrServerClosed
		},
	}

	gateway := NewGateway(router, transporter, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status %d, got %d", http.StatusBadGateway, w.Code)
	}
}

func TestGateway_ServeHTTP_WithPathParams(t *testing.T) {
	// パスパラメータを含むルート
	router := routing.NewRouter()
	backendURL, _ := url.Parse("http://backend.example.com")
	route := &routing.Route{
		Path:    "/api/v1/users/:id",
		Methods: []string{http.MethodGet},
		Backend: &routing.Backend{
			URL:     backendURL,
			Timeout: 30 * time.Second,
		},
		Middleware: []config.MiddlewareConfig{},
		Priority:   10,
	}
	router.AddRoute(route)

	transporter := &mockTransporter{
		transportFunc: func(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *transport.Backend) error {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"123"}`))
			return nil
		},
	}

	gateway := NewGateway(router, transporter, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	w := httptest.NewRecorder()

	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	expected := `{"id":"123"}`
	if string(body) != expected {
		t.Errorf("expected body %s, got %s", expected, string(body))
	}
}

func TestGateway_convertToTransportBackend(t *testing.T) {
	gateway := NewGateway(routing.NewRouter(), &mockTransporter{}, nil, slog.Default())

	backendURL, _ := url.Parse("http://backend.example.com")
	routingBackend := &routing.Backend{
		URL:     backendURL,
		Timeout: 30 * time.Second,
	}

	transportBackend := gateway.convertToTransportBackend(routingBackend)

	if transportBackend.URL.String() != backendURL.String() {
		t.Errorf("expected URL %s, got %s", backendURL.String(), transportBackend.URL.String())
	}

	if transportBackend.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %s", transportBackend.Timeout)
	}

	if transportBackend.Headers == nil {
		t.Error("Headers should be initialized")
	}
}
