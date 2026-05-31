// Package config содержит конфигурацию сервера GophKeeper.
package config

import (
	"flag"
	"os"
	"time"
)

// ServerConfig описывает параметры запуска сервера.
type ServerConfig struct {
	RunAddr      string
	LogLevel     string
	DatabaseDSN  string
	JWTSecret    string
	JWTTokenTTL  time.Duration
}

// ParseServerFlags читает флаги и переменные окружения сервера.
func ParseServerFlags() ServerConfig {
	var cfg ServerConfig
	var tokenTTLHours int

	flag.StringVar(&cfg.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database dsn")
	flag.StringVar(&cfg.JWTSecret, "j", "", "jwt secret key")
	flag.IntVar(&tokenTTLHours, "t", 24, "jwt token ttl in hours")
	flag.Parse()

	cfg.JWTTokenTTL = time.Duration(tokenTTLHours) * time.Hour

	if value := os.Getenv("SERVER_ADDRESS"); value != "" {
		cfg.RunAddr = value
	}
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		cfg.LogLevel = value
	}
	if value := os.Getenv("DATABASE_DSN"); value != "" {
		cfg.DatabaseDSN = value
	}
	if value := os.Getenv("JWT_SECRET"); value != "" {
		cfg.JWTSecret = value
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "gophkeeper-dev-secret"
	}

	return cfg
}
