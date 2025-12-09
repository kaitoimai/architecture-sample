package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockMiddleware はテスト用のミドルウェア
type mockMiddleware struct {
	processFunc func(ctx context.Context, req *http.Request) (context.Context, error)
}

func (m *mockMiddleware) Process(ctx context.Context, req *http.Request) (context.Context, error) {
	return m.processFunc(ctx, req)
}

func TestNewChain(t *testing.T) {
	mw1 := &mockMiddleware{}
	mw2 := &mockMiddleware{}

	chain := NewChain(mw1, mw2)

	if chain == nil {
		t.Fatal("NewChain returned nil")
	}

	if chain.Len() != 2 {
		t.Errorf("expected 2 middlewares, got %d", chain.Len())
	}
}

func TestChain_Execute_Success(t *testing.T) {
	executionOrder := []string{}

	mw1 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw1")
			return context.WithValue(ctx, "mw1", "executed"), nil
		},
	}

	mw2 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw2")
			return context.WithValue(ctx, "mw2", "executed"), nil
		},
	}

	mw3 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw3")
			return context.WithValue(ctx, "mw3", "executed"), nil
		},
	}

	chain := NewChain(mw1, mw2, mw3)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.Background()

	resultCtx, err := chain.Execute(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 実行順序の検証
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != "mw1" || executionOrder[1] != "mw2" || executionOrder[2] != "mw3" {
		t.Errorf("unexpected execution order: %v", executionOrder)
	}

	// コンテキストの検証
	if resultCtx.Value("mw1") != "executed" {
		t.Error("mw1 context value not set")
	}
	if resultCtx.Value("mw2") != "executed" {
		t.Error("mw2 context value not set")
	}
	if resultCtx.Value("mw3") != "executed" {
		t.Error("mw3 context value not set")
	}
}

func TestChain_Execute_WithError(t *testing.T) {
	expectedErr := errors.New("middleware error")
	executionOrder := []string{}

	mw1 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw1")
			return ctx, nil
		},
	}

	mw2 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw2")
			return ctx, expectedErr
		},
	}

	mw3 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw3")
			return ctx, nil
		},
	}

	chain := NewChain(mw1, mw2, mw3)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.Background()

	_, err := chain.Execute(ctx, req)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// mw2でエラーが発生したため、mw3は実行されないはず
	if len(executionOrder) != 2 {
		t.Errorf("expected 2 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != "mw1" || executionOrder[1] != "mw2" {
		t.Errorf("unexpected execution order: %v", executionOrder)
	}
}

func TestChain_Append(t *testing.T) {
	mw1 := &mockMiddleware{}
	mw2 := &mockMiddleware{}
	mw3 := &mockMiddleware{}

	chain := NewChain(mw1)
	if chain.Len() != 1 {
		t.Errorf("expected 1 middleware, got %d", chain.Len())
	}

	chain.Append(mw2, mw3)
	if chain.Len() != 3 {
		t.Errorf("expected 3 middlewares, got %d", chain.Len())
	}
}

func TestChain_Prepend(t *testing.T) {
	executionOrder := []string{}

	mw1 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw1")
			return ctx, nil
		},
	}

	mw2 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw2")
			return ctx, nil
		},
	}

	mw3 := &mockMiddleware{
		processFunc: func(ctx context.Context, req *http.Request) (context.Context, error) {
			executionOrder = append(executionOrder, "mw3")
			return ctx, nil
		},
	}

	chain := NewChain(mw3)
	chain.Prepend(mw1, mw2)

	if chain.Len() != 3 {
		t.Errorf("expected 3 middlewares, got %d", chain.Len())
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.Background()

	_, err := chain.Execute(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Prependされたミドルウェアが先に実行される
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != "mw1" || executionOrder[1] != "mw2" || executionOrder[2] != "mw3" {
		t.Errorf("unexpected execution order: %v", executionOrder)
	}
}

func TestChain_EmptyChain(t *testing.T) {
	chain := NewChain()

	if chain.Len() != 0 {
		t.Errorf("expected 0 middlewares, got %d", chain.Len())
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.Background()

	resultCtx, err := chain.Execute(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resultCtx != ctx {
		t.Error("context should be unchanged")
	}
}
