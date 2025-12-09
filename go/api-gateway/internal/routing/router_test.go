package routing

import (
	"net/url"
	"testing"
	"time"

	"api-gateway/internal/config"
)

func TestNewRouter(t *testing.T) {
	router := NewRouter()
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}

	if router.root == nil {
		t.Error("root node is nil")
	}
}

func TestAddRoute(t *testing.T) {
	tests := []struct {
		name    string
		route   *Route
		wantErr bool
	}{
		{
			name: "valid route",
			route: &Route{
				Path:    "/api/v1/users",
				Methods: []string{"GET", "POST"},
				Backend: &Backend{
					URL:     mustParseURL("https://example.com"),
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "route with parameter",
			route: &Route{
				Path:    "/api/v1/users/:id",
				Methods: []string{"GET"},
				Backend: &Backend{
					URL:     mustParseURL("https://example.com"),
					Timeout: 30 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil route",
			route:   nil,
			wantErr: true,
		},
		{
			name: "empty path",
			route: &Route{
				Path:    "",
				Methods: []string{"GET"},
				Backend: &Backend{
					URL:     mustParseURL("https://example.com"),
					Timeout: 30 * time.Second,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			err := router.AddRoute(tt.route)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddRoute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.route != nil {
				// ルートが正しく追加されたか確認
				result, err := router.Match("GET", tt.route.Path)
				if err != nil {
					t.Errorf("Match() after AddRoute() failed: %v", err)
				}
				if result.Route.Path != tt.route.Path {
					t.Errorf("Route path = %s, want %s", result.Route.Path, tt.route.Path)
				}
			}
		})
	}
}

func TestMatch(t *testing.T) {
	router := NewRouter()

	// テスト用のルートを追加
	routes := []*Route{
		{
			Path:    "/api/v1/users",
			Methods: []string{"GET", "POST"},
			Backend: &Backend{URL: mustParseURL("https://user-service.com")},
		},
		{
			Path:    "/api/v1/users/:id",
			Methods: []string{"GET", "PUT", "DELETE"},
			Backend: &Backend{URL: mustParseURL("https://user-service.com")},
		},
		{
			Path:    "/api/v1/orders/:orderId/items/:itemId",
			Methods: []string{"GET"},
			Backend: &Backend{URL: mustParseURL("https://order-service.com")},
		},
		{
			Path:    "/health",
			Methods: []string{"GET"},
			Backend: &Backend{URL: mustParseURL("https://localhost:8080")},
		},
	}

	for _, route := range routes {
		if err := router.AddRoute(route); err != nil {
			t.Fatalf("failed to add route: %v", err)
		}
	}

	tests := []struct {
		name        string
		method      string
		path        string
		wantErr     bool
		wantPath    string
		wantParams  map[string]string
	}{
		{
			name:       "static path match",
			method:     "GET",
			path:       "/api/v1/users",
			wantErr:    false,
			wantPath:   "/api/v1/users",
			wantParams: map[string]string{},
		},
		{
			name:     "path parameter match",
			method:   "GET",
			path:     "/api/v1/users/123",
			wantErr:  false,
			wantPath: "/api/v1/users/:id",
			wantParams: map[string]string{
				"id": "123",
			},
		},
		{
			name:     "multiple path parameters",
			method:   "GET",
			path:     "/api/v1/orders/456/items/789",
			wantErr:  false,
			wantPath: "/api/v1/orders/:orderId/items/:itemId",
			wantParams: map[string]string{
				"orderId": "456",
				"itemId":  "789",
			},
		},
		{
			name:       "health check",
			method:     "GET",
			path:       "/health",
			wantErr:    false,
			wantPath:   "/health",
			wantParams: map[string]string{},
		},
		{
			name:    "path not found",
			method:  "GET",
			path:    "/api/v1/nonexistent",
			wantErr: true,
		},
		{
			name:    "method not allowed",
			method:  "DELETE",
			path:    "/api/v1/users",
			wantErr: true,
		},
		{
			name:       "trailing slash",
			method:     "GET",
			path:       "/api/v1/users/",
			wantErr:    false,
			wantPath:   "/api/v1/users",
			wantParams: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.Match(tt.method, tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if result.Route.Path != tt.wantPath {
				t.Errorf("Route path = %s, want %s", result.Route.Path, tt.wantPath)
			}

			// パラメータの確認
			if len(result.Params) != len(tt.wantParams) {
				t.Errorf("Params length = %d, want %d", len(result.Params), len(tt.wantParams))
			}

			for key, expectedValue := range tt.wantParams {
				if actualValue, ok := result.Params[key]; !ok {
					t.Errorf("Param %s not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Param %s = %s, want %s", key, actualValue, expectedValue)
				}
			}
		})
	}
}

func TestLoadFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.RoutingFileConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.RoutingFileConfig{
				Routes: []config.Route{
					{
						Path:    "/api/v1/users",
						Methods: []string{"GET", "POST"},
						Backend: config.BackendConfig{
							URL:     "https://user-service.com",
							Timeout: 30 * time.Second,
						},
						Priority: 10,
					},
					{
						Path:    "/api/v1/users/:id",
						Methods: []string{"GET"},
						Backend: config.BackendConfig{
							URL:     "https://user-service.com",
							Timeout: 30 * time.Second,
						},
						Priority: 20,
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid backend URL",
			config: &config.RoutingFileConfig{
				Routes: []config.Route{
					{
						Path:    "/api/v1/test",
						Methods: []string{"GET"},
						Backend: config.BackendConfig{
							URL:     "://invalid-url",
							Timeout: 30 * time.Second,
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter()
			err := router.LoadFromConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAllRoutes(t *testing.T) {
	router := NewRouter()

	routes := []*Route{
		{
			Path:    "/api/v1/users",
			Methods: []string{"GET"},
			Backend: &Backend{URL: mustParseURL("https://example.com")},
		},
		{
			Path:    "/api/v1/orders",
			Methods: []string{"GET"},
			Backend: &Backend{URL: mustParseURL("https://example.com")},
		},
	}

	for _, route := range routes {
		if err := router.AddRoute(route); err != nil {
			t.Fatalf("failed to add route: %v", err)
		}
	}

	allRoutes := router.GetAllRoutes()
	if len(allRoutes) != len(routes) {
		t.Errorf("GetAllRoutes() returned %d routes, want %d", len(allRoutes), len(routes))
	}
}

func TestRouteHasMethod(t *testing.T) {
	tests := []struct {
		name    string
		route   *Route
		method  string
		want    bool
	}{
		{
			name: "method exists",
			route: &Route{
				Methods: []string{"GET", "POST"},
			},
			method: "GET",
			want:   true,
		},
		{
			name: "method does not exist",
			route: &Route{
				Methods: []string{"GET", "POST"},
			},
			method: "DELETE",
			want:   false,
		},
		{
			name: "empty methods allows all",
			route: &Route{
				Methods: []string{},
			},
			method: "GET",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.route.HasMethod(tt.method)
			if got != tt.want {
				t.Errorf("HasMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
