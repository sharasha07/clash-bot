package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sharasha07/clash-bot/internal/assert"
)

func TestMetrics(t *testing.T) {
	totalRequestsReceived.Set(0)
	totalResponsesSent.Set(0)
	totalProcessingTimeMicroseconds.Set(0)
	totalResponsesSentByStatus.Init()

	stub1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	stub2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	app := newTestApplication()

	serve := func(stub http.HandlerFunc) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()

		app.metrics(stub).ServeHTTP(rr, req)
	}

	serve(stub1)
	serve(stub1)
	serve(stub2)

	assert.Equal(t, "3", totalRequestsReceived.String())
	assert.Equal(t, "3", totalResponsesSent.String())
	assert.Equal(t, "2", totalResponsesSentByStatus.Get("200").String())
	assert.Equal(t, "1", totalResponsesSentByStatus.Get("404").String())
	assert.Equal(t, true, totalProcessingTimeMicroseconds.Value() > 0)
}

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
