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

func TestChat(t *testing.T) {
	client := http.Client{Timeout: 10 * time.Second}

	var userID int64

	t.Run("create user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/users",
			strings.NewReader(`{"username": "chatsuser", "password": "chats1234"}`))

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

		assert.Equal(t, "chatsuser", result.User.Username)
		require.NotZero(t, result.User.ID)
		userID = result.User.ID
	})

	var accessToken string
	t.Run("user login", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/auth/login",
			strings.NewReader(`{"username": "chatsuser", "password": "chats1234"}`))

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

	var firstChatID int64
	t.Run("create chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/chats",
			strings.NewReader(`{"name": "  general  "}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Chat struct {
				ID        int64  `json:"id"`
				UserID    int64  `json:"user_id"`
				Name      string `json:"name"`
				CreatedAt string `json:"created_at"`
				UpdatedAt string `json:"updated_at"`
			} `json:"chat"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, userID, result.Chat.UserID)
		assert.Equal(t, "general", result.Chat.Name)
		assert.NotEmpty(t, result.Chat.CreatedAt)
		assert.NotEmpty(t, result.Chat.UpdatedAt)
		require.NotZero(t, result.Chat.ID)
		firstChatID = result.Chat.ID
	})

	var secondChatID int64
	t.Run("create second chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/chats",
			strings.NewReader(`{"name": "random"}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result struct {
			Chat struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"chat"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, "random", result.Chat.Name)
		require.NotZero(t, result.Chat.ID)
		secondChatID = result.Chat.ID
	})

	t.Run("show chats", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, apiURL+"/v1/chats?sort=id", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Chats []struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"chats"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		require.Equal(t, 2, len(result.Chats))
		assert.Equal(t, firstChatID, result.Chats[0].ID)
		assert.Equal(t, "general", result.Chats[0].Name)
		assert.Equal(t, secondChatID, result.Chats[1].ID)
		assert.Equal(t, "random", result.Chats[1].Name)
	})

	t.Run("show chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/chats/%d", apiURL, firstChatID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Chat struct {
				ID     int64  `json:"id"`
				UserID int64  `json:"user_id"`
				Name   string `json:"name"`
			} `json:"chat"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, firstChatID, result.Chat.ID)
		assert.Equal(t, userID, result.Chat.UserID)
		assert.Equal(t, "general", result.Chat.Name)
	})

	t.Run("update chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/v1/chats/%d", apiURL, firstChatID),
			strings.NewReader(`{"name": "  updated  "}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			Chat struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"chat"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, firstChatID, result.Chat.ID)
		assert.Equal(t, "updated", result.Chat.Name)
	})

	t.Run("delete chat", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/v1/chats/%d", apiURL, firstChatID), nil)
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
