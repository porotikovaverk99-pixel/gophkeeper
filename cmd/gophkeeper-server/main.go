package main

import (
	"log"
	"net/http"

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

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Log.Sync()
	}()

	if cfg.DatabaseDSN == "" {
		logger.Log.Fatal("DATABASE_DSN is required")
	}

	storage, err := repository.NewPostgresStorage(cfg.DatabaseDSN)
	if err != nil {
		logger.Log.Fatal("failed to initialize storage", zap.Error(err))
	}

	authManager := auth.NewManager(cfg.JWTSecret, cfg.JWTTokenTTL)
	keeperService := service.NewKeeperService(storage, authManager)
	keeperHandler := handler.NewKeeperHandler(keeperService)

	httpServer := server.New(cfg.RunAddr)
	router := httpServer.Router()
	router.Use(func(next http.Handler) http.Handler {
		return logger.RequestLogger(packgzip.GzipMiddleware(next))
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

	logger.Log.Info("running gophkeeper server", zap.String("address", cfg.RunAddr))

	if err := httpServer.Run(); err != nil {
		logger.Log.Fatal("server stopped", zap.Error(err))
	}
}
