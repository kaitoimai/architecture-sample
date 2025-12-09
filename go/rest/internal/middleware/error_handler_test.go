package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/kaitoimai/go-sample/rest/internal/pkg/logger"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
)

// TestConvertOgenError_SecurityError tests SecurityError conversion to UnauthorizedError
func TestConvertOgenError_SecurityError(t *testing.T) {
	secErr := &ogenerrors.SecurityError{
		Security: "bearer",
		Err:      fmt.Errorf("missing authorization header"),
	}

	result := ConvertOgenError(secErr)

	var unauthorized *myerrors.UnauthorizedError
	if !errors.As(result, &unauthorized) {
		t.Errorf("expected UnauthorizedError, got %T", result)
	}

	if myerrors.ToHTTPStatus(result) != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", myerrors.ToHTTPStatus(result))
	}
}

// TestConvertOgenError_DecodeParamError tests DecodeParamError conversion
func TestConvertOgenError_DecodeParamError(t *testing.T) {
	tests := []struct {
		name              string
		decodeParamErr    *ogenerrors.DecodeParamError
		expectedCode      myerrors.ValidationErrorCode
		expectedUserMsg   string
		rawMessageContain string
	}{
		{
			name: "name contains exceed",
			decodeParamErr: &ogenerrors.DecodeParamError{
				Name: "name",
				In:   "query",
				Err:  fmt.Errorf("value exceeds maximum length"),
			},
			expectedCode:      myerrors.ValidationNameTooLong,
			expectedUserMsg:   "名前は100文字以内で入力してください",
			rawMessageContain: "invalid parameter: name",
		},
		{
			name: "name contains short",
			decodeParamErr: &ogenerrors.DecodeParamError{
				Name: "name",
				In:   "query",
				Err:  fmt.Errorf("value is too short"),
			},
			expectedCode:      myerrors.ValidationNameTooShort,
			expectedUserMsg:   "名前は1文字以上で入力してください",
			rawMessageContain: "too short",
		},
		{
			name: "unknown parameter error",
			decodeParamErr: &ogenerrors.DecodeParamError{
				Name: "unknown_param",
				In:   "query",
				Err:  fmt.Errorf("some error"),
			},
			expectedCode:      myerrors.ValidationParameterInvalid,
			expectedUserMsg:   "パラメータの形式が正しくありません",
			rawMessageContain: "unknown_param",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertOgenError(tt.decodeParamErr)

			var invalidArg *myerrors.InvalidArgumentError
			if !errors.As(result, &invalidArg) {
				t.Fatalf("expected InvalidArgumentError, got %T", result)
			}

			if invalidArg.ValidationCode() != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, invalidArg.ValidationCode())
			}

			if invalidArg.UserMessage() != tt.expectedUserMsg {
				t.Errorf("expected user message %q, got %q", tt.expectedUserMsg, invalidArg.UserMessage())
			}

			rawMsg := invalidArg.RawMessage()
			if !strings.Contains(rawMsg, tt.rawMessageContain) {
				t.Errorf("expected raw message to contain %q, got %q", tt.rawMessageContain, rawMsg)
			}

			if myerrors.ToHTTPStatus(result) != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", myerrors.ToHTTPStatus(result))
			}
		})
	}
}

// TestConvertOgenError_DecodeParamsError tests DecodeParamsError conversion
func TestConvertOgenError_DecodeParamsError(t *testing.T) {
	decParamsErr := &ogenerrors.DecodeParamsError{
		OperationContext: ogenerrors.OperationContext{
			Name: "V1GetHello",
		},
		Err: fmt.Errorf("query: \"name\": string: len 101 greater than maximum 100"),
	}

	result := ConvertOgenError(decParamsErr)

	var invalidArg *myerrors.InvalidArgumentError
	if !errors.As(result, &invalidArg) {
		t.Fatalf("expected InvalidArgumentError, got %T", result)
	}

	if invalidArg.ValidationCode() != myerrors.ValidationNameTooLong {
		t.Errorf("expected code ValidationNameTooLong, got %s", invalidArg.ValidationCode())
	}

	if invalidArg.UserMessage() != "名前は100文字以内で入力してください" {
		t.Errorf("unexpected user message: %s", invalidArg.UserMessage())
	}

	rawMsg := invalidArg.RawMessage()
	if !strings.Contains(rawMsg, "V1GetHello") || !strings.Contains(rawMsg, "greater than maximum") {
		t.Errorf("expected raw message to contain operation and error details, got %q", rawMsg)
	}
}

// TestConvertOgenError_DecodeBodyError tests DecodeBodyError conversion
func TestConvertOgenError_DecodeBodyError(t *testing.T) {
	decBodyErr := &ogenerrors.DecodeBodyError{
		Err: fmt.Errorf("required field missing"),
	}

	result := ConvertOgenError(decBodyErr)

	var invalidArg *myerrors.InvalidArgumentError
	if !errors.As(result, &invalidArg) {
		t.Fatalf("expected InvalidArgumentError, got %T", result)
	}

	if invalidArg.ValidationCode() != myerrors.ValidationBodyRequired {
		t.Errorf("expected code ValidationBodyRequired, got %s", invalidArg.ValidationCode())
	}
}

// TestConvertOgenError_UnknownError tests that unknown errors are wrapped with stack
func TestConvertOgenError_UnknownError(t *testing.T) {
	originalErr := fmt.Errorf("unknown error")

	result := ConvertOgenError(originalErr)

	// Should be wrapped with stack from error_handler.go
	stackTrace := fmt.Sprintf("%+v", result)
	if !strings.Contains(stackTrace, "error_handler.go") && !strings.Contains(stackTrace, "internal/middleware/error_handler.go") {
		t.Errorf("expected error to be wrapped with stack trace from error_handler.go, got: %s", stackTrace)
	}

	// Original error message should be preserved
	if result.Error() != originalErr.Error() {
		t.Errorf("expected error message %q, got %q", originalErr.Error(), result.Error())
	}
}

// TestClassify tests error classification
func TestClassify(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedTitle  string
		expectedDetail string
		rawMessageNil  bool
	}{
		{
			name:           "InvalidArgumentError with code",
			err:            myerrors.NewInvalidArgumentWithCode(myerrors.ValidationNameTooLong, "raw: name too long"),
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "入力内容に誤りがあります",
			expectedDetail: "名前は100文字以内で入力してください",
			rawMessageNil:  false,
		},
		{
			name:           "InvalidArgumentError without code",
			err:            myerrors.NewInvalidArgument("カスタムメッセージ"),
			expectedStatus: http.StatusBadRequest,
			expectedTitle:  "入力内容に誤りがあります",
			expectedDetail: "カスタムメッセージ",
			rawMessageNil:  true,
		},
		{
			name:           "UnauthorizedError",
			err:            myerrors.NewUnauthorized("認証が必要です"),
			expectedStatus: http.StatusUnauthorized,
			expectedTitle:  "認証が必要です",
			expectedDetail: "認証が必要です",
			rawMessageNil:  true,
		},
		{
			name:           "ForbiddenError",
			err:            myerrors.NewForbidden("アクセスが許可されていません"),
			expectedStatus: http.StatusForbidden,
			expectedTitle:  "アクセスが許可されていません。再ログインしてください",
			expectedDetail: "アクセスが許可されていません",
			rawMessageNil:  true,
		},
		{
			name:           "NotFoundError",
			err:            myerrors.NewNotFound("User", 123),
			expectedStatus: http.StatusNotFound,
			expectedTitle:  "リソースが見つかりません",
			expectedDetail: "User not found: 123",
			rawMessageNil:  true,
		},
		{
			name:           "ConflictError",
			err:            myerrors.NewConflict("データが競合しています"),
			expectedStatus: http.StatusConflict,
			expectedTitle:  "リクエストが競合しています",
			expectedDetail: "データが競合しています",
			rawMessageNil:  true,
		},
		{
			name:           "UnprocessableEntityError",
			err:            myerrors.NewUnprocessableEntity("処理できません", nil),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedTitle:  "処理できないリクエストです",
			expectedDetail: "処理できません",
			rawMessageNil:  true,
		},
		{
			name:           "SystemError",
			err:            myerrors.NewSystemError("サーバーエラー", "database connection failed", nil),
			expectedStatus: http.StatusInternalServerError,
			expectedTitle:  "サーバーエラーが発生しました",
			expectedDetail: "サーバーエラー",
			rawMessageNil:  true,
		},
		{
			name:           "Unknown error",
			err:            fmt.Errorf("unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedTitle:  "サーバーエラーが発生しました",
			expectedDetail: "サーバーエラーが発生しました",
			rawMessageNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, title, detail, rawMessage := classify(tt.err)

			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}

			if title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, title)
			}

			if detail != tt.expectedDetail {
				t.Errorf("expected detail %q, got %q", tt.expectedDetail, detail)
			}

			if tt.rawMessageNil && rawMessage != "" {
				t.Errorf("expected empty raw message, got %q", rawMessage)
			} else if !tt.rawMessageNil && rawMessage == "" {
				t.Error("expected non-empty raw message")
			}
		})
	}
}

// TestBuildProblemDetails tests Problem Details construction
func TestBuildProblemDetails(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		title          string
		detail         string
		path           string
		expectedTitle  string
		expectedDetail string
	}{
		{
			name:           "with title and detail",
			status:         400,
			title:          "Bad Request",
			detail:         "Invalid input",
			path:           "/v1/hello",
			expectedTitle:  "Bad Request",
			expectedDetail: "Invalid input",
		},
		{
			name:           "empty title uses default",
			status:         400,
			title:          "",
			detail:         "Invalid input",
			path:           "/v1/hello",
			expectedTitle:  "入力内容に誤りがあります",
			expectedDetail: "Invalid input",
		},
		{
			name:           "empty detail uses title",
			status:         500,
			title:          "Server Error",
			detail:         "",
			path:           "/v1/hello",
			expectedTitle:  "Server Error",
			expectedDetail: "Server Error",
		},
		{
			name:           "both empty use default",
			status:         404,
			title:          "",
			detail:         "",
			path:           "/v1/users/123",
			expectedTitle:  "リソースが見つかりません",
			expectedDetail: "リソースが見つかりません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			pd := buildProblemDetails(req, tt.status, tt.title, tt.detail)

			if pd["type"] != "about:blank" {
				t.Errorf("expected type 'about:blank', got %v", pd["type"])
			}

			if pd["title"] != tt.expectedTitle {
				t.Errorf("expected title %q, got %v", tt.expectedTitle, pd["title"])
			}

			if pd["status"] != tt.status {
				t.Errorf("expected status %d, got %v", tt.status, pd["status"])
			}

			if pd["detail"] != tt.expectedDetail {
				t.Errorf("expected detail %q, got %v", tt.expectedDetail, pd["detail"])
			}

			if pd["instance"] != tt.path {
				t.Errorf("expected instance %q, got %v", tt.path, pd["instance"])
			}
		})
	}
}

// TestErrorHandler_InvalidArgumentError tests ErrorHandler with InvalidArgumentError
func TestErrorHandler_InvalidArgumentError(t *testing.T) {
	// Setup logger
	log := logger.New(logger.LevelWarn)
	ctx := logger.NewContext(context.Background(), log)

	// Create test request and response recorder
	req := httptest.NewRequest(http.MethodGet, "/v1/hello?name=toolongname", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Create error
	err := myerrors.NewInvalidArgumentWithCode(
		myerrors.ValidationNameTooLong,
		"invalid parameters for operation V1GetHello: query: \"name\": string: len 101 greater than maximum 100",
	)

	// Execute
	ErrorHandler(ctx, w, req, err)

	// Verify HTTP response
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/problem+json" {
		t.Errorf("expected Content-Type application/problem+json, got %s", w.Header().Get("Content-Type"))
	}

	var respPD ProblemDetails
	if err := json.NewDecoder(w.Body).Decode(&respPD); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respPD["title"] != "入力内容に誤りがあります" {
		t.Errorf("expected title '入力内容に誤りがあります', got %v", respPD["title"])
	}

	if respPD["detail"] != "名前は100文字以内で入力してください" {
		t.Errorf("expected detail '名前は100文字以内で入力してください', got %v", respPD["detail"])
	}

	// Verify response does NOT contain raw_err
	if _, exists := respPD["raw_err"]; exists {
		t.Error("response should not contain raw_err field")
	}

	// Note: We cannot verify log output easily without more sophisticated mocking
	// In a real scenario, you would use a custom slog.Handler to capture log entries
}

// TestErrorHandler_SystemError tests ErrorHandler with SystemError
func TestErrorHandler_SystemError(t *testing.T) {
	ctx := logger.NewContext(context.Background(), logger.New(logger.LevelError))

	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	err := myerrors.NewSystemError(
		"サーバーエラーが発生しました",
		"database connection failed: timeout",
		fmt.Errorf("connection timeout"),
	)

	ErrorHandler(ctx, w, req, err)

	// Verify HTTP response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var respPD ProblemDetails
	if err := json.NewDecoder(w.Body).Decode(&respPD); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respPD["title"] != "サーバーエラーが発生しました" {
		t.Errorf("expected title 'サーバーエラーが発生しました', got %v", respPD["title"])
	}

	// Verify response does NOT contain stack or raw_err
	if _, exists := respPD["stack"]; exists {
		t.Error("response should not contain stack field")
	}
	if _, exists := respPD["raw_err"]; exists {
		t.Error("response should not contain raw_err field")
	}
}

// TestErrorHandler_OgenDecodeParamError tests ErrorHandler with ogen DecodeParamError
func TestErrorHandler_OgenDecodeParamError(t *testing.T) {
	ctx := logger.NewContext(context.Background(), logger.New(logger.LevelWarn))

	req := httptest.NewRequest(http.MethodGet, "/v1/hello?name=", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	// Simulate ogen DecodeParamError (use "short" keyword to trigger ValidationNameTooShort)
	ogenErr := &ogenerrors.DecodeParamError{
		Name: "name",
		In:   "query",
		Err:  fmt.Errorf("value is too short"),
	}

	ErrorHandler(ctx, w, req, ogenErr)

	// Verify normalization to 400
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var respPD ProblemDetails
	if err := json.NewDecoder(w.Body).Decode(&respPD); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if respPD["title"] != "入力内容に誤りがあります" {
		t.Errorf("expected title '入力内容に誤りがあります', got %v", respPD["title"])
	}

	if respPD["detail"] != "名前は1文字以上で入力してください" {
		t.Errorf("expected detail '名前は1文字以上で入力してください', got %v", respPD["detail"])
	}
}

// TestErrorHandler_NilError tests ErrorHandler with nil error (early return)
func TestErrorHandler_NilError(t *testing.T) {
	ctx := context.Background()
	req := httptest.NewRequest(http.MethodGet, "/v1/hello", nil)
	w := httptest.NewRecorder()

	ErrorHandler(ctx, w, req, nil)

	// Verify no response was written
	if w.Code != http.StatusOK {
		t.Errorf("expected no status code set (200 default), got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("expected empty response body, got %d bytes", w.Body.Len())
	}
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "llo wo", true},
		{"hello world", "xyz", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s contains %s", tt.s, tt.substr), func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
