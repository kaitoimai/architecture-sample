package middleware

import (
	"context"
	"net/http"
)

// Middleware はHTTPリクエストを処理するミドルウェアのインターフェース
type Middleware interface {
	// Process はミドルウェアの処理を実行する
	// 処理が成功した場合は更新されたcontextとnilを返す
	// 処理が失敗した場合はエラーを返す
	Process(ctx context.Context, req *http.Request) (context.Context, error)
}

// Chain は複数のミドルウェアを順次実行するチェーン
type Chain struct {
	middlewares []Middleware
}

// NewChain は新しいミドルウェアチェーンを作成する
func NewChain(middlewares ...Middleware) *Chain {
	return &Chain{
		middlewares: middlewares,
	}
}

// Execute はチェーン内のすべてのミドルウェアを順次実行する
// いずれかのミドルウェアがエラーを返した場合、処理を中断してエラーを返す
func (c *Chain) Execute(ctx context.Context, req *http.Request) (context.Context, error) {
	for _, mw := range c.middlewares {
		var err error
		ctx, err = mw.Process(ctx, req)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// Append は既存のチェーンに新しいミドルウェアを追加する
func (c *Chain) Append(middlewares ...Middleware) {
	c.middlewares = append(c.middlewares, middlewares...)
}

// Prepend は既存のチェーンの先頭に新しいミドルウェアを追加する
func (c *Chain) Prepend(middlewares ...Middleware) {
	c.middlewares = append(middlewares, c.middlewares...)
}

// Len はチェーン内のミドルウェア数を返す
func (c *Chain) Len() int {
	return len(c.middlewares)
}
