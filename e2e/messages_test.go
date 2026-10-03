//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessages(t *testing.T) {
	client := http.Client{Timeout: 30 * time.Second}

	var userID int64

	t.Run("create user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/users",
			strings.NewReader(`{"username": "msguser", "password": "msguser1234"}`))

		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			User struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
			} `json:"user"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, "msguser", result.User.Username)

		require.NotZero(t, result.User.ID)
		userID = result.User.ID
	})

	var accessToken string
	t.Run("user login", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/auth/login",
			strings.NewReader(`{"username": "msguser", "password": "msguser1234"}`))

		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			AccessToken string `json:"access_token"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.NotEmpty(t, result.AccessToken)
		accessToken = result.AccessToken
	})

	var chatID int64
	t.Run("create chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/chats",
			strings.NewReader(`{"name": "clash"}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Chat struct {
				ID     int64  `json:"id"`
				Name   string `json:"name"`
				UserID int64  `json:"user_id"`
			} `json:"chat"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, userID, result.Chat.UserID)
		assert.Equal(t, "clash", result.Chat.Name)

		require.NotZero(t, result.Chat.ID)
		chatID = result.Chat.ID
	})

	var messageID int64
	t.Run("create message", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/chats/%d/messages", apiURL, chatID),
			strings.NewReader(`{"content": "  whats my next 5 chests??  "}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Message struct {
				ID      int64  `json:"id"`
				ChatID  int64  `json:"chat_id"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			Reply *struct {
				ID      int64  `json:"id"`
				ChatID  int64  `json:"chat_id"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"reply"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, chatID, result.Message.ChatID)
		assert.Equal(t, "user", result.Message.Role)
		assert.Equal(t, "whats my next 5 chests??", result.Message.Content)

		require.NotZero(t, result.Message.ID)
		messageID = result.Message.ID

		require.NotNil(t, result.Reply)
		assert.Equal(t, chatID, result.Reply.ChatID)
		assert.Equal(t, "assistant", result.Reply.Role)
		assert.NotEmpty(t, result.Reply.Content)
		assert.Greater(t, result.Reply.ID, result.Message.ID)
	})

	t.Run("show messages", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/chats/%d/messages?sort=id", apiURL, chatID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Messages []struct {
				ID      int64  `json:"id"`
				ChatID  int64  `json:"chat_id"`
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		require.Equal(t, 2, len(result.Messages))
		assert.Equal(t, messageID, result.Messages[0].ID)
		assert.Equal(t, "user", result.Messages[0].Role)
		assert.Equal(t, "whats my next 5 chests??", result.Messages[0].Content)
		assert.Equal(t, "assistant", result.Messages[1].Role)
		assert.NotEmpty(t, result.Messages[1].Content)
	})

	t.Run("delete chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/chats/%d", apiURL, chatID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("delete user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/users/%d", apiURL, userID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}
