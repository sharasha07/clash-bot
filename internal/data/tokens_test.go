//go:build integration

package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokens(t *testing.T) {
	pool := newTestPool(t)
	users := UserModel{pool: pool}
	tokens := TokenModel{pool: pool}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	user := User{Username: "saba", Password: password{"saba123", "hash123"}}
	require.NoError(t, users.Insert(ctx, &user))

	t.Run("Insert", func(t *testing.T) {
		require.NoError(t, tokens.Insert(ctx, "saba-token", user.ID, time.Hour))
	})

	t.Run("GetUserID", func(t *testing.T) {
		userID, err := tokens.GetUserID(ctx, "saba-token")
		require.NoError(t, err)
		assert.Equal(t, user.ID, userID)

		_, err = tokens.GetUserID(ctx, "unknown-token")
		require.ErrorIs(t, err, ErrNoRecord)

		require.NoError(t, tokens.Insert(ctx, "expired-token", user.ID, -time.Hour))

		_, err = tokens.GetUserID(ctx, "expired-token")
		require.ErrorIs(t, err, ErrNoRecord)
	})

	t.Run("Delete", func(t *testing.T) {
		require.NoError(t, tokens.Delete(ctx, "saba-token"))

		_, err := tokens.GetUserID(ctx, "saba-token")
		require.ErrorIs(t, err, ErrNoRecord)
	})
}
