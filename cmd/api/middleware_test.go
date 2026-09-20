package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharasha07/clash-bot/internal/assert"
)

func TestRecoverPanic(t *testing.T) {
	tests := []struct {
		name       string
		stub       http.HandlerFunc
		wantCode   int
		wantHeader http.Header
	}{
		{
			name: "No panic",
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			wantCode:   http.StatusOK,
			wantHeader: nil,
		},
		{
			name: "Panic",
			stub: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("Panic attack!")
			}),
			wantCode:   http.StatusInternalServerError,
			wantHeader: http.Header{"Connection": []string{"close"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rr := httptest.NewRecorder()

			app := newTestApplication()
			app.recoverPanic(tt.stub).ServeHTTP(rr, req)

			resp := rr.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)

			assert.EqualHeaders(t, tt.wantHeader, resp.Header)

		})
	}
}
