package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

// Storage описывает хранилище данных, необходимое KeeperService.
type Storage interface {
	CreateUser(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetUserByLogin(ctx context.Context, login string) (*model.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*model.User, error)
	CreateEntry(ctx context.Context, entry *model.Entry) error
	UpdateEntry(ctx context.Context, entry *model.Entry) error
	GetEntry(ctx context.Context, userID, entryID uuid.UUID) (*model.Entry, error)
	ListEntries(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error)
	DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error
	Ping(ctx context.Context) error
}
