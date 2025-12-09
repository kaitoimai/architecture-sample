package routing

import (
	"fmt"
	"sort"

	"api-gateway/internal/config"
	"api-gateway/internal/errors"
)

// Router はルーティングを管理する
type Router struct {
	root *node
}

// NewRouter は新しいRouterを作成する
func NewRouter() *Router {
	return &Router{
		root: newNode(""),
	}
}

// AddRoute はルートを追加する
func (r *Router) AddRoute(route *Route) error {
	if route == nil {
		return fmt.Errorf("route is nil")
	}

	if route.Path == "" {
		return fmt.Errorf("route path is empty")
	}

	segments := SplitPath(route.Path)
	current := r.root

	// パスの各セグメントに対してノードを作成または取得
	for _, segment := range segments {
		current = current.addChild(segment)
	}

	// ルートを設定
	if current.route != nil {
		return fmt.Errorf("route already exists for path: %s", route.Path)
	}

	current.route = route
	return nil
}

// Match はパスとメソッドにマッチするルートを検索する
func (r *Router) Match(method, path string) (*MatchResult, error) {
	segments := SplitPath(path)
	params := make(map[string]string)

	route := r.findRoute(r.root, segments, params)
	if route == nil {
		return nil, errors.NewNotFoundError(fmt.Sprintf("no route found for path: %s", path))
	}

	// HTTPメソッドのチェック
	if !route.HasMethod(method) {
		return nil, errors.NewError(405, "METHOD_NOT_ALLOWED", fmt.Sprintf("method %s not allowed", method))
	}

	return &MatchResult{
		Route:  route,
		Params: params,
	}, nil
}

// findRoute は再帰的にルートを検索する
func (r *Router) findRoute(current *node, segments []string, params map[string]string) *Route {
	// すべてのセグメントを処理した場合
	if len(segments) == 0 {
		return current.route
	}

	segment := segments[0]
	remaining := segments[1:]

	// マッチする子ノードを検索
	child, found := current.findMatchingChild(segment)
	if !found {
		return nil
	}

	// パラメータノードの場合、パラメータを記録
	if child.nodeType == paramNode {
		params[child.paramName] = segment
	}

	// 再帰的に次のセグメントを処理
	return r.findRoute(child, remaining, params)
}

// LoadFromConfig は設定ファイルからルートを読み込む
func (r *Router) LoadFromConfig(cfg *config.RoutingFileConfig) error {
	if cfg == nil {
		return fmt.Errorf("routing config is nil")
	}

	// 優先度でソート（昇順）
	routes := make([]config.Route, len(cfg.Routes))
	copy(routes, cfg.Routes)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Priority < routes[j].Priority
	})

	// ルートを登録
	for _, routeCfg := range routes {
		route, err := NewRoute(routeCfg)
		if err != nil {
			return fmt.Errorf("failed to create route for %s: %w", routeCfg.Path, err)
		}

		if err := r.AddRoute(route); err != nil {
			return fmt.Errorf("failed to add route %s: %w", routeCfg.Path, err)
		}
	}

	return nil
}

// GetAllRoutes はすべてのルートを取得する（デバッグ用）
func (r *Router) GetAllRoutes() []*Route {
	var routes []*Route
	r.collectRoutes(r.root, &routes)
	return routes
}

// collectRoutes は再帰的にすべてのルートを収集する
func (r *Router) collectRoutes(current *node, routes *[]*Route) {
	if current.route != nil {
		*routes = append(*routes, current.route)
	}

	for _, child := range current.children {
		r.collectRoutes(child, routes)
	}
}
