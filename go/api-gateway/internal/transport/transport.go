package transport

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"api-gateway/internal/errors"
)

// Transporter はバックエンドへのHTTPリクエスト転送を行うインターフェース
type Transporter interface {
	// Transport はリクエストをバックエンドに転送する
	Transport(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *Backend) error
}

// Backend はバックエンドサービスの情報
type Backend struct {
	// URL はバックエンドのURL
	URL *url.URL

	// Timeout はリクエストのタイムアウト
	Timeout time.Duration

	// Headers はバックエンドに追加するヘッダー
	Headers map[string]string
}

// HTTPTransporter は標準的なHTTPリバースプロキシによる転送を行う
type HTTPTransporter struct {
	// ErrorHandler はプロキシエラー時のハンドラ
	ErrorHandler func(w http.ResponseWriter, req *http.Request, err error)
}

// NewHTTPTransporter は新しいHTTPTransporterを作成する
func NewHTTPTransporter() *HTTPTransporter {
	return &HTTPTransporter{
		ErrorHandler: defaultErrorHandler,
	}
}

// Transport はリクエストをバックエンドに転送する
func (t *HTTPTransporter) Transport(ctx context.Context, w http.ResponseWriter, req *http.Request, backend *Backend) error {
	if backend == nil || backend.URL == nil {
		return errors.NewBadGatewayError("invalid backend configuration")
	}

	// タイムアウト設定
	if backend.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, backend.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// リクエストURLをバックエンドURLに変更
	originalURL := req.URL
	req.URL = &url.URL{
		Scheme:   backend.URL.Scheme,
		Host:     backend.URL.Host,
		Path:     backend.URL.Path + originalURL.Path,
		RawQuery: originalURL.RawQuery,
	}
	req.Host = backend.URL.Host

	// カスタムヘッダーを追加
	for key, value := range backend.Headers {
		req.Header.Set(key, value)
	}

	// リバースプロキシで転送
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			// Director内では何もしない（事前にreqを設定済み）
		},
		ErrorHandler: t.ErrorHandler,
	}

	proxy.ServeHTTP(w, req)

	return nil
}

// defaultErrorHandler はデフォルトのエラーハンドラ
func defaultErrorHandler(w http.ResponseWriter, req *http.Request, err error) {
	gatewayErr := errors.NewBadGatewayError(err.Error())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(gatewayErr.StatusCode())
	w.Write(errors.ToJSON(gatewayErr))
}

// NewBackend は新しいBackendを作成する
func NewBackend(urlStr string, timeout time.Duration) (*Backend, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &Backend{
		URL:     u,
		Timeout: timeout,
		Headers: make(map[string]string),
	}, nil
}

// AddHeader はバックエンドに送信するヘッダーを追加する
func (b *Backend) AddHeader(key, value string) {
	if b.Headers == nil {
		b.Headers = make(map[string]string)
	}
	b.Headers[key] = value
}
