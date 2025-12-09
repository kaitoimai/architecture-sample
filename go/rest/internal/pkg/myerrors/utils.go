package myerrors

import (
	"net/http"

	"github.com/cockroachdb/errors"
)

// ToHTTPStatus converts an error to an appropriate HTTP status code
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for specific error types
	var (
		invalidArg    *InvalidArgumentError
		unauthorized  *UnauthorizedError
		forbidden     *ForbiddenError
		notFound      *NotFoundError
		conflict      *ConflictError
		unprocessable *UnprocessableEntityError
		system        *SystemError
	)

	switch {
	case errors.As(err, &invalidArg):
		return http.StatusBadRequest
	case errors.As(err, &unauthorized):
		return http.StatusUnauthorized
	case errors.As(err, &forbidden):
		return http.StatusForbidden
	case errors.As(err, &notFound):
		return http.StatusNotFound
	case errors.As(err, &conflict):
		return http.StatusConflict
	case errors.As(err, &unprocessable):
		return http.StatusUnprocessableEntity
	case errors.As(err, &system):
		return http.StatusInternalServerError
	default:
		// Default to 500 for unknown errors
		return http.StatusInternalServerError
	}
}

// GetUserMessage extracts the user-friendly message from an error
func GetUserMessage(err error) string {
	if err == nil {
		return ""
	}
	// Known custom error types expose userMessage via struct field
	var (
		invalidArg    *InvalidArgumentError
		unauthorized  *UnauthorizedError
		forbidden     *ForbiddenError
		notFound      *NotFoundError
		conflict      *ConflictError
		unprocessable *UnprocessableEntityError
		system        *SystemError
	)
	switch {
	case errors.As(err, &invalidArg):
		return invalidArg.userMessage
	case errors.As(err, &unauthorized):
		return unauthorized.userMessage
	case errors.As(err, &forbidden):
		return forbidden.userMessage
	case errors.As(err, &notFound):
		return notFound.userMessage
	case errors.As(err, &conflict):
		return conflict.userMessage
	case errors.As(err, &unprocessable):
		return unprocessable.userMessage
	case errors.As(err, &system):
		return system.userMessage
	default:
		// Fallback generic message
		return "An unexpected error occurred"
	}
}

// GetDetailMessage extracts the detail message for logging
func GetDetailMessage(err error) string {
	if err == nil {
		return ""
	}

	var sysErr *SystemError
	if errors.As(err, &sysErr) {
		return sysErr.DetailMessage()
	}

	return err.Error()
}

// FlattenErrors flattens joined errors for easier handling
func FlattenErrors(err error) []error {
	if err == nil {
		return nil
	}

	// Check if it's a joined error
	if joinedErr, ok := err.(interface{ Unwrap() []error }); ok {
		errs := joinedErr.Unwrap()
		result := make([]error, 0, len(errs))
		for _, e := range errs {
			result = append(result, FlattenErrors(e)...)
		}
		return result
	}

	return []error{err}
}
