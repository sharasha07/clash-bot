package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserHandler(t *testing.T) {
	app := newTestApplication(t)
	app.models.Users.(*mocks.MockUserRepository).EXPECT().Insert(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, user *data.User) error {
			if user.Username == "shaba" {
				return data.ErrDuplicateUsersUsername
			}

			user.ID = 5
			return nil
		}).Times(2)

	tests := []struct {
		name      string
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "empty username, small password",
			input:    `{"username": "", "password": "saba123"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["username"])
				assert.Equal(t, "must be more than 8 characters", result.Error["password"])
			},
		},
		{
			name:     "big username, empty password",
			input:    `{"username": "shabashabashaba1", "password": ""}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must be a maximum of 15 characters", result.Error["username"])
				assert.Equal(t, "must not be empty", result.Error["password"])
			},
		},
		{
			name:     "whitespace username, big password",
			input:    fmt.Sprintf(`{"username": "    ", "password": "%s"}`, strings.Repeat("a", 41)),
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 2, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["username"])
				assert.Equal(t, "must be a maximum of 40 characters", result.Error["password"])
			},
		},
		{
			name:     "duplicate username",
			input:    `{"username": "shaba", "password": "shaba1234"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be unique", result.Error["username"])
			},
		},
		{
			name:     "successful input",
			input:    `{"username": "luka", "password": "lukaluka123"}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, body io.Reader) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(5), result.User.ID)
				assert.Equal(t, "luka", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(tt.input))

			app.createUserHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			if tt.checkBody != nil {
				tt.checkBody(t, resp.Body)
			}
		})
	}
}
