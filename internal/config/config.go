// Package config содержит конфигурацию сервера GophKeeper.
package config

import (
	"flag"
	"os"
	"time"
)

// ServerConfig описывает параметры запуска сервера.
type ServerConfig struct {
	RunAddr     string
	LogLevel    string
	DatabaseDSN string
	JWTSecret   string
	JWTTokenTTL time.Duration
}

// ParseServerFlags читает конфигурацию сервера из переменных окружения и флагов.
// Env задаёт значения по умолчанию, флаги командной строки их перезаписывают.
func ParseServerFlags() ServerConfig {
	return parseServerConfig(nil)
}

func parseServerConfig(args []string) ServerConfig {
	var cfg ServerConfig
	var tokenTTLHours int

	if args == nil {
		args = os.Args[1:]
	}

	fs := flag.NewFlagSet("gophkeeper-server", flag.ContinueOnError)

	fs.StringVar(&cfg.RunAddr, "a", envOrDefault("SERVER_ADDRESS", ":8080"), "address and port to run server")
	fs.StringVar(&cfg.LogLevel, "l", envOrDefault("LOG_LEVEL", "info"), "log level")
	fs.StringVar(&cfg.DatabaseDSN, "d", envOrDefault("DATABASE_DSN", ""), "database dsn")
	fs.StringVar(&cfg.JWTSecret, "j", envOrDefault("JWT_SECRET", ""), "jwt secret key")
	fs.IntVar(&tokenTTLHours, "t", 24, "jwt token ttl in hours")

	_ = fs.Parse(args)

	cfg.JWTTokenTTL = time.Duration(tokenTTLHours) * time.Hour

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
