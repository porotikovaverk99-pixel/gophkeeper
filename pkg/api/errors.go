package api

import "fmt"

// APIError описывает ошибку HTTP-ответа сервера GophKeeper.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}
