package http

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"anabul/internal/tools"
)

const maxBodyBytes = 1 << 20

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to write the json response", "error", err)
	}
}

// writeError maps a domain error onto an HTTP status and a JSON body.
func writeError(w http.ResponseWriter, err error) {
	if domainErr := tools.AsError(err); domainErr != nil {
		writeJSON(w, tools.HTTPStatus(err), domainErr)
		return
	}
	slog.Error("internal error", "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "ada gangguan di server, coba lagi ya"})
}

// decodeJSON caps the body size and rejects unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body JSON tidak valid: " + err.Error()})
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body JSON hanya boleh satu objek"})
		return false
	}
	return true
}

func pathInt(r *http.Request, name string) (int64, bool) {
	raw := r.PathValue(name)
	var value int64
	if raw == "" {
		return 0, false
	}
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		value = value*10 + int64(ch-'0')
	}
	return value, true
}
