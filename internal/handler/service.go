package handler

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

// KeeperService описывает бизнес-логику, используемую HTTP-обработчиками.
type KeeperService interface {
	Register(ctx context.Context, login, password string) (string, *model.User, error)
	Login(ctx context.Context, login, password string) (string, *model.User, error)
	CreateEntry(ctx context.Context, userID uuid.UUID, entry *model.Entry) error
	UpdateEntry(ctx context.Context, userID uuid.UUID, entry *model.Entry) error
	GetEntry(ctx context.Context, userID, entryID uuid.UUID) (*model.Entry, error)
	SyncEntries(ctx context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error)
	DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error
	Ping(ctx context.Context) error
}
