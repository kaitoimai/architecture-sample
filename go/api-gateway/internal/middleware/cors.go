package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig はCORSミドルウェアの設定
type CORSConfig struct {
	// AllowedOrigins は許可するオリジンのリスト
	// "*" を指定すると全てのオリジンを許可する
	AllowedOrigins []string

	// AllowedMethods は許可するHTTPメソッドのリスト
	AllowedMethods []string

	// AllowedHeaders は許可するHTTPヘッダーのリスト
	AllowedHeaders []string

	// ExposedHeaders はブラウザに公開するレスポンスヘッダーのリスト
	ExposedHeaders []string

	// AllowCredentials はクレデンシャル（Cookie等）の送信を許可するか
	AllowCredentials bool

	// MaxAge はプリフライトリクエストの結果をキャッシュする秒数
	MaxAge int
}

// CORSMiddleware はCORSヘッダーを設定するミドルウェア
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware は新しいCORSミドルウェアを作成する
func NewCORSMiddleware(config CORSConfig) *CORSMiddleware {
	// デフォルト値の設定
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Accept", "Content-Type", "Authorization"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 3600
	}

	return &CORSMiddleware{
		config: config,
	}
}

// Process はCORSヘッダーを設定する
// このミドルウェアはリクエストを検証するだけで、実際のヘッダー設定はハンドラ側で行う必要がある
func (m *CORSMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	// Originヘッダーを取得
	origin := req.Header.Get("Origin")
	if origin == "" {
		// Originヘッダーがない場合は通常のリクエストとして処理
		return ctx, nil
	}

	// オリジンの検証
	if !m.isOriginAllowed(origin) {
		// 許可されていないオリジンの場合でもエラーにはせず、
		// CORSヘッダーを設定しないことで拒否する
		return ctx, nil
	}

	// CORSヘッダー情報をコンテキストに保存
	ctx = m.setCORSContext(ctx, origin)

	return ctx, nil
}

// isOriginAllowed はオリジンが許可されているか確認する
func (m *CORSMiddleware) isOriginAllowed(origin string) bool {
	// ワイルドカードの場合は全て許可
	if len(m.config.AllowedOrigins) == 1 && m.config.AllowedOrigins[0] == "*" {
		return true
	}

	// 許可されたオリジンのリストをチェック
	for _, allowedOrigin := range m.config.AllowedOrigins {
		if allowedOrigin == origin {
			return true
		}
	}

	return false
}

// setCORSContext はCORS情報をコンテキストに保存する
func (m *CORSMiddleware) setCORSContext(ctx context.Context, origin string) context.Context {
	corsHeaders := make(map[string]string)

	// Access-Control-Allow-Origin
	if len(m.config.AllowedOrigins) == 1 && m.config.AllowedOrigins[0] == "*" {
		corsHeaders["Access-Control-Allow-Origin"] = "*"
	} else {
		corsHeaders["Access-Control-Allow-Origin"] = origin
	}

	// Access-Control-Allow-Methods
	if len(m.config.AllowedMethods) > 0 {
		corsHeaders["Access-Control-Allow-Methods"] = strings.Join(m.config.AllowedMethods, ", ")
	}

	// Access-Control-Allow-Headers
	if len(m.config.AllowedHeaders) > 0 {
		corsHeaders["Access-Control-Allow-Headers"] = strings.Join(m.config.AllowedHeaders, ", ")
	}

	// Access-Control-Expose-Headers
	if len(m.config.ExposedHeaders) > 0 {
		corsHeaders["Access-Control-Expose-Headers"] = strings.Join(m.config.ExposedHeaders, ", ")
	}

	// Access-Control-Allow-Credentials
	if m.config.AllowCredentials {
		corsHeaders["Access-Control-Allow-Credentials"] = "true"
	}

	// Access-Control-Max-Age
	if m.config.MaxAge > 0 {
		corsHeaders["Access-Control-Max-Age"] = strconv.Itoa(m.config.MaxAge)
	}

	// コンテキストに保存
	ctx = context.WithValue(ctx, corsHeadersKey, corsHeaders)

	return ctx
}

// contextKey はコンテキストのキー型
type corsContextKey string

const (
	// corsHeadersKey はCORSヘッダーを格納するコンテキストキー
	corsHeadersKey corsContextKey = "cors_headers"
)

// GetCORSHeaders はコンテキストからCORSヘッダーを取得する
func GetCORSHeaders(ctx context.Context) map[string]string {
	headers, ok := ctx.Value(corsHeadersKey).(map[string]string)
	if !ok {
		return nil
	}
	return headers
}
