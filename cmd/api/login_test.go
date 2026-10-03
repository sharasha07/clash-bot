package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLoginHandler(t *testing.T) {
	app := newTestApplication(t)
	app.cfg.JWT.Secret = "secret"
	app.cfg.JWT.AccessTTL = time.Hour
	app.cfg.JWT.RefreshTTL = 24 * time.Hour

	hash, err := argon2id.CreateHash("shaba1234", argon2id.DefaultParams)
	require.NoError(t, err)

	app.models.Users.(*mocks.MockUserRepository).EXPECT().
		GetByUsername(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, username string) (data.User, error) {
			if username == "unknown" {
				return data.User{}, data.ErrNoRecord
			}

			var user data.User
			user.ID = 5
			user.Username = username
			user.Password.Hash = hash

			return user, nil
		}).Times(3)

	app.models.Tokens.(*mocks.MockTokenRepository).EXPECT().
		Insert(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	tests := []struct {
		name      string
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "user does not exist",
			input:    `{"username": "unknown", "password": "unknown123"}`,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "invalid credentials", result.Error)
			},
		},
		{
			name:     "password mismatch",
			input:    `{"username": "shaba", "password": "lukaluka123"}`,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "invalid credentials", result.Error)
			},
		},
		{
			name:     "successful login",
			input:    `{"username": "shaba", "password": "shaba1234"}`,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					AccessToken  string `json:"access_token"`
					RefreshToken string `json:"refresh_token"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/users/login", strings.NewReader(tt.input))

			app.loginHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}

func TestRefreshTokenHandler(t *testing.T) {
	app := newTestApplication(t)
	app.cfg.JWT.Secret = "secret"
	app.cfg.JWT.AccessTTL = time.Hour

	app.models.Tokens.(*mocks.MockTokenRepository).EXPECT().
		GetUserID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, token string) (int64, error) {
			if token == "invalid" {
				return 0, data.ErrNoRecord
			}

			return 5, nil
		}).Times(2)

	tests := []struct {
		name      string
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "missing refresh token",
			input:    `{"refresh_token": ""}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be provided", result.Error["refresh_token"])
			},
		},
		{
			name:     "refresh token does not exist",
			input:    `{"refresh_token": "invalid"}`,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "invalid authentication token", result.Error)
			},
		},
		{
			name:     "successful refresh",
			input:    `{"refresh_token": "valid"}`,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					AccessToken string `json:"access_token"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.NotEmpty(t, result.AccessToken)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/users/refresh-token", strings.NewReader(tt.input))

			app.refreshTokenHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}
