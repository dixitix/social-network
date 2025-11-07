package handlers

import "net/http"

// RespondJSONForTest exposes respondJSON to external tests.
func RespondJSONForTest(w http.ResponseWriter, status int, payload any) {
	respondJSON(w, status, payload)
}
