package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

func init() {
	defaultLogger = New(LevelInfo)
}

type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

func New(level Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func NewFromEnv() *slog.Logger {
	levelStr := os.Getenv("LOG_LEVEL")
	level := parseLogLevel(levelStr)
	return New(level)
}

func parseLogLevel(levelStr string) Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return LevelDebug
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

func SetDefault(l *slog.Logger) {
	defaultLogger = l
	slog.SetDefault(l)
}

func Default() *slog.Logger {
	return defaultLogger
}

func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

func WithContext(ctx context.Context) *slog.Logger {
	return defaultLogger
}

// context integration
type ctxKey struct{}

// NewContext stores the given logger in context and returns the derived context.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	if l == nil {
		l = defaultLogger
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext retrieves a logger from context. If not found, returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	if v := ctx.Value(ctxKey{}); v != nil {
		if l, ok := v.(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return defaultLogger
}
