package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		input     string
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "status created",
			method:   http.MethodPost,
			input:    `{"username": "giorgi", "password": "giorgi123"}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					User struct {
						Username       string  `json:"username"`
						GameTag        *string `json:"game_tag"`
						ProfilePicture *string `json:"profile_picture"`
					} `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "giorgi", result.User.Username)
				assert.Nil(t, result.User.GameTag)
				assert.Nil(t, result.User.ProfilePicture)
			},
		},
		{
			name:     "short password input",
			method:   http.MethodPost,
			input:    `{"username": "saba", "password": "saba"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must be at least 8 characters", result.Error["password"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, apiURL+"/v1/users", strings.NewReader(tt.input))
			require.NoError(t, err)

			client := http.Client{Timeout: 3 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.checkBody != nil {
				tt.checkBody(t, resp)
			}
		})
	}
}

func TestShowUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		path      string
		wantCode  int
		chechBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "ok status",
			method:   http.MethodGet,
			path:     "/v1/users/1",
			wantCode: http.StatusOK,
			chechBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.User.ID)
				assert.Equal(t, "nika", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, apiURL+tt.path, nil)
			require.NoError(t, err)

			client := http.Client{Timeout: 3 * time.Second}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.chechBody != nil {
				tt.chechBody(t, resp)
			}
		})
	}
}
