package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// LoggingConfig はログミドルウェアの設定
type LoggingConfig struct {
	// LogRequestBody はリクエストボディをログに記録するか
	LogRequestBody bool

	// LogResponseBody はレスポンスボディをログに記録するか
	LogResponseBody bool

	// SkipPaths はログ記録をスキップするパスのリスト
	SkipPaths []string
}

// LoggingMiddleware はアクセスログを記録するミドルウェア
type LoggingMiddleware struct {
	logger *slog.Logger
	config LoggingConfig
}

// NewLoggingMiddleware は新しいログミドルウェアを作成する
func NewLoggingMiddleware(logger *slog.Logger, config LoggingConfig) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
		config: config,
	}
}

// contextKey はコンテキストのキー型
type loggingContextKey string

const (
	// requestIDKey はリクエストIDを格納するコンテキストキー
	requestIDKey loggingContextKey = "request_id"

	// requestStartTimeKey はリクエスト開始時刻を格納するコンテキストキー
	requestStartTimeKey loggingContextKey = "request_start_time"
)

// Process はアクセスログを記録する
func (m *LoggingMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	// スキップパスのチェック
	if m.shouldSkipPath(req.URL.Path) {
		return ctx, nil
	}

	// リクエストIDの生成
	requestID := uuid.New().String()
	ctx = context.WithValue(ctx, requestIDKey, requestID)

	// リクエスト開始時刻を記録
	startTime := time.Now()
	ctx = context.WithValue(ctx, requestStartTimeKey, startTime)

	// リクエストログの記録
	m.logRequest(req, requestID)

	return ctx, nil
}

// logRequest はリクエスト情報をログに記録する
func (m *LoggingMiddleware) logRequest(req *http.Request, requestID string) {
	attrs := []any{
		slog.String("request_id", requestID),
		slog.String("method", req.Method),
		slog.String("path", req.URL.Path),
		slog.String("remote_addr", req.RemoteAddr),
		slog.String("user_agent", req.UserAgent()),
	}

	// クエリパラメータが存在する場合は記録
	if req.URL.RawQuery != "" {
		attrs = append(attrs, slog.String("query", req.URL.RawQuery))
	}

	m.logger.Info("incoming request", attrs...)
}

// shouldSkipPath はパスがスキップ対象か確認する
func (m *LoggingMiddleware) shouldSkipPath(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// GetRequestID はコンテキストからリクエストIDを取得する
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

// GetRequestStartTime はコンテキストからリクエスト開始時刻を取得する
func GetRequestStartTime(ctx context.Context) (time.Time, bool) {
	startTime, ok := ctx.Value(requestStartTimeKey).(time.Time)
	return startTime, ok
}

// LogResponse はレスポンス情報をログに記録するヘルパー関数
// この関数はハンドラ側から呼び出される
func LogResponse(logger *slog.Logger, ctx context.Context, statusCode int, bytesWritten int) {
	requestID, _ := GetRequestID(ctx)
	startTime, ok := GetRequestStartTime(ctx)

	attrs := []any{
		slog.String("request_id", requestID),
		slog.Int("status_code", statusCode),
		slog.Int("bytes_written", bytesWritten),
	}

	if ok {
		duration := time.Since(startTime)
		attrs = append(attrs, slog.Duration("duration", duration))
	}

	logger.Info("response sent", attrs...)
}
