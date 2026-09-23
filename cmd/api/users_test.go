package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/data/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCode  int
		setMock   func(t *testing.T, users *mocks.MockUserModelInterface)
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "status created with user in the request body",
			input:    `{"username": "luka", "password": "luka1234"}`,
			wantCode: http.StatusCreated,
			setMock: func(t *testing.T, users *mocks.MockUserModelInterface) {
				t.Helper()

				users.EXPECT().Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, u *data.User) error {
						assert.Equal(t, "luka", u.Username)
						return nil
					})
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
			},
		},
		{
			name:     "Long username, valid password",
			input:    `{"username": "lukalukaluka", "password": "lukaluka123"}`,
			wantCode: http.StatusUnprocessableEntity,
			setMock:  nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must not be more than 10 characters", result.Error["username"])
				assert.Equal(t, 1, len(result.Error))
			},
		},
		{
			name:     "empty username, empty password",
			input:    `{"username": "", "password": ""}`,
			wantCode: http.StatusUnprocessableEntity,
			setMock:  nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must be provided", result.Error["username"])
				assert.Equal(t, "must be provided", result.Error["password"])
				assert.Equal(t, 2, len(result.Error))
			},
		},
		{
			name:     "Invalid username, invalid password",
			input:    fmt.Sprintf(`{"username": "lukalukaluka", "password": "%s"}`, strings.Repeat("luka", 8)),
			wantCode: http.StatusUnprocessableEntity,
			setMock:  nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must not be more than 10 characters", result.Error["username"])
				assert.Equal(t, "must not be more than 30 characters", result.Error["password"])
				assert.Equal(t, 2, len(result.Error))
			},
		},
		{
			name:     "Username unique violation",
			input:    `{"username": "shaba", "password": "luka1234"}`,
			wantCode: http.StatusUnprocessableEntity,
			setMock: func(t *testing.T, users *mocks.MockUserModelInterface) {
				t.Helper()
				users.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(data.ErrUniqueViolation)
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			if tt.setMock != nil {
				users := mocks.NewMockUserModelInterface(gomock.NewController(t))
				tt.setMock(t, users)
				app.models.Users = users
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(tt.input))
			rr := httptest.NewRecorder()

			app.createUserHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
			assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			if tt.checkBody != nil {
				tt.checkBody(t, resp)
			}
		})
	}
}
