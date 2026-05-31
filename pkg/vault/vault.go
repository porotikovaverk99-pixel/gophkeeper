// Package vault реализует шифрование и расшифровку записей на стороне клиента.
package vault

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
)

// Manager управляет шифрованием payload записей мастер-паролем.
type Manager struct {
	key []byte
}

// NewManager создаёт менеджер vault из мастер-пароля и соли.
func NewManager(masterPassword, encodedSalt string) (*Manager, error) {
	salt, err := decodeSalt(encodedSalt)
	if err != nil {
		return nil, err
	}

	return &Manager{
		key: crypto.DeriveKey(masterPassword, salt),
	}, nil
}

// NewManagerWithSalt создаёт менеджер vault и возвращает новую соль для сохранения.
func NewManagerWithSalt(masterPassword string) (*Manager, string, error) {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, "", err
	}

	return &Manager{key: crypto.DeriveKey(masterPassword, salt)}, encodeSalt(salt), nil
}

// EncryptPayload шифрует произвольную структуру данных записи.
func (m *Manager) EncryptPayload(payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	encoded, err := crypto.Encrypt(m.key, raw)
	if err != nil {
		return nil, err
	}

	return []byte(encoded), nil
}

// DecryptPayload расшифровывает payload записи в указанную структуру.
func (m *Manager) DecryptPayload(payload []byte, target any) error {
	raw, err := crypto.Decrypt(m.key, string(payload))
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	return nil
}

// BuildEntry создаёт зашифрованную запись указанного типа.
func (m *Manager) BuildEntry(entryType model.EntryType, metadata string, payload any) (*model.Entry, error) {
	encrypted, err := m.EncryptPayload(payload)
	if err != nil {
		return nil, err
	}

	return &model.Entry{
		ID:       uuid.New(),
		Type:     entryType,
		Metadata: metadata,
		Payload:  encrypted,
	}, nil
}

func encodeSalt(salt []byte) string {
	return fmt.Sprintf("%x", salt)
}

func decodeSalt(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, fmt.Errorf("master salt is required")
	}

	salt := make([]byte, len(encoded)/2)
	for i := 0; i < len(salt); i++ {
		var b byte
		if _, err := fmt.Sscanf(encoded[i*2:i*2+2], "%02x", &b); err != nil {
			return nil, fmt.Errorf("decode salt: %w", err)
		}
		salt[i] = b
	}

	return salt, nil
}
