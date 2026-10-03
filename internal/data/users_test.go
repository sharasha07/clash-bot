//go:build integration

package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsers(t *testing.T) {
	m := UserModel{pool: newTestPool(t)}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	t.Run("Insert", func(t *testing.T) {
		err := m.Insert(ctx, &User{Username: "saba", Password: password{"saba123", "hash123"}})
		require.NoError(t, err)

		err = m.Insert(ctx, &User{Username: "Saba", Password: password{"123", "hash123"}})
		assert.ErrorIs(t, err, ErrDuplicateUsersUsername)
	})

	t.Run("GetByID", func(t *testing.T) {
		user, err := m.GetByID(ctx, 1)
		require.NoError(t, err)

		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "saba", user.Username)
		assert.Equal(t, "hash123", user.Password.Hash)
		assert.Equal(t, int32(1), user.Version)

		user, err = m.GetByID(ctx, 2)
		assert.ErrorIs(t, err, ErrNoRecord)
	})

	t.Run("GetByUsername", func(t *testing.T) {
		user, err := m.GetByUsername(ctx, "saba")
		require.NoError(t, err)

		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "saba", user.Username)
		assert.Equal(t, "hash123", user.Password.Hash)
		assert.Equal(t, int32(1), user.Version)

		user, err = m.GetByUsername(ctx, "luka")
		assert.ErrorIs(t, err, ErrNoRecord)
	})

	t.Run("Update", func(t *testing.T) {
		tag := "#tag"
		err := m.Update(ctx, &User{ID: 1, GameTag: &tag, Version: 1})
		assert.NoError(t, err)

		err = m.Update(ctx, &User{ID: 1, Username: "saba", Version: 1})
		assert.ErrorIs(t, err, ErrEditConflict)
	})

	t.Run("Delete", func(t *testing.T) {
		err := m.Delete(ctx, 1)
		require.NoError(t, err)

		err = m.Delete(ctx, 1)
		assert.ErrorIs(t, err, ErrNoRecord)
	})
}
