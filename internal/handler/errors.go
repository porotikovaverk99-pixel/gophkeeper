package handler

import (
	"encoding/json"
	"net/http"
)

// JSONError отправляет JSON-ответ с описанием ошибки.
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
