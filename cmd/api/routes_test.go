package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutes(t *testing.T) {
	t.Parallel()

	app := newTestApplication(t)

	tests := []struct {
		name      string
		method    string
		path      string
		wantCode  int
		checkBody func(t *testing.T, body io.Reader)
	}{
		{
			name:     "method not allowed",
			method:   http.MethodDelete,
			path:     "/health",
			wantCode: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Error string `json:"error"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "DELETE method is not allowed for path: /health", result.Error)
			},
		},
		{
			name:     "path not registered",
			method:   http.MethodGet,
			path:     "/healthsz",
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
			name:     "health check",
			method:   http.MethodGet,
			path:     "/health",
			wantCode: http.StatusOK,
			checkBody: func(t *testing.T, body io.Reader) {
				t.Helper()

				var result struct {
					Status string `json:"status"`
				}

				err := json.NewDecoder(body).Decode(&result)
				require.NoError(t, err)

				assert.Equal(t, "available", result.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			app.routes().ServeHTTP(rr, req)

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
