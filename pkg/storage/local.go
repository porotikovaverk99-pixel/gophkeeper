// Package storage управляет локальным кешем записей клиента GophKeeper.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

// LocalStorage хранит локальную копию записей и параметры сессии клиента.
type LocalStorage struct {
	mu         sync.RWMutex
	path       string
	Token      string    `json:"token"`
	Login      string    `json:"login"`
	LastSync   time.Time `json:"last_sync"`
	MasterSalt     string        `json:"master_salt"`
	MasterVerifier string        `json:"master_verifier,omitempty"`
	Entries        []model.Entry `json:"entries"`
}

// OpenLocalStorage загружает или создаёт локальное хранилище клиента.
func OpenLocalStorage(path string) (*LocalStorage, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(home, ".gophkeeper", "vault.json")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	storage := &LocalStorage{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storage, nil
		}
		return nil, fmt.Errorf("read storage: %w", err)
	}

	if err := json.Unmarshal(data, storage); err != nil {
		return nil, fmt.Errorf("decode storage: %w", err)
	}

	return storage, nil
}

// Save сохраняет локальное хранилище на диск.
func (s *LocalStorage) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal storage: %w", err)
	}

	if err := writeFileAtomic(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write storage: %w", err)
	}

	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// UpsertEntry добавляет или обновляет запись в локальном хранилище.
func (s *LocalStorage) UpsertEntry(entry model.Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.Entries {
		if existing.ID == entry.ID {
			s.Entries[i] = entry
			return
		}
	}

	s.Entries = append(s.Entries, entry)
}

// ListEntries возвращает все неудалённые записи из локального хранилища.
func (s *LocalStorage) ListEntries() []model.Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Entry, 0, len(s.Entries))
	for _, entry := range s.Entries {
		if !entry.Deleted {
			result = append(result, entry)
		}
	}

	return result
}

// GetEntry возвращает запись по идентификатору.
func (s *LocalStorage) GetEntry(id uuid.UUID) (model.Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, entry := range s.Entries {
		if entry.ID == id && !entry.Deleted {
			return entry, true
		}
	}

	return model.Entry{}, false
}

// RemoveEntry помечает запись как удалённую в локальном хранилище.
func (s *LocalStorage) RemoveEntry(id uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, entry := range s.Entries {
		if entry.ID == id {
			entry.Deleted = true
			entry.UpdatedAt = time.Now().UTC()
			s.Entries[i] = entry
			return true
		}
	}

	return false
}
