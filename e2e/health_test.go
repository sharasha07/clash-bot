package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "status ok",
			method:   http.MethodGet,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Status string `json:"status"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "available", result.Status)
			},
		},
		{
			name:     "invalid method",
			method:   http.MethodPost,
			wantCode: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "POST method is not allowed for endpoint: /health", result.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, apiURL+"/health", nil)
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
