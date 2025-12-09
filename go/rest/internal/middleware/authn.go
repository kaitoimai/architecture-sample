package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/kaitoimai/go-sample/rest/internal/auth"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
	"github.com/ogen-go/ogen/middleware"
)

// AuthnMiddleware は API Gateway で検証済みの JWT から Claims を抽出するミドルウェア
// 注意: JWT署名検証はAPI Gatewayで実施済みのため、本サービスでは署名検証を行わず、
//       ペイロード部分のみをデコードしてユーザー情報を取得する
type AuthnMiddleware struct{}

// NewAuthnMiddleware creates a new authentication middleware
func NewAuthnMiddleware() *AuthnMiddleware {
	return &AuthnMiddleware{}
}

// Handle は API Gateway から渡された JWT トークンを抽出し、Context に保存する
func (m *AuthnMiddleware) Handle(req middleware.Request, next middleware.Next) (middleware.Response, error) {
	// API Gateway から渡される Authorization ヘッダーの存在確認
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

	// API Gateway で署名検証済みの JWT からペイロードを抽出
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

// extractClaims は API Gateway で検証済みの JWT からペイロードを抽出する
// JWT format: header.payload.signature
// 注意: 署名検証は行わず、ペイロード（第2セグメント）のBase64デコードのみ実施
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
