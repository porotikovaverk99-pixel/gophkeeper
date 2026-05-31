package api_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/handler"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/middleware"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/repository"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/api"
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
		if entry.UserID == userID {
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

func newTestServer(t *testing.T) (*httptest.Server, *api.Client) {
	t.Helper()

	authManager := auth.NewManager("secret", time.Hour)
	h := handler.NewKeeperHandler(service.NewKeeperService(newMockStorage(), authManager))

	router := chi.NewRouter()
	router.Post("/api/v1/register", h.Register)
	router.Post("/api/v1/login", h.Login)
	router.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authManager))
		r.Post("/api/v1/data", h.CreateEntry)
		r.Get("/api/v1/data/{id}", h.GetEntry)
		r.Put("/api/v1/data/{id}", h.UpdateEntry)
		r.Delete("/api/v1/data/{id}", h.DeleteEntry)
		r.Post("/api/v1/sync", h.SyncEntries)
	})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := api.NewClient(server.URL)
	client.SetToken("")
	return server, client
}

func TestClientRegisterAndCreateEntry(t *testing.T) {
	_, client := newTestServer(t)

	token, err := client.Register(context.Background(), "alice", "password")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	entry := &model.Entry{Type: model.EntryTypeText, Metadata: "note", Payload: []byte("secret")}
	require.NoError(t, client.CreateEntry(context.Background(), entry))

	fetched, err := client.GetEntry(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, "note", fetched.Metadata)

	entry.Metadata = "updated"
	require.NoError(t, client.UpdateEntry(context.Background(), entry))

	entries, err := client.SyncEntries(context.Background(), time.Time{})
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	require.NoError(t, client.DeleteEntry(context.Background(), entry.ID))
}

func TestClientLoginInvalidCredentials(t *testing.T) {
	_, client := newTestServer(t)
	_, err := client.Login(context.Background(), "missing", "password")
	require.Error(t, err)
}
