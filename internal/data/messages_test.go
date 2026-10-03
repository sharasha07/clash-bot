//go:build integration

package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessages(t *testing.T) {
	pool := newTestPool(t)
	users := UserModel{pool: pool}
	chats := ChatModel{pool: pool}
	m := MessageModel{pool: pool}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	user := User{Username: "saba", Password: password{"saba123", "hash123"}}
	require.NoError(t, users.Insert(ctx, &user))

	chat := Chat{UserID: user.ID, Name: "chat"}
	require.NoError(t, chats.Insert(ctx, &chat))

	t.Run("Insert", func(t *testing.T) {
		message := Message{ChatID: chat.ID, Role: RoleUser, Content: "hello"}
		require.NoError(t, m.Insert(ctx, &message))

		assert.Equal(t, chat.ID, message.ChatID)
		assert.Equal(t, RoleUser, message.Role)
		assert.Equal(t, "hello", message.Content)
	})

	t.Run("GetAll", func(t *testing.T) {
		require.NoError(t, m.Insert(ctx, &Message{ChatID: chat.ID, Role: RoleAssistant, Content: "hello2"}))
		require.NoError(t, m.Insert(ctx, &Message{ChatID: chat.ID, Role: RoleAssistant, Content: "hello3"}))

		filters := Filters{Page: 1, PageSize: 10, Sort: "-id", SortSafeList: []string{"id", "-id"}}

		messages, err := m.GetAll(ctx, chat.ID, user.ID, filters)
		require.NoError(t, err)
		require.Len(t, messages, 3)
		assert.Equal(t, []string{"hello3", "hello2", "hello"}, []string{messages[0].Content, messages[1].Content, messages[2].Content})

		messages, err = m.GetAll(ctx, chat.ID, 2, filters)
		require.NoError(t, err)
		assert.Empty(t, messages)
	})
}
