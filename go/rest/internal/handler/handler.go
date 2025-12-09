package handler

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/kaitoimai/go-sample/rest/internal/oas"
	logx "github.com/kaitoimai/go-sample/rest/internal/pkg/logger"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
)

// OASHandler implements the oas.Handler interface
type OASHandler struct{}

// NewOASHandler creates a new OAS handler
func NewOASHandler() *OASHandler { return &OASHandler{} }

// GetRoot implements oas.Handler
func (h *OASHandler) GetRoot(ctx context.Context) (oas.GetRootOK, error) {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("Hello from ogen! (current time: %s)", currentTime)
	return oas.GetRootOK{
		Data: bytes.NewReader([]byte(msg)),
	}, nil
}

// GetHealth implements oas.Handler
func (h *OASHandler) GetHealth(ctx context.Context) (oas.GetHealthOK, error) {
	return oas.GetHealthOK{
		Data: bytes.NewReader([]byte("OK")),
	}, nil
}

// V1GetHello implements oas.Handler
func (h *OASHandler) V1GetHello(ctx context.Context, params oas.V1GetHelloParams) (oas.V1GetHelloRes, error) {
	name := "World"
	if params.Name.IsSet() {
		name = params.Name.Value
		// Example validation: reject if name is "error"
		if name == "error" {
			return nil, myerrors.NewInvalidArgument("名前に'error'は使用できません")
		}
		// Example server error: reject if name is "panic"
		if name == "panic" {
			return nil, myerrors.NewSystemError(
				"サーバーエラーが発生しました",
				"panic triggered by test input",
				nil,
			)
		}
	}

	response := &oas.HelloResponse{
		Message:   fmt.Sprintf("Hello, %s!", name),
		Timestamp: time.Now(),
	}

	logx.FromContext(ctx).Info("v1GetHello called", "name", name, "timestamp", response.Timestamp)

	return response, nil
}
