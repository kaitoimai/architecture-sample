package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"api-gateway/internal/errors"
	"api-gateway/internal/repository"

	"github.com/golang-jwt/jwt/v5"
)

// LogoutConfig はLogoutハンドラの設定
type LogoutConfig struct {
	Repository     repository.SessionRepository
	UserIDClaim    string        // ユーザーIDのクレーム名（デフォルト: "sub")
	JWTExpiration  time.Duration // JWTの有効期限（Redis TTL用、デフォルト: 10時間)
	Logger         *slog.Logger
}

// LogoutHandler はログアウト処理を行うハンドラ
type LogoutHandler struct {
	repository    repository.SessionRepository
	userIDClaim   string
	jwtExpiration time.Duration
	logger        *slog.Logger
}

// NewLogoutHandler は新しいLogoutHandlerを作成する
func NewLogoutHandler(config LogoutConfig) *LogoutHandler {
	// デフォルト値の設定
	if config.UserIDClaim == "" {
		config.UserIDClaim = "sub"
	}
	if config.JWTExpiration == 0 {
		config.JWTExpiration = 10 * time.Hour
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &LogoutHandler{
		repository:    config.Repository,
		userIDClaim:   config.UserIDClaim,
		jwtExpiration: config.JWTExpiration,
		logger:        config.Logger,
	}
}

// ServeHTTP はHTTPリクエストを処理する
func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// DELETEメソッドのみ許可
	if req.Method != http.MethodDelete {
		h.writeError(w, errors.NewError(http.StatusMethodNotAllowed, "MethodNotAllowed", "only DELETE method is allowed"))
		return
	}

	// Authorizationヘッダーからトークンを取得
	token, err := h.extractToken(req)
	if err != nil {
		h.logger.Warn("failed to extract token", "error", err)
		h.writeError(w, errors.NewError(http.StatusUnauthorized, "Unauthorized", "missing or invalid authorization header"))
		return
	}

	// JWTをパース（検証なし）
	// Gateway経由なのでGatewayで既に検証済みを前提
	claims, err := h.parseTokenUnverified(token)
	if err != nil {
		h.logger.Warn("failed to parse token", "error", err)
		h.writeError(w, errors.NewError(http.StatusUnauthorized, "Unauthorized", "invalid token format"))
		return
	}

	// ユーザーIDを取得
	userID, err := h.getUserID(claims)
	if err != nil {
		h.logger.Warn("failed to get user id from claims", "error", err)
		h.writeError(w, errors.NewError(http.StatusUnauthorized, "Unauthorized", "invalid token claims"))
		return
	}

	// 現在時刻を失効時刻としてRedisに保存
	revokedTime := time.Now()
	expiration := h.jwtExpiration

	if err := h.repository.SetRevokedTime(req.Context(), userID, revokedTime, expiration); err != nil {
		h.logger.Error("failed to set revoked time", "error", err, "user_id", userID)
		h.writeError(w, errors.NewError(http.StatusInternalServerError, "InternalServerError", "failed to process logout"))
		return
	}

	h.logger.Info("user logged out successfully",
		"user_id", userID,
		"revoked_at", revokedTime.Format(time.RFC3339),
		"expires_at", revokedTime.Add(expiration).Format(time.RFC3339))

	// 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// extractToken はAuthorizationヘッダーからトークンを抽出する
func (h *LogoutHandler) extractToken(req *http.Request) (string, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.Split(authHeader, "Bearer ")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// parseTokenUnverified はJWTを検証せずにパースする
func (h *LogoutHandler) parseTokenUnverified(tokenString string) (jwt.MapClaims, error) {
	parser := jwt.NewParser()
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// getUserID はClaimsからユーザーIDを取得する
func (h *LogoutHandler) getUserID(claims jwt.MapClaims) (string, error) {
	userIDRaw, ok := claims[h.userIDClaim]
	if !ok {
		return "", fmt.Errorf("claim %s not found", h.userIDClaim)
	}

	userID, ok := userIDRaw.(string)
	if !ok {
		return "", fmt.Errorf("claim %s is not a string", h.userIDClaim)
	}

	if userID == "" {
		return "", fmt.Errorf("claim %s is empty", h.userIDClaim)
	}

	return userID, nil
}

// writeError はエラーレスポンスを書き込む
func (h *LogoutHandler) writeError(w http.ResponseWriter, err errors.GatewayError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())
	w.Write(errors.ToJSON(err))
}
