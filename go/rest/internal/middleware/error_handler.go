package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/kaitoimai/go-sample/rest/internal/pkg/logger"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
)

// ErrorHandler handles errors from ogen handlers and converts them to appropriate HTTP responses
func ErrorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	// ogen エラーを正規化（スタック付きエラーを生成）
	err = ConvertOgenError(err)

	// 単一の分類ポイントで正規化（status, title, detail, extensions）
	statusCode, title, detail, rawMessage := classify(err)

	// Problem Details: title=要約（ユーザー向け）, detail=詳細（ユーザー向け）
	pd := buildProblemDetails(r, statusCode, title, detail)

	// ログ出力（Problem Detailsと補助情報）
	log := logger.FromContext(ctx)
	logErr := make(ProblemDetails, len(pd)+2)
	maps.Copy(logErr, pd)
	if rawMessage != "" {
		logErr["raw_err"] = rawMessage
	}
	logFields := []any{"err", logErr}
	if statusCode >= http.StatusInternalServerError {
		logFields = append(logFields, "stack", fmt.Sprintf("%+v", err))
		log.Error("", logFields...)
	} else {
		log.Warn("", logFields...)
	}

	// RFC 9457 Problem Details (application/problem+json) で応答
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)
	if encErr := json.NewEncoder(w).Encode(pd); encErr != nil {
		log.Error("failed to write error response", "err", encErr)
	}
}

// ProblemDetails represents RFC 9457 Problem Details.
type ProblemDetails map[string]any

// buildProblemDetails builds a RFC 9457 Problem Details payload.
// Standard members: type, title(要約/ユーザー向け), status, detail(詳細/ユーザー向け), instance
func buildProblemDetails(r *http.Request, status int, title string, detail string) ProblemDetails {
	if title == "" {
		title = myerrors.GetDefaultMessage(status)
	}
	if detail == "" || detail == "An unexpected error occurred" {
		detail = title
	}
	pd := ProblemDetails{
		"type":   "about:blank",
		"title":  title,
		"status": status,
		"detail": detail,
	}
	if r != nil && r.URL != nil {
		pd["instance"] = r.URL.Path
	}
	return pd
}

// classify: エラーを正規化し、HTTPステータス/ユーザー向けタイトル・詳細/拡張/生メッセージを返す
// 注: ConvertOgenErrorは呼び出し側（ErrorHandler）で事前に実行済みであること
func classify(err error) (status int, title string, detail string, rawMessage string) {
	status = myerrors.ToHTTPStatus(err)
	title = myerrors.GetDefaultMessage(status)
	detail = myerrors.GetUserMessage(err)
	if detail == "" || detail == "An unexpected error occurred" {
		detail = title
	}

	var invalidArg *myerrors.InvalidArgumentError
	if errors.As(err, &invalidArg) {
		if msg := invalidArg.UserMessage(); msg != "" {
			detail = msg
		}
		rawMessage = invalidArg.RawMessage()
	}

	return status, title, detail, rawMessage
}

// ConvertOgenError converts ogen-specific errors to myerrors types
func ConvertOgenError(err error) error {
	if err == nil {
		return nil
	}

	// Check for ogen security errors
	var secErr *ogenerrors.SecurityError
	if errors.As(err, &secErr) {
		return myerrors.NewUnauthorized("認証が必要です")
	}

	// Check for validation errors
	if errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied) {
		return myerrors.NewUnauthorized("認証が必要です")
	}

	// Parameter decoding/validation errors → 400
	var (
		decParamsErr *ogenerrors.DecodeParamsError
		decParamErr  *ogenerrors.DecodeParamError
		decBodyErr   *ogenerrors.DecodeBodyError
	)
	switch {
	case errors.As(err, &decParamsErr):
		code, rawMsg := mapOgenParamsError(decParamsErr)
		return myerrors.NewInvalidArgumentWithCode(code, rawMsg)
	case errors.As(err, &decParamErr):
		code, rawMsg := mapOgenParamError(decParamErr)
		return myerrors.NewInvalidArgumentWithCode(code, rawMsg)
	case errors.As(err, &decBodyErr):
		code, rawMsg := mapOgenBodyError(decBodyErr)
		return myerrors.NewInvalidArgumentWithCode(code, rawMsg)
	}

	// Default to wrapping with system error
	return errors.WithStack(err)
}

// mapOgenParamsError maps DecodeParamsError to validation code and raw message
func mapOgenParamsError(err *ogenerrors.DecodeParamsError) (myerrors.ValidationErrorCode, string) {
	rawMsg := fmt.Sprintf("invalid parameters for operation %s: %s", err.Name, err.Err.Error())

	// 内部エラーメッセージを解析してフィールド名とエラー内容を特定
	innerMsg := err.Err.Error()

	// "query: \"name\": ..." のパターンをチェック
	if contains(innerMsg, "query: \"name\"") {
		if contains(innerMsg, "greater than maximum") || contains(innerMsg, "maxLength") {
			return myerrors.ValidationNameTooLong, rawMsg
		}
		if contains(innerMsg, "less than minimum") || contains(innerMsg, "minLength") {
			return myerrors.ValidationNameTooShort, rawMsg
		}
		if contains(innerMsg, "required") {
			return myerrors.ValidationNameRequired, rawMsg
		}
	}

	return myerrors.ValidationParameterInvalid, rawMsg
}

// mapOgenParamError maps DecodeParamError to validation code and raw message
func mapOgenParamError(err *ogenerrors.DecodeParamError) (myerrors.ValidationErrorCode, string) {
	rawMsg := fmt.Sprintf("invalid parameter: %s (%s): %s", err.Name, err.In, err.Err.Error())

	// フィールド名とエラー内容から適切なコードを判定
	errMsg := err.Err.Error()
	paramName := err.Name

	if paramName == "name" {
		// minLength/maxLength/required判定
		if contains(errMsg, "required") {
			return myerrors.ValidationNameRequired, rawMsg
		}
		if contains(errMsg, "minLength") || contains(errMsg, "short") {
			return myerrors.ValidationNameTooShort, rawMsg
		}
		if contains(errMsg, "maxLength") || contains(errMsg, "exceed") || contains(errMsg, "long") {
			return myerrors.ValidationNameTooLong, rawMsg
		}
		if contains(errMsg, "format") {
			return myerrors.ValidationNameInvalidFormat, rawMsg
		}
	}

	// 未知のパラメータエラー
	return myerrors.ValidationParameterInvalid, rawMsg
}

// mapOgenBodyError maps DecodeBodyError to validation code and raw message
func mapOgenBodyError(err *ogenerrors.DecodeBodyError) (myerrors.ValidationErrorCode, string) {
	rawMsg := fmt.Sprintf("invalid request body: %s", err.Err.Error())
	errMsg := err.Err.Error()

	if contains(errMsg, "required") {
		return myerrors.ValidationBodyRequired, rawMsg
	}
	return myerrors.ValidationBodyInvalidFormat, rawMsg
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
