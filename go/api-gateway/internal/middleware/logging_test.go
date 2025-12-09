package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewLoggingMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	config := LoggingConfig{
		LogRequestBody:  true,
		LogResponseBody: false,
		SkipPaths:       []string{"/health"},
	}

	m := NewLoggingMiddleware(logger, config)
	if m == nil {
		t.Fatal("NewLoggingMiddleware returned nil")
	}

	if m.logger != logger {
		t.Error("logger not set correctly")
	}

	if m.config.LogRequestBody != config.LogRequestBody {
		t.Error("LogRequestBody not set correctly")
	}

	if m.config.LogResponseBody != config.LogResponseBody {
		t.Error("LogResponseBody not set correctly")
	}

	if len(m.config.SkipPaths) != len(config.SkipPaths) {
		t.Error("SkipPaths not set correctly")
	}
}

func TestLoggingMiddleware_Process(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		skipPaths      []string
		wantRequestID  bool
		wantStartTime  bool
		wantLogMessage bool
	}{
		{
			name:           "通常のリクエスト",
			path:           "/api/users",
			skipPaths:      []string{},
			wantRequestID:  true,
			wantStartTime:  true,
			wantLogMessage: true,
		},
		{
			name:           "スキップパスのリクエスト",
			path:           "/health",
			skipPaths:      []string{"/health", "/metrics"},
			wantRequestID:  false,
			wantStartTime:  false,
			wantLogMessage: false,
		},
		{
			name:           "クエリパラメータ付きリクエスト",
			path:           "/api/users",
			skipPaths:      []string{},
			wantRequestID:  true,
			wantStartTime:  true,
			wantLogMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			m := NewLoggingMiddleware(logger, LoggingConfig{
				SkipPaths: tt.skipPaths,
			})

			url := "http://localhost" + tt.path
			if tt.name == "クエリパラメータ付きリクエスト" {
				url += "?page=1&limit=10"
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "test-agent")

			ctx := context.Background()
			newCtx, err := m.Process(ctx, req)
			if err != nil {
				t.Errorf("Process() error = %v", err)
				return
			}

			// リクエストIDの確認
			requestID, hasRequestID := GetRequestID(newCtx)
			if hasRequestID != tt.wantRequestID {
				t.Errorf("hasRequestID = %v, want %v", hasRequestID, tt.wantRequestID)
			}
			if tt.wantRequestID && requestID == "" {
				t.Error("requestID is empty")
			}

			// 開始時刻の確認
			startTime, hasStartTime := GetRequestStartTime(newCtx)
			if hasStartTime != tt.wantStartTime {
				t.Errorf("hasStartTime = %v, want %v", hasStartTime, tt.wantStartTime)
			}
			if tt.wantStartTime && startTime.IsZero() {
				t.Error("startTime is zero")
			}

			// ログメッセージの確認
			logOutput := buf.String()
			hasLogMessage := strings.Contains(logOutput, "incoming request")
			if hasLogMessage != tt.wantLogMessage {
				t.Errorf("hasLogMessage = %v, want %v", hasLogMessage, tt.wantLogMessage)
			}

			// ログの詳細確認
			if tt.wantLogMessage {
				if !strings.Contains(logOutput, "method=GET") {
					t.Error("log does not contain method")
				}
				if !strings.Contains(logOutput, "path="+tt.path) {
					t.Error("log does not contain path")
				}
				if !strings.Contains(logOutput, "remote_addr=192.168.1.1:12345") {
					t.Error("log does not contain remote_addr")
				}
				if !strings.Contains(logOutput, "user_agent=test-agent") {
					t.Error("log does not contain user_agent")
				}

				// クエリパラメータの確認
				if tt.name == "クエリパラメータ付きリクエスト" {
					if !strings.Contains(logOutput, "query=") {
						t.Error("log does not contain query field")
					}
					if !strings.Contains(logOutput, "page=1") {
						t.Error("log does not contain query parameters")
					}
				}
			}
		})
	}
}

func TestLoggingMiddleware_shouldSkipPath(t *testing.T) {
	tests := []struct {
		name      string
		skipPaths []string
		path      string
		want      bool
	}{
		{
			name:      "スキップパスに該当",
			skipPaths: []string{"/health", "/metrics"},
			path:      "/health",
			want:      true,
		},
		{
			name:      "スキップパスに該当しない",
			skipPaths: []string{"/health", "/metrics"},
			path:      "/api/users",
			want:      false,
		},
		{
			name:      "空のスキップパスリスト",
			skipPaths: []string{},
			path:      "/health",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &LoggingMiddleware{
				config: LoggingConfig{
					SkipPaths: tt.skipPaths,
				},
			}

			if got := m.shouldSkipPath(tt.path); got != tt.want {
				t.Errorf("shouldSkipPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantID    string
		wantFound bool
	}{
		{
			name:      "リクエストIDが設定されている場合",
			ctx:       context.WithValue(context.Background(), requestIDKey, "test-request-id"),
			wantID:    "test-request-id",
			wantFound: true,
		},
		{
			name:      "リクエストIDが設定されていない場合",
			ctx:       context.Background(),
			wantID:    "",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotFound := GetRequestID(tt.ctx)

			if gotFound != tt.wantFound {
				t.Errorf("GetRequestID() found = %v, want %v", gotFound, tt.wantFound)
			}

			if gotID != tt.wantID {
				t.Errorf("GetRequestID() id = %v, want %v", gotID, tt.wantID)
			}
		})
	}
}

func TestGetRequestStartTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		ctx       context.Context
		wantTime  time.Time
		wantFound bool
	}{
		{
			name:      "開始時刻が設定されている場合",
			ctx:       context.WithValue(context.Background(), requestStartTimeKey, now),
			wantTime:  now,
			wantFound: true,
		},
		{
			name:      "開始時刻が設定されていない場合",
			ctx:       context.Background(),
			wantTime:  time.Time{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotFound := GetRequestStartTime(tt.ctx)

			if gotFound != tt.wantFound {
				t.Errorf("GetRequestStartTime() found = %v, want %v", gotFound, tt.wantFound)
			}

			if !gotTime.Equal(tt.wantTime) {
				t.Errorf("GetRequestStartTime() time = %v, want %v", gotTime, tt.wantTime)
			}
		})
	}
}

func TestLogResponse(t *testing.T) {
	tests := []struct {
		name         string
		ctx          context.Context
		statusCode   int
		bytesWritten int
		wantRequestID bool
		wantDuration bool
	}{
		{
			name: "完全なコンテキスト",
			ctx: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, requestIDKey, "test-id")
				ctx = context.WithValue(ctx, requestStartTimeKey, time.Now().Add(-100*time.Millisecond))
				return ctx
			}(),
			statusCode:    200,
			bytesWritten:  1024,
			wantRequestID: true,
			wantDuration:  true,
		},
		{
			name:          "空のコンテキスト",
			ctx:           context.Background(),
			statusCode:    404,
			bytesWritten:  0,
			wantRequestID: false,
			wantDuration:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))

			LogResponse(logger, tt.ctx, tt.statusCode, tt.bytesWritten)

			logOutput := buf.String()

			if !strings.Contains(logOutput, "response sent") {
				t.Error("log does not contain 'response sent'")
			}

			if !strings.Contains(logOutput, "status_code=") {
				t.Error("log does not contain status_code")
			}

			if tt.wantRequestID {
				if !strings.Contains(logOutput, "request_id=test-id") {
					t.Error("log does not contain request_id")
				}
			}

			if tt.wantDuration {
				if !strings.Contains(logOutput, "duration=") {
					t.Error("log does not contain duration")
				}
			}
		})
	}
}

func TestLoggingMiddleware_Process_Race(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	m := NewLoggingMiddleware(logger, LoggingConfig{})

	// 並行実行テスト
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "http://localhost/test", nil)
			ctx := context.Background()
			_, _ = m.Process(ctx, req)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
