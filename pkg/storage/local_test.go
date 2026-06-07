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

func TestListEntriesSkipsDeleted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.json")

	store, err := storage.OpenLocalStorage(path)
	require.NoError(t, err)

	activeID := uuid.New()
	deletedID := uuid.New()
	store.UpsertEntry(model.Entry{ID: activeID, Type: model.EntryTypeText, Metadata: "keep"})
	store.UpsertEntry(model.Entry{ID: deletedID, Type: model.EntryTypeText, Deleted: true})

	entries := store.ListEntries()
	require.Len(t, entries, 1)
	require.Equal(t, activeID, entries[0].ID)
}

func TestUpsertEntryUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.json")

	store, err := storage.OpenLocalStorage(path)
	require.NoError(t, err)

	entryID := uuid.New()
	store.UpsertEntry(model.Entry{ID: entryID, Type: model.EntryTypeText, Metadata: "first"})
	store.UpsertEntry(model.Entry{ID: entryID, Type: model.EntryTypeText, Metadata: "second"})

	entry, ok := store.GetEntry(entryID)
	require.True(t, ok)
	require.Equal(t, "second", entry.Metadata)
}

func TestOpenLocalStorageDefaultPath(t *testing.T) {
	store, err := storage.OpenLocalStorage("")
	require.NoError(t, err)
	require.NotNil(t, store)
}
