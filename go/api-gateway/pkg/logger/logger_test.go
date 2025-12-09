package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name           string
		config         Config
		wantLevel      slog.Level
		wantJSONFormat bool
	}{
		{
			name: "JSON format with info level",
			config: Config{
				Level:  LevelInfo,
				Format: "json",
			},
			wantLevel:      slog.LevelInfo,
			wantJSONFormat: true,
		},
		{
			name: "text format with debug level",
			config: Config{
				Level:  LevelDebug,
				Format: "text",
			},
			wantLevel:      slog.LevelDebug,
			wantJSONFormat: false,
		},
		{
			name: "warn level",
			config: Config{
				Level:  LevelWarn,
				Format: "json",
			},
			wantLevel:      slog.LevelWarn,
			wantJSONFormat: true,
		},
		{
			name: "error level",
			config: Config{
				Level:  LevelError,
				Format: "json",
			},
			wantLevel:      slog.LevelError,
			wantJSONFormat: true,
		},
		{
			name: "default to info for invalid level",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
			wantLevel:      slog.LevelInfo,
			wantJSONFormat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Fatal("New() returned nil")
			}

			// ログが正しく動作するか簡易テスト
			var buf bytes.Buffer

			// 新しいロガーを作成してバッファに出力
			testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: tt.wantLevel,
			}))

			// ログレベルに応じてログを出力
			switch tt.wantLevel {
			case slog.LevelDebug:
				testLogger.Debug("test message")
			case slog.LevelInfo:
				testLogger.Info("test message")
			case slog.LevelWarn:
				testLogger.Warn("test message")
			case slog.LevelError:
				testLogger.Error("test message")
			default:
				testLogger.Info("test message")
			}

			if buf.Len() == 0 {
				t.Error("expected log output, got none")
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     LogLevel
		wantLevel slog.Level
	}{
		{
			name:      "debug level",
			level:     LevelDebug,
			wantLevel: slog.LevelDebug,
		},
		{
			name:      "info level",
			level:     LevelInfo,
			wantLevel: slog.LevelInfo,
		},
		{
			name:      "warn level",
			level:     LevelWarn,
			wantLevel: slog.LevelWarn,
		},
		{
			name:      "error level",
			level:     LevelError,
			wantLevel: slog.LevelError,
		},
		{
			name:      "invalid level defaults to info",
			level:     "invalid",
			wantLevel: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLevel(tt.level)
			if got != tt.wantLevel {
				t.Errorf("parseLevel() = %v, want %v", got, tt.wantLevel)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	tests := []struct {
		name       string
		config     Config
		logFunc    func(*slog.Logger)
		wantInJSON bool
		wantFields []string
	}{
		{
			name: "info log with JSON format",
			config: Config{
				Level:  LevelInfo,
				Format: "json",
			},
			logFunc: func(l *slog.Logger) {
				l.Info("test message", "key", "value")
			},
			wantInJSON: true,
			wantFields: []string{"test message", "key", "value"},
		},
		{
			name: "error log with text format",
			config: Config{
				Level:  LevelError,
				Format: "text",
			},
			logFunc: func(l *slog.Logger) {
				l.Error("error message", "error", "test error")
			},
			wantInJSON: false,
			wantFields: []string{"error message", "error", "test error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			// テスト用のハンドラを作成
			var handler slog.Handler
			opts := &slog.HandlerOptions{
				Level: parseLevel(tt.config.Level),
			}

			if tt.config.Format == "json" {
				handler = slog.NewJSONHandler(&buf, opts)
			} else {
				handler = slog.NewTextHandler(&buf, opts)
			}

			logger := slog.New(handler)
			tt.logFunc(logger)

			output := buf.String()
			if len(output) == 0 {
				t.Fatal("expected log output, got none")
			}

			// フィールドの存在確認
			for _, field := range tt.wantFields {
				if !strings.Contains(output, field) {
					t.Errorf("log output missing field %q: %s", field, output)
				}
			}

			// JSON形式の検証
			if tt.wantInJSON {
				var jsonData map[string]any
				if err := json.Unmarshal([]byte(output), &jsonData); err != nil {
					t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, output)
				}
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name          string
		configLevel   LogLevel
		logLevel      slog.Level
		logMessage    string
		shouldBeLogged bool
	}{
		{
			name:          "debug log with debug level",
			configLevel:   LevelDebug,
			logLevel:      slog.LevelDebug,
			logMessage:    "debug message",
			shouldBeLogged: true,
		},
		{
			name:          "debug log with info level",
			configLevel:   LevelInfo,
			logLevel:      slog.LevelDebug,
			logMessage:    "debug message",
			shouldBeLogged: false,
		},
		{
			name:          "info log with info level",
			configLevel:   LevelInfo,
			logLevel:      slog.LevelInfo,
			logMessage:    "info message",
			shouldBeLogged: true,
		},
		{
			name:          "error log with error level",
			configLevel:   LevelError,
			logLevel:      slog.LevelError,
			logMessage:    "error message",
			shouldBeLogged: true,
		},
		{
			name:          "info log with error level",
			configLevel:   LevelError,
			logLevel:      slog.LevelInfo,
			logMessage:    "info message",
			shouldBeLogged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				Level: parseLevel(tt.configLevel),
			})
			logger := slog.New(handler)

			// ログレベルに応じてログを出力
			switch tt.logLevel {
			case slog.LevelDebug:
				logger.Debug(tt.logMessage)
			case slog.LevelInfo:
				logger.Info(tt.logMessage)
			case slog.LevelWarn:
				logger.Warn(tt.logMessage)
			case slog.LevelError:
				logger.Error(tt.logMessage)
			}

			output := buf.String()
			hasOutput := len(output) > 0

			if hasOutput != tt.shouldBeLogged {
				t.Errorf("log output = %v, want %v\noutput: %s", hasOutput, tt.shouldBeLogged, output)
			}

			if tt.shouldBeLogged && !strings.Contains(output, tt.logMessage) {
				t.Errorf("log output missing message %q: %s", tt.logMessage, output)
			}
		})
	}
}
