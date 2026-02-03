package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

    "github.com/Fipaan/gosp/lexer"
)

const maxBodyBytes = 1 << 20 // 1MB

type APIError struct {
	Loc     *lexer.Location `json:"loc,omitempty"`
	Message  string    `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteAPIError(w http.ResponseWriter, status int, loc *lexer.Location,
                   msg_fmt string, msg_args ...any) {
    WriteJSON(w, status, APIError{
		Loc:     loc,
		Message: fmt.Sprintf(msg_fmt, msg_args...),
	})
}

func ReadJSONBody(w http.ResponseWriter, r *http.Request, cont any) bool {
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "application/json") {
		WriteAPIError(w, http.StatusUnsupportedMediaType, nil,
                      "Content-Type must be application/json")
		return false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(cont); err != nil {
		WriteAPIError(w, http.StatusBadRequest, nil,
                      "Invalid JSON body: %s", err.Error())
		return false
	}
	if dec.More() {
		WriteAPIError(w, http.StatusBadRequest, nil, 
                      "Invalid JSON body: unexpected trailing data")
		return false
	}
	return true
}
