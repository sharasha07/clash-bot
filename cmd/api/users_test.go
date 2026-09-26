package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCode  int
		setupMock func(repo *mocks.MockUserRepository)
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:      "empty input",
			input:     `{"username": "", "password": ""}`,
			wantCode:  http.StatusUnprocessableEntity,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must be provided", result.Error["username"])
				assert.Equal(t, "must be provided", result.Error["password"])
			},
		},
		{
			name:      "long username, short password",
			input:     `{"username": "sabasabasaba", "password": "saba"}`,
			wantCode:  http.StatusUnprocessableEntity,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must not be more than 10 characters", result.Error["username"])
				assert.Equal(t, "must be at least 8 characters", result.Error["password"])
			},
		},
		{
			name:      "whitespace username",
			input:     `{"username": "               ", "password": "saba1234"}`,
			wantCode:  http.StatusUnprocessableEntity,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must be provided", result.Error["username"])
			},
		},
		{
			name:     "duplicate username",
			input:    `{"username": "saba", "password": "saba1234"}`,
			wantCode: http.StatusUnprocessableEntity,
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(data.ErrDuplicateUsersUsername)
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "must be unique", result.Error["username"])
			},
		},
		{
			name:     "success",
			input:    `{"username": "saba", "password": "saba1234"}`,
			wantCode: http.StatusCreated,
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().Insert(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, user *data.User) error {
						user.ID = 10
						return nil
					})
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(10), result.User.ID)
				assert.Equal(t, "saba", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			if tt.setupMock != nil {
				tt.setupMock(app.models.Users.(*mocks.MockUserRepository))
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

func TestShowUserHandler(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantCode  int
		setupMock func(repo *mocks.MockUserRepository)
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:      "negative id",
			id:        "-5",
			wantCode:  http.StatusNotFound,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:      "decimal id",
			id:        "2.3",
			wantCode:  http.StatusNotFound,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:      "character id",
			id:        "saba",
			wantCode:  http.StatusNotFound,
			setupMock: nil,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:     "valid id, not found",
			id:       "1",
			wantCode: http.StatusNotFound,
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(data.User{}, data.ErrNoRecord)
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:     "success",
			id:       "1",
			wantCode: http.StatusOK,
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().GetByID(gomock.Any(), int64(1)).Return(data.User{
					ID:       1,
					Username: "saba",
				}, nil)
			},
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.User.ID)
				assert.Equal(t, "saba", result.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApplication(t)
			if tt.setupMock != nil {
				tt.setupMock(app.models.Users.(*mocks.MockUserRepository))
			}

			req := httptest.NewRequest(http.MethodGet, "/v1/users/"+tt.id, nil)
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey,
				httprouter.Params{{Key: "id", Value: tt.id}}))

			rr := httptest.NewRecorder()

			app.showUserHandler(rr, req)

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
