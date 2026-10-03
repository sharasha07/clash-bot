package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharasha07/clash-bot/internal/data"
	"github.com/sharasha07/clash-bot/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestShowChatsHandler(t *testing.T) {
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
