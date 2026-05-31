package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/config"
)

func TestParseServerFlagsFromEnv(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_DSN", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")

	cfg := config.ParseServerFlags()
	require.Equal(t, ":9090", cfg.RunAddr)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "postgres://localhost/test", cfg.DatabaseDSN)
	require.Equal(t, "secret", cfg.JWTSecret)
	require.Equal(t, 24*time.Hour, cfg.JWTTokenTTL)

	_ = os.Unsetenv("SERVER_ADDRESS")
}
