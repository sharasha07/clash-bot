package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		url       string
		body      io.Reader
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:     "Success",
			method:   http.MethodGet,
			url:      apiURL + "/health",
			body:     nil,
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Status string `json:"status"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "available", result.Status)
			},
		},
		{
			name:     "Invalid Method",
			method:   http.MethodPost,
			url:      apiURL + "/health",
			body:     nil,
			wantCode: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "POST method is not allowed for endpoint: /health", result.Error)
			},
		},
		{
			name:     "Invalid Path",
			method:   http.MethodGet,
			url:      apiURL + "/thealth",
			body:     nil,
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, resp *http.Response) {
				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(resp.Body).Decode(&result)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, "resource not found", result.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			if tt.checkBody != nil {
				tt.checkBody(t, resp)
			}
		})
	}
}
