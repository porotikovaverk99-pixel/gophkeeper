// Package crypto предоставляет функции шифрования данных на стороне клиента.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	saltSize   = 16
	nonceSize  = 12
	keySize    = 32
	iterations = 100_000
)

var (
	// ErrInvalidCiphertext возвращается при повреждённом или некорректном шифротексте.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// DeriveKey создаёт ключ шифрования из мастер-пароля и соли с помощью PBKDF2.
func DeriveKey(masterPassword string, salt []byte) []byte {
	return pbkdf2.Key([]byte(masterPassword), salt, iterations, keySize, sha256.New)
}

// GenerateSalt генерирует случайную соль для деривации ключа.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt шифрует plaintext с помощью AES-GCM и возвращает base64-представление nonce+ciphertext.
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt расшифровывает данные, зашифрованные функцией Encrypt.
func Decrypt(key []byte, encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}

// HashPassword хеширует пароль пользователя для хранения на сервере.
func HashPassword(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}

	key := DeriveKey(password, salt)
	return base64.StdEncoding.EncodeToString(salt) + ":" + base64.StdEncoding.EncodeToString(key), nil
}

// VerifyPassword проверяет соответствие пароля сохранённому хешу.
func VerifyPassword(password, storedHash string) (bool, error) {
	parts := splitOnce(storedHash, ':')
	if len(parts) != 2 {
		return false, errors.New("invalid password hash format")
	}

	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}

	expected, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}

	actual := DeriveKey(password, salt)
	if len(actual) != len(expected) {
		return false, nil
	}

	var result byte
	for i := range actual {
		result |= actual[i] ^ expected[i]
	}

	return result == 0, nil
}

func splitOnce(value string, sep byte) []string {
	for i := 0; i < len(value); i++ {
		if value[i] == sep {
			return []string{value[:i], value[i+1:]}
		}
	}
	return []string{value}
}
