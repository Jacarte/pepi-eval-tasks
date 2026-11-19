package api

import (
	"encoding/json"
	"net/http"

	"payments/internal/apperr"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError sends {"error":{"code","message"}}.
func writeError(w http.ResponseWriter, status int, code apperr.Code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": string(code), "message": message}})
}

// decodeBody reads a JSON body into v and reports a malformed one to the client.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, apperr.ErrMalformedBody, "request body is not valid JSON")
		return false
	}
	return true
}
