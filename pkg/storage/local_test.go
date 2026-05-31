package storage_test

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/storage"
)

func TestLocalStorageSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.json")

	store, err := storage.OpenLocalStorage(path)
	require.NoError(t, err)

	entry := model.Entry{
		ID:       uuid.New(),
		Type:     model.EntryTypeText,
		Metadata: "note",
		Payload:  []byte("encrypted"),
	}
	store.UpsertEntry(entry)
	require.NoError(t, store.Save())

	reloaded, err := storage.OpenLocalStorage(path)
	require.NoError(t, err)

	loaded, ok := reloaded.GetEntry(entry.ID)
	require.True(t, ok)
	require.Equal(t, entry.Metadata, loaded.Metadata)
}

func TestRemoveEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.json")

	store, err := storage.OpenLocalStorage(path)
	require.NoError(t, err)

	entryID := uuid.New()
	store.UpsertEntry(model.Entry{ID: entryID, Type: model.EntryTypeText})
	require.True(t, store.RemoveEntry(entryID))

	_, ok := store.GetEntry(entryID)
	require.False(t, ok)
}
