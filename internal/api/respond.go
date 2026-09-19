package api

import (
	"encoding/json"
	"net/http"
)

// errorBody is the single error shape the BFF returns (PRD § 5).
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encoding failures here are programming errors (unsupported types);
	// there is nothing useful to do for the client once headers are sent.
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Code: code, Message: message})
}
