//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsers(t *testing.T) {
	client := http.Client{Timeout: 10 * time.Second}

	var userID int64

	t.Run("create user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/users",
			strings.NewReader(`{"username": "shaba", "password": "shaba1234"}`))

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

		assert.Equal(t, "shaba", result.User.Username)

		require.NotZero(t, result.User.ID)
		userID = result.User.ID
	})

	var accessToken string
	t.Run("user login", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, apiURL+"/v1/auth/login",
			strings.NewReader(`{"username": "shaba", "password": "shaba1234"}`))

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

	t.Run("upload profile picture", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		var jpegBuf bytes.Buffer

		jpeg.Encode(&jpegBuf, img, nil)

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		part, err := writer.CreateFormFile("avatar", "image.jpg")
		require.NoError(t, err)

		_, err = io.Copy(part, &jpegBuf)
		require.NoError(t, err)

		require.NoError(t, writer.Close())

		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/users/%d/profile-picture", apiURL, userID), &body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			User struct {
				ProfilePicture *string `json:"profile_picture"`
			} `json:"user"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		require.NotNil(t, result.User.ProfilePicture)
		assert.NotEmpty(t, result.User.ProfilePicture)
	})

	t.Run("update user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/v1/users/%d", apiURL, userID),
			strings.NewReader(`{"username": "shabaninio", "game_tag": "#tag"}`))

		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})

	t.Run("show user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/users/%d", apiURL, userID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var result struct {
			User struct {
				ID             int64   `json:"id"`
				Username       string  `json:"username"`
				GameTag        *string `json:"game_tag"`
				ProfilePicture *string `json:"profile_picture"`
			} `json:"user"`
		}

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

		assert.Equal(t, userID, result.User.ID)
		assert.Equal(t, "shabaninio", result.User.Username)
		assert.Equal(t, "#tag", *result.User.GameTag)
		assert.NotNil(t, result.User.ProfilePicture)
		assert.NotEmpty(t, result.User.ProfilePicture)
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
