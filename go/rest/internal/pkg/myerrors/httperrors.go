package myerrors

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
)

// DefaultMessages contains standard error messages for each HTTP status code.
// These messages are consistent with OpenAPI specification examples.
var DefaultMessages = map[int]string{
	http.StatusBadRequest:          "入力内容に誤りがあります",
	http.StatusUnauthorized:        "認証が必要です",
	http.StatusForbidden:           "アクセスが許可されていません。再ログインしてください",
	http.StatusNotFound:            "リソースが見つかりません",
	http.StatusConflict:            "リクエストが競合しています",
	http.StatusUnprocessableEntity: "処理できないリクエストです",
	http.StatusInternalServerError: "サーバーエラーが発生しました",
}

// ValidationErrorCode represents a validation error code for mapping to user messages
type ValidationErrorCode string

const (
	// Query parameter validation errors
	ValidationNameRequired      ValidationErrorCode = "name.required"
	ValidationNameTooShort      ValidationErrorCode = "name.too_short"
	ValidationNameTooLong       ValidationErrorCode = "name.too_long"
	ValidationNameInvalidFormat ValidationErrorCode = "name.invalid_format"

	// Body validation errors
	ValidationBodyRequired      ValidationErrorCode = "body.required"
	ValidationBodyInvalidFormat ValidationErrorCode = "body.invalid_format"

	// Generic validation errors
	ValidationParameterRequired ValidationErrorCode = "parameter.required"
	ValidationParameterInvalid  ValidationErrorCode = "parameter.invalid"
	ValidationUnknown           ValidationErrorCode = "validation.unknown"
)

// ValidationMessages maps validation error codes to user-friendly messages
var ValidationMessages = map[ValidationErrorCode]string{
	ValidationNameRequired:      "名前を入力してください",
	ValidationNameTooShort:      "名前は1文字以上で入力してください",
	ValidationNameTooLong:       "名前は100文字以内で入力してください",
	ValidationNameInvalidFormat: "名前の形式が正しくありません",

	ValidationBodyRequired:      "リクエストボディを入力してください",
	ValidationBodyInvalidFormat: "リクエストボディの形式が正しくありません",

	ValidationParameterRequired: "必須パラメータが不足しています",
	ValidationParameterInvalid:  "パラメータの形式が正しくありません",
	ValidationUnknown:           "入力内容に誤りがあります",
}

// GetValidationMessage returns the user-friendly message for a validation error code
func GetValidationMessage(code ValidationErrorCode) string {
	if message, ok := ValidationMessages[code]; ok {
		return message
	}
	return ValidationMessages[ValidationUnknown]
}

// GetDefaultMessage returns the default error message for a given HTTP status code
func GetDefaultMessage(statusCode int) string {
	if message, ok := DefaultMessages[statusCode]; ok {
		return message
	}
	return "エラーが発生しました"
}

// baseHTTPError provides common implementation for HTTP errors
type baseHTTPError struct {
	userMessage string
}

func (e *baseHTTPError) Error() string { return e.userMessage }

// no Unwrap: we do not chain causes in custom errors to avoid leaking internals

// InvalidArgumentError represents a 400 Bad Request error
type InvalidArgumentError struct {
	baseHTTPError
	validationCode ValidationErrorCode
	rawMessage     string // ogen生メッセージ（ログ専用）
}

// NewInvalidArgument creates a new InvalidArgumentError
func NewInvalidArgument(userMessage string, details ...any) error {
	var cause error
	if len(details) > 0 {
		if err, ok := details[len(details)-1].(error); ok {
			cause = err
			details = details[:len(details)-1]
		}
	}

	if len(details) > 0 {
		userMessage = fmt.Sprintf(userMessage, details...)
	}

	err := &InvalidArgumentError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	_ = cause // cause is intentionally ignored for client responses
	// 返却するエラーは必ず自分の型を保ったままスタックを付与する
	return errors.WithStack(err)
}

// NewInvalidArgumentWithCode creates a new InvalidArgumentError with validation code
func NewInvalidArgumentWithCode(code ValidationErrorCode, rawMessage string) error {
	err := &InvalidArgumentError{
		baseHTTPError: baseHTTPError{
			userMessage: GetValidationMessage(code),
		},
		validationCode: code,
		rawMessage:     rawMessage,
	}
	return errors.WithStack(err)
}

// UserMessage returns the client-facing message
func (e *InvalidArgumentError) UserMessage() string {
	if e == nil {
		return ""
	}
	return e.userMessage
}

// ValidationCode returns the validation error code
func (e *InvalidArgumentError) ValidationCode() ValidationErrorCode {
	return e.validationCode
}

// RawMessage returns the raw ogen error message for logging
func (e *InvalidArgumentError) RawMessage() string {
	return e.rawMessage
}

// UnauthorizedError represents a 401 Unauthorized error
type UnauthorizedError struct {
	baseHTTPError
}

// NewUnauthorized creates a new UnauthorizedError
func NewUnauthorized(userMessage string) error {
	err := &UnauthorizedError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	return errors.WithStack(err)
}

// ForbiddenError represents a 403 Forbidden error
type ForbiddenError struct {
	baseHTTPError
}

// NewForbidden creates a new ForbiddenError
func NewForbidden(userMessage string) error {
	err := &ForbiddenError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	return errors.WithStack(err)
}

// NotFoundError represents a 404 Not Found error
type NotFoundError struct {
	baseHTTPError
}

// NewNotFound creates a new NotFoundError
func NewNotFound(resource string, id any) error {
	userMessage := fmt.Sprintf("%s not found: %v", resource, id)
	err := &NotFoundError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	return errors.WithStack(err)
}

// ConflictError represents a 409 Conflict error
type ConflictError struct {
	baseHTTPError
}

// NewConflict creates a new ConflictError
func NewConflict(userMessage string) error {
	err := &ConflictError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	return errors.WithStack(err)
}

// UnprocessableEntityError represents a 422 Unprocessable Entity error
type UnprocessableEntityError struct {
	baseHTTPError
}

// NewUnprocessableEntity creates a new UnprocessableEntityError
func NewUnprocessableEntity(userMessage string, cause error) error {
	err := &UnprocessableEntityError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
	}
	_ = cause // cause is intentionally ignored for client responses
	return errors.WithStack(err)
}

// SystemError represents a 500 Internal Server Error
type SystemError struct {
	baseHTTPError
	detailMessage string
}

// NewSystemError creates a new SystemError with separate user and detail messages
func NewSystemError(userMessage string, detailMessage string, cause error) error {
	err := &SystemError{
		baseHTTPError: baseHTTPError{
			userMessage: userMessage,
		},
		detailMessage: detailMessage,
	}
	_ = cause // cause is intentionally ignored for client responses
	return errors.WithStack(err)
}

func (e *SystemError) DetailMessage() string {
	return e.detailMessage
}
