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

func TestShowMessagesHandler(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, id, userID int64) (data.Chat, error) {
			if id == 2 {
				return data.Chat{}, data.ErrNoRecord
			}

			return data.Chat{ID: 1, UserID: 1, Name: "chat"}, nil
		}).Times(6)

	app.models.Messages.(*mocks.MockMessageRepository).EXPECT().
		GetAll(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]data.Message{{ID: 1, ChatID: 1, Role: data.RoleUser, Content: "hello"}}, nil).Times(1)

	tests := []struct {
		name      string
		userID    int64
		id        string
		query     string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "anonymous user",
			userID:   0,
			id:       "1",
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
			name:     "chat does not exist",
			userID:   2,
			id:       "2",
			query:    "",
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
			name:     "page is not an integer",
			userID:   1,
			id:       "1",
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
			name:     "page below range",
			userID:   1,
			id:       "1",
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
			name:     "page size above range",
			userID:   1,
			id:       "1",
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
			id:       "1",
			query:    "?sort=content",
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
			id:       "1",
			query:    "?page=2&page_size=5&sort=-id",
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Messages []data.Message `json:"messages"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Messages))
				assert.Equal(t, int64(1), result.Messages[0].ID)
				assert.Equal(t, data.RoleUser, result.Messages[0].Role)
				assert.Equal(t, "hello", result.Messages[0].Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/chats/%s/messages%s", tt.id, tt.query), nil)

			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.showMessagesHandler(rr, req)

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

func TestCreateMessageHandler(t *testing.T) {
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

	app.models.Messages.(*mocks.MockMessageRepository).EXPECT().
		Insert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	app.models.Chats.(*mocks.MockChatRepository).EXPECT().
		Touch(gomock.Any(), gomock.Any(), gomock.Any()).Return(data.Chat{}, nil).Times(1)

	app.models.Messages.(*mocks.MockMessageRepository).EXPECT().
		GetAll(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]data.Message{}, nil).Times(1)

	app.llmClient.(*mocks.MockLLM).EXPECT().
		GenerateReply(gomock.Any(), gomock.Any(), gomock.Any()).Return("hello there", nil).Times(1)

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
			input:    `{"content": "hello"}`,
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
			input:    `{"content": "hello"}`,
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
			name:     "whitespace content",
			userID:   1,
			id:       "1",
			input:    `{"content": "    "}`,
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must not be empty", result.Error["content"])
			},
		},
		{
			name:     "big content",
			userID:   1,
			id:       "1",
			input:    fmt.Sprintf(`{"content": "%s"}`, strings.Repeat("a", 4001)),
			wantCode: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error map[string]string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, 1, len(result.Error))
				assert.Equal(t, "must be a maximum of 4000 characters", result.Error["content"])
			},
		},
		{
			name:     "successful input",
			userID:   1,
			id:       "1",
			input:    `{"content": "  hello  "}`,
			wantCode: http.StatusCreated,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Message data.Message  `json:"message"`
					Reply   *data.Message `json:"reply"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, data.RoleUser, result.Message.Role)
				assert.Equal(t, "hello", result.Message.Content)

				require.NotNil(t, result.Reply)
				assert.Equal(t, data.RoleAssistant, result.Reply.Role)
				assert.Equal(t, "hello there", result.Reply.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/v1/chats/%s/messages", tt.id), strings.NewReader(tt.input))

			params := httprouter.Params{{Key: "id", Value: tt.id}}
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))

			if tt.userID != 0 {
				req = contextSetUser(req, &data.User{ID: tt.userID, Username: "luka"})
			} else {
				req = contextSetUser(req, data.AnonymousUser)
			}

			app.createMessageHandler(rr, req)

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
