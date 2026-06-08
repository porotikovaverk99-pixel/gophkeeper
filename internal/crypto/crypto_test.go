package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/crypto"
)

func TestEncryptDecrypt(t *testing.T) {
	salt, err := crypto.GenerateSalt()
	require.NoError(t, err)

	key := crypto.DeriveKey("master-password", salt)
	encrypted, err := crypto.Encrypt(key, []byte("secret-data"))
	require.NoError(t, err)

	decrypted, err := crypto.Decrypt(key, encrypted)
	require.NoError(t, err)
	require.Equal(t, "secret-data", string(decrypted))
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := crypto.HashPassword("account-password")
	require.NoError(t, err)

	ok, err := crypto.VerifyPassword("account-password", hash)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = crypto.VerifyPassword("wrong-password", hash)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	salt, err := crypto.GenerateSalt()
	require.NoError(t, err)

	key := crypto.DeriveKey("master-password", salt)
	_, err = crypto.Decrypt(key, "YWJjZA==")
	require.ErrorIs(t, err, crypto.ErrInvalidCiphertext)
}
