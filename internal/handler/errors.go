package handler

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// JSONError отправляет JSON-ответ с описанием ошибки.
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func (h *KeeperHandler) respondInternalError(w http.ResponseWriter, err error) {
	h.log.Error("internal server error", zap.Error(err))
	JSONError(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
