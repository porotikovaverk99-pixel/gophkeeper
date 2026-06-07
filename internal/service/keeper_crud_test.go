package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/service"
)

func TestEntryCRUD(t *testing.T) {
	storage := newMockStorage()
	svc := service.NewKeeperService(storage, auth.NewManager("secret", time.Hour))

	_, user, err := svc.Register(context.Background(), "alice", "password")
	require.NoError(t, err)

	entry := &model.Entry{
		Type:     model.EntryTypeText,
		Metadata: "note",
		Payload:  []byte("encrypted"),
	}
	require.NoError(t, svc.CreateEntry(context.Background(), user.ID, entry))

	fetched, err := svc.GetEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)
	require.Equal(t, "note", fetched.Metadata)

	entries, err := svc.SyncEntries(context.Background(), user.ID, time.Time{})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	require.NoError(t, svc.DeleteEntry(context.Background(), user.ID, entry.ID))
}

func TestUpdateEntry(t *testing.T) {
	storage := newMockStorage()
	svc := service.NewKeeperService(storage, auth.NewManager("secret", time.Hour))

	_, user, err := svc.Register(context.Background(), "alice", "password")
	require.NoError(t, err)

	entry := &model.Entry{
		Type:     model.EntryTypeText,
		Metadata: "note",
		Payload:  []byte("encrypted"),
	}
	require.NoError(t, svc.CreateEntry(context.Background(), user.ID, entry))

	entry.Metadata = "updated"
	require.NoError(t, svc.UpdateEntry(context.Background(), user.ID, entry))

	updated, err := svc.GetEntry(context.Background(), user.ID, entry.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", updated.Metadata)
}

func TestGetEntryNotFound(t *testing.T) {
	svc := service.NewKeeperService(newMockStorage(), auth.NewManager("secret", time.Hour))

	_, err := svc.GetEntry(context.Background(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, service.ErrEntryNotFound)
}

func TestPing(t *testing.T) {
	svc := service.NewKeeperService(newMockStorage(), auth.NewManager("secret", time.Hour))
	require.NoError(t, svc.Ping(context.Background()))
}

func TestRegisterDuplicateUser(t *testing.T) {
	svc := service.NewKeeperService(newMockStorage(), auth.NewManager("secret", time.Hour))
	_, _, err := svc.Register(context.Background(), "alice", "password")
	require.NoError(t, err)

	_, _, err = svc.Register(context.Background(), "alice", "password")
	require.Error(t, err)
}

func TestCreateEntryRequiresUser(t *testing.T) {
	svc := service.NewKeeperService(newMockStorage(), auth.NewManager("secret", time.Hour))
	entry := &model.Entry{Type: model.EntryTypeText}
	require.NoError(t, svc.CreateEntry(context.Background(), uuid.New(), entry))
}
