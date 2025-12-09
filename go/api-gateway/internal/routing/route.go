package routing

import (
	"net/url"
	"time"

	"api-gateway/internal/config"
)

// Route はルーティング情報を保持する
type Route struct {
	Path       string
	Methods    []string
	Backend    *Backend
	Middleware []config.MiddlewareConfig
	Priority   int
}

// Backend はバックエンドサービスの情報
type Backend struct {
	URL     *url.URL
	Timeout time.Duration
}

// MatchResult はルーティングマッチの結果
type MatchResult struct {
	Route  *Route
	Params map[string]string // パスパラメータ（:id など）
}

// NewRoute は新しいRouteを作成する
func NewRoute(cfg config.Route) (*Route, error) {
	backendURL, err := url.Parse(cfg.Backend.URL)
	if err != nil {
		return nil, err
	}

	return &Route{
		Path:    cfg.Path,
		Methods: cfg.Methods,
		Backend: &Backend{
			URL:     backendURL,
			Timeout: cfg.Backend.Timeout,
		},
		Middleware: cfg.Middleware,
		Priority:   cfg.Priority,
	}, nil
}

// HasMethod はRouteが指定されたHTTPメソッドをサポートしているか確認する
func (r *Route) HasMethod(method string) bool {
	if len(r.Methods) == 0 {
		return true // メソッド指定がない場合は全メソッドを許可
	}

	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}
