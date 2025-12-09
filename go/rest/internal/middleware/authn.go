package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/kaitoimai/go-sample/rest/internal/auth"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
	"github.com/ogen-go/ogen/middleware"
)

// AuthnMiddleware は JWT 存在チェック + Claims 抽出を行うミドルウェア
// 注意: JWT署名検証はAPI Gatewayで実施済みのため、本サービスでは行わない
type AuthnMiddleware struct{}

// NewAuthnMiddleware creates a new authentication middleware
func NewAuthnMiddleware() *AuthnMiddleware {
	return &AuthnMiddleware{}
}

// Handle processes the authentication middleware
func (m *AuthnMiddleware) Handle(req middleware.Request, next middleware.Next) (middleware.Response, error) {
	// 全てのAPIリクエストでJWT存在チェックを行う
	// Authorization ヘッダーの存在確認（ガードレール）
	authHeader := req.Raw.Header.Get("Authorization")
	if authHeader == "" {
		return middleware.Response{}, myerrors.NewUnauthorized("認証トークンが必要です")
	}

	// Bearer 形式の確認
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return middleware.Response{}, myerrors.NewUnauthorized("認証形式が不正です")
	}

	tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
	if tokenString == "" {
		return middleware.Response{}, myerrors.NewUnauthorized("認証トークンが空です")
	}

	// JWT Claims抽出（署名検証なし）
	claims, err := extractClaims(tokenString)
	if err != nil {
		return middleware.Response{}, myerrors.NewUnauthorized("トークンの解析に失敗しました")
	}

	// ロールの妥当性確認
	if !auth.IsValidRole(claims.Role) {
		return middleware.Response{}, myerrors.NewUnauthorized("無効なロールです")
	}

	// ClaimsをContextに保存
	req.Context = auth.NewContext(req.Context, claims)

	return next(req)
}

// extractClaims extracts claims from JWT payload without signature verification
// JWT format: header.payload.signature
// 注意: API Gatewayで署名検証済みのため、ペイロード部分のみをデコード
func extractClaims(tokenString string) (*auth.Claims, error) {
	// JWT を '.' で分割
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, myerrors.NewUnauthorized("トークン形式が不正です")
	}

	// ペイロード部分（第2セグメント）をデコード
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, myerrors.NewUnauthorized("ペイロードのデコードに失敗しました")
	}

	// ClaimsにJSONデシリアライズ
	var claims auth.Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, myerrors.NewUnauthorized("Claimsの解析に失敗しました")
	}

	return &claims, nil
}
