package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GatewayError はAPI Gatewayのエラーインターフェース
type GatewayError interface {
	error
	StatusCode() int
	ErrorCode() string
	Details() map[string]any
}

// gatewayError はGatewayErrorの実装
type gatewayError struct {
	statusCode int
	errorCode  string
	message    string
	details    map[string]any
}

func (e *gatewayError) Error() string {
	return e.message
}

func (e *gatewayError) StatusCode() int {
	return e.statusCode
}

func (e *gatewayError) ErrorCode() string {
	return e.errorCode
}

func (e *gatewayError) Details() map[string]any {
	return e.details
}

// ErrorResponse はエラーレスポンスのJSON構造
type ErrorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

// ToJSON はエラーをJSON形式に変換する
func ToJSON(err GatewayError) []byte {
	resp := ErrorResponse{}
	resp.Error.Code = err.ErrorCode()
	resp.Error.Message = err.Error()
	resp.Error.Details = err.Details()

	data, _ := json.Marshal(resp)
	return data
}

// NewError はエラーを生成する
func NewError(statusCode int, errorCode, message string) GatewayError {
	return &gatewayError{
		statusCode: statusCode,
		errorCode:  errorCode,
		message:    message,
		details:    nil,
	}
}

// NewErrorWithDetails は詳細情報付きのエラーを生成する
func NewErrorWithDetails(statusCode int, errorCode, message string, details map[string]any) GatewayError {
	return &gatewayError{
		statusCode: statusCode,
		errorCode:  errorCode,
		message:    message,
		details:    details,
	}
}

// 定義済みエラー

// NewBadRequestError は400エラーを生成する
func NewBadRequestError(message string) GatewayError {
	return NewError(http.StatusBadRequest, "BAD_REQUEST", message)
}

// NewUnauthorizedError は401エラーを生成する
func NewUnauthorizedError(message string) GatewayError {
	return NewError(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// NewForbiddenError は403エラーを生成する
func NewForbiddenError(message string) GatewayError {
	return NewError(http.StatusForbidden, "FORBIDDEN", message)
}

// NewNotFoundError は404エラーを生成する
func NewNotFoundError(message string) GatewayError {
	return NewError(http.StatusNotFound, "NOT_FOUND", message)
}

// NewInternalServerError は500エラーを生成する
func NewInternalServerError(message string) GatewayError {
	return NewError(http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message)
}

// NewBadGatewayError は502エラーを生成する
func NewBadGatewayError(message string) GatewayError {
	return NewError(http.StatusBadGateway, "BAD_GATEWAY", message)
}

// NewGatewayTimeoutError は504エラーを生成する
func NewGatewayTimeoutError(message string) GatewayError {
	return NewError(http.StatusGatewayTimeout, "GATEWAY_TIMEOUT", message)
}

// WrapError は既存のエラーをGatewayErrorにラップする
func WrapError(err error, statusCode int, errorCode string) GatewayError {
	if err == nil {
		return nil
	}

	if ge, ok := err.(GatewayError); ok {
		return ge
	}

	return NewError(statusCode, errorCode, err.Error())
}

// IsGatewayError はエラーがGatewayErrorかどうかを判定する
func IsGatewayError(err error) bool {
	_, ok := err.(GatewayError)
	return ok
}

// WithContext はエラーにコンテキスト情報を追加する
func WithContext(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}
