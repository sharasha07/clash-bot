package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		path      string
		wantCode  int
		checkBody func(t *testing.T, resp *http.Response)
	}{
		{
			name:      "Success request on /health",
			method:    http.MethodGet,
			path:      "/health",
			wantCode:  http.StatusOK,
			checkBody: nil,
		},
		{
			name:     "Invalid method on /health",
			method:   http.MethodPost,
			path:     "/health",
			wantCode: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

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
			name:     "Invalid path request",
			method:   http.MethodGet,
			path:     "/healthinio",
			wantCode: http.StatusNotFound,
			checkBody: func(t *testing.T, resp *http.Response) {
				t.Helper()

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
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			app := newTestApplication(t)
			app.routes().ServeHTTP(rr, req)

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
