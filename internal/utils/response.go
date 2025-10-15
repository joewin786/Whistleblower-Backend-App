package utils

import (
	"encoding/json"
	"net/http"
)

// RespondWithJSON writes a JSON response with a given status code and payload.
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	// Marshal the payload into a JSON string.
	response, err := json.Marshal(payload)
	if err != nil {
		// If marshalling fails, send a generic server error.
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	// Set the content type header and write the response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// RespondWithError writes a standardized JSON error message.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	// Use the generic JSON responder to send a structured error.
	RespondWithJSON(w, code, map[string]string{"error": message})
}
