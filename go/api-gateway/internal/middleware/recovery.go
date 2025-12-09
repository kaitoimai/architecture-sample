package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"api-gateway/internal/errors"
)

// RecoveryConfig はリカバリーミドルウェアの設定
type RecoveryConfig struct {
	// EnableStackTrace はスタックトレースをログに出力するか
	EnableStackTrace bool
}

// RecoveryMiddleware はパニックから回復するミドルウェア
type RecoveryMiddleware struct {
	logger *slog.Logger
	config RecoveryConfig
}

// NewRecoveryMiddleware は新しいリカバリーミドルウェアを作成する
func NewRecoveryMiddleware(logger *slog.Logger, config RecoveryConfig) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
		config: config,
	}
}

// Process はパニックから回復する
func (m *RecoveryMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	// パニックハンドリングはミドルウェアチェーン内では難しいため、
	// このミドルウェアは主にパニック発生時の情報をコンテキストに保存する役割を持つ
	// 実際のパニック回復はハンドラ層で行う必要がある
	return ctx, nil
}

// Recover はパニックから回復し、エラーを返す
// この関数はミドルウェアチェーンの実行をラップする形で使用される
func (m *RecoveryMiddleware) Recover(ctx context.Context, req *http.Request, next func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// パニックをログに記録
			requestID, _ := GetRequestID(ctx)

			attrs := []any{
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Any("panic", r),
			}

			// スタックトレースの追加
			if m.config.EnableStackTrace {
				attrs = append(attrs, slog.String("stack", string(debug.Stack())))
			}

			m.logger.Error("panic recovered", attrs...)

			// パニックをエラーに変換
			err = errors.NewInternalServerError(fmt.Sprintf("panic recovered: %v", r))
		}
	}()

	return next()
}
