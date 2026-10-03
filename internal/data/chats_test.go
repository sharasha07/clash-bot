//go:build integration

package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChats(t *testing.T) {
	pool := newTestPool(t)
	users := UserModel{pool: pool}
	m := ChatModel{pool: pool}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	user := User{Username: "saba", Password: password{"saba123", "hash123"}}
	require.NoError(t, users.Insert(ctx, &user))

	t.Run("Insert", func(t *testing.T) {
		chat := Chat{UserID: user.ID, Name: "chat"}
		require.NoError(t, m.Insert(ctx, &chat))

		assert.Equal(t, int64(1), chat.ID)
		assert.Equal(t, user.ID, chat.UserID)
		assert.Equal(t, "chat", chat.Name)
		assert.Equal(t, int32(1), chat.Version)
	})

	t.Run("GetAll", func(t *testing.T) {
		require.NoError(t, m.Insert(ctx, &Chat{UserID: user.ID, Name: "chat2"}))
		require.NoError(t, m.Insert(ctx, &Chat{UserID: user.ID, Name: "chat3"}))

		filters := Filters{Page: 1, PageSize: 10, Sort: "-id", SortSafeList: []string{"id", "-id"}}

		chats, err := m.GetAll(ctx, user.ID, "", filters)
		require.NoError(t, err)
		require.Len(t, chats, 3)
		assert.Equal(t, []string{"chat3", "chat2", "chat"}, []string{chats[0].Name, chats[1].Name, chats[2].Name})
	})

	t.Run("Get", func(t *testing.T) {
		chat, err := m.Get(ctx, int64(1), user.ID)
		require.NoError(t, err)
		assert.Equal(t, "chat", chat.Name)

		_, err = m.Get(ctx, int64(4), user.ID)
		require.ErrorIs(t, err, ErrNoRecord)
	})

	t.Run("Update", func(t *testing.T) {
		chat, err := m.Get(ctx, int64(1), user.ID)
		require.NoError(t, err)

		chat.Name = "renamed"
		require.NoError(t, m.Update(ctx, &chat))
		assert.Equal(t, int32(2), chat.Version)

		got, err := m.Get(ctx, int64(1), user.ID)
		require.NoError(t, err)
		assert.Equal(t, "renamed", got.Name)

		chat.Version++
		require.ErrorIs(t, ErrEditConflict, m.Update(ctx, &chat))
	})

	t.Run("Touch", func(t *testing.T) {
		_, err := m.Touch(ctx, int64(1), user.ID)
		require.NoError(t, err)

		_, err = m.Touch(ctx, int64(4), 2)
		require.ErrorIs(t, err, ErrNoRecord)
	})

	t.Run("Delete", func(t *testing.T) {
		require.ErrorIs(t, m.Delete(ctx, int64(1), 2), ErrNoRecord)

		require.NoError(t, m.Delete(ctx, int64(1), user.ID))

		_, err := m.Get(ctx, int64(1), user.ID)
		require.ErrorIs(t, err, ErrNoRecord)
	})
}
