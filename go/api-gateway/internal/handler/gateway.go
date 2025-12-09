package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"api-gateway/internal/config"
	"api-gateway/internal/errors"
	"api-gateway/internal/middleware"
	"api-gateway/internal/routing"
	"api-gateway/internal/transport"
)

// Gateway はAPI Gatewayのメインハンドラ
type Gateway struct {
	router            *routing.Router
	transporter       transport.Transporter
	middlewareFactory *middleware.Factory
	logger            *slog.Logger
}

// NewGateway は新しいGatewayを作成する
func NewGateway(router *routing.Router, transporter transport.Transporter, middlewareFactory *middleware.Factory, logger *slog.Logger) *Gateway {
	if logger == nil {
		logger = slog.Default()
	}

	return &Gateway{
		router:            router,
		transporter:       transporter,
		middlewareFactory: middlewareFactory,
		logger:            logger,
	}
}

// ServeHTTP はhttp.Handlerインターフェースの実装
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// OPTIONSリクエストの処理（CORSプリフライト）
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ルーティング解決
	matchResult, err := g.router.Match(r.Method, r.URL.Path)
	if err != nil {
		g.handleError(w, r, errors.WrapError(err, http.StatusNotFound, "ROUTING_ERROR"))
		return
	}

	g.logger.Debug("route matched",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Any("params", matchResult.Params),
	)

	// ミドルウェアチェーンの構築と実行
	ctx := r.Context()
	if len(matchResult.Route.Middleware) > 0 {
		chain, err := g.buildMiddlewareChain(matchResult.Route.Middleware)
		if err != nil {
			g.handleError(w, r, errors.WrapError(err, http.StatusInternalServerError, "MIDDLEWARE_SETUP_ERROR"))
			return
		}

		ctx, err = chain.Execute(ctx, r)
		if err != nil {
			g.handleError(w, r, errors.WrapError(err, http.StatusUnauthorized, "MIDDLEWARE_ERROR"))
			return
		}

		r = r.WithContext(ctx)
	}

	// バックエンドへの転送
	backend := g.convertToTransportBackend(matchResult.Route.Backend)
	if err := g.transporter.Transport(ctx, w, r, backend); err != nil {
		g.handleError(w, r, errors.WrapError(err, http.StatusBadGateway, "TRANSPORT_ERROR"))
		return
	}

	g.logger.Debug("request completed successfully",
		slog.String("path", r.URL.Path),
		slog.String("backend", backend.URL.String()),
	)
}

// buildMiddlewareChain はミドルウェアチェーンを構築する
func (g *Gateway) buildMiddlewareChain(configs []config.MiddlewareConfig) (*middleware.Chain, error) {
	if g.middlewareFactory == nil {
		return middleware.NewChain(), nil
	}

	middlewares := make([]middleware.Middleware, 0, len(configs))

	for _, cfg := range configs {
		m, err := g.middlewareFactory.Create(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware type=%s: %w", cfg.Type, err)
		}
		middlewares = append(middlewares, m)
	}

	return middleware.NewChain(middlewares...), nil
}

// convertToTransportBackend はrouting.Backendをtransport.Backendに変換する
func (g *Gateway) convertToTransportBackend(routingBackend *routing.Backend) *transport.Backend {
	return &transport.Backend{
		URL:     routingBackend.URL,
		Timeout: routingBackend.Timeout,
		Headers: make(map[string]string),
	}
}

// handleError はエラーレスポンスを処理する
func (g *Gateway) handleError(w http.ResponseWriter, r *http.Request, err error) {
	var gatewayErr errors.GatewayError
	if errors.IsGatewayError(err) {
		gatewayErr = err.(errors.GatewayError)
	} else {
		gatewayErr = errors.NewInternalServerError(err.Error())
	}

	g.logger.Error("request failed",
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.String("error_code", gatewayErr.ErrorCode()),
		slog.String("error", gatewayErr.Error()),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(gatewayErr.StatusCode())
	w.Write(errors.ToJSON(gatewayErr))
}
