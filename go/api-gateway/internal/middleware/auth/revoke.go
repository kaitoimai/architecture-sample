package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"api-gateway/internal/errors"
	"api-gateway/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

// RevokeConfig はRevokeミドルウェアの設定
type RevokeConfig struct {
	Repository     repository.SessionRepository
	UserIDClaim    string // ユーザーIDのクレーム名（デフォルト: "sub")
	IssuedAtClaim  string // 発行時刻のクレーム名（デフォルト: "iat")
	FailOpen       bool   // Redis接続エラー時に通過させるか（デフォルト: false)
	Logger         *slog.Logger
}

// RevokeMiddleware はJWT Revokeをチェックするミドルウェア
type RevokeMiddleware struct {
	repository    repository.SessionRepository
	userIDClaim   string
	issuedAtClaim string
	failOpen      bool
	logger        *slog.Logger
}

// NewRevokeMiddleware は新しいRevokeMiddlewareを作成する
func NewRevokeMiddleware(config RevokeConfig) *RevokeMiddleware {
	// デフォルト値の設定
	if config.UserIDClaim == "" {
		config.UserIDClaim = "sub"
	}
	if config.IssuedAtClaim == "" {
		config.IssuedAtClaim = "iat"
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &RevokeMiddleware{
		repository:    config.Repository,
		userIDClaim:   config.UserIDClaim,
		issuedAtClaim: config.IssuedAtClaim,
		failOpen:      config.FailOpen,
		logger:        config.Logger,
	}
}

// Process はRevokeチェックを実行する
func (m *RevokeMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	// コンテキストからClaimsを取得
	claims, ok := GetClaimsFromContext(ctx)
	if !ok {
		// JWTミドルウェアの後に実行されることを想定
		// Claimsが存在しない場合はスキップ
		return ctx, nil
	}

	// ユーザーIDの取得
	userID, err := m.getUserID(claims)
	if err != nil {
		m.logger.Warn("failed to get user id from claims", "error", err)
		return ctx, errors.NewError(http.StatusUnauthorized, "Unauthorized", "invalid token claims")
	}

	// 発行時刻の取得
	issuedAt, err := m.getIssuedAt(claims)
	if err != nil {
		m.logger.Warn("failed to get issued at from claims", "error", err, "user_id", userID)
		return ctx, errors.NewError(http.StatusUnauthorized, "Unauthorized", "invalid token claims")
	}

	// Redisから失効時刻を取得
	revokedTime, err := m.repository.GetRevokedTime(ctx, userID)
	if err != nil {
		m.logger.Error("failed to get revoked time from redis", "error", err, "user_id", userID)

		// Redis接続エラー時の挙動
		if m.failOpen {
			// Fail Open: エラー時は通過させる（可用性優先）
			m.logger.Warn("redis error, allowing request (fail-open mode)", "user_id", userID)
			return ctx, nil
		}
		// Fail Close: エラー時は拒否（セキュリティ優先）
		return ctx, errors.NewError(http.StatusServiceUnavailable, "ServiceUnavailable", "session service unavailable")
	}

	// 失効時刻が設定されていない場合は通過
	if revokedTime.IsZero() {
		return ctx, nil
	}

	// 発行時刻が失効時刻より前の場合は拒否
	if issuedAt.Before(revokedTime) {
		m.logger.Info("token revoked",
			"user_id", userID,
			"issued_at", issuedAt.Format(time.RFC3339),
			"revoked_at", revokedTime.Format(time.RFC3339))
		return ctx, errors.NewError(http.StatusUnauthorized, "Unauthorized", "token has been revoked")
	}

	return ctx, nil
}

// getUserID はClaimsからユーザーIDを取得する
func (m *RevokeMiddleware) getUserID(claims jwt.MapClaims) (string, error) {
	userIDRaw, ok := claims[m.userIDClaim]
	if !ok {
		return "", fmt.Errorf("claim %s not found", m.userIDClaim)
	}

	userID, ok := userIDRaw.(string)
	if !ok {
		return "", fmt.Errorf("claim %s is not a string", m.userIDClaim)
	}

	if userID == "" {
		return "", fmt.Errorf("claim %s is empty", m.userIDClaim)
	}

	return userID, nil
}

// getIssuedAt はClaimsから発行時刻を取得する
func (m *RevokeMiddleware) getIssuedAt(claims jwt.MapClaims) (time.Time, error) {
	iatRaw, ok := claims[m.issuedAtClaim]
	if !ok {
		return time.Time{}, fmt.Errorf("claim %s not found", m.issuedAtClaim)
	}

	// iatは通常float64またはint64
	switch v := iatRaw.(type) {
	case float64:
		return time.Unix(int64(v), 0), nil
	case int64:
		return time.Unix(v, 0), nil
	case int:
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, fmt.Errorf("claim %s has invalid type: %T", m.issuedAtClaim, iatRaw)
	}
}
