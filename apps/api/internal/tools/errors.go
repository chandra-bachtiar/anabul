package tools

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

// Error codes are shared by the REST API (HTTP status) and the agent (structured replies).
const (
	CodeInvalid      = "invalid"
	CodeNotFound     = "not_found"
	CodeLocked       = "locked"
	CodeConflict     = "conflict"
	CodeAmbiguous    = "ambiguous"
	CodeNeedsConfirm = "needs_confirmation"
)

// Error carries a message that is safe to show the user.
type Error struct {
	Code      string   `json:"code"`
	Message   string   `json:"error"`
	Available []string `json:"available,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func invalid(format string, args ...any) *Error {
	return &Error{Code: CodeInvalid, Message: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...any) *Error {
	return &Error{Code: CodeNotFound, Message: fmt.Sprintf(format, args...)}
}

func locked(month string) *Error {
	return &Error{
		Code:    CodeLocked,
		Message: fmt.Sprintf("bulan %s sudah ditutup, buka kembali dulu kalau mau diubah", month),
	}
}

// WithAvailable lists the valid options so the user can pick one.
func (e *Error) WithAvailable(items ...string) *Error {
	e.Available = items
	return e
}

// AsError returns the domain error, or nil when err is not one.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// NewInvalidError and NewNotFoundError keep validation errors from other
// packages (agent) consistent with this layer.
func NewInvalidError(format string, args ...any) *Error {
	return invalid(format, args...)
}

func NewNotFoundError(format string, args ...any) *Error {
	return notFound(format, args...)
}

func NewConflictError(format string, args ...any) *Error {
	return &Error{Code: CodeConflict, Message: fmt.Sprintf(format, args...)}
}

// HTTPStatus maps a domain error code to an HTTP status.
func HTTPStatus(err error) int {
	e := AsError(err)
	if e == nil {
		return 500
	}
	switch e.Code {
	case CodeInvalid:
		return 400
	case CodeNotFound:
		return 404
	case CodeLocked, CodeConflict:
		return 409
	case CodeAmbiguous, CodeNeedsConfirm:
		return 409
	default:
		return 400
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
