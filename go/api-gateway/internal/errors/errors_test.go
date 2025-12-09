package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorCode  string
		message    string
	}{
		{
			name:       "create 400 error",
			statusCode: http.StatusBadRequest,
			errorCode:  "BAD_REQUEST",
			message:    "invalid request",
		},
		{
			name:       "create 500 error",
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			message:    "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.statusCode, tt.errorCode, tt.message)

			if err.StatusCode() != tt.statusCode {
				t.Errorf("StatusCode() = %d, want %d", err.StatusCode(), tt.statusCode)
			}

			if err.ErrorCode() != tt.errorCode {
				t.Errorf("ErrorCode() = %s, want %s", err.ErrorCode(), tt.errorCode)
			}

			if err.Error() != tt.message {
				t.Errorf("Error() = %s, want %s", err.Error(), tt.message)
			}

			if err.Details() != nil {
				t.Errorf("Details() = %v, want nil", err.Details())
			}
		})
	}
}

func TestNewErrorWithDetails(t *testing.T) {
	details := map[string]any{
		"field": "email",
		"issue": "invalid format",
	}

	err := NewErrorWithDetails(
		http.StatusBadRequest,
		"VALIDATION_ERROR",
		"validation failed",
		details,
	)

	if err.StatusCode() != http.StatusBadRequest {
		t.Errorf("StatusCode() = %d, want %d", err.StatusCode(), http.StatusBadRequest)
	}

	if err.ErrorCode() != "VALIDATION_ERROR" {
		t.Errorf("ErrorCode() = %s, want VALIDATION_ERROR", err.ErrorCode())
	}

	gotDetails := err.Details()
	if gotDetails == nil {
		t.Fatal("Details() = nil, want non-nil")
	}

	if gotDetails["field"] != "email" {
		t.Errorf("Details()[field] = %v, want email", gotDetails["field"])
	}

	if gotDetails["issue"] != "invalid format" {
		t.Errorf("Details()[issue] = %v, want invalid format", gotDetails["issue"])
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name       string
		createErr  func(string) GatewayError
		wantStatus int
		wantCode   string
	}{
		{
			name:       "BadRequestError",
			createErr:  NewBadRequestError,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "UnauthorizedError",
			createErr:  NewUnauthorizedError,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "UNAUTHORIZED",
		},
		{
			name:       "ForbiddenError",
			createErr:  NewForbiddenError,
			wantStatus: http.StatusForbidden,
			wantCode:   "FORBIDDEN",
		},
		{
			name:       "NotFoundError",
			createErr:  NewNotFoundError,
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "InternalServerError",
			createErr:  NewInternalServerError,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_SERVER_ERROR",
		},
		{
			name:       "BadGatewayError",
			createErr:  NewBadGatewayError,
			wantStatus: http.StatusBadGateway,
			wantCode:   "BAD_GATEWAY",
		},
		{
			name:       "GatewayTimeoutError",
			createErr:  NewGatewayTimeoutError,
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   "GATEWAY_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := "test message"
			err := tt.createErr(message)

			if err.StatusCode() != tt.wantStatus {
				t.Errorf("StatusCode() = %d, want %d", err.StatusCode(), tt.wantStatus)
			}

			if err.ErrorCode() != tt.wantCode {
				t.Errorf("ErrorCode() = %s, want %s", err.ErrorCode(), tt.wantCode)
			}

			if err.Error() != message {
				t.Errorf("Error() = %s, want %s", err.Error(), message)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		name    string
		err     GatewayError
		wantErr bool
	}{
		{
			name: "simple error",
			err:  NewBadRequestError("invalid input"),
		},
		{
			name: "error with details",
			err: NewErrorWithDetails(
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				"validation failed",
				map[string]any{
					"field": "email",
					"value": "invalid",
				},
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := ToJSON(tt.err)

			if len(jsonData) == 0 {
				t.Fatal("ToJSON() returned empty data")
			}

			var response ErrorResponse
			if err := json.Unmarshal(jsonData, &response); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			if response.Error.Code != tt.err.ErrorCode() {
				t.Errorf("JSON code = %s, want %s", response.Error.Code, tt.err.ErrorCode())
			}

			if response.Error.Message != tt.err.Error() {
				t.Errorf("JSON message = %s, want %s", response.Error.Message, tt.err.Error())
			}

			if tt.err.Details() != nil {
				if response.Error.Details == nil {
					t.Error("JSON details = nil, want non-nil")
				}
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		errorCode  string
		wantNil    bool
	}{
		{
			name:       "wrap standard error",
			err:        errors.New("standard error"),
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			wantNil:    false,
		},
		{
			name:       "wrap nil error",
			err:        nil,
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			wantNil:    true,
		},
		{
			name:       "wrap GatewayError returns same",
			err:        NewBadRequestError("bad request"),
			statusCode: http.StatusInternalServerError,
			errorCode:  "INTERNAL_ERROR",
			wantNil:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WrapError(tt.err, tt.statusCode, tt.errorCode)

			if tt.wantNil {
				if got != nil {
					t.Errorf("WrapError() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("WrapError() = nil, want non-nil")
			}

			// GatewayErrorをラップした場合、元のエラーがそのまま返される
			if ge, ok := tt.err.(GatewayError); ok {
				if got.ErrorCode() != ge.ErrorCode() {
					t.Errorf("wrapped GatewayError code = %s, want %s", got.ErrorCode(), ge.ErrorCode())
				}
			} else {
				if got.StatusCode() != tt.statusCode {
					t.Errorf("StatusCode() = %d, want %d", got.StatusCode(), tt.statusCode)
				}
				if got.ErrorCode() != tt.errorCode {
					t.Errorf("ErrorCode() = %s, want %s", got.ErrorCode(), tt.errorCode)
				}
			}
		})
	}
}

func TestIsGatewayError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "GatewayError returns true",
			err:  NewBadRequestError("bad request"),
			want: true,
		},
		{
			name: "standard error returns false",
			err:  errors.New("standard error"),
			want: false,
		},
		{
			name: "nil returns false",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGatewayError(tt.err)
			if got != tt.want {
				t.Errorf("IsGatewayError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		context string
		wantNil bool
		wantMsg string
	}{
		{
			name:    "add context to error",
			err:     errors.New("original error"),
			context: "operation failed",
			wantNil: false,
			wantMsg: "operation failed: original error",
		},
		{
			name:    "nil error returns nil",
			err:     nil,
			context: "operation failed",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithContext(tt.err, tt.context)

			if tt.wantNil {
				if got != nil {
					t.Errorf("WithContext() = %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("WithContext() = nil, want non-nil")
			}

			if got.Error() != tt.wantMsg {
				t.Errorf("WithContext() message = %s, want %s", got.Error(), tt.wantMsg)
			}
		})
	}
}
