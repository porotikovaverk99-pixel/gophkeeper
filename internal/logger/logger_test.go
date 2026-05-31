package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/logger"
)

func TestInitializeAndRequestLogger(t *testing.T) {
	require.NoError(t, logger.Initialize("info"))

	called := false
	handler := logger.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestInitializeInvalidLevel(t *testing.T) {
	err := logger.Initialize("not-a-level")
	require.Error(t, err)
}
