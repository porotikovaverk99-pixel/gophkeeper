package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseServerFlagsFromEnv(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_DSN", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "secret")

	cfg := parseServerConfig([]string{})
	require.Equal(t, ":9090", cfg.RunAddr)
	require.Equal(t, "debug", cfg.LogLevel)
	require.Equal(t, "postgres://localhost/test", cfg.DatabaseDSN)
	require.Equal(t, "secret", cfg.JWTSecret)
	require.Equal(t, 24*time.Hour, cfg.JWTTokenTTL)
}

func TestParseServerFlagsFlagOverridesEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "from-env")
	t.Setenv("SERVER_ADDRESS", ":9090")

	cfg := parseServerConfig([]string{"-j", "from-flag", "-a", ":3000"})
	require.Equal(t, "from-flag", cfg.JWTSecret)
	require.Equal(t, ":3000", cfg.RunAddr)
}
