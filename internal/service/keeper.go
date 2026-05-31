// Package service содержит бизнес-логику сервера GophKeeper.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/repository"
)

var (
	// ErrInvalidCredentials возвращается при неверном логине или пароле.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// KeeperService реализует регистрацию, аутентификацию и CRUD для приватных данных.
type KeeperService struct {
	storage repository.Storage
	auth    *auth.Manager
}

// NewKeeperService создаёт сервис GophKeeper.
func NewKeeperService(storage repository.Storage, authManager *auth.Manager) *KeeperService {
	return &KeeperService{
		storage: storage,
		auth:    authManager,
	}
}

// Register создаёт нового пользователя и возвращает JWT-токен.
func (s *KeeperService) Register(ctx context.Context, login, password string) (string, *model.User, error) {
	if login == "" || password == "" {
		return "", nil, fmt.Errorf("login and password are required")
	}

	hash, err := crypto.HashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.storage.CreateUser(ctx, login, hash)
	if err != nil {
		return "", nil, err
	}

	token, err := s.auth.GenerateToken(user.ID, user.Login)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// Login аутентифицирует пользователя и возвращает JWT-токен.
func (s *KeeperService) Login(ctx context.Context, login, password string) (string, *model.User, error) {
	user, err := s.storage.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	ok, err := crypto.VerifyPassword(password, user.Password)
	if err != nil || !ok {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.auth.GenerateToken(user.ID, user.Login)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// CreateEntry сохраняет новую запись пользователя.
func (s *KeeperService) CreateEntry(ctx context.Context, userID uuid.UUID, entry *model.Entry) error {
	entry.UserID = userID
	return s.storage.CreateEntry(ctx, entry)
}

// UpdateEntry обновляет существующую запись пользователя.
func (s *KeeperService) UpdateEntry(ctx context.Context, userID uuid.UUID, entry *model.Entry) error {
	entry.UserID = userID
	return s.storage.UpdateEntry(ctx, entry)
}

// GetEntry возвращает запись пользователя по идентификатору.
func (s *KeeperService) GetEntry(ctx context.Context, userID, entryID uuid.UUID) (*model.Entry, error) {
	return s.storage.GetEntry(ctx, userID, entryID)
}

// SyncEntries возвращает все записи, изменённые после указанного времени.
func (s *KeeperService) SyncEntries(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	return s.storage.ListEntries(ctx, userID, since)
}

// DeleteEntry удаляет запись пользователя.
func (s *KeeperService) DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	return s.storage.DeleteEntry(ctx, userID, entryID)
}

// Ping проверяет доступность хранилища.
func (s *KeeperService) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}
