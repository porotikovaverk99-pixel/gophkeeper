package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/logger"
)

func TestNewAndRequestLogger(t *testing.T) {
	zlog, err := logger.New("info")
	require.NoError(t, err)

	called := false
	handler := logger.RequestLogger(zlog, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))
	require.True(t, called)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestNewInvalidLevel(t *testing.T) {
	_, err := logger.New("not-a-level")
	require.Error(t, err)
}

func TestRequestLoggerWithNopLogger(t *testing.T) {
	handler := logger.RequestLogger(zap.NewNop(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.Equal(t, http.StatusNoContent, rec.Code)
}
