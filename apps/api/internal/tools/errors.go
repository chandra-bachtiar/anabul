package tools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// Kode error dipakai REST API (HTTP status) dan agent (balasan terstruktur).
const (
	CodeInvalid      = "invalid"
	CodeNotFound     = "not_found"
	CodeLocked       = "locked"
	CodeConflict     = "conflict"
	CodeAmbiguous    = "ambiguous"
	CodeNeedsConfirm = "needs_confirmation"
)

// Error adalah error domain dengan pesan yang aman ditampilkan ke pengguna.
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

// WithAvailable melengkapi error dengan daftar pilihan yang valid.
func (e *Error) WithAvailable(items ...string) *Error {
	e.Available = items
	return e
}

// AsError mengambil *Error dari sebuah error (nil jika bukan error domain).
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// NewInvalidError dan NewNotFoundError dipakai paket lain (agent) supaya pesan
// validasi tetap seragam dengan layer tools.
func NewInvalidError(format string, args ...any) *Error {
	return invalid(format, args...)
}

func NewNotFoundError(format string, args ...any) *Error {
	return notFound(format, args...)
}

// NewConflictError menandai aksi yang bentrok dengan state sekarang.
func NewConflictError(format string, args ...any) *Error {
	return &Error{Code: CodeConflict, Message: fmt.Sprintf(format, args...)}
}

// HTTPStatus memetakan kode error ke status HTTP.
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

func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

// isUniqueViolation menandai pelanggaran constraint UNIQUE di PostgreSQL.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
