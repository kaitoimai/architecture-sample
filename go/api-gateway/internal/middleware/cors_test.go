package middleware

import (
	"context"
	"net/http"
	"testing"
)

func TestNewCORSMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		config         CORSConfig
		wantMethods    []string
		wantHeaders    []string
		wantMaxAge     int
		wantCredentials bool
	}{
		{
			name: "デフォルト値が設定される",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
			},
			wantMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			wantHeaders: []string{"Accept", "Content-Type", "Authorization"},
			wantMaxAge:  3600,
		},
		{
			name: "カスタム値が設定される",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowedMethods:   []string{"GET", "POST"},
				AllowedHeaders:   []string{"Content-Type"},
				MaxAge:           7200,
				AllowCredentials: true,
			},
			wantMethods:     []string{"GET", "POST"},
			wantHeaders:     []string{"Content-Type"},
			wantMaxAge:      7200,
			wantCredentials: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(tt.config)
			if m == nil {
				t.Fatal("NewCORSMiddleware returned nil")
			}

			if len(m.config.AllowedMethods) != len(tt.wantMethods) {
				t.Errorf("AllowedMethods = %v, want %v", m.config.AllowedMethods, tt.wantMethods)
			}

			if len(m.config.AllowedHeaders) != len(tt.wantHeaders) {
				t.Errorf("AllowedHeaders = %v, want %v", m.config.AllowedHeaders, tt.wantHeaders)
			}

			if m.config.MaxAge != tt.wantMaxAge {
				t.Errorf("MaxAge = %d, want %d", m.config.MaxAge, tt.wantMaxAge)
			}

			if m.config.AllowCredentials != tt.wantCredentials {
				t.Errorf("AllowCredentials = %v, want %v", m.config.AllowCredentials, tt.wantCredentials)
			}
		})
	}
}

func TestCORSMiddleware_Process(t *testing.T) {
	tests := []struct {
		name           string
		config         CORSConfig
		origin         string
		wantHeaders    bool
		wantOrigin     string
		wantMethods    string
		wantAllowCreds string
	}{
		{
			name: "Originヘッダーがない場合は何もしない",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
			},
			origin:      "",
			wantHeaders: false,
		},
		{
			name: "許可されたオリジンの場合はヘッダーを設定",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:      "https://example.com",
			wantHeaders: true,
			wantOrigin:  "https://example.com",
			wantMethods: "GET, POST",
		},
		{
			name: "ワイルドカードの場合は全てのオリジンを許可",
			config: CORSConfig{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST"},
			},
			origin:      "https://example.com",
			wantHeaders: true,
			wantOrigin:  "*",
			wantMethods: "GET, POST",
		},
		{
			name: "許可されていないオリジンの場合はヘッダーを設定しない",
			config: CORSConfig{
				AllowedOrigins: []string{"https://example.com"},
			},
			origin:      "https://evil.com",
			wantHeaders: false,
		},
		{
			name: "AllowCredentialsが有効な場合",
			config: CORSConfig{
				AllowedOrigins:   []string{"https://example.com"},
				AllowCredentials: true,
			},
			origin:         "https://example.com",
			wantHeaders:    true,
			wantOrigin:     "https://example.com",
			wantAllowCreds: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewCORSMiddleware(tt.config)

			req, err := http.NewRequest("GET", "http://localhost/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			ctx := context.Background()
			newCtx, err := m.Process(ctx, req)
			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}

			headers := GetCORSHeaders(newCtx)

			if tt.wantHeaders {
				if headers == nil {
					t.Error("Expected CORS headers to be set, but got nil")
					return
				}

				if tt.wantOrigin != "" {
					if got := headers["Access-Control-Allow-Origin"]; got != tt.wantOrigin {
						t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, tt.wantOrigin)
					}
				}

				if tt.wantMethods != "" {
					if got := headers["Access-Control-Allow-Methods"]; got != tt.wantMethods {
						t.Errorf("Access-Control-Allow-Methods = %v, want %v", got, tt.wantMethods)
					}
				}

				if tt.wantAllowCreds != "" {
					if got := headers["Access-Control-Allow-Credentials"]; got != tt.wantAllowCreds {
						t.Errorf("Access-Control-Allow-Credentials = %v, want %v", got, tt.wantAllowCreds)
					}
				}
			} else {
				if headers != nil {
					t.Errorf("Expected no CORS headers, but got %v", headers)
				}
			}
		})
	}
}

func TestCORSMiddleware_isOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		want           bool
	}{
		{
			name:           "ワイルドカードで全て許可",
			allowedOrigins: []string{"*"},
			origin:         "https://example.com",
			want:           true,
		},
		{
			name:           "許可されたオリジン",
			allowedOrigins: []string{"https://example.com", "https://example.org"},
			origin:         "https://example.com",
			want:           true,
		},
		{
			name:           "許可されていないオリジン",
			allowedOrigins: []string{"https://example.com"},
			origin:         "https://evil.com",
			want:           false,
		},
		{
			name:           "空のオリジンリスト",
			allowedOrigins: []string{},
			origin:         "https://example.com",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &CORSMiddleware{
				config: CORSConfig{
					AllowedOrigins: tt.allowedOrigins,
				},
			}

			if got := m.isOriginAllowed(tt.origin); got != tt.want {
				t.Errorf("isOriginAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCORSHeaders(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		want    map[string]string
		wantNil bool
	}{
		{
			name: "CORSヘッダーが設定されている場合",
			ctx: context.WithValue(context.Background(), corsHeadersKey, map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			}),
			want: map[string]string{
				"Access-Control-Allow-Origin": "https://example.com",
			},
			wantNil: false,
		},
		{
			name:    "CORSヘッダーが設定されていない場合",
			ctx:     context.Background(),
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCORSHeaders(tt.ctx)

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetCORSHeaders() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("GetCORSHeaders() returned nil, want non-nil")
			}

			for key, wantValue := range tt.want {
				if gotValue, ok := got[key]; !ok || gotValue != wantValue {
					t.Errorf("GetCORSHeaders()[%s] = %v, want %v", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	m := NewCORSMiddleware(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Another-Header"},
	})

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("Origin", "https://example.com")

	ctx := context.Background()
	newCtx, err := m.Process(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	headers := GetCORSHeaders(newCtx)
	if headers == nil {
		t.Fatal("Expected CORS headers to be set")
	}

	want := "X-Custom-Header, X-Another-Header"
	if got := headers["Access-Control-Expose-Headers"]; got != want {
		t.Errorf("Access-Control-Expose-Headers = %v, want %v", got, want)
	}
}

func TestCORSMiddleware_MaxAge(t *testing.T) {
	m := NewCORSMiddleware(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         7200,
	})

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("Origin", "https://example.com")

	ctx := context.Background()
	newCtx, err := m.Process(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	headers := GetCORSHeaders(newCtx)
	if headers == nil {
		t.Fatal("Expected CORS headers to be set")
	}

	want := "7200"
	if got := headers["Access-Control-Max-Age"]; got != want {
		t.Errorf("Access-Control-Max-Age = %v, want %v", got, want)
	}
}
