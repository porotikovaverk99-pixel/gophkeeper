package vault_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/model"
	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/vault"
)

func TestBuildAndDecryptEntry(t *testing.T) {
	manager, salt, err := vault.NewManagerWithSalt("master")
	require.NoError(t, err)

	entry, err := manager.BuildEntry(model.EntryTypeText, "note", model.TextPayload{Text: "hello"})
	require.NoError(t, err)
	require.NotEmpty(t, entry.Payload)

	manager2, err := vault.NewManager("master", salt)
	require.NoError(t, err)

	var payload model.TextPayload
	require.NoError(t, manager2.DecryptPayload(entry.Payload, &payload))
	require.Equal(t, "hello", payload.Text)
}

func TestMasterPasswordVerifier(t *testing.T) {
	manager, salt, err := vault.NewManagerWithSalt("correct")
	require.NoError(t, err)

	verifier, err := manager.CreateVerifier()
	require.NoError(t, err)

	wrongManager, err := vault.NewManager("wrong", salt)
	require.NoError(t, err)
	require.ErrorIs(t, wrongManager.ValidateVerifier(verifier), vault.ErrWrongMasterPassword)

	correctManager, err := vault.NewManager("correct", salt)
	require.NoError(t, err)
	require.NoError(t, correctManager.ValidateVerifier(verifier))
}
