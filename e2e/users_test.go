package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

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
					User data.User `json:"user"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "giorgi", result.User.Username)
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

			resp, err := http.DefaultClient.Do(req)
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
