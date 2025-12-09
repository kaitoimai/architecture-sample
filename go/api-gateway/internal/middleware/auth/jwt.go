package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"api-gateway/internal/errors"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey はコンテキストのキー型
type contextKey string

const (
	// ClaimsContextKey はJWTクレームを格納するコンテキストキー
	ClaimsContextKey contextKey = "jwt_claims"
)

// JWTConfig はJWT認証ミドルウェアの設定
type JWTConfig struct {
	// PublicKeys はJWT検証用の公開鍵マップ (kid → 公開鍵)
	PublicKeys map[string]*rsa.PublicKey

	// SkipValidation はtrueの場合、JWT検証をスキップする（開発環境用）
	SkipValidation bool

	// RequiredClaims は必須のクレーム
	RequiredClaims []string
}

// JWTMiddleware はJWT認証を行うミドルウェア
type JWTMiddleware struct {
	config JWTConfig
}

// NewJWTMiddleware は新しいJWT認証ミドルウェアを作成する
func NewJWTMiddleware(config JWTConfig) *JWTMiddleware {
	return &JWTMiddleware{
		config: config,
	}
}

// NewJWTMiddlewareFromPEMs はPEM形式の公開鍵マップからJWT認証ミドルウェアを作成する
func NewJWTMiddlewareFromPEMs(publicKeyPEMs map[string]string, skipValidation bool, requiredClaims []string) (*JWTMiddleware, error) {
	if skipValidation {
		return &JWTMiddleware{
			config: JWTConfig{
				SkipValidation: true,
				RequiredClaims: requiredClaims,
			},
		}, nil
	}

	publicKeys := make(map[string]*rsa.PublicKey)
	for kid, pemStr := range publicKeyPEMs {
		publicKey, err := parsePublicKeyFromPEM(pemStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key for kid=%s: %w", kid, err)
		}
		publicKeys[kid] = publicKey
	}

	return &JWTMiddleware{
		config: JWTConfig{
			PublicKeys:     publicKeys,
			SkipValidation: false,
			RequiredClaims: requiredClaims,
		},
	}, nil
}

// Process はJWT認証を実行する
func (m *JWTMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	// Authorizationヘッダーからトークンを取得
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		return ctx, errors.NewUnauthorizedError("missing authorization header")
	}

	// "Bearer "プレフィックスを削除
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return ctx, errors.NewUnauthorizedError("invalid authorization header format")
	}

	// 検証をスキップする場合は、トークンをパースせずにコンテキストに保存
	if m.config.SkipValidation {
		ctx = context.WithValue(ctx, ClaimsContextKey, jwt.MapClaims{
			"skip_validation": true,
		})
		return ctx, nil
	}

	// JWTトークンをパースして検証
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// アルゴリズムの確認
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// kidヘッダーから公開鍵を取得
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		publicKey, ok := m.config.PublicKeys[kid]
		if !ok {
			return nil, fmt.Errorf("public key not found for kid: %s", kid)
		}

		return publicKey, nil
	})

	if err != nil {
		return ctx, errors.NewUnauthorizedError(fmt.Sprintf("invalid token: %v", err))
	}

	if !token.Valid {
		return ctx, errors.NewUnauthorizedError("token is not valid")
	}

	// クレームを取得
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx, errors.NewUnauthorizedError("invalid token claims")
	}

	// 必須クレームの検証
	if err := m.validateRequiredClaims(claims); err != nil {
		return ctx, err
	}

	// クレームをコンテキストに保存
	ctx = context.WithValue(ctx, ClaimsContextKey, claims)

	return ctx, nil
}

// validateRequiredClaims は必須クレームが存在するか検証する
func (m *JWTMiddleware) validateRequiredClaims(claims jwt.MapClaims) error {
	for _, requiredClaim := range m.config.RequiredClaims {
		if _, ok := claims[requiredClaim]; !ok {
			return errors.NewUnauthorizedError(fmt.Sprintf("missing required claim: %s", requiredClaim))
		}
	}
	return nil
}

// parsePublicKeyFromPEM はPEM形式の文字列からRSA公開鍵をパースする
func parsePublicKeyFromPEM(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// GetClaimsFromContext はコンテキストからJWTクレームを取得する
func GetClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(jwt.MapClaims)
	return claims, ok
}
