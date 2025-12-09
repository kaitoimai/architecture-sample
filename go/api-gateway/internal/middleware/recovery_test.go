package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"api-gateway/internal/errors"
)

func TestNewRecoveryMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	config := RecoveryConfig{
		EnableStackTrace: true,
	}

	m := NewRecoveryMiddleware(logger, config)
	if m == nil {
		t.Fatal("NewRecoveryMiddleware returned nil")
	}

	if m.logger != logger {
		t.Error("logger not set correctly")
	}

	if m.config.EnableStackTrace != config.EnableStackTrace {
		t.Error("EnableStackTrace not set correctly")
	}
}

func TestRecoveryMiddleware_Process(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	m := NewRecoveryMiddleware(logger, RecoveryConfig{})

	req, err := http.NewRequest("GET", "http://localhost/test", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	newCtx, err := m.Process(ctx, req)
	if err != nil {
		t.Errorf("Process() error = %v", err)
	}

	if newCtx == nil {
		t.Error("Process() returned nil context")
	}
}

func TestRecoveryMiddleware_Recover_NoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	m := NewRecoveryMiddleware(logger, RecoveryConfig{})

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	ctx := context.Background()

	err := m.Recover(ctx, req, func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Recover() error = %v, want nil", err)
	}
}

func TestRecoveryMiddleware_Recover_WithPanic(t *testing.T) {
	tests := []struct {
		name             string
		enableStackTrace bool
		panicValue       any
		wantLogContains  []string
	}{
		{
			name:             "文字列パニック、スタックトレース無効",
			enableStackTrace: false,
			panicValue:       "something went wrong",
			wantLogContains: []string{
				"panic recovered",
				"something went wrong",
				"method=GET",
				"path=/test",
			},
		},
		{
			name:             "文字列パニック、スタックトレース有効",
			enableStackTrace: true,
			panicValue:       "something went wrong",
			wantLogContains: []string{
				"panic recovered",
				"something went wrong",
				"stack=",
			},
		},
		{
			name:             "エラーパニック",
			enableStackTrace: false,
			panicValue:       errors.NewBadRequestError("test error"),
			wantLogContains: []string{
				"panic recovered",
				"test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			m := NewRecoveryMiddleware(logger, RecoveryConfig{
				EnableStackTrace: tt.enableStackTrace,
			})

			req, _ := http.NewRequest("GET", "http://localhost/test", nil)
			ctx := context.Background()

			err := m.Recover(ctx, req, func() error {
				panic(tt.panicValue)
			})

			// エラーが返されることを確認
			if err == nil {
				t.Error("Recover() error = nil, want non-nil")
				return
			}

			// GatewayErrorであることを確認
			gwErr, ok := err.(errors.GatewayError)
			if !ok {
				t.Error("Recover() error is not GatewayError")
				return
			}

			// 500エラーであることを確認
			if gwErr.StatusCode() != 500 {
				t.Errorf("StatusCode = %d, want 500", gwErr.StatusCode())
			}

			// ログ出力の確認
			logOutput := buf.String()
			for _, want := range tt.wantLogContains {
				if !strings.Contains(logOutput, want) {
					t.Errorf("log does not contain %q\nlog output: %s", want, logOutput)
				}
			}
		})
	}
}

func TestRecoveryMiddleware_Recover_WithRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	m := NewRecoveryMiddleware(logger, RecoveryConfig{})

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	ctx := context.WithValue(context.Background(), requestIDKey, "test-request-id")

	err := m.Recover(ctx, req, func() error {
		panic("test panic")
	})

	if err == nil {
		t.Fatal("Recover() error = nil, want non-nil")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "request_id=test-request-id") {
		t.Error("log does not contain request_id")
	}
}

func TestRecoveryMiddleware_Recover_NextReturnsError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	m := NewRecoveryMiddleware(logger, RecoveryConfig{})

	req, _ := http.NewRequest("GET", "http://localhost/test", nil)
	ctx := context.Background()

	expectedErr := errors.NewBadRequestError("test error")
	err := m.Recover(ctx, req, func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Recover() error = %v, want %v", err, expectedErr)
	}
}

func TestRecoveryMiddleware_Recover_Race(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	m := NewRecoveryMiddleware(logger, RecoveryConfig{EnableStackTrace: true})

	// 並行実行テスト
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(index int) {
			req, _ := http.NewRequest("GET", "http://localhost/test", nil)
			ctx := context.Background()

			if index%2 == 0 {
				// パニックするケース
				_ = m.Recover(ctx, req, func() error {
					panic("test panic")
				})
			} else {
				// 正常終了するケース
				_ = m.Recover(ctx, req, func() error {
					return nil
				})
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
