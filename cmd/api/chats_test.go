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

	"github.com/julienschmidt/httprouter"
	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestShowChatsHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		GetAll(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]data.Chat{{ID: 1, UserID: 1, Name: "chat"}}, nil).Times(1)

	tests := []struct {
		name      string
		userID    int64
		query     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			query:    "",
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "page is not an integer",
			userID:   1,
			query:    "?page=abc",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be an integer value", result.Error["page"])
			},
		},
		{
			name:     "page size is not an integer",
			userID:   1,
			query:    "?page_size=abc",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be an integer value", result.Error["page_size"])
			},
		},
		{
			name:     "page below range",
			userID:   1,
			query:    "?page=0",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be greater than zero", result.Error["page"])
			},
		},
		{
			name:     "page above range",
			userID:   1,
			query:    "?page=101",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be a maximum of 100", result.Error["page"])
			},
		},
		{
			name:     "page size above range",
			userID:   1,
			query:    "?page_size=11",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be a maximum of 10", result.Error["page_size"])
			},
		},
		{
			name:     "unsafe sort",
			userID:   1,
			query:    "?sort=name",
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "invalid value", result.Error["sort"])
			},
		},
		{
			name:     "successful input",
			userID:   1,
			query:    "?name=general&page=2&page_size=5&sort=-updated_at",
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Chats []data.Chat `json:"chats"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Chats))
				assert.Equal(t, int64(1), result.Chats[0].ID)
				assert.Equal(t, "chat", result.Chats[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/v1/chats"+tt.query, nil)

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.showChatsHandler(rr, req)

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

func TestCreateChatHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Insert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	tests := []struct {
		name      string
		userID    int64
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			input:    `{"name": "chat"}`,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "whitespace name",
			userID:   1,
			input:    `{"name": "    "}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["name"])
			},
		},
		{
			name:     "big name",
			userID:   1,
			input:    `{"name": "shabashabashaba"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be a maximum of 10", result.Error["name"])
			},
		},
		{
			name:     "successful input",
			userID:   1,
			input:    `{"name": "  chat  "}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Chat data.Chat `json:"chat"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.Chat.UserID)
				assert.Equal(t, "chat", result.Chat.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/chats", strings.NewReader(tt.input))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.createChatHandler(rr, req)

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

func TestShowChatHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id, userID int64) (data.Chat, error) {
			if id == 2 {
				return data.Chat{}, data.ErrNoRecord
			}

			return data.Chat{ID: 1, UserID: 1, Name: "chat"}, nil
		}).Times(2)

	tests := []struct {
		name      string
		userID    int64
		id        string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			id:       "1",
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "chat does not exist",
			userID:   2,
			id:       "2",
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:     "successful input",
			userID:   1,
			id:       "1",
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Chat data.Chat `json:"chat"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.Chat.ID)
				assert.Equal(t, "chat", result.Chat.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/chats/%s", tt.id), nil)

			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.showChatHandler(rr, req)

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

func TestUpdateChatHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id, userID int64) (data.Chat, error) {
			if id == 2 {
				return data.Chat{}, data.ErrNoRecord
			}

			return data.Chat{ID: 1, UserID: 1, Name: "general"}, nil
		}).Times(4)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Update(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	tests := []struct {
		name      string
		userID    int64
		id        string
		input     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			id:       "1",
			input:    `{"name": "chats"}`,
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "chat does not exist",
			userID:   2,
			id:       "2",
			input:    `{"name": "chats"}`,
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:     "whitespace name",
			userID:   1,
			id:       "1",
			input:    `{"name": "    "}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["name"])
			},
		},
		{
			name:     "big name",
			userID:   1,
			id:       "1",
			input:    `{"name": "shabashabashaba"}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be a maximum of 10", result.Error["name"])
			},
		},
		{
			name:     "successful input",
			userID:   1,
			id:       "1",
			input:    `{"name": "  chats  "}`,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Chat data.Chat `json:"chat"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, int64(1), result.Chat.ID)
				assert.Equal(t, "chats", result.Chat.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v1/chats/%s", tt.id), strings.NewReader(tt.input))

			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.updateChatHandler(rr, req)

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

func TestDeleteChatHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id, userID int64) error {
			if id == 2 {
				return data.ErrNoRecord
			}

			return nil
		}).Times(2)

	tests := []struct {
		name      string
		userID    int64
		id        string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			id:       "1",
			wantCode: http.StatusUnauthorized,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "authentication required to access this endpoint", result.Error)
			},
		},
		{
			name:     "chat does not exist",
			userID:   2,
			id:       "2",
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "resource not found", result.Error)
			},
		},
		{
			name:      "successful input",
			userID:    1,
			id:        "1",
			wantCode:  http.StatusNoContent,
			checkBody: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/chats/%s", tt.id), nil)

			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.deleteChatHandler(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			if tt.checkBody != nil {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
				tt.checkBody(t, resp.Body)
			}
		})
	}
}
