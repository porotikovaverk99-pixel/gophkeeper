package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/internal/auth"
)

func TestGenerateAndParseToken(t *testing.T) {
	manager := auth.NewManager("secret", time.Hour)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID, "alice")
	require.NoError(t, err)

	claims, err := manager.ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, userID, claims.UserID)
	require.Equal(t, "alice", claims.Login)
}

func TestParseInvalidToken(t *testing.T) {
	manager := auth.NewManager("secret", time.Hour)
	_, err := manager.ParseToken("invalid.token.value")
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}
