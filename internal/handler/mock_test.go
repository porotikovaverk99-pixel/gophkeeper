package handler_test

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/repository"
)

type mockStorage struct {
	users   map[string]*model.User
	entries map[uuid.UUID]*model.Entry
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		users:   make(map[string]*model.User),
		entries: make(map[uuid.UUID]*model.Entry),
	}
}

func (m *mockStorage) CreateUser(_ context.Context, login, passwordHash string) (*model.User, error) {
	if _, ok := m.users[login]; ok {
		return nil, repository.ErrUserAlreadyExists
	}
	user := &model.User{ID: uuid.New(), Login: login, Password: passwordHash, CreatedAt: time.Now().UTC()}
	m.users[login] = user
	return user, nil
}

func (m *mockStorage) GetUserByLogin(_ context.Context, login string) (*model.User, error) {
	user, ok := m.users[login]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	return user, nil
}

func (m *mockStorage) GetUserByID(_ context.Context, userID uuid.UUID) (*model.User, error) {
	for _, user := range m.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return nil, repository.ErrUserNotFound
}

func (m *mockStorage) CreateEntry(_ context.Context, entry *model.Entry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.UpdatedAt = time.Now().UTC()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockStorage) UpdateEntry(_ context.Context, entry *model.Entry) error {
	if _, ok := m.entries[entry.ID]; !ok {
		return repository.ErrEntryNotFound
	}
	entry.UpdatedAt = time.Now().UTC()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockStorage) GetEntry(_ context.Context, userID, entryID uuid.UUID) (*model.Entry, error) {
	entry, ok := m.entries[entryID]
	if !ok || entry.UserID != userID {
		return nil, repository.ErrEntryNotFound
	}
	return entry, nil
}

func (m *mockStorage) ListEntries(_ context.Context, userID uuid.UUID, since time.Time) ([]model.Entry, error) {
	var result []model.Entry
	for _, entry := range m.entries {
		if entry.UserID != userID {
			continue
		}
		if since.IsZero() || !entry.UpdatedAt.Before(since) {
			result = append(result, *entry)
		}
	}
	return result, nil
}

func (m *mockStorage) DeleteEntry(ctx context.Context, userID, entryID uuid.UUID) error {
	entry, err := m.GetEntry(ctx, userID, entryID)
	if err != nil {
		return err
	}
	entry.Deleted = true
	return m.UpdateEntry(ctx, entry)
}

func (m *mockStorage) Ping(context.Context) error {
	return nil
}
