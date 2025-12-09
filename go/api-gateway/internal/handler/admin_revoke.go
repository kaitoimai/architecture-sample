package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"api-gateway/internal/errors"
	"api-gateway/internal/repository"
)

// AdminRevokeConfig はAdminRevokeハンドラの設定
type AdminRevokeConfig struct {
	Repository    repository.SessionRepository
	APIKey        string        // 管理者APIキー
	JWTExpiration time.Duration // JWTの有効期限（Redis TTL用、デフォルト: 10時間)
	Logger        *slog.Logger
}

// AdminRevokeHandler は管理者による強制Revoke処理を行うハンドラ
type AdminRevokeHandler struct {
	repository    repository.SessionRepository
	apiKey        string
	jwtExpiration time.Duration
	logger        *slog.Logger
}

// RevokeRequest はRevoke APIのリクエストボディ
type RevokeRequest struct {
	UserID string `json:"user_id"`
}

// NewAdminRevokeHandler は新しいAdminRevokeHandlerを作成する
func NewAdminRevokeHandler(config AdminRevokeConfig) *AdminRevokeHandler {
	// デフォルト値の設定
	if config.JWTExpiration == 0 {
		config.JWTExpiration = 10 * time.Hour
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	return &AdminRevokeHandler{
		repository:    config.Repository,
		apiKey:        config.APIKey,
		jwtExpiration: config.JWTExpiration,
		logger:        config.Logger,
	}
}

// ServeHTTP はHTTPリクエストを処理する
func (h *AdminRevokeHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// POSTメソッドのみ許可
	if req.Method != http.MethodPost {
		h.writeError(w, errors.NewError(http.StatusMethodNotAllowed, "MethodNotAllowed", "only POST method is allowed"))
		return
	}

	// APIキー認証
	if err := h.authenticate(req); err != nil {
		h.logger.Warn("authentication failed", "error", err)
		h.writeError(w, errors.NewError(http.StatusUnauthorized, "Unauthorized", "invalid or missing API key"))
		return
	}

	// リクエストボディをパース
	var body RevokeRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		h.logger.Warn("failed to parse request body", "error", err)
		h.writeError(w, errors.NewError(http.StatusBadRequest, "BadRequest", "invalid request body"))
		return
	}

	// ユーザーIDのバリデーション
	if body.UserID == "" {
		h.logger.Warn("user_id is empty")
		h.writeError(w, errors.NewError(http.StatusBadRequest, "BadRequest", "user_id is required"))
		return
	}

	// 現在時刻を失効時刻としてRedisに保存
	revokedTime := time.Now()
	expiration := h.jwtExpiration

	if err := h.repository.SetRevokedTime(req.Context(), body.UserID, revokedTime, expiration); err != nil {
		h.logger.Error("failed to set revoked time", "error", err, "user_id", body.UserID)
		h.writeError(w, errors.NewError(http.StatusInternalServerError, "InternalServerError", "failed to process revoke"))
		return
	}

	h.logger.Info("user revoked successfully by admin",
		"user_id", body.UserID,
		"revoked_at", revokedTime.Format(time.RFC3339),
		"expires_at", revokedTime.Add(expiration).Format(time.RFC3339))

	// 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success":    true,
		"user_id":    body.UserID,
		"revoked_at": revokedTime.Format(time.RFC3339),
	})
}

// authenticate はAPIキー認証を行う
func (h *AdminRevokeHandler) authenticate(req *http.Request) error {
	apiKey := req.Header.Get("X-API-Key")
	if apiKey == "" {
		return fmt.Errorf("X-API-Key header is missing")
	}

	if apiKey != h.apiKey {
		return fmt.Errorf("invalid API key")
	}

	return nil
}

// writeError はエラーレスポンスを書き込む
func (h *AdminRevokeHandler) writeError(w http.ResponseWriter, err errors.GatewayError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())
	w.Write(errors.ToJSON(err))
}
