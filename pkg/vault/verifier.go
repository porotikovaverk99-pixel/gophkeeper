package vault

import (
	"errors"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
)

// ErrWrongMasterPassword возвращается при неверном мастер-пароле.
var ErrWrongMasterPassword = errors.New("wrong master password")

const verifierPlaintext = "gophkeeper-master-password-ok"

// CreateVerifier создаёт зашифрованную метку для проверки мастер-пароля.
func (m *Manager) CreateVerifier() (string, error) {
	return crypto.Encrypt(m.key, []byte(verifierPlaintext))
}

// ValidateVerifier проверяет соответствие мастер-пароля сохранённой метке.
func (m *Manager) ValidateVerifier(encoded string) error {
	if encoded == "" {
		return nil
	}

	plain, err := crypto.Decrypt(m.key, encoded)
	if err != nil || string(plain) != verifierPlaintext {
		return ErrWrongMasterPassword
	}

	return nil
}
