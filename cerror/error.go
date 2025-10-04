package cerror

import (
	"encoding/json"
	"main/shared"
	"net/http"
	"time"
)

func ThrowError(w http.ResponseWriter, message string, status int) bool {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(shared.APIError{
		Message:   message,
		Status:    status,
		Timestamp: time.Now().UTC(),
	})

	errorLogger(message, status)

	// if err != nil {
	// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// }

	return true
}
