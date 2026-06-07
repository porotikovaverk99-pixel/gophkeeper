package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/config"
	packgzip "github.com/porotikovaverk99-pixel/gophkeeper/internal/gzip"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/handler"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/logger"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/middleware"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/repository"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/server"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
	"go.uber.org/zap"
)

func main() {
	cfg := config.ParseServerFlags()

	zlog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		_ = zlog.Sync()
	}()

	if cfg.DatabaseDSN == "" {
		zlog.Fatal("DATABASE_DSN is required")
	}
	if cfg.JWTSecret == "" {
		zlog.Fatal("JWT_SECRET is required")
	}

	storage, err := repository.NewPostgresStorage(cfg.DatabaseDSN)
	if err != nil {
		zlog.Fatal("failed to initialize storage", zap.Error(err))
	}
	defer storage.Close()

	authManager := auth.NewManager(cfg.JWTSecret, cfg.JWTTokenTTL)
	keeperService := service.NewKeeperService(storage, authManager)
	keeperHandler := handler.NewKeeperHandler(keeperService, zlog)

	httpServer := server.New(cfg.RunAddr)
	router := httpServer.Router()
	router.Use(func(next http.Handler) http.Handler {
		return logger.RequestLogger(zlog, packgzip.GzipMiddleware(next))
	})

	router.Get("/ping", keeperHandler.Ping)
	router.Post("/api/v1/register", keeperHandler.Register)
	router.Post("/api/v1/login", keeperHandler.Login)

	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authManager))
		r.Post("/api/v1/data", keeperHandler.CreateEntry)
		r.Get("/api/v1/data/{id}", keeperHandler.GetEntry)
		r.Put("/api/v1/data/{id}", keeperHandler.UpdateEntry)
		r.Delete("/api/v1/data/{id}", keeperHandler.DeleteEntry)
		r.Post("/api/v1/sync", keeperHandler.SyncEntries)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	zlog.Info("running gophkeeper server", zap.String("address", cfg.RunAddr))

	if err := httpServer.Run(ctx); err != nil {
		zlog.Fatal("server stopped", zap.Error(err))
	}

	zlog.Info("server shutdown complete")
}
